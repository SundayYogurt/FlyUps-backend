package services

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type DisbursementService interface {
	CreateForMilestone(milestoneID uint) (*domain.Disbursement, error)
	ListAll() ([]dto.DisbursementItem, error)
	ListPending() ([]dto.DisbursementItem, error)
	Confirm(disbursementID uint, adminID uint, req dto.ConfirmDisbursementRequest) (*domain.Disbursement, error)
	ListMyPayouts(pioneerID uint) ([]dto.PioneerPayoutItem, error)
}

type disbursementService struct {
	disbursementRepo repository.DisbursementRepository
	projectRepo      repository.ProjectRepository
	userRepo         repository.UserRepository
	notifSvc         NotificationService
}

func NewDisbursementService(
	disbursementRepo repository.DisbursementRepository,
	projectRepo repository.ProjectRepository,
	userRepo repository.UserRepository,
	notifSvc NotificationService,
) DisbursementService {
	return &disbursementService{disbursementRepo, projectRepo, userRepo, notifSvc}
}

// CreateForMilestone สร้างรายการเบิกจ่ายเงิน (disbursement) แบบ pending ให้ milestone ที่ผ่านการโหวตแล้ว
// คำนวณยอดจาก current_funding ของโปรเจกต์ x percent_release ของ milestone; เรียกซ้ำได้อย่างปลอดภัยถ้ามีอยู่แล้ว
func (s *disbursementService) CreateForMilestone(milestoneID uint) (*domain.Disbursement, error) {
	existing, err := s.disbursementRepo.FindByMilestoneID(milestoneID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	milestone, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil {
		return nil, errors.New("milestone not found")
	}

	project, err := s.projectRepo.FindProjectByID(milestone.ProjectID)
	if err != nil {
		return nil, errors.New("project not found")
	}

	amount := project.CurrentFunding * float64(milestone.PercentRelease) / 100.0

	d := &domain.Disbursement{
		MilestoneID:    milestoneID,
		ProjectID:      project.ID,
		PioneerUserID:  project.OwnerUserID,
		Amount:         amount,
		PhaseNo:        milestone.PhaseNo,
		PercentRelease: milestone.PercentRelease,
		Status:         domain.DisbursementPending,
	}

	if err := s.disbursementRepo.Create(d); err != nil {
		return nil, err
	}

	return d, nil
}

// ListAll ดึงรายการเบิกจ่ายเงินทั้งหมด พร้อมข้อมูลโปรเจกต์/pioneer/บัญชีธนาคาร
func (s *disbursementService) ListAll() ([]dto.DisbursementItem, error) {
	list, err := s.disbursementRepo.ListAll()
	if err != nil {
		return nil, errors.New("internal server error")
	}
	return s.toItems(list), nil
}

// ListPending ดึงรายการเบิกจ่ายเงินที่ยังรอดำเนินการ (pending) เท่านั้น
func (s *disbursementService) ListPending() ([]dto.DisbursementItem, error) {
	list, err := s.disbursementRepo.ListByStatus(domain.DisbursementPending)
	if err != nil {
		return nil, errors.New("internal server error")
	}
	return s.toItems(list), nil
}

// Confirm ยืนยันว่าแอดมินได้โอนเงินเบิกจ่ายให้ pioneer เรียบร้อยแล้ว บันทึกเลขอ้างอิงการโอน แล้วแจ้งเตือน pioneer
func (s *disbursementService) Confirm(disbursementID uint, adminID uint, req dto.ConfirmDisbursementRequest) (*domain.Disbursement, error) {
	d, err := s.disbursementRepo.FindByID(disbursementID)
	if err != nil {
		return nil, errors.New("disbursement not found")
	}

	if d.Status == domain.DisbursementConfirmed {
		return nil, errors.New("disbursement already confirmed")
	}

	now := time.Now().UTC()
	d.Status = domain.DisbursementConfirmed
	d.TransferRef = req.TransferRef
	d.AdminNote = req.Note
	d.ConfirmedBy = &adminID
	d.ConfirmedAt = &now

	if err := s.disbursementRepo.Update(d); err != nil {
		return nil, errors.New("failed to confirm disbursement")
	}

	if s.notifSvc != nil {
		projectTitle := ""
		if project, err := s.projectRepo.FindProjectByID(d.ProjectID); err == nil && project != nil {
			projectTitle = project.Title
		}
		relatedID := d.ID
		relatedType := "disbursement"
		title := "ได้รับเงินจาก Milestone แล้ว"
		body := fmt.Sprintf("โอนเงิน Phase %d โปรเจกต์ %s จำนวน ฿%.2f เรียบร้อยแล้ว (ref: %s)",
			d.PhaseNo, projectTitle, d.Amount, d.TransferRef)
		_ = s.notifSvc.CreateAndPush(d.PioneerUserID, domain.NotifProfit, title, body, &relatedID, &relatedType)
	}

	return d, nil
}

// ListMyPayouts ดึงรายการเบิกจ่ายเงินทั้งหมดของ pioneer พร้อมระบุว่าโปรเจกต์ไหนผ่านครบทุก phase แล้วบ้าง
func (s *disbursementService) ListMyPayouts(pioneerID uint) ([]dto.PioneerPayoutItem, error) {
	list, err := s.disbursementRepo.ListByPioneerID(pioneerID)
	if err != nil {
		return nil, errors.New("internal server error")
	}

	// collect unique project IDs to check if all phases are complete
	projectIDs := map[uint]bool{}
	for _, d := range list {
		projectIDs[d.ProjectID] = true
	}
	allComplete := map[uint]bool{}
	for pid := range projectIDs {
		milestones, err := s.projectRepo.FindMilestonesByProjectID(pid)
		if err != nil {
			continue
		}
		done := len(milestones) > 0
		for _, m := range milestones {
			if m.Status != domain.MilestonePaid {
				done = false
				break
			}
		}
		allComplete[pid] = done
	}

	items := make([]dto.PioneerPayoutItem, 0, len(list))
	for _, d := range list {
		item := dto.PioneerPayoutItem{
			ID:                d.ID,
			MilestoneID:       d.MilestoneID,
			ProjectID:         d.ProjectID,
			PhaseNo:           d.PhaseNo,
			PercentRelease:    d.PercentRelease,
			Amount:            d.Amount,
			Status:            string(d.Status),
			TransferRef:       d.TransferRef,
			AdminNote:         d.AdminNote,
			CreatedAt:         d.CreatedAt,
			ConfirmedAt:       d.ConfirmedAt,
			AllPhasesComplete: allComplete[d.ProjectID],
		}
		if project, err := s.projectRepo.FindProjectByID(d.ProjectID); err == nil && project != nil {
			item.ProjectTitle = project.Title
		}
		items = append(items, item)
	}
	return items, nil
}

// toItems แปลง domain.Disbursement เป็น dto.DisbursementItem พร้อมเติมข้อมูลโปรเจกต์/pioneer/บัญชีธนาคารเริ่มต้น
func (s *disbursementService) toItems(list []domain.Disbursement) []dto.DisbursementItem {
	items := make([]dto.DisbursementItem, 0, len(list))
	for _, d := range list {
		item := dto.DisbursementItem{
			ID:             d.ID,
			MilestoneID:    d.MilestoneID,
			ProjectID:      d.ProjectID,
			PioneerUserID:  d.PioneerUserID,
			PhaseNo:        d.PhaseNo,
			PercentRelease: d.PercentRelease,
			Amount:         d.Amount,
			Status:         string(d.Status),
			TransferRef:    d.TransferRef,
			AdminNote:      d.AdminNote,
			CreatedAt:      d.CreatedAt,
			ConfirmedAt:    d.ConfirmedAt,
		}

		if project, err := s.projectRepo.FindProjectByID(d.ProjectID); err == nil && project != nil {
			item.ProjectTitle = project.Title
		}

		if pioneer, err := s.userRepo.FindUserById(d.PioneerUserID); err == nil && pioneer != nil {
			item.PioneerName = pioneer.FirstName + " " + pioneer.LastName
			item.PioneerEmail = pioneer.Email
		}

		if banks, err := s.userRepo.FindBankByUserId(d.PioneerUserID); err == nil && len(banks) > 0 {
			var b domain.BankAccount
			found := false
			for _, bank := range banks {
				if bank.IsDefault {
					b = bank
					found = true
					break
				}
			}
			if !found {
				b = banks[0]
			}
			item.BankAccount = &dto.DisbursementBankAccount{
				BankName:      b.BankName,
				AccountName:   b.AccountName,
				AccountNumber: b.AccountNumber,
			}
		}

		items = append(items, item)
	}
	return items
}

package service

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"time"

	"gorm.io/gorm"
)

type DisbursementService interface {
	CreateForMilestone(milestoneID uint) (*domain.Disbursement, error)
	ListAll() ([]dto.DisbursementItem, error)
	ListPending() ([]dto.DisbursementItem, error)
	Confirm(disbursementID uint, adminID uint, req dto.ConfirmDisbursementRequest) (*domain.Disbursement, error)
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

func (s *disbursementService) ListAll() ([]dto.DisbursementItem, error) {
	list, err := s.disbursementRepo.ListAll()
	if err != nil {
		return nil, errors.New("internal server error")
	}
	return s.toItems(list), nil
}

func (s *disbursementService) ListPending() ([]dto.DisbursementItem, error) {
	list, err := s.disbursementRepo.ListByStatus(domain.DisbursementPending)
	if err != nil {
		return nil, errors.New("internal server error")
	}
	return s.toItems(list), nil
}

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

	return d, nil
}

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
			b := banks[0]
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

package service

import (
	"errors"
	"fmt"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"math"
	"time"
)

const errInternalServer = "internal server error"

type ProfitPoolService interface {
	Create(adminID uint, req dto.CreateProfitPoolRequest) (*dto.ProfitPoolDetail, error)
	List() ([]dto.ProfitPoolListItem, error)
	GetDetail(poolID uint) (*dto.ProfitPoolDetail, error)
	ConfirmPayout(poolID uint, payoutID uint, adminID uint, req dto.ConfirmInvestorPayoutRequest) error
	PioneerSubmit(pioneerID uint, projectID uint, req dto.PioneerSubmitProfitRequest) (*dto.ProfitPoolDetail, error)
	GetPioneerPools(pioneerID uint) ([]dto.ProfitPoolListItem, error)
	GetMyProfitPayouts(userID uint) ([]dto.MyProfitPayoutItem, error)
}

type profitPoolService struct {
	repo        repository.ProfitPoolRepository
	projectRepo repository.ProjectRepository
	investRepo  repository.InvestmentRepository
	userRepo    repository.UserRepository
	notifSvc    NotificationService
}

func NewProfitPoolService(
	repo repository.ProfitPoolRepository,
	projectRepo repository.ProjectRepository,
	investRepo repository.InvestmentRepository,
	userRepo repository.UserRepository,
	notifSvc NotificationService,
) ProfitPoolService {
	return &profitPoolService{repo, projectRepo, investRepo, userRepo, notifSvc}
}

func (s *profitPoolService) Create(adminID uint, req dto.CreateProfitPoolRequest) (*dto.ProfitPoolDetail, error) {
	project, err := s.projectRepo.FindProjectByID(req.ProjectID)
	if err != nil {
		return nil, errors.New("project not found")
	}

	investors, err := s.investRepo.ListInvestorsByProjectID(req.ProjectID)
	if err != nil || len(investors) == 0 {
		return nil, errors.New("no investors found for this project")
	}

	var totalPrincipal float64
	for _, inv := range investors {
		totalPrincipal += inv.PrincipalAmount
	}
	if totalPrincipal == 0 {
		return nil, errors.New("total principal is zero")
	}

	pool := &domain.ProfitPool{
		ProjectID:     req.ProjectID,
		PioneerUserID: project.OwnerUserID,
		TotalAmount:   req.TotalAmount,
		TransferRef:   req.TransferRef,
		Status:        domain.ProfitPoolPending,
		AdminNote:     req.AdminNote,
		QuarterNo:     req.QuarterNo,
	}
	if err := s.repo.Create(pool); err != nil {
		return nil, errors.New("failed to create profit pool")
	}

	for _, inv := range investors {
		sharePct := math.Round((inv.PrincipalAmount/totalPrincipal)*10000) / 100
		amount := math.Round((inv.PrincipalAmount/totalPrincipal)*req.TotalAmount*100) / 100
		payout := &domain.InvestorProfitPayout{
			ProfitPoolID:  pool.ID,
			ProjectID:     req.ProjectID,
			BoosterUserID: inv.UserID,
			Amount:        amount,
			SharePct:      sharePct,
			Status:        domain.InvestorPayoutPending,
		}
		_ = s.repo.CreatePayout(payout)
	}

	return s.GetDetail(pool.ID)
}

func (s *profitPoolService) List() ([]dto.ProfitPoolListItem, error) {
	pools, err := s.repo.ListAll()
	if err != nil {
		return nil, errors.New(errInternalServer)
	}

	items := make([]dto.ProfitPoolListItem, 0, len(pools))
	for _, p := range pools {
		item := dto.ProfitPoolListItem{
			ID:          p.ID,
			ProjectID:   p.ProjectID,
			TotalAmount: p.TotalAmount,
			Status:      string(p.Status),
			QuarterNo:   p.QuarterNo,
			CreatedAt:   p.CreatedAt,
		}
		if project, err := s.projectRepo.FindProjectByID(p.ProjectID); err == nil {
			item.ProjectTitle = project.Title
		}
		if pioneer, err := s.userRepo.FindUserById(p.PioneerUserID); err == nil {
			item.PioneerName = pioneer.FirstName + " " + pioneer.LastName
		}
		payouts, _ := s.repo.ListPayoutsByPoolID(p.ID)
		item.InvestorCount = len(payouts)
		confirmed := 0
		for _, pay := range payouts {
			if pay.Status == domain.InvestorPayoutConfirmed {
				confirmed++
			}
		}
		item.ConfirmedCount = confirmed
		items = append(items, item)
	}
	return items, nil
}

func (s *profitPoolService) GetDetail(poolID uint) (*dto.ProfitPoolDetail, error) {
	pool, err := s.repo.FindByID(poolID)
	if err != nil {
		return nil, errors.New("profit pool not found")
	}

	detail := &dto.ProfitPoolDetail{
		ID:            pool.ID,
		ProjectID:     pool.ProjectID,
		PioneerUserID: pool.PioneerUserID,
		TotalAmount:   pool.TotalAmount,
		TransferRef:   pool.TransferRef,
		Status:        string(pool.Status),
		AdminNote:     pool.AdminNote,
		QuarterNo:     pool.QuarterNo,
		CreatedAt:     pool.CreatedAt,
	}
	if project, err := s.projectRepo.FindProjectByID(pool.ProjectID); err == nil {
		detail.ProjectTitle = project.Title
	}
	if pioneer, err := s.userRepo.FindUserById(pool.PioneerUserID); err == nil {
		detail.PioneerName = pioneer.FirstName + " " + pioneer.LastName
	}

	// pre-fetch investor principals once
	principalMap := map[uint]float64{}
	if investors, err := s.investRepo.ListInvestorsByProjectID(pool.ProjectID); err == nil {
		for _, inv := range investors {
			principalMap[inv.UserID] = inv.PrincipalAmount
		}
	}

	payouts, _ := s.repo.ListPayoutsByPoolID(poolID)
	detail.Payouts = make([]dto.InvestorPayoutDetail, 0, len(payouts))
	for _, p := range payouts {
		pd := dto.InvestorPayoutDetail{
			ID:              p.ID,
			BoosterUserID:   p.BoosterUserID,
			Amount:          p.Amount,
			SharePct:        p.SharePct,
			Status:          string(p.Status),
			TransferRef:     p.TransferRef,
			AdminNote:       p.AdminNote,
			ConfirmedAt:     p.ConfirmedAt,
			PrincipalAmount: principalMap[p.BoosterUserID],
		}
		if user, err := s.userRepo.FindUserById(p.BoosterUserID); err == nil {
			pd.FirstName = user.FirstName
			pd.LastName = user.LastName
			pd.Email = user.Email
		}
		if banks, err := s.userRepo.FindBankByUserId(p.BoosterUserID); err == nil && len(banks) > 0 {
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
			pd.BankAccount = &dto.DisbursementBankAccount{
				BankName:      b.BankName,
				AccountName:   b.AccountName,
				AccountNumber: b.AccountNumber,
			}
		}
		detail.Payouts = append(detail.Payouts, pd)
	}

	return detail, nil
}

func (s *profitPoolService) ConfirmPayout(poolID uint, payoutID uint, adminID uint, req dto.ConfirmInvestorPayoutRequest) error {
	pool, err := s.repo.FindByID(poolID)
	if err != nil {
		return errors.New("profit pool not found")
	}

	payout, err := s.repo.FindPayoutByID(payoutID)
	if err != nil {
		return errors.New("payout not found")
	}
	if payout.ProfitPoolID != poolID {
		return errors.New("payout does not belong to this pool")
	}
	if payout.Status == domain.InvestorPayoutConfirmed {
		return errors.New("payout already confirmed")
	}

	now := time.Now().UTC()
	payout.Status = domain.InvestorPayoutConfirmed
	payout.TransferRef = req.TransferRef
	payout.AdminNote = req.Note
	payout.ConfirmedAt = &now
	payout.ConfirmedBy = &adminID
	if err := s.repo.UpdatePayout(payout); err != nil {
		return errors.New("failed to confirm payout")
	}

	if s.notifSvc != nil {
		projectTitle := ""
		if project, err := s.projectRepo.FindProjectByID(pool.ProjectID); err == nil {
			projectTitle = project.Title
		}
		relatedID := payout.ID
		relatedType := "investor_payout"
		title := "ได้รับกำไรจากโปรเจกต์"
		body := fmt.Sprintf("โอนกำไรจากโปรเจกต์ %s จำนวน ฿%.2f เรียบร้อยแล้ว (%.2f%% ของทุนรวม)",
			projectTitle, payout.Amount, payout.SharePct)
		_ = s.notifSvc.CreateAndPush(payout.BoosterUserID, domain.NotifProfit, title, body, &relatedID, &relatedType)
	}

	// if all payouts confirmed → mark pool completed
	allPayouts, _ := s.repo.ListPayoutsByPoolID(poolID)
	allDone := len(allPayouts) > 0
	for _, p := range allPayouts {
		if p.Status != domain.InvestorPayoutConfirmed {
			allDone = false
			break
		}
	}
	if allDone {
		pool.Status = domain.ProfitPoolCompleted
		_ = s.repo.Update(pool)
	}

	return nil
}

func (s *profitPoolService) validatePioneerProjectForProfit(pioneerID, projectID uint, quarterNo int) (*domain.Project, error) {
	project, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return nil, errors.New("project not found")
	}
	if project.OwnerUserID != pioneerID {
		return nil, errors.New("permission denied")
	}
	if project.State != domain.StateExecuting && project.State != domain.StateClosed {
		return nil, errors.New("project must be in executing or closed state")
	}
	// ตรวจว่า milestone ครบ 4 phase และทุก phase เป็น paid ก่อนจ่ายปันผล
	milestones, err := s.projectRepo.FindMilestonesByProjectID(projectID)
	if err != nil || len(milestones) == 0 {
		return nil, errors.New("ยังไม่มีข้อมูล Milestone")
	}
	for _, m := range milestones {
		if m.Status != domain.MilestonePaid {
			return nil, errors.New("ต้องผ่านครบทุก Phase Milestone ก่อนจึงจะจ่ายปันผลได้")
		}
	}
	exists, err := s.repo.ExistsByProjectAndQuarter(projectID, quarterNo)
	if err != nil {
		return nil, errors.New(errInternalServer)
	}
	if exists {
		return nil, fmt.Errorf("ไตรมาสที่ %d ส่งไปแล้ว", quarterNo)
	}
	existing, _ := s.repo.ListByPioneerUserID(pioneerID)
	count := 0
	for _, p := range existing {
		if p.ProjectID == projectID {
			count++
		}
	}
	if count >= 4 {
		return nil, errors.New("ส่งครบ 4 ไตรมาสแล้ว")
	}
	return project, nil
}

func (s *profitPoolService) notifyAdminsNewProfit(pool *domain.ProfitPool, projectTitle string) {
	if s.notifSvc == nil {
		return
	}
	adminIDs, err := s.userRepo.FindAdminUserIDs()
	if err != nil {
		return
	}
	relatedID := pool.ID
	relatedType := "profit_pool"
	body := fmt.Sprintf("Pioneer โอนกำไร Q%d โปรเจกต์ %s จำนวน ฿%.2f รอการแจกจ่ายให้นักลงทุน", pool.QuarterNo, projectTitle, pool.TotalAmount)
	for _, aid := range adminIDs {
		_ = s.notifSvc.CreateAndPush(aid, domain.NotifProfit, "Pioneer โอนกำไรเข้าระบบ", body, &relatedID, &relatedType)
	}
}

func (s *profitPoolService) PioneerSubmit(pioneerID uint, projectID uint, req dto.PioneerSubmitProfitRequest) (*dto.ProfitPoolDetail, error) {
	project, err := s.validatePioneerProjectForProfit(pioneerID, projectID, req.QuarterNo)
	if err != nil {
		return nil, err
	}

	investors, err := s.investRepo.ListInvestorsByProjectID(projectID)
	if err != nil || len(investors) == 0 {
		return nil, errors.New("no investors found for this project")
	}
	var totalPrincipal float64
	for _, inv := range investors {
		totalPrincipal += inv.PrincipalAmount
	}

	pool := &domain.ProfitPool{
		ProjectID:     projectID,
		PioneerUserID: pioneerID,
		TotalAmount:   req.TotalAmount,
		TransferRef:   req.TransferRef,
		SlipImage:     req.SlipImage,
		Status:        domain.ProfitPoolPending,
		QuarterNo:     req.QuarterNo,
	}
	if err := s.repo.Create(pool); err != nil {
		return nil, errors.New("failed to create profit pool")
	}

	for _, inv := range investors {
		sharePct := math.Round((inv.PrincipalAmount/totalPrincipal)*10000) / 100
		amount := math.Round((inv.PrincipalAmount/totalPrincipal)*req.TotalAmount*100) / 100
		_ = s.repo.CreatePayout(&domain.InvestorProfitPayout{
			ProfitPoolID:  pool.ID,
			ProjectID:     projectID,
			BoosterUserID: inv.UserID,
			Amount:        amount,
			SharePct:      sharePct,
			Status:        domain.InvestorPayoutPending,
		})
	}

	s.notifyAdminsNewProfit(pool, project.Title)

	return s.GetDetail(pool.ID)
}

func (s *profitPoolService) GetPioneerPools(pioneerID uint) ([]dto.ProfitPoolListItem, error) {
	pools, err := s.repo.ListByPioneerUserID(pioneerID)
	if err != nil {
		return nil, errors.New(errInternalServer)
	}

	items := make([]dto.ProfitPoolListItem, 0, len(pools))
	for _, p := range pools {
		item := dto.ProfitPoolListItem{
			ID:          p.ID,
			ProjectID:   p.ProjectID,
			TotalAmount: p.TotalAmount,
			Status:      string(p.Status),
			QuarterNo:   p.QuarterNo,
			CreatedAt:   p.CreatedAt,
		}
		if project, err := s.projectRepo.FindProjectByID(p.ProjectID); err == nil {
			item.ProjectTitle = project.Title
		}
		payouts, _ := s.repo.ListPayoutsByPoolID(p.ID)
		item.InvestorCount = len(payouts)
		confirmed := 0
		for _, pay := range payouts {
			if pay.Status == domain.InvestorPayoutConfirmed {
				confirmed++
			}
		}
		item.ConfirmedCount = confirmed
		items = append(items, item)
	}
	return items, nil
}

func (s *profitPoolService) GetMyProfitPayouts(userID uint) ([]dto.MyProfitPayoutItem, error) {
	payouts, err := s.repo.ListPayoutsByBoosterUserID(userID)
	if err != nil {
		return nil, errors.New(errInternalServer)
	}
	items := make([]dto.MyProfitPayoutItem, 0, len(payouts))
	for _, p := range payouts {
		item := dto.MyProfitPayoutItem{
			ID:          p.ID,
			ProjectID:   p.ProjectID,
			Amount:      p.Amount,
			SharePct:    p.SharePct,
			Status:      string(p.Status),
			TransferRef: p.TransferRef,
			ConfirmedAt: p.ConfirmedAt,
			CreatedAt:   p.CreatedAt,
		}
		if pool, err := s.repo.FindByID(p.ProfitPoolID); err == nil {
			item.QuarterNo = pool.QuarterNo
		}
		if project, err := s.projectRepo.FindProjectByID(p.ProjectID); err == nil {
			item.ProjectTitle = project.Title
			item.CoverImage = project.CoverImage
		}
		items = append(items, item)
	}
	return items, nil
}

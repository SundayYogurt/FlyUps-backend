package services

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"fmt"
	"log"
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
	repo          repository.ProfitPoolRepository
	projectRepo   repository.ProjectRepository
	investRepo    repository.InvestmentRepository
	userRepo      repository.UserRepository
	investmentSvc InvestmentService
	notifSvc      NotificationService
}

func NewProfitPoolService(
	repo repository.ProfitPoolRepository,
	projectRepo repository.ProjectRepository,
	investRepo repository.InvestmentRepository,
	userRepo repository.UserRepository,
	investmentSvc InvestmentService,
	notifSvc NotificationService,
) ProfitPoolService {
	return &profitPoolService{repo, projectRepo, investRepo, userRepo, investmentSvc, notifSvc}
}

func (s *profitPoolService) Create(adminID uint, req dto.CreateProfitPoolRequest) (*dto.ProfitPoolDetail, error) {
	project, err := s.projectRepo.FindProjectByID(req.ProjectID)
	if err != nil {
		return nil, errors.New("project not found")
	}
	allPools, err := s.repo.ListAll()
	if err != nil {
		return nil, errors.New(errInternalServer)
	}
	projectPools := make([]domain.ProfitPool, 0)
	for _, existingPool := range allPools {
		if existingPool.ProjectID == req.ProjectID {
			projectPools = append(projectPools, existingPool)
		}
	}
	if err := s.validateProfitEligibility(project, req.QuarterNo, projectPools); err != nil {
		return nil, err
	}

	if err := s.investmentSvc.SyncProjectPrincipalAmounts(req.ProjectID); err != nil {
		log.Printf("[ProfitPool.Create] sync principal amounts error: %v", err)
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

	var totalAllocated float64
	for i, inv := range investors {
		sharePct := math.Round((inv.PrincipalAmount/totalPrincipal)*10000) / 100
		var amount float64
		if i == len(investors)-1 {
			// ให้ remainder แก่ investor คนสุดท้าย กัน rounding loss
			amount = math.Round((req.TotalAmount-totalAllocated)*100) / 100
		} else {
			amount = math.Round((inv.PrincipalAmount/totalPrincipal)*req.TotalAmount*100) / 100
			totalAllocated += amount
		}
		payout := &domain.InvestorProfitPayout{
			ProfitPoolID:  pool.ID,
			ProjectID:     req.ProjectID,
			BoosterUserID: inv.UserID,
			Amount:        amount,
			SharePct:      sharePct,
			Status:        domain.InvestorPayoutPending,
		}
		if err := s.repo.CreatePayout(payout); err != nil {
			return nil, errors.New("failed to create investor payout")
		}
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
		if err := s.repo.Update(pool); err != nil {
			return errors.New("failed to complete profit pool")
		}
		if pool.QuarterNo == 4 {
			s.completeProjectAfterFourQuarters(pool)
		}
	}

	return nil
}

func (s *profitPoolService) validateProfitEligibility(project *domain.Project, quarterNo int, existing []domain.ProfitPool) error {
	if project.State != domain.StateExecuting && project.State != domain.StateClosed {
		return errors.New("project must be in executing or closed state")
	}
	milestones, err := s.projectRepo.FindMilestonesByProjectID(project.ID)
	if err != nil || len(milestones) != 4 {
		return errors.New("โปรเจกต์ต้องมี Milestone ครบ 4 Phase ก่อนจ่ายปันผล")
	}
	for _, m := range milestones {
		if m.Status != domain.MilestonePaid {
			return errors.New("ต้องผ่านครบทุก Phase Milestone ก่อนจึงจะจ่ายปันผลได้")
		}
	}

	quarterMap := make(map[int]domain.ProfitPool, len(existing))
	for _, pool := range existing {
		quarterMap[pool.QuarterNo] = pool
	}
	if _, exists := quarterMap[quarterNo]; exists {
		return fmt.Errorf("ไตรมาสที่ %d ส่งไปแล้ว", quarterNo)
	}
	for q := 1; q < quarterNo; q++ {
		pool, exists := quarterMap[q]
		if !exists {
			return fmt.Errorf("ต้องส่งปันผลไตรมาสที่ %d ก่อน", q)
		}
		if pool.Status != domain.ProfitPoolCompleted {
			return fmt.Errorf("ต้องแจกจ่ายปันผลไตรมาสที่ %d ให้ครบก่อน", q)
		}
	}
	return nil
}

func (s *profitPoolService) completeProjectAfterFourQuarters(finalPool *domain.ProfitPool) {
	pools, err := s.repo.ListByPioneerUserID(finalPool.PioneerUserID)
	if err != nil {
		return
	}
	completed := make(map[int]bool, 4)
	for _, pool := range pools {
		if pool.ProjectID == finalPool.ProjectID && pool.Status == domain.ProfitPoolCompleted {
			completed[pool.QuarterNo] = true
		}
	}
	for quarter := 1; quarter <= 4; quarter++ {
		if !completed[quarter] {
			return
		}
	}

	project, err := s.projectRepo.FindProjectByID(finalPool.ProjectID)
	if err != nil || project == nil {
		return
	}
	project.State = domain.StateClosed
	project.Status = domain.StatusCompleted
	if _, err := s.projectRepo.UpdateProject(project); err != nil {
		log.Printf("[completeProjectAfterFourQuarters] update project %d error: %v", project.ID, err)
		return
	}

	if s.notifSvc != nil {
		relatedID := project.ID
		relatedType := "project"
		_ = s.notifSvc.CreateAndPush(
			project.OwnerUserID,
			domain.NotifProfit,
			"จ่ายปันผลครบ 4 ไตรมาสแล้ว",
			fmt.Sprintf("โปรเจกต์ %s แจกจ่ายปันผลให้นักลงทุนครบทั้ง 4 ไตรมาสแล้ว", project.Title),
			&relatedID,
			&relatedType,
		)
	}
}

func (s *profitPoolService) validatePioneerProjectForProfit(pioneerID, projectID uint, quarterNo int) (*domain.Project, error) {
	project, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return nil, errors.New("project not found")
	}
	if project.OwnerUserID != pioneerID {
		return nil, errors.New("permission denied")
	}
	existing, _ := s.repo.ListByPioneerUserID(pioneerID)
	projectPools := make([]domain.ProfitPool, 0)
	for _, p := range existing {
		if p.ProjectID == projectID {
			projectPools = append(projectPools, p)
		}
	}
	if err := s.validateProfitEligibility(project, quarterNo, projectPools); err != nil {
		return nil, err
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

	if err := s.investmentSvc.SyncProjectPrincipalAmounts(projectID); err != nil {
		log.Printf("[ProfitPool.PioneerSubmit] sync principal amounts error: %v", err)
	}

	investors, err := s.investRepo.ListInvestorsByProjectID(projectID)
	if err != nil || len(investors) == 0 {
		return nil, errors.New("no investors found for this project")
	}
	var totalPrincipal float64
	for _, inv := range investors {
		totalPrincipal += inv.PrincipalAmount
	}
	if totalPrincipal <= 0 {
		return nil, errors.New("total principal is zero")
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

	var totalAllocated float64
	for i, inv := range investors {
		sharePct := math.Round((inv.PrincipalAmount/totalPrincipal)*10000) / 100
		var amount float64
		if i == len(investors)-1 {
			amount = math.Round((req.TotalAmount-totalAllocated)*100) / 100
		} else {
			amount = math.Round((inv.PrincipalAmount/totalPrincipal)*req.TotalAmount*100) / 100
			totalAllocated += amount
		}
		if err := s.repo.CreatePayout(&domain.InvestorProfitPayout{
			ProfitPoolID:  pool.ID,
			ProjectID:     projectID,
			BoosterUserID: inv.UserID,
			Amount:        amount,
			SharePct:      sharePct,
			Status:        domain.InvestorPayoutPending,
		}); err != nil {
			return nil, errors.New("failed to create investor payout")
		}
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
			if project.CoverImage != nil {
				item.CoverImage = *project.CoverImage
			}
		}
		items = append(items, item)
	}
	return items, nil
}

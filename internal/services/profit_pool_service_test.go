package services

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// Mock: ProfitPoolRepository

type mockProfitPoolRepo struct{ mock.Mock }

func (m *mockProfitPoolRepo) Create(pool *domain.ProfitPool) error {
	return m.Called(pool).Error(0)
}

func (m *mockProfitPoolRepo) FindByID(id uint) (*domain.ProfitPool, error) {
	args := m.Called(id)
	if v := args.Get(0); v != nil {
		return v.(*domain.ProfitPool), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockProfitPoolRepo) ListAll() ([]domain.ProfitPool, error) {
	args := m.Called()
	return args.Get(0).([]domain.ProfitPool), args.Error(1)
}

func (m *mockProfitPoolRepo) ListByPioneerUserID(pioneerUserID uint) ([]domain.ProfitPool, error) {
	args := m.Called(pioneerUserID)
	return args.Get(0).([]domain.ProfitPool), args.Error(1)
}

func (m *mockProfitPoolRepo) ExistsByProjectAndQuarter(projectID uint, quarterNo int) (bool, error) {
	args := m.Called(projectID, quarterNo)
	return args.Get(0).(bool), args.Error(1)
}

func (m *mockProfitPoolRepo) Update(pool *domain.ProfitPool) error {
	return m.Called(pool).Error(0)
}

func (m *mockProfitPoolRepo) CreatePayout(p *domain.InvestorProfitPayout) error {
	return m.Called(p).Error(0)
}

func (m *mockProfitPoolRepo) FindPayoutByID(id uint) (*domain.InvestorProfitPayout, error) {
	args := m.Called(id)
	if v := args.Get(0); v != nil {
		return v.(*domain.InvestorProfitPayout), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockProfitPoolRepo) ListPayoutsByPoolID(poolID uint) ([]domain.InvestorProfitPayout, error) {
	args := m.Called(poolID)
	return args.Get(0).([]domain.InvestorProfitPayout), args.Error(1)
}

func (m *mockProfitPoolRepo) ListPayoutsByBoosterUserID(boosterUserID uint) ([]domain.InvestorProfitPayout, error) {
	args := m.Called(boosterUserID)
	return args.Get(0).([]domain.InvestorProfitPayout), args.Error(1)
}

func (m *mockProfitPoolRepo) UpdatePayout(p *domain.InvestorProfitPayout) error {
	return m.Called(p).Error(0)
}

// Mock: InvestmentService (stub, only SyncProjectPrincipalAmounts is used by profitPoolService)

type mockPPInvestmentSvc struct{ mock.Mock }

func (m *mockPPInvestmentSvc) SyncProjectPrincipalAmounts(projectID uint) error {
	return m.Called(projectID).Error(0)
}
func (m *mockPPInvestmentSvc) GetInvestment(boosterUserID, investmentID uint) (*domain.Investment, *domain.Transaction, error) {
	return nil, nil, nil
}
func (m *mockPPInvestmentSvc) GenerateContractHTML(boosterUserID, investmentID uint) ([]byte, error) {
	return nil, nil
}
func (m *mockPPInvestmentSvc) CreateInvestment(boosterUserID uint, boosterEmail string, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error) {
	return nil, nil
}
func (m *mockPPInvestmentSvc) ListUserInvestments(boosterUserID uint) ([]domain.Investment, error) {
	return nil, nil
}
func (m *mockPPInvestmentSvc) HandleStripeWebhook(payload []byte, sigHeader string) error {
	return nil
}
func (m *mockPPInvestmentSvc) RefundInvestment(boosterUserID, investmentID uint, note string) (*dto.RefundResponse, error) {
	return nil, nil
}
func (m *mockPPInvestmentSvc) ApproveRefund(investmentID uint) error           { return nil }
func (m *mockPPInvestmentSvc) ListRefundRequests() ([]dto.RefundRequestItem, error) {
	return nil, nil
}
func (m *mockPPInvestmentSvc) GetProjectInvestors(projectID uint) ([]dto.ProjectInvestorItem, error) {
	return nil, nil
}
func (m *mockPPInvestmentSvc) ListInvestedProjects(boosterUserID uint) ([]dto.InvestedProjectItem, error) {
	return nil, nil
}
func (m *mockPPInvestmentSvc) VoteMilestone(boosterUserID, milestoneID uint, choice domain.MilestoneVoteChoice) (*domain.MilestoneVote, error) {
	return nil, nil
}
func (m *mockPPInvestmentSvc) GetMyVote(boosterUserID, milestoneID uint) (*domain.MilestoneVote, error) {
	return nil, nil
}
func (m *mockPPInvestmentSvc) GetMilestoneVoters(pioneerUserID, milestoneID uint) ([]dto.MilestoneVoterItem, error) {
	return nil, nil
}
func (m *mockPPInvestmentSvc) RefundProjectInvestments(project domain.Project) {}
func (m *mockPPInvestmentSvc) GetCancelPreview(projectID uint) (*dto.CancelPreviewResponse, error) {
	return nil, nil
}
func (m *mockPPInvestmentSvc) FinalizeVotingIfExpired(milestoneID uint) error { return nil }
func (m *mockPPInvestmentSvc) GetTotalFunding() (float64, error)              { return 0, nil }
func (m *mockPPInvestmentSvc) GetUniqueBoostersCount() (int64, error)         { return 0, nil }

// Mock: NotificationService (stub, only CreateAndPush is used by profitPoolService)

type mockPPNotifSvc struct{ mock.Mock }

func (m *mockPPNotifSvc) CreateAndPush(userID uint, notifType domain.NotificationType, title, body string, relatedID *uint, relatedType *string) error {
	return m.Called(userID, notifType, title, body, relatedID, relatedType).Error(0)
}
func (m *mockPPNotifSvc) GetNotifications(userID uint, page, limit int) ([]domain.Notification, int64, error) {
	return nil, 0, nil
}
func (m *mockPPNotifSvc) MarkAsRead(userID, notifID uint) error  { return nil }
func (m *mockPPNotifSvc) MarkAllAsRead(userID uint) error        { return nil }
func (m *mockPPNotifSvc) CountUnread(userID uint) (int64, error) { return 0, nil }
func (m *mockPPNotifSvc) Subscribe(userID uint) chan *domain.Notification {
	return make(chan *domain.Notification, 1)
}
func (m *mockPPNotifSvc) Unsubscribe(userID uint, ch chan *domain.Notification) {}

// helpers

func newProfitPoolSvc(
	repo *mockProfitPoolRepo,
	projRepo *ProjectRepository,
	investRepo *mockInvestmentRepo,
	userRepo *mockUserRepository,
	investSvc InvestmentService,
	notifSvc NotificationService,
) ProfitPoolService {
	return NewProfitPoolService(repo, projRepo, investRepo, userRepo, investSvc, notifSvc)
}

// Create

func TestProfitPoolService_Create_ProjectNotFound(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	projRepo.On("FindProjectByID", uint(10)).Return(nil, gorm.ErrRecordNotFound)

	result, err := svc.Create(1, dto.CreateProfitPoolRequest{ProjectID: 10, TotalAmount: 10000})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "project not found")
}

func TestProfitPoolService_Create_NoInvestors(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	investSvc := new(mockPPInvestmentSvc)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, investSvc, nil)

	projRepo.On("FindProjectByID", uint(10)).Return(&domain.Project{ID: 10, OwnerUserID: 5}, nil)
	investSvc.On("SyncProjectPrincipalAmounts", uint(10)).Return(nil)
	investRepo.On("ListInvestorsByProjectID", uint(10)).Return([]dto.ProjectInvestorItem{}, nil)

	result, err := svc.Create(1, dto.CreateProfitPoolRequest{ProjectID: 10, TotalAmount: 10000})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no investors")
}

// List

func TestProfitPoolService_List_Empty(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	repo.On("ListAll").Return([]domain.ProfitPool{}, nil)

	items, err := svc.List()

	assert.NoError(t, err)
	assert.Empty(t, items)
}

func TestProfitPoolService_List_Success(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	pools := []domain.ProfitPool{
		{ID: 1, ProjectID: 10, PioneerUserID: 5, TotalAmount: 10000, Status: domain.ProfitPoolPending, QuarterNo: 1},
	}
	repo.On("ListAll").Return(pools, nil)
	projRepo.On("FindProjectByID", uint(10)).Return(&domain.Project{ID: 10, Title: "Project A"}, nil)
	userRepo.On("FindUserById", uint(5)).Return(&domain.User{ID: 5, FirstName: "Pioneer", LastName: "One"}, nil)
	repo.On("ListPayoutsByPoolID", uint(1)).Return([]domain.InvestorProfitPayout{
		{ID: 1, Status: domain.InvestorPayoutConfirmed},
		{ID: 2, Status: domain.InvestorPayoutPending},
	}, nil)

	items, err := svc.List()

	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Project A", items[0].ProjectTitle)
	assert.Equal(t, "Pioneer One", items[0].PioneerName)
	assert.Equal(t, 2, items[0].InvestorCount)
	assert.Equal(t, 1, items[0].ConfirmedCount)
}

func TestProfitPoolService_List_RepoError(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	repo.On("ListAll").Return([]domain.ProfitPool(nil), errors.New("db error"))

	items, err := svc.List()

	assert.Error(t, err)
	assert.Nil(t, items)
}

// GetDetail

func TestProfitPoolService_GetDetail_Success(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	pool := &domain.ProfitPool{ID: 1, ProjectID: 10, PioneerUserID: 5, TotalAmount: 10000, Status: domain.ProfitPoolPending, QuarterNo: 1}
	repo.On("FindByID", uint(1)).Return(pool, nil)
	projRepo.On("FindProjectByID", uint(10)).Return(&domain.Project{ID: 10, Title: "Project A"}, nil)
	userRepo.On("FindUserById", uint(5)).Return(&domain.User{ID: 5, FirstName: "Pioneer", LastName: "One"}, nil)
	investRepo.On("ListInvestorsByProjectID", uint(10)).Return([]dto.ProjectInvestorItem{
		{UserID: 2, PrincipalAmount: 5000},
	}, nil)
	repo.On("ListPayoutsByPoolID", uint(1)).Return([]domain.InvestorProfitPayout{
		{ID: 1, BoosterUserID: 2, Amount: 500, SharePct: 50, Status: domain.InvestorPayoutPending},
	}, nil)
	userRepo.On("FindUserById", uint(2)).Return(&domain.User{ID: 2, FirstName: "Booster", LastName: "Two", Email: "b@test.com"}, nil)
	userRepo.On("FindBankByUserId", uint(2)).Return([]domain.BankAccount{}, nil)

	detail, err := svc.GetDetail(1)

	assert.NoError(t, err)
	assert.NotNil(t, detail)
	assert.Equal(t, uint(1), detail.ID)
	assert.Equal(t, "Project A", detail.ProjectTitle)
	assert.Equal(t, "Pioneer One", detail.PioneerName)
	assert.Len(t, detail.Payouts, 1)
	assert.Equal(t, float64(5000), detail.Payouts[0].PrincipalAmount)
}

func TestProfitPoolService_GetDetail_NotFound(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	repo.On("FindByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)

	detail, err := svc.GetDetail(99)

	assert.Error(t, err)
	assert.Nil(t, detail)
	assert.Contains(t, err.Error(), "not found")
}

// ConfirmPayout

func TestProfitPoolService_ConfirmPayout_PoolNotFound(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	repo.On("FindByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)

	err := svc.ConfirmPayout(99, 1, 1, dto.ConfirmInvestorPayoutRequest{TransferRef: "REF"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "profit pool not found")
}

func TestProfitPoolService_ConfirmPayout_PayoutNotFound(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	repo.On("FindByID", uint(1)).Return(&domain.ProfitPool{ID: 1, ProjectID: 10}, nil)
	repo.On("FindPayoutByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)

	err := svc.ConfirmPayout(1, 99, 1, dto.ConfirmInvestorPayoutRequest{TransferRef: "REF"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payout not found")
}

func TestProfitPoolService_ConfirmPayout_WrongPool(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	repo.On("FindByID", uint(1)).Return(&domain.ProfitPool{ID: 1, ProjectID: 10}, nil)
	// payout belongs to pool 2, not pool 1
	repo.On("FindPayoutByID", uint(5)).Return(&domain.InvestorProfitPayout{ID: 5, ProfitPoolID: 2}, nil)

	err := svc.ConfirmPayout(1, 5, 1, dto.ConfirmInvestorPayoutRequest{TransferRef: "REF"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong")
}

func TestProfitPoolService_ConfirmPayout_AlreadyConfirmed(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	repo.On("FindByID", uint(1)).Return(&domain.ProfitPool{ID: 1, ProjectID: 10}, nil)
	repo.On("FindPayoutByID", uint(5)).Return(&domain.InvestorProfitPayout{
		ID: 5, ProfitPoolID: 1, Status: domain.InvestorPayoutConfirmed,
	}, nil)

	err := svc.ConfirmPayout(1, 5, 1, dto.ConfirmInvestorPayoutRequest{TransferRef: "REF"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already confirmed")
}

func TestProfitPoolService_ConfirmPayout_Success_AllDone(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	pool := &domain.ProfitPool{ID: 1, ProjectID: 10, Status: domain.ProfitPoolPending}
	repo.On("FindByID", uint(1)).Return(pool, nil)
	repo.On("FindPayoutByID", uint(5)).Return(&domain.InvestorProfitPayout{
		ID: 5, ProfitPoolID: 1, BoosterUserID: 2, Status: domain.InvestorPayoutPending,
	}, nil)
	repo.On("UpdatePayout", mock.Anything).Return(nil)
	// after update → all payouts are confirmed
	repo.On("ListPayoutsByPoolID", uint(1)).Return([]domain.InvestorProfitPayout{
		{ID: 5, Status: domain.InvestorPayoutConfirmed},
	}, nil)
	repo.On("Update", mock.Anything).Return(nil) // pool status → completed

	err := svc.ConfirmPayout(1, 5, 99, dto.ConfirmInvestorPayoutRequest{TransferRef: "REF-001", Note: "โอนแล้ว"})

	assert.NoError(t, err)
	// pool ต้องถูก update เป็น completed
	repo.AssertCalled(t, "Update", mock.Anything)
}

// PioneerSubmit - validation paths

func TestProfitPoolService_PioneerSubmit_PermissionDenied(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	// project owned by user 5, but pioneer is user 3
	projRepo.On("FindProjectByID", uint(10)).Return(&domain.Project{ID: 10, OwnerUserID: 5, State: domain.StateExecuting}, nil)

	result, err := svc.PioneerSubmit(3, 10, dto.PioneerSubmitProfitRequest{TotalAmount: 5000, QuarterNo: 1})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestProfitPoolService_PioneerSubmit_ProjectStateMismatch(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	projRepo.On("FindProjectByID", uint(10)).Return(&domain.Project{
		ID: 10, OwnerUserID: 5, State: domain.StateFunding,
	}, nil)

	result, err := svc.PioneerSubmit(5, 10, dto.PioneerSubmitProfitRequest{TotalAmount: 5000, QuarterNo: 1})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "executing or closed")
}

func TestProfitPoolService_PioneerSubmit_MilestoneNotAllPaid(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	projRepo.On("FindProjectByID", uint(10)).Return(&domain.Project{
		ID: 10, OwnerUserID: 5, State: domain.StateExecuting,
	}, nil)
	projRepo.On("FindMilestonesByProjectID", uint(10)).Return([]domain.Milestone{
		{ID: 1, Status: domain.MilestonePaid},
		{ID: 2, Status: domain.MilestoneActive}, // ยังไม่ paid
	}, nil)

	result, err := svc.PioneerSubmit(5, 10, dto.PioneerSubmitProfitRequest{TotalAmount: 5000, QuarterNo: 1})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Phase Milestone")
}

func TestProfitPoolService_PioneerSubmit_QuarterAlreadySubmitted(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	projRepo.On("FindProjectByID", uint(10)).Return(&domain.Project{
		ID: 10, OwnerUserID: 5, State: domain.StateExecuting,
	}, nil)
	projRepo.On("FindMilestonesByProjectID", uint(10)).Return([]domain.Milestone{
		{ID: 1, Status: domain.MilestonePaid},
	}, nil)
	repo.On("ExistsByProjectAndQuarter", uint(10), 1).Return(true, nil)

	result, err := svc.PioneerSubmit(5, 10, dto.PioneerSubmitProfitRequest{TotalAmount: 5000, QuarterNo: 1})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "ส่งไปแล้ว")
}

// GetPioneerPools

func TestProfitPoolService_GetPioneerPools_Success(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	pools := []domain.ProfitPool{
		{ID: 1, ProjectID: 10, PioneerUserID: 5, TotalAmount: 10000, Status: domain.ProfitPoolPending, QuarterNo: 1},
	}
	repo.On("ListByPioneerUserID", uint(5)).Return(pools, nil)
	projRepo.On("FindProjectByID", uint(10)).Return(&domain.Project{ID: 10, Title: "Project A"}, nil)
	repo.On("ListPayoutsByPoolID", uint(1)).Return([]domain.InvestorProfitPayout{
		{ID: 1, Status: domain.InvestorPayoutConfirmed},
		{ID: 2, Status: domain.InvestorPayoutPending},
	}, nil)

	items, err := svc.GetPioneerPools(5)

	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Project A", items[0].ProjectTitle)
	assert.Equal(t, 2, items[0].InvestorCount)
	assert.Equal(t, 1, items[0].ConfirmedCount)
}

func TestProfitPoolService_GetPioneerPools_Empty(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	repo.On("ListByPioneerUserID", uint(5)).Return([]domain.ProfitPool{}, nil)

	items, err := svc.GetPioneerPools(5)

	assert.NoError(t, err)
	assert.Empty(t, items)
}

func TestProfitPoolService_GetPioneerPools_RepoError(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	repo.On("ListByPioneerUserID", uint(5)).Return([]domain.ProfitPool(nil), errors.New("db error"))

	items, err := svc.GetPioneerPools(5)

	assert.Error(t, err)
	assert.Nil(t, items)
}

// GetMyProfitPayouts

func TestProfitPoolService_GetMyProfitPayouts_Success(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	payouts := []domain.InvestorProfitPayout{
		{ID: 1, ProfitPoolID: 10, ProjectID: 5, BoosterUserID: 2, Amount: 500, SharePct: 25, Status: domain.InvestorPayoutConfirmed},
	}
	repo.On("ListPayoutsByBoosterUserID", uint(2)).Return(payouts, nil)
	repo.On("FindByID", uint(10)).Return(&domain.ProfitPool{ID: 10, QuarterNo: 2}, nil)
	projRepo.On("FindProjectByID", uint(5)).Return(&domain.Project{ID: 5, Title: "Project B"}, nil)

	items, err := svc.GetMyProfitPayouts(2)

	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Project B", items[0].ProjectTitle)
	assert.Equal(t, 2, items[0].QuarterNo)
	assert.Equal(t, float64(500), items[0].Amount)
}

func TestProfitPoolService_GetMyProfitPayouts_Empty(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	repo.On("ListPayoutsByBoosterUserID", uint(2)).Return([]domain.InvestorProfitPayout{}, nil)

	items, err := svc.GetMyProfitPayouts(2)

	assert.NoError(t, err)
	assert.Empty(t, items)
}

func TestProfitPoolService_GetMyProfitPayouts_RepoError(t *testing.T) {
	repo := new(mockProfitPoolRepo)
	projRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	userRepo := new(mockUserRepository)
	svc := newProfitPoolSvc(repo, projRepo, investRepo, userRepo, nil, nil)

	repo.On("ListPayoutsByBoosterUserID", uint(2)).Return([]domain.InvestorProfitPayout(nil), errors.New("db error"))

	items, err := svc.GetMyProfitPayouts(2)

	assert.Error(t, err)
	assert.Nil(t, items)
}

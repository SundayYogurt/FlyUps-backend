package service

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// ─── Mock: InvestmentRepository ─────────────────────────────────────────────

type mockInvestmentRepo struct{ mock.Mock }

func (m *mockInvestmentRepo) Create(inv *domain.Investment) error {
	args := m.Called(inv)
	return args.Error(0)
}
func (m *mockInvestmentRepo) FindByID(id uint) (*domain.Investment, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.Investment), args.Error(1)
}
func (m *mockInvestmentRepo) FindByIDWithProject(id uint) (*domain.Investment, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.Investment), args.Error(1)
}
func (m *mockInvestmentRepo) FindByReferenceNumber(ref string) (*domain.Investment, error) {
	args := m.Called(ref)
	return args.Get(0).(*domain.Investment), args.Error(1)
}
func (m *mockInvestmentRepo) UpdateStatus(id uint, status domain.InvestmentStatus) error {
	args := m.Called(id, status)
	return args.Error(0)
}
func (m *mockInvestmentRepo) UpdatePaid(inv *domain.Investment) error {
	args := m.Called(inv)
	return args.Error(0)
}
func (m *mockInvestmentRepo) UpdateRefunded(inv *domain.Investment) error {
	args := m.Called(inv)
	return args.Error(0)
}
func (m *mockInvestmentRepo) ListByBoosterUserID(boosterUserID uint) ([]domain.Investment, error) {
	args := m.Called(boosterUserID)
	return args.Get(0).([]domain.Investment), args.Error(1)
}
func (m *mockInvestmentRepo) ListRefundPending() ([]domain.Investment, error) {
	args := m.Called()
	return args.Get(0).([]domain.Investment), args.Error(1)
}
func (m *mockInvestmentRepo) SumActiveByProjectID(projectID uint) (float64, error) {
	args := m.Called(projectID)
	return args.Get(0).(float64), args.Error(1)
}
func (m *mockInvestmentRepo) FindVerifiedByProjectID(projectID uint) ([]domain.Investment, error) {
	args := m.Called(projectID)
	return args.Get(0).([]domain.Investment), args.Error(1)
}
func (m *mockInvestmentRepo) IncrementProjectFunding(projectID uint, amount float64) error {
	args := m.Called(projectID, amount)
	return args.Error(0)
}
func (m *mockInvestmentRepo) ListInvestorsByProjectID(projectID uint) ([]dto.ProjectInvestorItem, error) {
	args := m.Called(projectID)
	return args.Get(0).([]dto.ProjectInvestorItem), args.Error(1)
}
func (m *mockInvestmentRepo) ListInvestedProjectsByUserID(boosterUserID uint) ([]dto.InvestedProjectItem, error) {
	args := m.Called(boosterUserID)
	return args.Get(0).([]dto.InvestedProjectItem), args.Error(1)
}
func (m *mockInvestmentRepo) SumTotalFunding() (float64, error) {
	args := m.Called()
	return args.Get(0).(float64), args.Error(1)
}
func (m *mockInvestmentRepo) CountUniqueBoosters() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// ─── Mock: TransactionRepository ────────────────────────────────────────────

type mockTransactionRepo struct{ mock.Mock }

func (m *mockTransactionRepo) Create(txn *domain.Transaction) error {
	args := m.Called(txn)
	return args.Error(0)
}
func (m *mockTransactionRepo) FindByPaymentIntentID(intentID string) (*domain.Transaction, error) {
	args := m.Called(intentID)
	return args.Get(0).(*domain.Transaction), args.Error(1)
}
func (m *mockTransactionRepo) FindByInvestmentID(investmentID uint) (*domain.Transaction, error) {
	args := m.Called(investmentID)
	return args.Get(0).(*domain.Transaction), args.Error(1)
}
func (m *mockTransactionRepo) UpdateStatus(id uint, status domain.TransactionStatus) error {
	args := m.Called(id, status)
	return args.Error(0)
}
func (m *mockTransactionRepo) UpdateStripeFeesAndNet(id uint, stripeFee, stripeFeeVAT, netAmount float64) error {
	args := m.Called(id, stripeFee, stripeFeeVAT, netAmount)
	return args.Error(0)
}

// ─── Helper to build service ─────────────────────────────────────────────────

func newTestInvestmentService(
	projectRepo *ProjectRepository,
	investRepo *mockInvestmentRepo,
	txnRepo *mockTransactionRepo,
	userRepo *mockUserRepository,
) InvestmentService {
	return NewInvestmentService(projectRepo, investRepo, txnRepo, userRepo, nil, "sk_test_dummy", "whsec_dummy", nil, nil)
}

// ─── GetInvestment ────────────────────────────────────────────────────────────

func TestGetInvestment_Success(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{ID: 1, BoosterUserID: 10}
	txn := &domain.Transaction{ID: 5, InvestmentID: 1}

	investRepo.On("FindByIDWithProject", uint(1)).Return(inv, nil)
	txnRepo.On("FindByInvestmentID", uint(1)).Return(txn, nil)

	gotInv, gotTxn, err := svc.GetInvestment(10, 1)

	assert.NoError(t, err)
	assert.Equal(t, inv, gotInv)
	assert.Equal(t, txn, gotTxn)
	investRepo.AssertExpectations(t)
	txnRepo.AssertExpectations(t)
}

func TestGetInvestment_NotFound(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	investRepo.On("FindByIDWithProject", uint(99)).Return(&domain.Investment{}, gorm.ErrRecordNotFound)

	_, _, err := svc.GetInvestment(10, 99)

	assert.EqualError(t, err, "investment not found")
}

func TestGetInvestment_DBError(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	investRepo.On("FindByIDWithProject", uint(1)).Return(&domain.Investment{}, errors.New("db error"))

	_, _, err := svc.GetInvestment(10, 1)

	assert.EqualError(t, err, "internal server error")
}

func TestGetInvestment_WrongOwner(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{ID: 1, BoosterUserID: 99} // owned by user 99
	investRepo.On("FindByIDWithProject", uint(1)).Return(inv, nil)

	_, _, err := svc.GetInvestment(10, 1) // request as user 10

	assert.EqualError(t, err, "investment not found")
}

func TestGetInvestment_NoTransaction(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{ID: 1, BoosterUserID: 10}
	investRepo.On("FindByIDWithProject", uint(1)).Return(inv, nil)
	txnRepo.On("FindByInvestmentID", uint(1)).Return(&domain.Transaction{}, gorm.ErrRecordNotFound)

	gotInv, gotTxn, err := svc.GetInvestment(10, 1)

	assert.NoError(t, err)
	assert.Equal(t, inv, gotInv)
	assert.Nil(t, gotTxn)
}

// ─── ListUserInvestments ──────────────────────────────────────────────────────

func TestListUserInvestments_Success(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	list := []domain.Investment{{ID: 1}, {ID: 2}}
	investRepo.On("ListByBoosterUserID", uint(10)).Return(list, nil)

	result, err := svc.ListUserInvestments(10)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestListUserInvestments_Error(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	investRepo.On("ListByBoosterUserID", uint(10)).Return([]domain.Investment{}, errors.New("db error"))

	_, err := svc.ListUserInvestments(10)

	assert.Error(t, err)
}

// ─── CreateInvestment (pre-Stripe paths) ─────────────────────────────────────

func TestCreateInvestment_ProjectNotFound(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID: 10,
		Role: "booster",
		IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved},
	}, nil)
	projectRepo.On("FindProjectByID", uint(1)).Return(&domain.Project{}, gorm.ErrRecordNotFound)

	_, err := svc.CreateInvestment(10, "user@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 1000})

	assert.EqualError(t, err, "project not found")
}

func TestCreateInvestment_AdminCannotInvest(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{ID: 10, Role: "admin"}, nil)

	_, err := svc.CreateInvestment(10, "admin@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 1000})

	assert.EqualError(t, err, "admin cannot invest")
	projectRepo.AssertNotCalled(t, "FindProjectByID", mock.Anything)
}

func TestCreateInvestment_OwnerCannotInvestSelf(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID: 10,
		Role: "pioneer",
		IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved},
	}, nil)
	project := &domain.Project{ID: 1, OwnerUserID: 10, State: domain.StateFunding}
	projectRepo.On("FindProjectByID", uint(1)).Return(project, nil)

	_, err := svc.CreateInvestment(10, "owner@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 1000})

	assert.EqualError(t, err, "cannot invest in your own project")
}

func TestCreateInvestment_ProjectNotInFunding(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID: 10,
		Role: "booster",
		IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved},
	}, nil)
	project := &domain.Project{ID: 1, State: domain.StateDraft}
	projectRepo.On("FindProjectByID", uint(1)).Return(project, nil)

	_, err := svc.CreateInvestment(10, "user@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 1000})

	assert.EqualError(t, err, "project is not open for investment")
}

func TestCreateInvestment_AmountBelowMin(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID: 10,
		Role: "booster",
		IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved},
	}, nil)
	project := &domain.Project{
		ID:              1,
		State:           domain.StateFunding,
		FundingGoal:     100000,
		MinInvestAmount: 500,
		MaxInvestAmount: 50000,
		PlatformFee:     2,
	}
	projectRepo.On("FindProjectByID", uint(1)).Return(project, nil)

	_, err := svc.CreateInvestment(10, "user@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 100})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "minimum investment")
}

func TestCreateInvestment_AmountAboveMax(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID: 10,
		Role: "booster",
		IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved},
	}, nil)
	project := &domain.Project{
		ID:              1,
		State:           domain.StateFunding,
		FundingGoal:     100000,
		MinInvestAmount: 500,
		MaxInvestAmount: 50000,
		PlatformFee:     2,
	}
	projectRepo.On("FindProjectByID", uint(1)).Return(project, nil)

	_, err := svc.CreateInvestment(10, "user@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 99999})

	assert.EqualError(t, err, "maximum investment is ฿50000")
}

func TestCreateInvestment_ExceedsMaxPerTransaction(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID: 10,
		Role: "booster",
		IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved},
	}, nil)
	project := &domain.Project{
		ID:              1,
		State:           domain.StateFunding,
		FundingGoal:     10_000_000,
		MinInvestAmount: 500,
		MaxInvestAmount: 1_000_000,
		PlatformFee:     2,
	}
	projectRepo.On("FindProjectByID", uint(1)).Return(project, nil)

	_, err := svc.CreateInvestment(10, "user@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 600_000})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ยอดลงทุนต่อรายการต้องไม่เกิน")
	investRepo.AssertNotCalled(t, "Create", mock.Anything)
}

func TestCreateInvestment_ExceedsFundingGoal(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID: 10,
		Role: "booster",
		IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved},
	}, nil)
	project := &domain.Project{
		ID:              1,
		State:           domain.StateFunding,
		FundingGoal:     10000,
		MinInvestAmount: 100,
		MaxInvestAmount: 0,
		PlatformFee:     2,
	}
	projectRepo.On("FindProjectByID", uint(1)).Return(project, nil)
	investRepo.On("SumActiveByProjectID", uint(1)).Return(float64(9500), nil)

	_, err := svc.CreateInvestment(10, "user@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 1000})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "investment exceeds funding goal")
	assert.Contains(t, err.Error(), "500")
}

// ─── RefundInvestment ─────────────────────────────────────────────────────────

func TestRefundInvestment_Success(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{
		ID:              1,
		BoosterUserID:   10,
		ProjectID:       5,
		Status:          domain.InvestmentVerified,
		TotalAmount:     1000,
		PrincipalAmount: 921.30,
		ReferenceNumber: "INV-ABCDEFGH",
	}
	project := &domain.Project{ID: 5, State: domain.StateFunding}

	investRepo.On("FindByID", uint(1)).Return(inv, nil)
	projectRepo.On("FindProjectByID", uint(5)).Return(project, nil)
	investRepo.On("UpdateRefunded", mock.AnythingOfType("*domain.Investment")).Return(nil)
	investRepo.On("IncrementProjectFunding", uint(5), -float64(1000)).Return(nil)

	resp, err := svc.RefundInvestment(10, 1, "ต้องการยกเลิกการลงทุน")

	assert.NoError(t, err)
	assert.Equal(t, uint(1), resp.InvestmentID)
	assert.Equal(t, float64(921.30), resp.RefundAmount)
	investRepo.AssertExpectations(t)
}

func TestRefundInvestment_InvestmentNotFound(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	investRepo.On("FindByID", uint(99)).Return(&domain.Investment{}, errors.New("not found"))

	_, err := svc.RefundInvestment(10, 99, "test note")

	assert.EqualError(t, err, "investment not found")
}

func TestRefundInvestment_WrongOwner(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{ID: 1, BoosterUserID: 99}
	investRepo.On("FindByID", uint(1)).Return(inv, nil)

	_, err := svc.RefundInvestment(10, 1, "test note")

	assert.EqualError(t, err, "investment not found")
}

func TestRefundInvestment_NotVerified(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{ID: 1, BoosterUserID: 10, Status: domain.InvestmentPending}
	investRepo.On("FindByID", uint(1)).Return(inv, nil)

	_, err := svc.RefundInvestment(10, 1, "test note")

	assert.EqualError(t, err, "only verified investments can be refunded")
}

func TestRefundInvestment_ProjectNotInFunding(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{ID: 1, BoosterUserID: 10, ProjectID: 5, Status: domain.InvestmentVerified}
	project := &domain.Project{ID: 5, State: domain.StateClosed}

	investRepo.On("FindByID", uint(1)).Return(inv, nil)
	projectRepo.On("FindProjectByID", uint(5)).Return(project, nil)

	_, err := svc.RefundInvestment(10, 1, "test note")

	assert.EqualError(t, err, "refund is only allowed while project project is in funding state")
}

// ─── ApproveRefund ────────────────────────────────────────────────────────────

func TestApproveRefund_Success(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{ID: 1, ProjectID: 5, Status: domain.InvestmentRefundPending, TotalAmount: 1000}
	investRepo.On("FindByID", uint(1)).Return(inv, nil)
	investRepo.On("UpdateRefunded", mock.AnythingOfType("*domain.Investment")).Return(nil)
	investRepo.On("IncrementProjectFunding", uint(5), -float64(1000)).Return(nil)

	err := svc.ApproveRefund(1)

	assert.NoError(t, err)
	investRepo.AssertExpectations(t)
}

func TestApproveRefund_InvestmentNotFound(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	investRepo.On("FindByID", uint(99)).Return(&domain.Investment{}, errors.New("not found"))

	err := svc.ApproveRefund(99)

	assert.EqualError(t, err, "investment not found")
}

func TestApproveRefund_NotRefundPending(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{ID: 1, Status: domain.InvestmentVerified}
	investRepo.On("FindByID", uint(1)).Return(inv, nil)

	err := svc.ApproveRefund(1)

	assert.EqualError(t, err, "investment is not pending refund")
}

func TestApproveRefund_DBUpdateError(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{ID: 1, ProjectID: 5, Status: domain.InvestmentRefundPending, TotalAmount: 1000}
	investRepo.On("FindByID", uint(1)).Return(inv, nil)
	investRepo.On("UpdateRefunded", mock.AnythingOfType("*domain.Investment")).Return(errors.New("db error"))

	err := svc.ApproveRefund(1)

	assert.EqualError(t, err, "failed to approve refund")
}

// ─── ListRefundRequests ───────────────────────────────────────────────────────

func TestListRefundRequests_Success(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	now := time.Now()
	investments := []domain.Investment{
		{ID: 1, ProjectID: 5, BoosterUserID: 10, ReferenceNumber: "INV-AAA", RefundAmount: 900, TotalAmount: 1000, RefundedAt: &now},
	}
	user := &domain.User{
		ID:        10,
		FirstName: "John",
		LastName:  "Doe",
		BankAccounts: []domain.BankAccount{
			{
				BankName:      "SCB",
				AccountName:   "John Doe",
				AccountNumber: "123456789",
				IsDefault:     true,
			},
		},
	}

	investRepo.On("ListRefundPending").Return(investments, nil)
	projectRepo.On("FindProjectByID", uint(5)).Return(&domain.Project{ID: 5, Title: "Test Project"}, nil)
	userRepo.On("FindUserById", uint(10)).Return(user, nil)

	result, err := svc.ListRefundRequests()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "John Doe", result[0].BoosterName)
	assert.NotNil(t, result[0].BankAccount)
	assert.Equal(t, "SCB", result[0].BankAccount.BankName)
}

func TestListRefundRequests_DBError(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	investRepo.On("ListRefundPending").Return([]domain.Investment{}, errors.New("db error"))

	_, err := svc.ListRefundRequests()

	assert.EqualError(t, err, "internal server error")
}

// ─── GetProjectInvestors ──────────────────────────────────────────────────────

func TestGetProjectInvestors_Success(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	project := &domain.Project{ID: 5}
	investors := []dto.ProjectInvestorItem{{UserID: 10, FirstName: "Alice"}}

	projectRepo.On("FindProjectByID", uint(5)).Return(project, nil)
	investRepo.On("ListInvestorsByProjectID", uint(5)).Return(investors, nil)

	result, err := svc.GetProjectInvestors(5)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Alice", result[0].FirstName)
}

func TestGetProjectInvestors_ProjectNotFound(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	projectRepo.On("FindProjectByID", uint(99)).Return(&domain.Project{}, errors.New("not found"))

	_, err := svc.GetProjectInvestors(99)

	assert.EqualError(t, err, "project not found")
}

// ─── ListInvestedProjects ─────────────────────────────────────────────────────

func TestListInvestedProjects_Success(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	items := []dto.InvestedProjectItem{{ProjectID: 1, Title: "Cool Project"}}
	investRepo.On("ListInvestedProjectsByUserID", uint(10)).Return(items, nil)

	result, err := svc.ListInvestedProjects(10)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Cool Project", result[0].Title)
}

// ─── validateAmount (helper) ──────────────────────────────────────────────────

func TestValidateAmount_Success(t *testing.T) {
	project := &domain.Project{
		FundingGoal:     100000,
		MinInvestAmount: 500,
		MaxInvestAmount: 50000,
	}
	err := validateAmount(project, 1000)
	assert.NoError(t, err)
}

func TestValidateAmount_BelowMin(t *testing.T) {
	project := &domain.Project{
		FundingGoal:     100000,
		MinInvestAmount: 1000,
		MaxInvestAmount: 0,
	}
	err := validateAmount(project, 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "minimum investment")
}

func TestValidateAmount_AboveMax(t *testing.T) {
	project := &domain.Project{
		FundingGoal:     100000,
		MinInvestAmount: 100,
		MaxInvestAmount: 10000,
	}
	err := validateAmount(project, 20000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum investment")
}

func TestValidateAmount_EffectiveMinFromFundingGoal(t *testing.T) {
	// effective min = max(minInvestAmount=0, fundingGoal*0.01=1000)
	project := &domain.Project{
		FundingGoal:     100000,
		MinInvestAmount: 0,
		MaxInvestAmount: 0,
	}
	err := validateAmount(project, 500) // below 1% of 100000
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "minimum investment")
}

// ─── calculateFees (helper) ───────────────────────────────────────────────────

func TestCalculateFees_Basic(t *testing.T) {
	// amount=10000, platformFee=2%
	// fee   = round(10000 * 0.02 * 100) / 100 = 200.00
	// vat   = round(200 * 0.07 * 100) / 100   = 14.00
	// principal = round((10000 - 200 - 14) * 100) / 100 = 9786.00
	fee, vat, principal := calculateFees(10000, 2)

	assert.Equal(t, 200.0, fee)
	assert.Equal(t, 14.0, vat)
	assert.Equal(t, 9786.0, principal)
}

func TestCalculateFees_ZeroFee(t *testing.T) {
	fee, vat, principal := calculateFees(5000, 0)

	assert.Equal(t, 0.0, fee)
	assert.Equal(t, 0.0, vat)
	assert.Equal(t, 5000.0, principal)
}

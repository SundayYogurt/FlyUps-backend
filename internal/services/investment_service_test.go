package services

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

// ─── Helper to build services ─────────────────────────────────────────────────

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
		ID:                 10,
		Role:               "booster",
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

func TestCreateInvestment_PioneerCannotInvest(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{ID: 10, Role: "pioneer"}, nil)

	_, err := svc.CreateInvestment(10, "pioneer@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 1000})

	assert.EqualError(t, err, "pioneer cannot invest")
	projectRepo.AssertNotCalled(t, "FindProjectByID", mock.Anything)
}

func TestCreateInvestment_OwnerCannotInvestSelf(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID:                 10,
		Role:               "booster",
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
		ID:                 10,
		Role:               "booster",
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
		ID:                 10,
		Role:               "booster",
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
	assert.Contains(t, err.Error(), "จำนวนเงินขั้นต่ำคือ")
}

func TestCreateInvestment_AmountAboveMax(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID:                 10,
		Role:               "booster",
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

	assert.EqualError(t, err, "จำนวนเงินสูงสุดคือ ฿50000.00")
}

func TestCreateInvestment_ExceedsMaxPerTransaction(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID:                 10,
		Role:               "booster",
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
		ID:                 10,
		Role:               "booster",
		IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved},
	}, nil)
	project := &domain.Project{
		ID:              1,
		State:           domain.StateFunding,
		FundingGoal:     10000,
		CurrentFunding:  9500,
		MinInvestAmount: 100,
		MaxInvestAmount: 0,
		PlatformFee:     2,
	}
	projectRepo.On("FindProjectByID", uint(1)).Return(project, nil)

	_, err := svc.CreateInvestment(10, "user@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 1000})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ยอดคงเหลือให้ลงทุนคือ")
	assert.Contains(t, err.Error(), "500")
	investRepo.AssertNotCalled(t, "Create", mock.Anything)
}

func TestCreateInvestment_BelowStripeMinimum(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID:                 10,
		Role:               "booster",
		IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved},
	}, nil)
	// softcap reached → 1% minimum is waived, but the ฿20 Stripe floor must still apply
	project := &domain.Project{
		ID:              1,
		State:           domain.StateFunding,
		FundingGoal:     100000,
		Softcap:         5000,
		CurrentFunding:  6000,
		MinInvestAmount: 0,
		MaxInvestAmount: 0,
		PlatformFee:     2,
	}
	projectRepo.On("FindProjectByID", uint(1)).Return(project, nil)

	_, err := svc.CreateInvestment(10, "user@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 10})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "จำนวนเงินขั้นต่ำต่อครั้งคือ")
	investRepo.AssertNotCalled(t, "SumActiveByProjectID", mock.Anything)
}

func TestCreateInvestment_LeavesDustBelowStripeMinimum(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID:                 10,
		Role:               "booster",
		IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved},
	}, nil)
	project := &domain.Project{
		ID:              1,
		State:           domain.StateFunding,
		FundingGoal:     10000,
		CurrentFunding:  9870,
		MinInvestAmount: 100,
		MaxInvestAmount: 0,
		PlatformFee:     2,
	}
	projectRepo.On("FindProjectByID", uint(1)).Return(project, nil)
	// remaining = 10000 - 9870 = 130; investing 120 leaves a ฿10 dust below the ฿20 Stripe minimum

	_, err := svc.CreateInvestment(10, "user@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 120})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ระดมต่อไม่ได้")
	assert.Contains(t, err.Error(), "ปิดยอดที่เหลือ")
	investRepo.AssertNotCalled(t, "Create", mock.Anything)
}

func TestCreateInvestment_ClosesFundingGoalExactly(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	userRepo.On("FindUserById", uint(10)).Return(&domain.User{
		ID:                 10,
		Role:               "booster",
		IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved},
	}, nil)
	project := &domain.Project{
		ID:              1,
		State:           domain.StateFunding,
		FundingGoal:     10000,
		CurrentFunding:  9900,
		MinInvestAmount: 100,
		MaxInvestAmount: 0,
		PlatformFee:     2,
	}
	projectRepo.On("FindProjectByID", uint(1)).Return(project, nil)
	// remaining = 10000 - 9900 = 100, investing exactly 100 closes the goal to 0 — must be allowed
	investRepo.On("Create", mock.Anything).Return(errors.New("stop before stripe call"))

	_, err := svc.CreateInvestment(10, "user@test.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 100})

	// should fail at the DB-create stage, not at amount validation
	assert.EqualError(t, err, "failed to create investment")
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
	txn := &domain.Transaction{ID: 7, InvestmentID: 1, NetAmount: 970}

	investRepo.On("FindByID", uint(1)).Return(inv, nil)
	projectRepo.On("FindProjectByID", uint(5)).Return(project, nil)
	txnRepo.On("FindByInvestmentID", uint(1)).Return(txn, nil)
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

func TestRefundInvestment_TransactionNotFound(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{ID: 1, BoosterUserID: 10, ProjectID: 5, Status: domain.InvestmentVerified, TotalAmount: 1000}
	project := &domain.Project{ID: 5, State: domain.StateFunding}

	investRepo.On("FindByID", uint(1)).Return(inv, nil)
	projectRepo.On("FindProjectByID", uint(5)).Return(project, nil)
	txnRepo.On("FindByInvestmentID", uint(1)).Return(&domain.Transaction{}, errors.New("not found"))

	_, err := svc.RefundInvestment(10, 1, "test note")

	assert.EqualError(t, err, "transaction not found")
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

func TestApproveRefund_TransactionNotFound(t *testing.T) {
	projectRepo := new(ProjectRepository)
	investRepo := new(mockInvestmentRepo)
	txnRepo := new(mockTransactionRepo)
	userRepo := new(mockUserRepository)

	svc := newTestInvestmentService(projectRepo, investRepo, txnRepo, userRepo)

	inv := &domain.Investment{ID: 1, ProjectID: 5, Status: domain.InvestmentRefundPending, TotalAmount: 1000}
	investRepo.On("FindByID", uint(1)).Return(inv, nil)
	txnRepo.On("FindByInvestmentID", uint(1)).Return(&domain.Transaction{}, errors.New("not found"))

	err := svc.ApproveRefund(1)

	assert.EqualError(t, err, "transaction not found")
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
//
// โปรเจกต์ทดสอบมาตรฐาน (ตาม Jira F2-116): goal 100,000 / softcap 50,000 / min 1,000 / max 10,000

func newValidateAmountTestProject(currentFunding float64) *domain.Project {
	return &domain.Project{
		FundingGoal:     100000,
		Softcap:         50000,
		CurrentFunding:  currentFunding,
		MinInvestAmount: 1000,
		MaxInvestAmount: 10000,
	}
}

func TestValidateAmount_TableDriven(t *testing.T) {
	tests := []struct {
		name           string
		currentFunding float64
		amount         float64
		wantErr        bool
		errContains    string
	}{
		{"amount 19 fails ข้อ 1 (hard floor ฿20)", 0, 19, true, "จำนวนเงินขั้นต่ำต่อครั้งคือ"},
		{"amount 20 but current low fails ข้อ 3 (below 1% min)", 0, 20, true, "จำนวนเงินขั้นต่ำคือ"},
		{"current 0, amount 999 fails ข้อ 3", 0, 999, true, "จำนวนเงินขั้นต่ำคือ"},
		{"current 0, amount 1000 passes (meets 1% min)", 0, 1000, false, ""},
		{"current 60000 (past softcap), amount 20 passes", 60000, 20, false, ""},
		{"current 60000 (past softcap), amount 19 still fails ข้อ 1", 60000, 19, true, "จำนวนเงินขั้นต่ำต่อครั้งคือ"},
		{"current 98990 (เหลือ 1010), amount 1000 fails ข้อ 5 (เหลือเศษ 10)", 98990, 1000, true, "ระดมต่อไม่ได้"},
		{"current 98980 (เหลือ 1020), amount 1000 passes (เหลือ 20 พอดี)", 98980, 1000, false, ""},
		{"current 99980 (เหลือ 20), amount 20 passes (ปิดยอด ต่ำกว่า 1% ได้)", 99980, 20, false, ""},
		{"current 99980 (เหลือ 20), amount 100 fails ข้อ 2", 99980, 100, true, "ยอดคงเหลือให้ลงทุนคือ"},
		{"current 88000 (เหลือ 12000 > max 10000), amount 11990 fails ข้อ 4", 88000, 11990, true, "จำนวนเงินสูงสุดคือ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := newValidateAmountTestProject(tt.currentFunding)
			err := validateAmount(project, tt.amount)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateAmount_DustRule_CannotCloseAloneDueToMaxCap ทดสอบ branch ที่สองของข้อ 5
// (remaining > MaxInvestAmount ปิดยอดคนเดียวไม่ได้เพราะติดเพดาน)
//
// หมายเหตุ: ตัวอย่างตัวเลขในทิกเก็ต (เหลือ 12,000 / max 10,000 / amount 11,985) ขัดกับลำดับการเช็คที่กำหนดไว้
// เพราะ amount 11,985 > max 10,000 จะเข้า error ข้อ 4 (เพดานสูงสุด) ก่อนถึงข้อ 5 เสมอ ทำให้ branch นี้ unreachable
// ด้วยตัวเลขดังกล่าว จึงปรับ current/amount ให้ amount ไม่เกิน max แต่ remaining ยังมากกว่า max เพื่อให้ทดสอบ
// branch นี้ได้จริง (ดูรายละเอียดเพิ่มเติมในสรุปท้ายงาน)
func TestValidateAmount_DustRule_CannotCloseAloneDueToMaxCap(t *testing.T) {
	project := newValidateAmountTestProject(89985) // remaining = 100000 - 89985 = 10015
	err := validateAmount(project, 10000)          // = max, เหลือเศษ ฿15; remaining(10015) > max(10000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ระดมต่อไม่ได้")
	assert.Contains(t, err.Error(), "ลดยอดเหลือไม่เกิน")
	assert.Contains(t, err.Error(), "9995")
}

func TestValidateAmount_SoftcapBoundary_ReachedExactlyAtEquality(t *testing.T) {
	// current == softcap พอดี ต้องถือว่า reached แล้ว (ยกเว้นขั้นต่ำ 1%)
	project := newValidateAmountTestProject(50000)
	err := validateAmount(project, 20) // ต่ำกว่า 1% ของเป้าหมาย (1000) แต่ softcap reached จึงต้องผ่าน
	assert.NoError(t, err)
}

func TestValidateAmount_SoftcapBoundary_ZeroNeverReached(t *testing.T) {
	// softcap = 0 ต้องไม่ถือว่า reached ไม่ว่ายอดระดมทุนจะสูงแค่ไหน
	project := newValidateAmountTestProject(90000)
	project.Softcap = 0
	err := validateAmount(project, 20) // ต่ำกว่า 1% ของเป้าหมาย และ softcap ไม่ถูกนับว่า reached
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "จำนวนเงินขั้นต่ำคือ")
}

func TestValidateAmount_ReproRealCase(t *testing.T) {
	// repro จริงจาก Jira F2-116: goal 300,000, current 299,975 (เหลือ 25)
	// เดิมระบบยอมให้ลง 20 บาท ทำให้เหลือ 5 บาทค้างตลอดกาลเพราะไม่มีใครโอนปิดได้
	project := &domain.Project{
		FundingGoal:     300000,
		Softcap:         150000, // โปรเจกต์ใกล้เต็มยอดแล้ว ถือว่าผ่าน softcap มานานแล้ว
		CurrentFunding:  299975,
		MinInvestAmount: 0,
		MaxInvestAmount: 0,
	}
	err := validateAmount(project, 20)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ระดมต่อไม่ได้")
	assert.Contains(t, err.Error(), "ปิดยอดที่เหลือ")
	assert.Contains(t, err.Error(), "25")
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

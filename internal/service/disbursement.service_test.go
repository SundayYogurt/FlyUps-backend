package service

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// ─── Mock: DisbursementRepository ────────────────────────────────────────────

type mockDisbursementRepo struct{ mock.Mock }

func (m *mockDisbursementRepo) Create(d *domain.Disbursement) error {
	args := m.Called(d)
	return args.Error(0)
}
func (m *mockDisbursementRepo) FindByID(id uint) (*domain.Disbursement, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Disbursement), args.Error(1)
}
func (m *mockDisbursementRepo) FindByMilestoneID(milestoneID uint) (*domain.Disbursement, error) {
	args := m.Called(milestoneID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Disbursement), args.Error(1)
}
func (m *mockDisbursementRepo) Update(d *domain.Disbursement) error {
	args := m.Called(d)
	return args.Error(0)
}
func (m *mockDisbursementRepo) ListAll() ([]domain.Disbursement, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Disbursement), args.Error(1)
}
func (m *mockDisbursementRepo) ListByStatus(status domain.DisbursementStatus) ([]domain.Disbursement, error) {
	args := m.Called(status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Disbursement), args.Error(1)
}
func (m *mockDisbursementRepo) ListByPioneerID(pioneerID uint) ([]domain.Disbursement, error) {
	args := m.Called(pioneerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Disbursement), args.Error(1)
}

// ─── Helper ──────────────────────────────────────────────────────────────────

func newTestDisbursementService(
	disbRepo *mockDisbursementRepo,
	projRepo *ProjectRepository,
	userRepo *mockUserRepository,
) DisbursementService {
	return NewDisbursementService(disbRepo, projRepo, userRepo, nil)
}

// ─── CreateForMilestone ──────────────────────────────────────────────────────

func TestCreateForMilestone_AlreadyExists_ReturnsExisting(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	existing := &domain.Disbursement{ID: 77, MilestoneID: 1}
	disbRepo.On("FindByMilestoneID", uint(1)).Return(existing, nil)

	got, err := svc.CreateForMilestone(1)

	assert.NoError(t, err)
	assert.Equal(t, existing, got)
	disbRepo.AssertNotCalled(t, "Create", mock.Anything)
}

func TestCreateForMilestone_Success(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	milestone := &domain.Milestone{
		ID:             10,
		ProjectID:      5,
		PhaseNo:        1,
		PercentRelease: 40,
	}
	project := &domain.Project{
		ID:             5,
		OwnerUserID:    99,
		CurrentFunding: 10000,
	}

	disbRepo.On("FindByMilestoneID", uint(10)).Return(nil, gorm.ErrRecordNotFound)
	projRepo.On("FindMilestoneByID", uint(10)).Return(milestone, nil)
	projRepo.On("FindProjectByID", uint(5)).Return(project, nil)
	disbRepo.On("Create", mock.MatchedBy(func(d *domain.Disbursement) bool {
		return d.MilestoneID == 10 &&
			d.ProjectID == 5 &&
			d.PioneerUserID == 99 &&
			d.Amount == 4000 && // 10000 * 40 / 100
			d.PhaseNo == 1 &&
			d.PercentRelease == 40 &&
			d.Status == domain.DisbursementPending
	})).Return(nil)

	got, err := svc.CreateForMilestone(10)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, float64(4000), got.Amount)
	assert.Equal(t, domain.DisbursementPending, got.Status)
	disbRepo.AssertExpectations(t)
	projRepo.AssertExpectations(t)
}

func TestCreateForMilestone_MilestoneNotFound(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	disbRepo.On("FindByMilestoneID", uint(10)).Return(nil, gorm.ErrRecordNotFound)
	projRepo.On("FindMilestoneByID", uint(10)).Return((*domain.Milestone)(nil), errors.New("not found"))

	_, err := svc.CreateForMilestone(10)

	assert.EqualError(t, err, "milestone not found")
}

func TestCreateForMilestone_ProjectNotFound(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	milestone := &domain.Milestone{ID: 10, ProjectID: 5}

	disbRepo.On("FindByMilestoneID", uint(10)).Return(nil, gorm.ErrRecordNotFound)
	projRepo.On("FindMilestoneByID", uint(10)).Return(milestone, nil)
	projRepo.On("FindProjectByID", uint(5)).Return((*domain.Project)(nil), errors.New("not found"))

	_, err := svc.CreateForMilestone(10)

	assert.EqualError(t, err, "project not found")
}

func TestCreateForMilestone_LookupDBError_Propagates(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	disbRepo.On("FindByMilestoneID", uint(10)).Return(nil, errors.New("db exploded"))

	_, err := svc.CreateForMilestone(10)

	assert.EqualError(t, err, "db exploded")
}

func TestCreateForMilestone_CreateDBError(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	milestone := &domain.Milestone{ID: 10, ProjectID: 5, PhaseNo: 1, PercentRelease: 25}
	project := &domain.Project{ID: 5, OwnerUserID: 99, CurrentFunding: 4000}

	disbRepo.On("FindByMilestoneID", uint(10)).Return(nil, gorm.ErrRecordNotFound)
	projRepo.On("FindMilestoneByID", uint(10)).Return(milestone, nil)
	projRepo.On("FindProjectByID", uint(5)).Return(project, nil)
	disbRepo.On("Create", mock.AnythingOfType("*domain.Disbursement")).Return(errors.New("insert failed"))

	_, err := svc.CreateForMilestone(10)

	assert.EqualError(t, err, "insert failed")
}

// ─── ListAll ─────────────────────────────────────────────────────────────────

func TestListAll_Success_EnrichesWithProjectAndPioneer(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	list := []domain.Disbursement{
		{ID: 1, MilestoneID: 10, ProjectID: 5, PioneerUserID: 99, Amount: 4000, Status: domain.DisbursementPending},
	}
	project := &domain.Project{ID: 5, Title: "FlyUp Prototype"}
	pioneer := &domain.User{ID: 99, FirstName: "Jane", LastName: "Smith", Email: "jane@test.com"}
	banks := []domain.BankAccount{
		{BankName: "KBank", AccountName: "Jane Smith", AccountNumber: "1234567890"},
	}

	disbRepo.On("ListAll").Return(list, nil)
	projRepo.On("FindProjectByID", uint(5)).Return(project, nil)
	userRepo.On("FindUserById", uint(99)).Return(pioneer, nil)
	userRepo.On("FindBankByUserId", uint(99)).Return(banks, nil)

	items, err := svc.ListAll()

	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "FlyUp Prototype", items[0].ProjectTitle)
	assert.Equal(t, "Jane Smith", items[0].PioneerName)
	assert.Equal(t, "jane@test.com", items[0].PioneerEmail)
	assert.NotNil(t, items[0].BankAccount)
	assert.Equal(t, "KBank", items[0].BankAccount.BankName)
}

func TestListAll_Success_Empty(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	disbRepo.On("ListAll").Return([]domain.Disbursement{}, nil)

	items, err := svc.ListAll()

	assert.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestListAll_Success_LookupFailuresTolerated(t *testing.T) {
	// project / user / bank lookups failing must not break the list —
	// the base row should still come through with empty enrichments
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	list := []domain.Disbursement{
		{ID: 1, ProjectID: 5, PioneerUserID: 99, Amount: 4000, Status: domain.DisbursementPending},
	}

	disbRepo.On("ListAll").Return(list, nil)
	projRepo.On("FindProjectByID", uint(5)).Return((*domain.Project)(nil), errors.New("gone"))
	userRepo.On("FindUserById", uint(99)).Return((*domain.User)(nil), errors.New("gone"))
	userRepo.On("FindBankByUserId", uint(99)).Return([]domain.BankAccount{}, errors.New("gone"))

	items, err := svc.ListAll()

	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "", items[0].ProjectTitle)
	assert.Equal(t, "", items[0].PioneerName)
	assert.Nil(t, items[0].BankAccount)
}

func TestListAll_DBError(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	disbRepo.On("ListAll").Return(nil, errors.New("db error"))

	_, err := svc.ListAll()

	assert.EqualError(t, err, "internal server error")
}

// ─── ListPending ─────────────────────────────────────────────────────────────

func TestListPending_Success(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	list := []domain.Disbursement{
		{ID: 1, ProjectID: 5, PioneerUserID: 99, Amount: 2000, Status: domain.DisbursementPending},
	}
	project := &domain.Project{ID: 5, Title: "Pending Project"}
	pioneer := &domain.User{ID: 99, FirstName: "Bob", LastName: "Lee", Email: "bob@test.com"}

	disbRepo.On("ListByStatus", domain.DisbursementPending).Return(list, nil)
	projRepo.On("FindProjectByID", uint(5)).Return(project, nil)
	userRepo.On("FindUserById", uint(99)).Return(pioneer, nil)
	userRepo.On("FindBankByUserId", uint(99)).Return([]domain.BankAccount{}, nil)

	items, err := svc.ListPending()

	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Pending Project", items[0].ProjectTitle)
	assert.Equal(t, "Bob Lee", items[0].PioneerName)
	assert.Nil(t, items[0].BankAccount) // empty bank list → nil
}

func TestListPending_DBError(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	disbRepo.On("ListByStatus", domain.DisbursementPending).Return(nil, errors.New("db error"))

	_, err := svc.ListPending()

	assert.EqualError(t, err, "internal server error")
}

// ─── Confirm ─────────────────────────────────────────────────────────────────

func TestConfirm_Success(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	d := &domain.Disbursement{ID: 1, Status: domain.DisbursementPending}
	adminID := uint(42)
	req := dto.ConfirmDisbursementRequest{TransferRef: "TRF-001", Note: "paid via SCB"}

	disbRepo.On("FindByID", uint(1)).Return(d, nil)
	disbRepo.On("Update", mock.MatchedBy(func(updated *domain.Disbursement) bool {
		return updated.ID == 1 &&
			updated.Status == domain.DisbursementConfirmed &&
			updated.TransferRef == "TRF-001" &&
			updated.AdminNote == "paid via SCB" &&
			updated.ConfirmedBy != nil && *updated.ConfirmedBy == adminID &&
			updated.ConfirmedAt != nil
	})).Return(nil)

	result, err := svc.Confirm(1, adminID, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, domain.DisbursementConfirmed, result.Status)
	assert.Equal(t, "TRF-001", result.TransferRef)
	disbRepo.AssertExpectations(t)
}

func TestConfirm_NotFound(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	disbRepo.On("FindByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)

	_, err := svc.Confirm(99, 42, dto.ConfirmDisbursementRequest{TransferRef: "X"})

	assert.EqualError(t, err, "disbursement not found")
}

func TestConfirm_AlreadyConfirmed(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	d := &domain.Disbursement{ID: 1, Status: domain.DisbursementConfirmed}
	disbRepo.On("FindByID", uint(1)).Return(d, nil)

	_, err := svc.Confirm(1, 42, dto.ConfirmDisbursementRequest{TransferRef: "X"})

	assert.EqualError(t, err, "disbursement already confirmed")
	disbRepo.AssertNotCalled(t, "Update", mock.Anything)
}

func TestConfirm_UpdateDBError(t *testing.T) {
	disbRepo := new(mockDisbursementRepo)
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	svc := newTestDisbursementService(disbRepo, projRepo, userRepo)

	d := &domain.Disbursement{ID: 1, Status: domain.DisbursementPending}
	disbRepo.On("FindByID", uint(1)).Return(d, nil)
	disbRepo.On("Update", mock.AnythingOfType("*domain.Disbursement")).Return(errors.New("db error"))

	_, err := svc.Confirm(1, 42, dto.ConfirmDisbursementRequest{TransferRef: "X"})

	assert.EqualError(t, err, "failed to confirm disbursement")
}

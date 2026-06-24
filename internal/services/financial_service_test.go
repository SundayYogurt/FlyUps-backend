package services

import (
	"errors"
	"flyup/internal/repository"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock: FinancialRepository

type mockFinancialRepo struct{ mock.Mock }

func (m *mockFinancialRepo) GetDisbursementStat(status string) (repository.FinancialStat, error) {
	args := m.Called(status)
	return args.Get(0).(repository.FinancialStat), args.Error(1)
}

func (m *mockFinancialRepo) GetInvestmentRefundStat(status string) (repository.FinancialStat, error) {
	args := m.Called(status)
	return args.Get(0).(repository.FinancialStat), args.Error(1)
}

func (m *mockFinancialRepo) GetTransactionFees() (float64, error) {
	args := m.Called()
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockFinancialRepo) GetTransactionRevenue() (float64, error) {
	args := m.Called()
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockFinancialRepo) GetProjectsFinancialRows() ([]repository.FinancialProjectRow, error) {
	args := m.Called()
	return args.Get(0).([]repository.FinancialProjectRow), args.Error(1)
}

// GetSummary

func TestFinancialService_GetSummary(t *testing.T) {
	repo := new(mockFinancialRepo)
	svc := NewFinancialService(repo, "")

	repo.On("GetDisbursementStat", "confirmed").Return(repository.FinancialStat{Total: 1000, Count: 5}, nil)
	repo.On("GetDisbursementStat", "pending").Return(repository.FinancialStat{Total: 500, Count: 2}, nil)
	repo.On("GetInvestmentRefundStat", "refund_pending").Return(repository.FinancialStat{Total: 200, Count: 1}, nil)
	repo.On("GetInvestmentRefundStat", "refunded").Return(repository.FinancialStat{Total: 150, Count: 3}, nil)
	repo.On("GetTransactionFees").Return(float64(80), nil)
	repo.On("GetTransactionRevenue").Return(float64(920), nil)

	summary, err := svc.GetSummary()

	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, float64(1000), summary.Disbursement.TotalConfirmed)
	assert.Equal(t, int64(5), summary.Disbursement.CountConfirmed)
	assert.Equal(t, float64(500), summary.Disbursement.TotalPending)
	assert.Equal(t, int64(2), summary.Disbursement.CountPending)
	assert.Equal(t, float64(200), summary.Refund.TotalPending)
	assert.Equal(t, int64(1), summary.Refund.CountPending)
	assert.Equal(t, float64(150), summary.Refund.TotalRefunded)
	assert.Equal(t, int64(3), summary.Refund.CountRefunded)
	assert.Equal(t, float64(80), summary.Platform.TotalFees)
	assert.Equal(t, float64(920), summary.Platform.TotalRevenue)
	repo.AssertExpectations(t)
}

// GetProjectsFinancial

func TestFinancialService_GetProjectsFinancial_Success(t *testing.T) {
	repo := new(mockFinancialRepo)
	svc := NewFinancialService(repo, "")

	confAt := "2024-01-15"
	rows := []repository.FinancialProjectRow{
		{ProjectID: 1, ProjectTitle: "Project A", State: "executing", FundingGoal: 100000, CurrentFunding: 80000, PhaseNo: 1, MilestoneTitle: "Phase 1", PercentRelease: 30, Amount: 30000, DisbStatus: "confirmed", ConfirmedAt: &confAt},
		{ProjectID: 1, ProjectTitle: "Project A", State: "executing", FundingGoal: 100000, CurrentFunding: 80000, PhaseNo: 2, MilestoneTitle: "Phase 2", PercentRelease: 70, Amount: 0, DisbStatus: "not_started", ConfirmedAt: nil},
		{ProjectID: 2, ProjectTitle: "Project B", State: "funded", FundingGoal: 50000, CurrentFunding: 50000, PhaseNo: 1, MilestoneTitle: "Phase 1", PercentRelease: 100, Amount: 50000, DisbStatus: "pending", ConfirmedAt: nil},
	}
	repo.On("GetProjectsFinancialRows").Return(rows, nil)

	result, err := svc.GetProjectsFinancial()

	assert.NoError(t, err)
	assert.Len(t, result, 2)

	assert.Equal(t, uint(1), result[0].ProjectID)
	assert.Equal(t, "Project A", result[0].ProjectTitle)
	assert.Len(t, result[0].Phases, 2)
	assert.Equal(t, 1, result[0].Phases[0].PhaseNo)
	assert.Equal(t, "confirmed", result[0].Phases[0].Status)
	assert.Equal(t, &confAt, result[0].Phases[0].ConfirmedAt)
	assert.Equal(t, "not_started", result[0].Phases[1].Status)

	assert.Equal(t, uint(2), result[1].ProjectID)
	assert.Equal(t, "Project B", result[1].ProjectTitle)
	assert.Len(t, result[1].Phases, 1)
	assert.Equal(t, "pending", result[1].Phases[0].Status)

	repo.AssertExpectations(t)
}

func TestFinancialService_GetProjectsFinancial_Empty(t *testing.T) {
	repo := new(mockFinancialRepo)
	svc := NewFinancialService(repo, "")

	repo.On("GetProjectsFinancialRows").Return([]repository.FinancialProjectRow{}, nil)

	result, err := svc.GetProjectsFinancial()

	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestFinancialService_GetProjectsFinancial_RepoError(t *testing.T) {
	repo := new(mockFinancialRepo)
	svc := NewFinancialService(repo, "")

	repo.On("GetProjectsFinancialRows").Return([]repository.FinancialProjectRow(nil), errors.New("connection failed"))

	result, err := svc.GetProjectsFinancial()

	assert.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

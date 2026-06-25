package services

import (
	"errors"
	"flyup/internal/dto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock: AdminBadgeRepository ---

type mockAdminBadgeRepo struct{ mock.Mock }

func (m *mockAdminBadgeRepo) CountPendingProjects() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockAdminBadgeRepo) CountSubmittedMilestones() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockAdminBadgeRepo) CountPendingCancelReqs() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockAdminBadgeRepo) CountOpenComplaints() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockAdminBadgeRepo) CountPendingRefunds() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockAdminBadgeRepo) CountPendingDisbursements() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockAdminBadgeRepo) CountPendingProfitPools() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockAdminBadgeRepo) CountPendingVerifications() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

// --- Mock: BoosterBadgeRepository ---

type mockBoosterBadgeRepo struct{ mock.Mock }

func (m *mockBoosterBadgeRepo) CountPendingVotes(boosterID uint) (int64, error) {
	args := m.Called(boosterID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockBoosterBadgeRepo) CountUpcomingMeetings(boosterID uint, today string) (int64, error) {
	args := m.Called(boosterID, today)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockBoosterBadgeRepo) CountPendingRefunds(boosterID uint) (int64, error) {
	args := m.Called(boosterID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockBoosterBadgeRepo) CountOpenComplaints(boosterID uint) (int64, error) {
	args := m.Called(boosterID)
	return args.Get(0).(int64), args.Error(1)
}

// --- Mock: PioneerBadgeRepository ---

type mockPioneerBadgeRepo struct{ mock.Mock }

func (m *mockPioneerBadgeRepo) CountActiveMilestones(pioneerID uint) (int64, error) {
	args := m.Called(pioneerID)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockPioneerBadgeRepo) CountUpcomingMeetings(pioneerID uint, today string) (int64, error) {
	args := m.Called(pioneerID, today)
	return args.Get(0).(int64), args.Error(1)
}
func (m *mockPioneerBadgeRepo) CountPendingPayouts(pioneerID uint) (int64, error) {
	args := m.Called(pioneerID)
	return args.Get(0).(int64), args.Error(1)
}

// --- AdminBadgeService tests ---

func TestAdminBadgeService_GetCounts_Success(t *testing.T) {
	repo := new(mockAdminBadgeRepo)
	svc := NewAdminBadgeService(repo)

	repo.On("CountPendingProjects").Return(int64(3), nil)
	repo.On("CountSubmittedMilestones").Return(int64(1), nil)
	repo.On("CountPendingCancelReqs").Return(int64(2), nil)
	repo.On("CountOpenComplaints").Return(int64(4), nil)
	repo.On("CountPendingRefunds").Return(int64(5), nil)
	repo.On("CountPendingDisbursements").Return(int64(6), nil)
	repo.On("CountPendingProfitPools").Return(int64(7), nil)
	repo.On("CountPendingVerifications").Return(int64(8), nil)

	counts, err := svc.GetCounts()

	assert.NoError(t, err)
	assert.Equal(t, dto.AdminBadgeCounts{
		PendingProjects:      3,
		SubmittedMilestones:  1,
		PendingCancelReqs:    2,
		OpenComplaints:       4,
		PendingRefunds:       5,
		PendingDisbursements: 6,
		PendingProfitPools:   7,
		PendingVerifications: 8,
	}, counts)
	repo.AssertExpectations(t)
}

func TestAdminBadgeService_GetCounts_RepoError_StopsEarly(t *testing.T) {
	repo := new(mockAdminBadgeRepo)
	svc := NewAdminBadgeService(repo)

	repo.On("CountPendingProjects").Return(int64(3), nil)
	repo.On("CountSubmittedMilestones").Return(int64(0), errors.New("db error"))
	// ต้องไม่เรียก method ที่เหลือเมื่อ error แล้ว

	counts, err := svc.GetCounts()

	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
	assert.Equal(t, int64(3), counts.PendingProjects)  // field ก่อน error ยังคงค่าเดิม
	assert.Equal(t, int64(0), counts.SubmittedMilestones)
	repo.AssertExpectations(t)
}

// --- BoosterBadgeService tests ---

func TestBoosterBadgeService_GetCounts_Success(t *testing.T) {
	repo := new(mockBoosterBadgeRepo)
	svc := NewBoosterBadgeService(repo)

	repo.On("CountPendingVotes", uint(1)).Return(int64(2), nil)
	repo.On("CountUpcomingMeetings", uint(1), mock.AnythingOfType("string")).Return(int64(3), nil)
	repo.On("CountPendingRefunds", uint(1)).Return(int64(1), nil)
	repo.On("CountOpenComplaints", uint(1)).Return(int64(0), nil)

	counts, err := svc.GetCounts(1)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), counts.PendingVotes)
	assert.Equal(t, int64(3), counts.UpcomingMeetings)
	assert.Equal(t, int64(1), counts.PendingRefunds)
	assert.Equal(t, int64(0), counts.OpenComplaints)
	repo.AssertExpectations(t)
}

func TestBoosterBadgeService_GetCounts_RepoError(t *testing.T) {
	repo := new(mockBoosterBadgeRepo)
	svc := NewBoosterBadgeService(repo)

	repo.On("CountPendingVotes", uint(1)).Return(int64(0), errors.New("db error"))

	_, err := svc.GetCounts(1)

	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestBoosterBadgeService_GetCounts_AllZero(t *testing.T) {
	repo := new(mockBoosterBadgeRepo)
	svc := NewBoosterBadgeService(repo)

	repo.On("CountPendingVotes", uint(5)).Return(int64(0), nil)
	repo.On("CountUpcomingMeetings", uint(5), mock.AnythingOfType("string")).Return(int64(0), nil)
	repo.On("CountPendingRefunds", uint(5)).Return(int64(0), nil)
	repo.On("CountOpenComplaints", uint(5)).Return(int64(0), nil)

	counts, err := svc.GetCounts(5)

	assert.NoError(t, err)
	assert.Equal(t, dto.BoosterBadgeCounts{}, counts)
	repo.AssertExpectations(t)
}

// --- PioneerBadgeService tests ---

func TestPioneerBadgeService_GetCounts_Success(t *testing.T) {
	repo := new(mockPioneerBadgeRepo)
	svc := NewPioneerBadgeService(repo)

	repo.On("CountActiveMilestones", uint(2)).Return(int64(4), nil)
	repo.On("CountUpcomingMeetings", uint(2), mock.AnythingOfType("string")).Return(int64(1), nil)
	repo.On("CountPendingPayouts", uint(2)).Return(int64(2), nil)

	counts, err := svc.GetCounts(2)

	assert.NoError(t, err)
	assert.Equal(t, int64(4), counts.ActiveMilestones)
	assert.Equal(t, int64(1), counts.UpcomingMeetings)
	assert.Equal(t, int64(2), counts.PendingPayouts)
	repo.AssertExpectations(t)
}

func TestPioneerBadgeService_GetCounts_RepoError(t *testing.T) {
	repo := new(mockPioneerBadgeRepo)
	svc := NewPioneerBadgeService(repo)

	repo.On("CountActiveMilestones", uint(2)).Return(int64(0), errors.New("db error"))

	_, err := svc.GetCounts(2)

	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestPioneerBadgeService_GetCounts_AllZero(t *testing.T) {
	repo := new(mockPioneerBadgeRepo)
	svc := NewPioneerBadgeService(repo)

	repo.On("CountActiveMilestones", uint(9)).Return(int64(0), nil)
	repo.On("CountUpcomingMeetings", uint(9), mock.AnythingOfType("string")).Return(int64(0), nil)
	repo.On("CountPendingPayouts", uint(9)).Return(int64(0), nil)

	counts, err := svc.GetCounts(9)

	assert.NoError(t, err)
	assert.Equal(t, dto.PioneerBadgeCounts{}, counts)
	repo.AssertExpectations(t)
}

package services

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock: AdminLogRepository

type mockAdminLogRepo struct{ mock.Mock }

func (m *mockAdminLogRepo) Create(log *domain.AdminLog) error {
	return m.Called(log).Error(0)
}

func (m *mockAdminLogRepo) ListAll(filter dto.AdminLogFilter) ([]domain.AdminLog, int64, error) {
	args := m.Called(filter)
	return args.Get(0).([]domain.AdminLog), args.Get(1).(int64), args.Error(2)
}

// LogAction

func TestAdminLogService_LogAction_Success(t *testing.T) {
	repo := new(mockAdminLogRepo)
	svc := NewAdminLogService(repo)

	repo.On("Create", mock.Anything).Return(nil)

	svc.LogAction(1, "approve_project", "project", nil, nil)

	repo.AssertExpectations(t)
}

func TestAdminLogService_LogAction_RepoError_Swallowed(t *testing.T) {
	repo := new(mockAdminLogRepo)
	svc := NewAdminLogService(repo)

	// error จาก repo ต้องไม่ panic และไม่ return ออกมา
	repo.On("Create", mock.Anything).Return(errors.New("db error"))

	assert.NotPanics(t, func() {
		svc.LogAction(1, "action", "type", nil, nil)
	})

	repo.AssertExpectations(t)
}

// ListLogs

func TestAdminLogService_ListLogs_Success(t *testing.T) {
	repo := new(mockAdminLogRepo)
	svc := NewAdminLogService(repo)

	adminUser := &domain.User{ID: 1, FirstName: "Admin", LastName: "User", Email: "admin@test.com"}
	logs := []domain.AdminLog{
		{ID: 1, AdminID: 1, Action: "approve_project", TargetType: "project", Admin: adminUser},
		{ID: 2, AdminID: 1, Action: "reject_project", TargetType: "project", Admin: nil},
	}
	filter := dto.AdminLogFilter{Page: 1, PageSize: 20}
	repo.On("ListAll", filter).Return(logs, int64(2), nil)

	items, total, err := svc.ListLogs(filter)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, items, 2)
	assert.Equal(t, "approve_project", items[0].Action)
	assert.NotNil(t, items[0].Admin)
	assert.Equal(t, "Admin", items[0].Admin.FirstName)
	assert.Nil(t, items[1].Admin)
	repo.AssertExpectations(t)
}

func TestAdminLogService_ListLogs_Empty(t *testing.T) {
	repo := new(mockAdminLogRepo)
	svc := NewAdminLogService(repo)

	filter := dto.AdminLogFilter{Page: 1, PageSize: 20}
	repo.On("ListAll", filter).Return([]domain.AdminLog{}, int64(0), nil)

	items, total, err := svc.ListLogs(filter)

	assert.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, items)
	repo.AssertExpectations(t)
}

func TestAdminLogService_ListLogs_RepoError(t *testing.T) {
	repo := new(mockAdminLogRepo)
	svc := NewAdminLogService(repo)

	filter := dto.AdminLogFilter{}
	repo.On("ListAll", filter).Return([]domain.AdminLog(nil), int64(0), errors.New("db error"))

	items, total, err := svc.ListLogs(filter)

	assert.Error(t, err)
	assert.Nil(t, items)
	assert.Equal(t, int64(0), total)
	repo.AssertExpectations(t)
}

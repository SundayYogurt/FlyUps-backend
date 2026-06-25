package services

import (
	"errors"
	"flyup/internal/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock: NotificationRepository

type mockNotificationRepo struct{ mock.Mock }

func (m *mockNotificationRepo) Create(n *domain.Notification) error {
	return m.Called(n).Error(0)
}

func (m *mockNotificationRepo) FindByUserID(userID uint, limit, offset int) ([]domain.Notification, int64, error) {
	args := m.Called(userID, limit, offset)
	return args.Get(0).([]domain.Notification), args.Get(1).(int64), args.Error(2)
}

func (m *mockNotificationRepo) MarkAsRead(userID uint, notifID uint) error {
	return m.Called(userID, notifID).Error(0)
}

func (m *mockNotificationRepo) MarkAllAsRead(userID uint) error {
	return m.Called(userID).Error(0)
}

func (m *mockNotificationRepo) CountUnread(userID uint) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

// CreateAndPush

func TestNotificationService_CreateAndPush_Success(t *testing.T) {
	repo := new(mockNotificationRepo)
	svc := NewNotificationService(repo)

	repo.On("Create", mock.Anything).Return(nil)

	err := svc.CreateAndPush(1, domain.NotifProfit, "title", "body", nil, nil)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestNotificationService_CreateAndPush_RepoError(t *testing.T) {
	repo := new(mockNotificationRepo)
	svc := NewNotificationService(repo)

	repo.On("Create", mock.Anything).Return(errors.New("db error"))

	err := svc.CreateAndPush(1, domain.NotifProfit, "title", "body", nil, nil)

	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestNotificationService_CreateAndPush_BroadcastToSubscriber(t *testing.T) {
	repo := new(mockNotificationRepo)
	svc := NewNotificationService(repo)

	repo.On("Create", mock.Anything).Return(nil)

	ch := svc.Subscribe(1)
	defer svc.Unsubscribe(1, ch)

	_ = svc.CreateAndPush(1, domain.NotifProfit, "กำไร", "body", nil, nil)

	select {
	case notif := <-ch:
		assert.Equal(t, uint(1), notif.UserID)
		assert.Equal(t, "กำไร", notif.Title)
	case <-time.After(time.Second):
		t.Fatal("timeout: notification not received on channel")
	}
}

// GetNotifications

func TestNotificationService_GetNotifications_Success(t *testing.T) {
	repo := new(mockNotificationRepo)
	svc := NewNotificationService(repo)

	notifs := []domain.Notification{{ID: 1, UserID: 1, Title: "test"}}
	repo.On("FindByUserID", uint(1), 20, 0).Return(notifs, int64(1), nil)

	result, total, err := svc.GetNotifications(1, 1, 20)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}

func TestNotificationService_GetNotifications_PageNormalized(t *testing.T) {
	repo := new(mockNotificationRepo)
	svc := NewNotificationService(repo)

	// page <= 0 ถูก normalize เป็น 1 → offset = 0
	repo.On("FindByUserID", uint(1), 10, 0).Return([]domain.Notification{}, int64(0), nil)

	_, _, err := svc.GetNotifications(1, 0, 10)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// MarkAsRead

func TestNotificationService_MarkAsRead_Success(t *testing.T) {
	repo := new(mockNotificationRepo)
	svc := NewNotificationService(repo)

	repo.On("MarkAsRead", uint(1), uint(5)).Return(nil)

	err := svc.MarkAsRead(1, 5)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestNotificationService_MarkAsRead_RepoError(t *testing.T) {
	repo := new(mockNotificationRepo)
	svc := NewNotificationService(repo)

	repo.On("MarkAsRead", uint(1), uint(5)).Return(errors.New("db error"))

	err := svc.MarkAsRead(1, 5)

	assert.Error(t, err)
	repo.AssertExpectations(t)
}

// MarkAllAsRead

func TestNotificationService_MarkAllAsRead_Success(t *testing.T) {
	repo := new(mockNotificationRepo)
	svc := NewNotificationService(repo)

	repo.On("MarkAllAsRead", uint(1)).Return(nil)

	err := svc.MarkAllAsRead(1)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// CountUnread

func TestNotificationService_CountUnread_Success(t *testing.T) {
	repo := new(mockNotificationRepo)
	svc := NewNotificationService(repo)

	repo.On("CountUnread", uint(1)).Return(int64(3), nil)

	count, err := svc.CountUnread(1)

	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
	repo.AssertExpectations(t)
}

// Subscribe / Unsubscribe

func TestNotificationService_Subscribe_ReturnsChannel(t *testing.T) {
	repo := new(mockNotificationRepo)
	svc := NewNotificationService(repo)

	ch := svc.Subscribe(1)

	assert.NotNil(t, ch)

	// cleanup
	svc.Unsubscribe(1, ch)
}

func TestNotificationService_Unsubscribe_ClosesChannel(t *testing.T) {
	repo := new(mockNotificationRepo)
	svc := NewNotificationService(repo)

	ch := svc.Subscribe(1)
	svc.Unsubscribe(1, ch)

	// channel ถูก close แล้ว → receive จะได้ zero value ทันทีโดยไม่ block
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed after Unsubscribe")
}

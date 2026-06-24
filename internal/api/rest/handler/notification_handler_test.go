package handler

import (
	"flyup/internal/domain"
	"flyup/internal/helper"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock: NotificationService

type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) CreateAndPush(userID uint, notifType domain.NotificationType, title, body string, relatedID *uint, relatedType *string) error {
	return m.Called(userID, notifType, title, body, relatedID, relatedType).Error(0)
}

func (m *MockNotificationService) GetNotifications(userID uint, page, limit int) ([]domain.Notification, int64, error) {
	args := m.Called(userID, page, limit)
	return args.Get(0).([]domain.Notification), args.Get(1).(int64), args.Error(2)
}

func (m *MockNotificationService) MarkAsRead(userID uint, notifID uint) error {
	return m.Called(userID, notifID).Error(0)
}

func (m *MockNotificationService) MarkAllAsRead(userID uint) error {
	return m.Called(userID).Error(0)
}

func (m *MockNotificationService) CountUnread(userID uint) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotificationService) Subscribe(userID uint) chan *domain.Notification {
	args := m.Called(userID)
	return args.Get(0).(chan *domain.Notification)
}

func (m *MockNotificationService) Unsubscribe(userID uint, ch chan *domain.Notification) {
	m.Called(userID, ch)
}

// Setup

func setupNotificationTest(t *testing.T) (*fiber.App, *MockNotificationService, *NotificationHandler) {
	app := fiber.New()
	mockSvc := new(MockNotificationService)
	h := &NotificationHandler{
		svc:  mockSvc,
		auth: helper.Auth{Secret: testSecret},
		rh:   nil,
	}
	return app, mockSvc, h
}

// List

func TestNotificationHandler_List(t *testing.T) {
	app, mockSvc, h := setupNotificationTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/notifications", h.List)

	mockSvc.On("GetNotifications", uint(1), 1, 20).Return([]domain.Notification{}, int64(0), nil)
	mockSvc.On("CountUnread", uint(1)).Return(int64(0), nil)
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNotificationHandler_List_Unauthorized(t *testing.T) {
	app, _, h := setupNotificationTest(t)
	app.Get("/notifications", h.List)

	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// MarkAsRead

func TestNotificationHandler_MarkAsRead(t *testing.T) {
	app, mockSvc, h := setupNotificationTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Patch("/notifications/:id/read", h.MarkAsRead)

	mockSvc.On("MarkAsRead", uint(1), uint(5)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/notifications/5/read", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNotificationHandler_MarkAsRead_Unauthorized(t *testing.T) {
	app, _, h := setupNotificationTest(t)
	app.Patch("/notifications/:id/read", h.MarkAsRead)

	req := httptest.NewRequest(http.MethodPatch, "/notifications/5/read", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestNotificationHandler_MarkAsRead_InvalidID(t *testing.T) {
	app, _, h := setupNotificationTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Patch("/notifications/:id/read", h.MarkAsRead)

	req := httptest.NewRequest(http.MethodPatch, "/notifications/abc/read", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// MarkAllAsRead

func TestNotificationHandler_MarkAllAsRead(t *testing.T) {
	app, mockSvc, h := setupNotificationTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Patch("/notifications/read-all", h.MarkAllAsRead)

	mockSvc.On("MarkAllAsRead", uint(1)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/notifications/read-all", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNotificationHandler_MarkAllAsRead_Unauthorized(t *testing.T) {
	app, _, h := setupNotificationTest(t)
	app.Patch("/notifications/read-all", h.MarkAllAsRead)

	req := httptest.NewRequest(http.MethodPatch, "/notifications/read-all", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

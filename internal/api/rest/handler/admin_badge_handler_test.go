package handler

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock: AdminBadgeService

type MockAdminBadgeService struct {
	mock.Mock
}

func (m *MockAdminBadgeService) GetCounts() (dto.AdminBadgeCounts, error) {
	args := m.Called()
	return args.Get(0).(dto.AdminBadgeCounts), args.Error(1)
}

// Setup

func setupAdminBadgeTest(t *testing.T) (*fiber.App, *MockAdminBadgeService, *AdminBadgeHandler) {
	app := fiber.New()
	mockSvc := new(MockAdminBadgeService)
	h := &AdminBadgeHandler{
		svc:  mockSvc,
		auth: helper.Auth{Secret: testSecret},
	}
	return app, mockSvc, h
}

func TestAdminBadgeHandler_GetBadgeCounts(t *testing.T) {
	app, mockSvc, h := setupAdminBadgeTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/badges", h.GetBadgeCounts)

	mockSvc.On("GetCounts").Return(dto.AdminBadgeCounts{
		PendingProjects:    3,
		OpenComplaints:     1,
		PendingRefunds:     2,
		PendingProfitPools: 0,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/badges", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAdminBadgeHandler_GetBadgeCounts_Unauthorized(t *testing.T) {
	app, _, h := setupAdminBadgeTest(t)
	app.Get("/admin/badges", h.GetBadgeCounts)

	req := httptest.NewRequest(http.MethodGet, "/admin/badges", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdminBadgeHandler_GetBadgeCounts_ServiceError(t *testing.T) {
	app, mockSvc, h := setupAdminBadgeTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/badges", h.GetBadgeCounts)

	mockSvc.On("GetCounts").Return(dto.AdminBadgeCounts{}, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/admin/badges", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

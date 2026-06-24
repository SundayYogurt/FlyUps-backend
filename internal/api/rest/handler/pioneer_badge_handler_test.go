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

// Mock: PioneerBadgeService

type MockPioneerBadgeService struct {
	mock.Mock
}

func (m *MockPioneerBadgeService) GetCounts(userID uint) (dto.PioneerBadgeCounts, error) {
	args := m.Called(userID)
	return args.Get(0).(dto.PioneerBadgeCounts), args.Error(1)
}

// Setup

func setupPioneerBadgeTest(t *testing.T) (*fiber.App, *MockPioneerBadgeService, *PioneerBadgeHandler) {
	app := fiber.New()
	mockSvc := new(MockPioneerBadgeService)
	h := &PioneerBadgeHandler{
		svc:  mockSvc,
		auth: helper.Auth{Secret: testSecret},
	}
	return app, mockSvc, h
}

func TestPioneerBadgeHandler_GetBadgeCounts(t *testing.T) {
	app, mockSvc, h := setupPioneerBadgeTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 3, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/pioneer/badges", h.GetBadgeCounts)

	mockSvc.On("GetCounts", uint(3)).Return(dto.PioneerBadgeCounts{
		ActiveMilestones: 2,
		UpcomingMeetings: 1,
		PendingPayouts:   3,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/pioneer/badges", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPioneerBadgeHandler_GetBadgeCounts_Unauthorized(t *testing.T) {
	app, _, h := setupPioneerBadgeTest(t)
	app.Get("/pioneer/badges", h.GetBadgeCounts)

	req := httptest.NewRequest(http.MethodGet, "/pioneer/badges", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestPioneerBadgeHandler_GetBadgeCounts_ServiceError(t *testing.T) {
	app, mockSvc, h := setupPioneerBadgeTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 3, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/pioneer/badges", h.GetBadgeCounts)

	mockSvc.On("GetCounts", uint(3)).Return(dto.PioneerBadgeCounts{}, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/pioneer/badges", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

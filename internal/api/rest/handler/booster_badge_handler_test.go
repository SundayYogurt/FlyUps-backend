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

// Mock: BoosterBadgeService

type MockBoosterBadgeService struct {
	mock.Mock
}

func (m *MockBoosterBadgeService) GetCounts(userID uint) (dto.BoosterBadgeCounts, error) {
	args := m.Called(userID)
	return args.Get(0).(dto.BoosterBadgeCounts), args.Error(1)
}

// Setup

func setupBoosterBadgeTest(t *testing.T) (*fiber.App, *MockBoosterBadgeService, *BoosterBadgeHandler) {
	app := fiber.New()
	mockSvc := new(MockBoosterBadgeService)
	h := &BoosterBadgeHandler{
		svc:  mockSvc,
		auth: helper.Auth{Secret: testSecret},
	}
	return app, mockSvc, h
}

func TestBoosterBadgeHandler_GetBadgeCounts(t *testing.T) {
	app, mockSvc, h := setupBoosterBadgeTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 2, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/booster/badges", h.GetBadgeCounts)

	mockSvc.On("GetCounts", uint(2)).Return(dto.BoosterBadgeCounts{
		PendingVotes:     1,
		UpcomingMeetings: 2,
		PendingRefunds:   0,
		OpenComplaints:   1,
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/booster/badges", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestBoosterBadgeHandler_GetBadgeCounts_Unauthorized(t *testing.T) {
	app, _, h := setupBoosterBadgeTest(t)
	app.Get("/booster/badges", h.GetBadgeCounts)

	req := httptest.NewRequest(http.MethodGet, "/booster/badges", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestBoosterBadgeHandler_GetBadgeCounts_ServiceError(t *testing.T) {
	app, mockSvc, h := setupBoosterBadgeTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 2, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/booster/badges", h.GetBadgeCounts)

	mockSvc.On("GetCounts", uint(2)).Return(dto.BoosterBadgeCounts{}, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/booster/badges", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

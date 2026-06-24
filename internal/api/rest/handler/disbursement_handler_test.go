package handler

import (
	"bytes"
	"encoding/json"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock: DisbursementService

type MockDisbursementService struct {
	mock.Mock
}

func (m *MockDisbursementService) CreateForMilestone(milestoneID uint) (*domain.Disbursement, error) {
	args := m.Called(milestoneID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Disbursement), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDisbursementService) ListAll() ([]dto.DisbursementItem, error) {
	args := m.Called()
	return args.Get(0).([]dto.DisbursementItem), args.Error(1)
}

func (m *MockDisbursementService) ListPending() ([]dto.DisbursementItem, error) {
	args := m.Called()
	return args.Get(0).([]dto.DisbursementItem), args.Error(1)
}

func (m *MockDisbursementService) Confirm(disbursementID uint, adminID uint, req dto.ConfirmDisbursementRequest) (*domain.Disbursement, error) {
	args := m.Called(disbursementID, adminID, req)
	if v := args.Get(0); v != nil {
		return v.(*domain.Disbursement), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockDisbursementService) ListMyPayouts(pioneerID uint) ([]dto.PioneerPayoutItem, error) {
	args := m.Called(pioneerID)
	return args.Get(0).([]dto.PioneerPayoutItem), args.Error(1)
}

// Setup

func setupDisbursementTest(t *testing.T) (*fiber.App, *MockDisbursementService, *DisbursementHandler) {
	app := fiber.New()
	mockSvc := new(MockDisbursementService)
	h := &DisbursementHandler{
		svc:       mockSvc,
		validator: validator.New(),
		auth:      helper.Auth{Secret: testSecret},
	}
	return app, mockSvc, h
}

// ListAll

func TestDisbursementHandler_ListAll(t *testing.T) {
	app, mockSvc, h := setupDisbursementTest(t)
	app.Get("/admin/disbursements", h.ListAll)

	mockSvc.On("ListAll").Return([]dto.DisbursementItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/disbursements", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ListPending

func TestDisbursementHandler_ListPending(t *testing.T) {
	app, mockSvc, h := setupDisbursementTest(t)
	app.Get("/admin/disbursements/pending", h.ListPending)

	mockSvc.On("ListPending").Return([]dto.DisbursementItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/disbursements/pending", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// Confirm

func TestDisbursementHandler_Confirm(t *testing.T) {
	app, mockSvc, h := setupDisbursementTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/disbursements/:id/confirm", h.Confirm)

	body := dto.ConfirmDisbursementRequest{TransferRef: "REF-001", Note: "โอนแล้ว"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("Confirm", uint(5), uint(99), mock.Anything).Return(&domain.Disbursement{}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/disbursements/5/confirm", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDisbursementHandler_Confirm_Unauthorized(t *testing.T) {
	app, _, h := setupDisbursementTest(t)
	app.Patch("/admin/disbursements/:id/confirm", h.Confirm)

	body := dto.ConfirmDisbursementRequest{TransferRef: "REF-001"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/admin/disbursements/5/confirm", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestDisbursementHandler_Confirm_InvalidID(t *testing.T) {
	app, _, h := setupDisbursementTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/disbursements/:id/confirm", h.Confirm)

	body := dto.ConfirmDisbursementRequest{TransferRef: "REF-001"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/admin/disbursements/abc/confirm", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDisbursementHandler_Confirm_ValidationFail(t *testing.T) {
	app, _, h := setupDisbursementTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/disbursements/:id/confirm", h.Confirm)

	body := dto.ConfirmDisbursementRequest{} // ขาด transfer_ref
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/admin/disbursements/5/confirm", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ListMyPayouts

func TestDisbursementHandler_ListMyPayouts(t *testing.T) {
	app, mockSvc, h := setupDisbursementTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/pioneer/payouts", h.ListMyPayouts)

	mockSvc.On("ListMyPayouts", uint(1)).Return([]dto.PioneerPayoutItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/pioneer/payouts", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDisbursementHandler_ListMyPayouts_Unauthorized(t *testing.T) {
	app, _, h := setupDisbursementTest(t)
	app.Get("/pioneer/payouts", h.ListMyPayouts)

	req := httptest.NewRequest(http.MethodGet, "/pioneer/payouts", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

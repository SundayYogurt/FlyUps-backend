package handler

import (
	"bytes"
	"encoding/json"
	"errors"
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

// Mock: ProfitPoolService

type MockProfitPoolService struct {
	mock.Mock
}

func (m *MockProfitPoolService) Create(adminID uint, req dto.CreateProfitPoolRequest) (*dto.ProfitPoolDetail, error) {
	args := m.Called(adminID, req)
	if v := args.Get(0); v != nil {
		return v.(*dto.ProfitPoolDetail), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProfitPoolService) List() ([]dto.ProfitPoolListItem, error) {
	args := m.Called()
	return args.Get(0).([]dto.ProfitPoolListItem), args.Error(1)
}

func (m *MockProfitPoolService) GetDetail(poolID uint) (*dto.ProfitPoolDetail, error) {
	args := m.Called(poolID)
	if v := args.Get(0); v != nil {
		return v.(*dto.ProfitPoolDetail), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProfitPoolService) ConfirmPayout(poolID uint, payoutID uint, adminID uint, req dto.ConfirmInvestorPayoutRequest) error {
	return m.Called(poolID, payoutID, adminID, req).Error(0)
}

func (m *MockProfitPoolService) PioneerSubmit(pioneerID uint, projectID uint, req dto.PioneerSubmitProfitRequest) (*dto.ProfitPoolDetail, error) {
	args := m.Called(pioneerID, projectID, req)
	if v := args.Get(0); v != nil {
		return v.(*dto.ProfitPoolDetail), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProfitPoolService) GetPioneerPools(pioneerID uint) ([]dto.ProfitPoolListItem, error) {
	args := m.Called(pioneerID)
	return args.Get(0).([]dto.ProfitPoolListItem), args.Error(1)
}

func (m *MockProfitPoolService) GetMyProfitPayouts(userID uint) ([]dto.MyProfitPayoutItem, error) {
	args := m.Called(userID)
	return args.Get(0).([]dto.MyProfitPayoutItem), args.Error(1)
}

// Setup

func setupProfitPoolTest(t *testing.T) (*fiber.App, *MockProfitPoolService, *ProfitPoolHandler) {
	app := fiber.New()
	mockSvc := new(MockProfitPoolService)
	h := &ProfitPoolHandler{
		svc:       mockSvc,
		validator: validator.New(),
		auth:      helper.Auth{Secret: "test-secret-key-for-testing-1234"},
	}
	return app, mockSvc, h
}

func TestProfitHandlersRejectMissingSlip(t *testing.T) {
	for _, tc := range []struct{ name, method, path, role, body string }{
		{"admin pool", "POST", "/admin/profit-pools", "admin", `{"project_id":1,"quarter_no":1,"total_amount":100,"transfer_ref":"MANUAL"}`},
		{"pioneer", "POST", "/pioneer/profit-pools/1", "pioneer", `{"quarter_no":1,"total_amount":100,"transfer_ref":"MANUAL"}`},
		{"admin payout", "PATCH", "/admin/profit-pools/1/payouts/2/confirm", "admin", `{"transfer_ref":"MANUAL"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, svc, h := setupProfitPoolTest(t)
			app.Use(func(c fiber.Ctx) error { c.Locals("user", domain.User{ID: 99, Role: tc.role}); return c.Next() })
			app.Post("/admin/profit-pools", h.Create)
			app.Post("/pioneer/profit-pools/:projectId", h.PioneerSubmit)
			app.Patch("/admin/profit-pools/:id/payouts/:payoutId/confirm", h.ConfirmPayout)
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			assert.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			assert.Empty(t, svc.Calls)
		})
	}
}

// Create

func TestProfitPoolHandler_Create(t *testing.T) {
	app, mockSvc, h := setupProfitPoolTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Post("/admin/profit-pools", h.Create)

	body := dto.CreateProfitPoolRequest{ProjectID: 1, TotalAmount: 5000, TransferRef: "REF-001", QuarterNo: 1, SlipImage: "https://res.cloudinary.com/flyup/image/upload/slip.png"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("Create", uint(99), mock.Anything).Return(&dto.ProfitPoolDetail{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/profit-pools", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProfitPoolHandler_Create_Unauthorized(t *testing.T) {
	app, _, h := setupProfitPoolTest(t)
	app.Post("/admin/profit-pools", h.Create)

	body := dto.CreateProfitPoolRequest{ProjectID: 1, TotalAmount: 5000, TransferRef: "REF-001", QuarterNo: 1, SlipImage: "https://res.cloudinary.com/flyup/image/upload/slip.png"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/profit-pools", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProfitPoolHandler_Create_ValidationFail(t *testing.T) {
	app, _, h := setupProfitPoolTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Post("/admin/profit-pools", h.Create)

	body := dto.CreateProfitPoolRequest{} // ขาด required fields
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/profit-pools", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// List

func TestProfitPoolHandler_List(t *testing.T) {
	app, mockSvc, h := setupProfitPoolTest(t)
	app.Get("/admin/profit-pools", h.List)

	mockSvc.On("List").Return([]dto.ProfitPoolListItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/profit-pools", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetDetail

func TestProfitPoolHandler_GetDetail(t *testing.T) {
	app, mockSvc, h := setupProfitPoolTest(t)
	app.Get("/admin/profit-pools/:id", h.GetDetail)

	mockSvc.On("GetDetail", uint(2)).Return(&dto.ProfitPoolDetail{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/profit-pools/2", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProfitPoolHandler_GetDetail_InvalidID(t *testing.T) {
	app, _, h := setupProfitPoolTest(t)
	app.Get("/admin/profit-pools/:id", h.GetDetail)

	req := httptest.NewRequest(http.MethodGet, "/admin/profit-pools/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProfitPoolHandler_GetDetail_NotFound(t *testing.T) {
	app, mockSvc, h := setupProfitPoolTest(t)
	app.Get("/admin/profit-pools/:id", h.GetDetail)

	mockSvc.On("GetDetail", uint(99)).Return(nil, errors.New("profit pool not found"))
	req := httptest.NewRequest(http.MethodGet, "/admin/profit-pools/99", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// PioneerSubmit

func TestProfitPoolHandler_PioneerSubmit(t *testing.T) {
	app, mockSvc, h := setupProfitPoolTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/profit-pools/:projectId", h.PioneerSubmit)

	body := dto.PioneerSubmitProfitRequest{TotalAmount: 10000, TransferRef: "REF-P001", QuarterNo: 1, SlipImage: "https://res.cloudinary.com/flyup/image/upload/slip.png"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("PioneerSubmit", uint(1), uint(5), mock.Anything).Return(&dto.ProfitPoolDetail{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/profit-pools/5", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProfitPoolHandler_PioneerSubmit_Unauthorized(t *testing.T) {
	app, _, h := setupProfitPoolTest(t)
	app.Post("/pioneer/profit-pools/:projectId", h.PioneerSubmit)

	body := dto.PioneerSubmitProfitRequest{TotalAmount: 10000, TransferRef: "REF-P001", QuarterNo: 1, SlipImage: "https://res.cloudinary.com/flyup/image/upload/slip.png"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/profit-pools/5", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProfitPoolHandler_PioneerSubmit_InvalidProjectID(t *testing.T) {
	app, _, h := setupProfitPoolTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/profit-pools/:projectId", h.PioneerSubmit)

	body := dto.PioneerSubmitProfitRequest{TotalAmount: 10000, TransferRef: "REF-P001", QuarterNo: 1, SlipImage: "https://res.cloudinary.com/flyup/image/upload/slip.png"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/profit-pools/abc", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// PioneerList

func TestProfitPoolHandler_PioneerList(t *testing.T) {
	app, mockSvc, h := setupProfitPoolTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/pioneer/profit-pools", h.PioneerList)

	mockSvc.On("GetPioneerPools", uint(1)).Return([]dto.ProfitPoolListItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/pioneer/profit-pools", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProfitPoolHandler_PioneerList_Unauthorized(t *testing.T) {
	app, _, h := setupProfitPoolTest(t)
	app.Get("/pioneer/profit-pools", h.PioneerList)

	req := httptest.NewRequest(http.MethodGet, "/pioneer/profit-pools", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// MyProfitPayouts

func TestProfitPoolHandler_MyProfitPayouts(t *testing.T) {
	app, mockSvc, h := setupProfitPoolTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/me/profit-payouts", h.MyProfitPayouts)

	mockSvc.On("GetMyProfitPayouts", uint(1)).Return([]dto.MyProfitPayoutItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/me/profit-payouts", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProfitPoolHandler_MyProfitPayouts_Unauthorized(t *testing.T) {
	app, _, h := setupProfitPoolTest(t)
	app.Get("/me/profit-payouts", h.MyProfitPayouts)

	req := httptest.NewRequest(http.MethodGet, "/me/profit-payouts", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ConfirmPayout

func TestProfitPoolHandler_ConfirmPayout(t *testing.T) {
	app, mockSvc, h := setupProfitPoolTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/profit-pools/:id/payouts/:payoutId/confirm", h.ConfirmPayout)

	body := dto.ConfirmInvestorPayoutRequest{TransferRef: "REF-INV-001", SlipImage: "https://res.cloudinary.com/flyup/image/upload/slip.png"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("ConfirmPayout", uint(2), uint(7), uint(99), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/profit-pools/2/payouts/7/confirm", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProfitPoolHandler_ConfirmPayout_Unauthorized(t *testing.T) {
	app, _, h := setupProfitPoolTest(t)
	app.Patch("/admin/profit-pools/:id/payouts/:payoutId/confirm", h.ConfirmPayout)

	body := dto.ConfirmInvestorPayoutRequest{TransferRef: "REF-INV-001", SlipImage: "https://res.cloudinary.com/flyup/image/upload/slip.png"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/admin/profit-pools/2/payouts/7/confirm", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProfitPoolHandler_ConfirmPayout_InvalidPoolID(t *testing.T) {
	app, _, h := setupProfitPoolTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/profit-pools/:id/payouts/:payoutId/confirm", h.ConfirmPayout)

	body := dto.ConfirmInvestorPayoutRequest{TransferRef: "REF-INV-001", SlipImage: "https://res.cloudinary.com/flyup/image/upload/slip.png"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/admin/profit-pools/abc/payouts/7/confirm", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

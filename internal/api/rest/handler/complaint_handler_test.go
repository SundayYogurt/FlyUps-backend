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

// Mock: ComplaintService

type MockComplaintService struct {
	mock.Mock
}

func (m *MockComplaintService) Create(userID uint, req dto.CreateComplaintRequest) (*domain.Complaint, error) {
	args := m.Called(userID, req)
	if v := args.Get(0); v != nil {
		return v.(*domain.Complaint), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockComplaintService) ListMine(userID uint) ([]dto.ComplaintItem, error) {
	args := m.Called(userID)
	return args.Get(0).([]dto.ComplaintItem), args.Error(1)
}

func (m *MockComplaintService) AdminList(status *domain.ComplaintStatus) ([]dto.ComplaintItem, error) {
	args := m.Called(status)
	return args.Get(0).([]dto.ComplaintItem), args.Error(1)
}

func (m *MockComplaintService) AdminGet(id uint) (*dto.ComplaintItem, error) {
	args := m.Called(id)
	if v := args.Get(0); v != nil {
		return v.(*dto.ComplaintItem), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockComplaintService) AdminResolve(id, adminID uint, note string) (*domain.Complaint, error) {
	args := m.Called(id, adminID, note)
	if v := args.Get(0); v != nil {
		return v.(*domain.Complaint), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockComplaintService) AdminReject(id, adminID uint, note string) (*domain.Complaint, error) {
	args := m.Called(id, adminID, note)
	if v := args.Get(0); v != nil {
		return v.(*domain.Complaint), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockComplaintService) GetProjectStats(projectID uint) (*dto.ProjectComplaintStats, error) {
	args := m.Called(projectID)
	if v := args.Get(0); v != nil {
		return v.(*dto.ProjectComplaintStats), args.Error(1)
	}
	return nil, args.Error(1)
}

// Setup

func setupComplaintTest(t *testing.T) (*fiber.App, *MockComplaintService, *ComplaintHandler) {
	app := fiber.New()
	mockSvc := new(MockComplaintService)
	h := &ComplaintHandler{
		svc:       mockSvc,
		validator: validator.New(),
		auth:      helper.Auth{Secret: testSecret},
	}
	return app, mockSvc, h
}

// Create

func TestComplaintHandler_Create(t *testing.T) {
	app, mockSvc, h := setupComplaintTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/complaints", h.Create)

	body := dto.CreateComplaintRequest{ProjectID: 10, Subject: "ปัญหาโปรเจกต์", Body: "มีปัญหาบางอย่างเกิดขึ้นกับโปรเจกต์นี้"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("Create", uint(1), mock.Anything).Return(&domain.Complaint{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/complaints", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestComplaintHandler_Create_Unauthorized(t *testing.T) {
	app, _, h := setupComplaintTest(t)
	app.Post("/complaints", h.Create)

	body := dto.CreateComplaintRequest{ProjectID: 10, Subject: "ปัญหา", Body: "มีปัญหาบางอย่างเกิดขึ้นกับโปรเจกต์นี้"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/complaints", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestComplaintHandler_Create_Duplicate(t *testing.T) {
	app, mockSvc, h := setupComplaintTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/complaints", h.Create)

	body := dto.CreateComplaintRequest{ProjectID: 10, Subject: "ปัญหาโปรเจกต์", Body: "มีปัญหาบางอย่างเกิดขึ้นกับโปรเจกต์นี้"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("Create", uint(1), mock.Anything).Return(nil, errors.New("you have already filed a complaint for this project"))
	req := httptest.NewRequest(http.MethodPost, "/complaints", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

// ListMine

func TestComplaintHandler_ListMine(t *testing.T) {
	app, mockSvc, h := setupComplaintTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/complaints/me", h.ListMine)

	mockSvc.On("ListMine", uint(1)).Return([]dto.ComplaintItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/complaints/me", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestComplaintHandler_ListMine_Unauthorized(t *testing.T) {
	app, _, h := setupComplaintTest(t)
	app.Get("/complaints/me", h.ListMine)

	req := httptest.NewRequest(http.MethodGet, "/complaints/me", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// AdminList

func TestComplaintHandler_AdminList(t *testing.T) {
	app, mockSvc, h := setupComplaintTest(t)
	app.Get("/admin/complaints", h.AdminList)

	mockSvc.On("AdminList", (*domain.ComplaintStatus)(nil)).Return([]dto.ComplaintItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/complaints", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestComplaintHandler_AdminList_InvalidStatus(t *testing.T) {
	app, _, h := setupComplaintTest(t)
	app.Get("/admin/complaints", h.AdminList)

	req := httptest.NewRequest(http.MethodGet, "/admin/complaints?status=invalid-status", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// AdminGet

func TestComplaintHandler_AdminGet(t *testing.T) {
	app, mockSvc, h := setupComplaintTest(t)
	app.Get("/admin/complaints/:id", h.AdminGet)

	mockSvc.On("AdminGet", uint(3)).Return(&dto.ComplaintItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/complaints/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestComplaintHandler_AdminGet_InvalidID(t *testing.T) {
	app, _, h := setupComplaintTest(t)
	app.Get("/admin/complaints/:id", h.AdminGet)

	req := httptest.NewRequest(http.MethodGet, "/admin/complaints/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestComplaintHandler_AdminGet_NotFound(t *testing.T) {
	app, mockSvc, h := setupComplaintTest(t)
	app.Get("/admin/complaints/:id", h.AdminGet)

	mockSvc.On("AdminGet", uint(99)).Return(nil, errors.New("not found"))
	req := httptest.NewRequest(http.MethodGet, "/admin/complaints/99", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// AdminResolve

func TestComplaintHandler_AdminResolve(t *testing.T) {
	app, mockSvc, h := setupComplaintTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/complaints/:id/resolve", h.AdminResolve)

	body := dto.ResolveComplaintRequest{AdminNote: "ตรวจสอบแล้วพบปัญหาจริง"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("AdminResolve", uint(3), uint(99), "ตรวจสอบแล้วพบปัญหาจริง").Return(&domain.Complaint{}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/complaints/3/resolve", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestComplaintHandler_AdminResolve_Unauthorized(t *testing.T) {
	app, _, h := setupComplaintTest(t)
	app.Patch("/admin/complaints/:id/resolve", h.AdminResolve)

	body := dto.ResolveComplaintRequest{AdminNote: "ตรวจสอบแล้ว"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/admin/complaints/3/resolve", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// AdminReject

func TestComplaintHandler_AdminReject(t *testing.T) {
	app, mockSvc, h := setupComplaintTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/complaints/:id/reject", h.AdminReject)

	body := dto.ResolveComplaintRequest{AdminNote: "ไม่พบปัญหาตามที่แจ้ง"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("AdminReject", uint(3), uint(99), "ไม่พบปัญหาตามที่แจ้ง").Return(&domain.Complaint{}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/complaints/3/reject", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// AdminProjectStats

func TestComplaintHandler_AdminProjectStats(t *testing.T) {
	app, mockSvc, h := setupComplaintTest(t)
	app.Get("/admin/complaints/project-stats/:project_id", h.AdminProjectStats)

	mockSvc.On("GetProjectStats", uint(5)).Return(&dto.ProjectComplaintStats{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/complaints/project-stats/5", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestComplaintHandler_AdminProjectStats_InvalidID(t *testing.T) {
	app, _, h := setupComplaintTest(t)
	app.Get("/admin/complaints/project-stats/:project_id", h.AdminProjectStats)

	req := httptest.NewRequest(http.MethodGet, "/admin/complaints/project-stats/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

package handler

import (
	"errors"
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

// Mock: FinancialService

type MockFinancialService struct {
	mock.Mock
}

func (m *MockFinancialService) GetSummary() (*dto.FinancialSummary, error) {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.(*dto.FinancialSummary), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockFinancialService) GetProjectsFinancial() ([]dto.ProjectFinancialItem, error) {
	args := m.Called()
	return args.Get(0).([]dto.ProjectFinancialItem), args.Error(1)
}

// Setup

func setupFinancialTest(t *testing.T) (*fiber.App, *MockFinancialService, *FinancialHandler) {
	app := fiber.New()
	mockSvc := new(MockFinancialService)
	h := &FinancialHandler{
		svc:  mockSvc,
		auth: helper.Auth{Secret: testSecret},
	}
	return app, mockSvc, h
}

// GetSummary

func TestFinancialHandler_GetSummary(t *testing.T) {
	app, mockSvc, h := setupFinancialTest(t)
	app.Get("/admin/financial/summary", h.GetSummary)

	mockSvc.On("GetSummary").Return(&dto.FinancialSummary{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/financial/summary", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFinancialHandler_GetSummary_Error(t *testing.T) {
	app, mockSvc, h := setupFinancialTest(t)
	app.Get("/admin/financial/summary", h.GetSummary)

	mockSvc.On("GetSummary").Return(nil, errors.New("db error"))
	req := httptest.NewRequest(http.MethodGet, "/admin/financial/summary", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// GetProjectsFinancial

func TestFinancialHandler_GetProjectsFinancial(t *testing.T) {
	app, mockSvc, h := setupFinancialTest(t)
	app.Get("/admin/financial/projects", h.GetProjectsFinancial)

	mockSvc.On("GetProjectsFinancial").Return([]dto.ProjectFinancialItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/financial/projects", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFinancialHandler_GetProjectsFinancial_Error(t *testing.T) {
	app, mockSvc, h := setupFinancialTest(t)
	app.Get("/admin/financial/projects", h.GetProjectsFinancial)

	mockSvc.On("GetProjectsFinancial").Return([]dto.ProjectFinancialItem(nil), errors.New("db error"))
	req := httptest.NewRequest(http.MethodGet, "/admin/financial/projects", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

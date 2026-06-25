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

// Mock: InvestmentService

type MockInvestmentService struct {
	mock.Mock
}

func (m *MockInvestmentService) GetInvestment(boosterUserID uint, investmentID uint) (*domain.Investment, *domain.Transaction, error) {
	args := m.Called(boosterUserID, investmentID)
	inv, _ := args.Get(0).(*domain.Investment)
	txn, _ := args.Get(1).(*domain.Transaction)
	return inv, txn, args.Error(2)
}

func (m *MockInvestmentService) GenerateContractHTML(boosterUserID uint, investmentID uint) ([]byte, error) {
	args := m.Called(boosterUserID, investmentID)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockInvestmentService) CreateInvestment(boosterUserID uint, boosterEmail string, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error) {
	args := m.Called(boosterUserID, boosterEmail, req)
	if v := args.Get(0); v != nil {
		return v.(*dto.InvestmentResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInvestmentService) ListUserInvestments(boosterUserID uint) ([]domain.Investment, error) {
	args := m.Called(boosterUserID)
	return args.Get(0).([]domain.Investment), args.Error(1)
}

func (m *MockInvestmentService) HandleStripeWebhook(payload []byte, sigHeader string) error {
	return m.Called(payload, sigHeader).Error(0)
}

func (m *MockInvestmentService) RefundInvestment(boosterUserID uint, investmentID uint, note string) (*dto.RefundResponse, error) {
	args := m.Called(boosterUserID, investmentID, note)
	if v := args.Get(0); v != nil {
		return v.(*dto.RefundResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInvestmentService) ApproveRefund(investmentID uint) error {
	return m.Called(investmentID).Error(0)
}

func (m *MockInvestmentService) ListRefundRequests() ([]dto.RefundRequestItem, error) {
	args := m.Called()
	return args.Get(0).([]dto.RefundRequestItem), args.Error(1)
}

func (m *MockInvestmentService) GetProjectInvestors(projectID uint) ([]dto.ProjectInvestorItem, error) {
	args := m.Called(projectID)
	return args.Get(0).([]dto.ProjectInvestorItem), args.Error(1)
}

func (m *MockInvestmentService) ListInvestedProjects(boosterUserID uint) ([]dto.InvestedProjectItem, error) {
	args := m.Called(boosterUserID)
	return args.Get(0).([]dto.InvestedProjectItem), args.Error(1)
}

func (m *MockInvestmentService) VoteMilestone(boosterUserID uint, milestoneID uint, choice domain.MilestoneVoteChoice) (*domain.MilestoneVote, error) {
	args := m.Called(boosterUserID, milestoneID, choice)
	if v := args.Get(0); v != nil {
		return v.(*domain.MilestoneVote), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInvestmentService) GetMyVote(boosterUserID uint, milestoneID uint) (*domain.MilestoneVote, error) {
	args := m.Called(boosterUserID, milestoneID)
	if v := args.Get(0); v != nil {
		return v.(*domain.MilestoneVote), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInvestmentService) GetMilestoneVoters(pioneerUserID uint, milestoneID uint) ([]dto.MilestoneVoterItem, error) {
	args := m.Called(pioneerUserID, milestoneID)
	return args.Get(0).([]dto.MilestoneVoterItem), args.Error(1)
}

func (m *MockInvestmentService) RefundProjectInvestments(project domain.Project) {
	m.Called(project)
}

func (m *MockInvestmentService) GetCancelPreview(projectID uint) (*dto.CancelPreviewResponse, error) {
	args := m.Called(projectID)
	if v := args.Get(0); v != nil {
		return v.(*dto.CancelPreviewResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockInvestmentService) FinalizeVotingIfExpired(milestoneID uint) error {
	return m.Called(milestoneID).Error(0)
}

func (m *MockInvestmentService) GetTotalFunding() (float64, error) {
	args := m.Called()
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockInvestmentService) GetUniqueBoostersCount() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockInvestmentService) SyncProjectPrincipalAmounts(projectID uint) error {
	return m.Called(projectID).Error(0)
}

// Setup

func setupInvestmentTest(t *testing.T) (*fiber.App, *MockInvestmentService, *InvestmentHandler) {
	app := fiber.New()
	mockSvc := new(MockInvestmentService)
	h := &InvestmentHandler{
		svc:       mockSvc,
		validator: validator.New(),
		auth:      helper.Auth{Secret: testSecret},
	}
	return app, mockSvc, h
}

// ListMyInvestments

func TestInvestmentHandler_ListMyInvestments(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/investments", h.ListMyInvestments)

	mockSvc.On("ListUserInvestments", uint(1)).Return([]domain.Investment{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/investments", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvestmentHandler_ListMyInvestments_Unauthorized(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Get("/investments", h.ListMyInvestments)

	req := httptest.NewRequest(http.MethodGet, "/investments", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// CreateInvestment

func TestInvestmentHandler_CreateInvestment(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/investments", h.CreateInvestment)

	body := dto.CreateInvestmentRequest{ProjectID: 10, Amount: 1000}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("CreateInvestment", uint(1), "booster@test.com", mock.Anything).Return(&dto.InvestmentResponse{InvestmentID: 1}, nil)
	req := httptest.NewRequest(http.MethodPost, "/investments", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestInvestmentHandler_CreateInvestment_Unauthorized(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Post("/investments", h.CreateInvestment)

	body := dto.CreateInvestmentRequest{ProjectID: 10, Amount: 1000}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/investments", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestInvestmentHandler_CreateInvestment_ValidationFail(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/investments", h.CreateInvestment)

	body := dto.CreateInvestmentRequest{} // ขาด required fields
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/investments", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// GetInvestment

func TestInvestmentHandler_GetInvestment(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/investments/:id", h.GetInvestment)

	mockSvc.On("GetInvestment", uint(1), uint(5)).Return(&domain.Investment{}, &domain.Transaction{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/investments/5", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvestmentHandler_GetInvestment_Unauthorized(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Get("/investments/:id", h.GetInvestment)

	req := httptest.NewRequest(http.MethodGet, "/investments/5", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestInvestmentHandler_GetInvestment_InvalidID(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/investments/:id", h.GetInvestment)

	req := httptest.NewRequest(http.MethodGet, "/investments/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ListRefundRequests

func TestInvestmentHandler_ListRefundRequests(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Get("/admin/investments/refund-requests", h.ListRefundRequests)

	mockSvc.On("ListRefundRequests").Return([]dto.RefundRequestItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/investments/refund-requests", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ApproveRefund

func TestInvestmentHandler_ApproveRefund(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Patch("/admin/investments/:id/approve-refund", h.ApproveRefund)

	mockSvc.On("ApproveRefund", uint(3)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/investments/3/approve-refund", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvestmentHandler_ApproveRefund_InvalidID(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Patch("/admin/investments/:id/approve-refund", h.ApproveRefund)

	req := httptest.NewRequest(http.MethodPatch, "/admin/investments/abc/approve-refund", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// GetProjectInvestors

func TestInvestmentHandler_GetProjectInvestors(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Get("/investments/projects/:projectId/investors", h.GetProjectInvestors)

	mockSvc.On("GetProjectInvestors", uint(7)).Return([]dto.ProjectInvestorItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/investments/projects/7/investors", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvestmentHandler_GetProjectInvestors_InvalidID(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Get("/investments/projects/:projectId/investors", h.GetProjectInvestors)

	req := httptest.NewRequest(http.MethodGet, "/investments/projects/abc/investors", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ListMyInvestedProjects

func TestInvestmentHandler_ListMyInvestedProjects(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/investments/my-projects", h.ListMyInvestedProjects)

	mockSvc.On("ListInvestedProjects", uint(1)).Return([]dto.InvestedProjectItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/investments/my-projects", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvestmentHandler_ListMyInvestedProjects_Unauthorized(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Get("/investments/my-projects", h.ListMyInvestedProjects)

	req := httptest.NewRequest(http.MethodGet, "/investments/my-projects", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// VoteMilestone

func TestInvestmentHandler_VoteMilestone(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/investments/milestones/:milestone_id/vote", h.VoteMilestone)

	body := dto.VoteMilestoneRequest{Choice: "approve"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("VoteMilestone", uint(1), uint(2), domain.MilestoneVoteChoice("approve")).Return(&domain.MilestoneVote{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/investments/milestones/2/vote", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvestmentHandler_VoteMilestone_Unauthorized(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Post("/investments/milestones/:milestone_id/vote", h.VoteMilestone)

	body := dto.VoteMilestoneRequest{Choice: "approve"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/investments/milestones/2/vote", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestInvestmentHandler_VoteMilestone_ServiceError(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/investments/milestones/:milestone_id/vote", h.VoteMilestone)

	body := dto.VoteMilestoneRequest{Choice: "approve"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("VoteMilestone", uint(1), uint(2), domain.MilestoneVoteChoice("approve")).Return(nil, errors.New("already voted"))
	req := httptest.NewRequest(http.MethodPost, "/investments/milestones/2/vote", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// RefundInvestment

func TestInvestmentHandler_RefundInvestment(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/investments/:id/refund", h.RefundInvestment)

	body := dto.RefundInvestmentRequest{Note: "ขอคืนเงิน"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("RefundInvestment", uint(1), uint(5), "ขอคืนเงิน").Return(&dto.RefundResponse{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/investments/5/refund", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvestmentHandler_RefundInvestment_MissingNote(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/investments/:id/refund", h.RefundInvestment)

	body := dto.RefundInvestmentRequest{Note: ""} // note ว่าง
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/investments/5/refund", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// GetMyMilestoneVote

func TestInvestmentHandler_GetMyMilestoneVote(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/investments/milestones/:milestone_id/my-vote", h.GetMyMilestoneVote)

	mockSvc.On("GetMyVote", uint(1), uint(3)).Return(&domain.MilestoneVote{ID: 1}, nil)
	req := httptest.NewRequest(http.MethodGet, "/investments/milestones/3/my-vote", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvestmentHandler_GetMyMilestoneVote_Unauthorized(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Get("/investments/milestones/:milestone_id/my-vote", h.GetMyMilestoneVote)

	req := httptest.NewRequest(http.MethodGet, "/investments/milestones/3/my-vote", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestInvestmentHandler_GetMyMilestoneVote_InvalidID(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/investments/milestones/:milestone_id/my-vote", h.GetMyMilestoneVote)

	req := httptest.NewRequest(http.MethodGet, "/investments/milestones/abc/my-vote", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// GetMilestoneVoters

func TestInvestmentHandler_GetMilestoneVoters(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/investments/milestones/:milestone_id/voters", h.GetMilestoneVoters)

	mockSvc.On("GetMilestoneVoters", uint(1), uint(3)).Return([]dto.MilestoneVoterItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/investments/milestones/3/voters", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvestmentHandler_GetMilestoneVoters_Unauthorized(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Get("/investments/milestones/:milestone_id/voters", h.GetMilestoneVoters)

	req := httptest.NewRequest(http.MethodGet, "/investments/milestones/3/voters", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestInvestmentHandler_GetMilestoneVoters_InvalidID(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/investments/milestones/:milestone_id/voters", h.GetMilestoneVoters)

	req := httptest.NewRequest(http.MethodGet, "/investments/milestones/xyz/voters", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// DownloadContract

func TestInvestmentHandler_DownloadContract(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/investments/:id/contract", h.DownloadContract)

	mockSvc.On("GenerateContractHTML", uint(1), uint(7)).Return([]byte("<html>contract</html>"), nil)
	req := httptest.NewRequest(http.MethodGet, "/investments/7/contract", nil)
	resp, _ := app.Test(req)
	t.Logf("status=%d content-type=%s", resp.StatusCode, resp.Header.Get("Content-Type"))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "contract-INV-7.html")
}

func TestInvestmentHandler_DownloadContract_Unauthorized(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Get("/investments/:id/contract", h.DownloadContract)

	req := httptest.NewRequest(http.MethodGet, "/investments/7/contract", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestInvestmentHandler_DownloadContract_InvalidID(t *testing.T) {
	app, _, h := setupInvestmentTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/investments/:id/contract", h.DownloadContract)

	req := httptest.NewRequest(http.MethodGet, "/investments/bad/contract", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// StripeWebhook

func TestInvestmentHandler_StripeWebhook(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Post("/stripe/webhook", h.StripeWebhook)

	payload := []byte(`{"type":"payment_intent.succeeded"}`)
	mockSvc.On("HandleStripeWebhook", payload, "whsec_test").Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/stripe/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "whsec_test")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestInvestmentHandler_StripeWebhook_InvalidSignature(t *testing.T) {
	app, mockSvc, h := setupInvestmentTest(t)
	app.Post("/stripe/webhook", h.StripeWebhook)

	payload := []byte(`{"type":"payment_intent.succeeded"}`)
	mockSvc.On("HandleStripeWebhook", payload, "bad_sig").Return(errors.New("invalid signature"))

	req := httptest.NewRequest(http.MethodPost, "/stripe/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "bad_sig")
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

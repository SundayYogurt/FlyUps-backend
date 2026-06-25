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

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock: ChatService

type MockChatService struct {
	mock.Mock
}

func (m *MockChatService) SendMessage(userID uint, req dto.SendChatMessageRequest) (*dto.SendChatMessageResponse, error) {
	args := m.Called(userID, req)
	if v := args.Get(0); v != nil {
		return v.(*dto.SendChatMessageResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockChatService) ConfirmAction(userID uint, req dto.ConfirmChatActionRequest) (*dto.ConfirmChatActionResponse, error) {
	args := m.Called(userID, req)
	if v := args.Get(0); v != nil {
		return v.(*dto.ConfirmChatActionResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

// Setup

func setupChatTest(t *testing.T) (*fiber.App, *MockChatService, *ChatHandler) {
	app := fiber.New()
	mockSvc := new(MockChatService)
	h := &ChatHandler{
		chatService: mockSvc,
		auth:        helper.Auth{Secret: testSecret},
	}
	return app, mockSvc, h
}

// SendMessage

func TestChatHandler_SendMessage(t *testing.T) {
	app, mockSvc, h := setupChatTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/chat/messages", h.SendMessage)

	reqBody := dto.SendChatMessageRequest{Message: "สวัสดี"}
	b, _ := json.Marshal(reqBody)

	mockSvc.On("SendMessage", uint(1), reqBody).Return(&dto.SendChatMessageResponse{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/chat/messages", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestChatHandler_SendMessage_Unauthorized(t *testing.T) {
	app, _, h := setupChatTest(t)
	app.Post("/chat/messages", h.SendMessage)

	reqBody := dto.SendChatMessageRequest{Message: "สวัสดี"}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/chat/messages", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestChatHandler_SendMessage_ServiceError(t *testing.T) {
	app, mockSvc, h := setupChatTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/chat/messages", h.SendMessage)

	reqBody := dto.SendChatMessageRequest{Message: "error"}
	b, _ := json.Marshal(reqBody)

	mockSvc.On("SendMessage", uint(1), reqBody).Return(nil, errors.New("AI unavailable"))

	req := httptest.NewRequest(http.MethodPost, "/chat/messages", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

// ConfirmAction

func TestChatHandler_ConfirmAction(t *testing.T) {
	app, mockSvc, h := setupChatTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/chat/actions/confirm", h.ConfirmAction)

	reqBody := dto.ConfirmChatActionRequest{ActionID: 5, Confirm: true}
	b, _ := json.Marshal(reqBody)

	mockSvc.On("ConfirmAction", uint(1), reqBody).Return(&dto.ConfirmChatActionResponse{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/chat/actions/confirm", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(body))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestChatHandler_ConfirmAction_Unauthorized(t *testing.T) {
	app, _, h := setupChatTest(t)
	app.Post("/chat/actions/confirm", h.ConfirmAction)

	reqBody := dto.ConfirmChatActionRequest{ActionID: 5, Confirm: true}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/chat/actions/confirm", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestChatHandler_ConfirmAction_ServiceError(t *testing.T) {
	app, mockSvc, h := setupChatTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/chat/actions/confirm", h.ConfirmAction)

	reqBody := dto.ConfirmChatActionRequest{ActionID: 5, Confirm: true}
	b, _ := json.Marshal(reqBody)

	mockSvc.On("ConfirmAction", uint(1), reqBody).Return(nil, errors.New("action not found"))

	req := httptest.NewRequest(http.MethodPost, "/chat/actions/confirm", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

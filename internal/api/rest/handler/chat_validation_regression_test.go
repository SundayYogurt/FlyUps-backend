package handler

import (
	"flyup/internal/domain"
	"flyup/internal/helper"
	"flyup/internal/services"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBlankChatReturnsBadRequest(t *testing.T) {
	for _, body := range []string{`{}`, `{"message":""}`, `{"message":"   "}`} {
		app := fiber.New()
		h := NewChatHandler(services.NewChatService(nil, nil, nil, nil, nil, nil, nil), helper.Auth{Secret: testSecret})
		app.Use(func(c fiber.Ctx) error { c.Locals("user", domain.User{ID: 1, Role: "booster"}); return c.Next() })
		app.Post("/chat/messages", h.SendMessage)
		req := httptest.NewRequest("POST", "/chat/messages", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		res, err := app.Test(req)
		require.NoError(t, err)
		res.Body.Close()
		require.Equal(t, 400, res.StatusCode)
	}
}

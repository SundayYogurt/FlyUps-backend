package handler

import (
	"flyup/internal/domain"
	"flyup/internal/helper"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

// Setup

func setupAdminLogTest(t *testing.T) (*fiber.App, *AdminLogHandler) {
	app := fiber.New()
	h := &AdminLogHandler{
		svc:  &MockAdminLogService{},
		auth: helper.Auth{Secret: testSecret},
	}
	return app, h
}

// ListLogs

func TestAdminLogHandler_ListLogs(t *testing.T) {
	app, h := setupAdminLogTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/logs", h.ListLogs)

	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestAdminLogHandler_ListLogs_Unauthorized(t *testing.T) {
	app, h := setupAdminLogTest(t)
	app.Get("/admin/logs", h.ListLogs)

	req := httptest.NewRequest(http.MethodGet, "/admin/logs", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestAdminLogHandler_ListLogs_WithFilters(t *testing.T) {
	app, h := setupAdminLogTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/logs", h.ListLogs)

	req := httptest.NewRequest(http.MethodGet, "/admin/logs?page=2&page_size=10&action=approve_project&target_type=project", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

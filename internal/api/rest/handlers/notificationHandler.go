package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"flyup/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	svc  service.NotificationService
	auth helper.Auth
	rh   *rest.RestHandler
}

func SetupNotificationRoutes(rh *rest.RestHandler) {
	h := &NotificationHandler{
		svc:  rh.NotifSvc,
		auth: rh.Auth,
		rh:   rh,
	}

	// SSE stream — no Authorize middleware; authenticated via one-time sse_token query param
	rh.App.Get("/notifications/stream", h.Stream)

	priv := rh.App.Group("/notifications")
	priv.Get("/", rh.Middlewares.Authorize, h.List)
	priv.Post("/sse-token", rh.Middlewares.Authorize, h.IssueSSEToken)
	priv.Patch("/read-all", rh.Middlewares.Authorize, h.MarkAllAsRead)
	priv.Patch("/:id/read", rh.Middlewares.Authorize, h.MarkAsRead)
}

// IssueSSEToken issues a short-lived (60s) one-time token for SSE connection.
// The token is stored in Redis; it is deleted on first use.
func (h *NotificationHandler) IssueSSEToken(c fiber.Ctx) error {
	user := h.auth.GetCurrentUser(c)
	if user.ID == 0 {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	token := uuid.New().String()
	key := "sse_token:" + token
	ctx := context.Background()
	if err := h.rh.Cache.Set(ctx, key, strconv.FormatUint(uint64(user.ID), 10), 60*time.Second); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"message": "failed to issue token"})
	}

	return c.JSON(fiber.Map{"token": token})
}

// List godoc
func (h *NotificationHandler) List(c fiber.Ctx) error {
	user := h.auth.GetCurrentUser(c)
	if user.ID == 0 {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	notifications, total, err := h.svc.GetNotifications(user.ID, page, limit)
	if err != nil {
		return rest.InternalError(c, err)
	}

	unread, _ := h.svc.CountUnread(user.ID)

	return rest.SuccessResponse(c, "success", fiber.Map{
		"notifications": notifications,
		"total":         total,
		"unread":        unread,
		"page":          page,
		"limit":         limit,
	})
}

// Stream — SSE endpoint authenticated via one-time sse_token query param (NOT JWT in URL)
func (h *NotificationHandler) Stream(c fiber.Ctx) error {
	sseToken := c.Query("sse_token")
	if sseToken == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "sse_token required"})
	}

	ctx := context.Background()
	key := "sse_token:" + sseToken

	// Validate and consume (delete) the one-time token
	userIDStr, err := h.rh.Cache.Get(ctx, key)
	if err != nil || userIDStr == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "invalid or expired sse_token"})
	}
	_ = h.rh.Cache.Del(ctx, key) // one-time use

	userID64, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "invalid sse_token"})
	}
	userID := uint(userID64)

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no")

	ch := h.svc.Subscribe(userID)

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer h.svc.Unsubscribe(userID, ch)

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		fmt.Fprintf(w, ": ping\n\n")
		if err := w.Flush(); err != nil {
			return
		}

		for {
			select {
			case notif, ok := <-ch:
				if !ok {
					return
				}
				data, _ := json.Marshal(notif)
				fmt.Fprintf(w, "data: %s\n\n", data)
				if err := w.Flush(); err != nil {
					return
				}
			case <-ticker.C:
				fmt.Fprintf(w, ": ping\n\n")
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	})

	return nil
}

// MarkAsRead godoc
func (h *NotificationHandler) MarkAsRead(c fiber.Ctx) error {
	user := h.auth.GetCurrentUser(c)
	if user.ID == 0 {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return rest.BadRequestError(c, "invalid notification id")
	}

	if err := h.svc.MarkAsRead(user.ID, uint(id)); err != nil {
		return rest.InternalError(c, err)
	}

	return rest.SuccessResponse(c, "marked as read", nil)
}

// MarkAllAsRead godoc
func (h *NotificationHandler) MarkAllAsRead(c fiber.Ctx) error {
	user := h.auth.GetCurrentUser(c)
	if user.ID == 0 {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	if err := h.svc.MarkAllAsRead(user.ID); err != nil {
		return rest.InternalError(c, err)
	}

	return rest.SuccessResponse(c, "all marked as read", nil)
}

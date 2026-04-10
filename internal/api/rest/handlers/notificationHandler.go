package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"flyup/internal/service"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
)

type NotificationHandler struct {
	svc  service.NotificationService
	auth helper.Auth
}

func SetupNotificationRoutes(rh *rest.RestHandler) {
	h := &NotificationHandler{
		svc:  rh.NotifSvc,
		auth: rh.Auth,
	}

	priv := rh.App.Group("/notifications", rh.Middlewares.Authorize)
	priv.Get("/", h.List)
	priv.Get("/stream", h.Stream)
	priv.Patch("/read-all", h.MarkAllAsRead)
	priv.Patch("/:id/read", h.MarkAsRead)
}

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

func (h *NotificationHandler) Stream(c fiber.Ctx) error {
	user := h.auth.GetCurrentUser(c)
	if user.ID == 0 {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("X-Accel-Buffering", "no")

	ch := h.svc.Subscribe(user.ID)

	c.Response().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer h.svc.Unsubscribe(user.ID, ch)

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		// initial ping to confirm connection
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

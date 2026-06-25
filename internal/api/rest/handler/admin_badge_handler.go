package handler

import (
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"flyup/internal/services"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

type AdminBadgeHandler struct {
	svc  services.AdminBadgeService
	auth helper.Auth
}

func SetupAdminBadgeRoutes(rh *rest.RestHandler) {
	svc := services.NewAdminBadgeService(rh.DB)
	h := &AdminBadgeHandler{svc: svc, auth: rh.Auth}
	rh.App.Get("/admin/badges", rh.Middlewares.AuthorizeAdmin, h.GetBadgeCounts)
}

func (h *AdminBadgeHandler) GetBadgeCounts(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	counts, err := h.svc.GetCounts()
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", counts)
}

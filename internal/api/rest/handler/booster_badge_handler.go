package handler

import (
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"flyup/internal/services"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

type BoosterBadgeHandler struct {
	svc  services.BoosterBadgeService
	auth helper.Auth
}

func SetupBoosterBadgeRoutes(rh *rest.RestHandler) {
	svc := services.NewBoosterBadgeService(repository.NewBoosterBadgeRepository(rh.DB))
	h := &BoosterBadgeHandler{svc: svc, auth: rh.Auth}
	rh.App.Get("/booster/badges", rh.Middlewares.Authorize, h.GetBadgeCounts)
}

func (h *BoosterBadgeHandler) GetBadgeCounts(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	counts, err := h.svc.GetCounts(user.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", counts)
}

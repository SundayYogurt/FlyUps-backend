package handler

import (
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
)

type PioneerBadgeHandler struct {
	repo repository.PioneerBadgeRepository
	auth helper.Auth
}

func SetupPioneerBadgeRoutes(rh *rest.RestHandler) {
	h := &PioneerBadgeHandler{repo: repository.NewPioneerBadgeRepository(rh.DB), auth: rh.Auth}
	rh.App.Get("/pioneer/badges", rh.Middlewares.Authorize, h.GetBadgeCounts)
}

type PioneerBadgeCounts struct {
	ActiveMilestones int64 `json:"active_milestones"`
	UpcomingMeetings int64 `json:"upcoming_meetings"`
	PendingPayouts   int64 `json:"pending_payouts"`
}

func (h *PioneerBadgeHandler) GetBadgeCounts(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	var counts PioneerBadgeCounts
	today := time.Now().UTC().Format("2006-01-02")

	counts.ActiveMilestones, _ = h.repo.CountActiveMilestones(user.ID)
	counts.UpcomingMeetings, _ = h.repo.CountUpcomingMeetings(user.ID, today)
	counts.PendingPayouts, _ = h.repo.CountPendingPayouts(user.ID)

	return rest.SuccessResponse(ctx, "success", counts)
}

package handler

import (
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
)

type BoosterBadgeHandler struct {
	repo repository.BoosterBadgeRepository
	auth helper.Auth
}

func SetupBoosterBadgeRoutes(rh *rest.RestHandler) {
	h := &BoosterBadgeHandler{repo: repository.NewBoosterBadgeRepository(rh.DB), auth: rh.Auth}
	rh.App.Get("/booster/badges", rh.Middlewares.Authorize, h.GetBadgeCounts)
}

type BoosterBadgeCounts struct {
	PendingVotes     int64 `json:"pending_votes"`
	UpcomingMeetings int64 `json:"upcoming_meetings"`
	PendingRefunds   int64 `json:"pending_refunds"`
	OpenComplaints   int64 `json:"open_complaints"`
}

func (h *BoosterBadgeHandler) GetBadgeCounts(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	var counts BoosterBadgeCounts
	today := time.Now().UTC().Format("2006-01-02")

	counts.PendingVotes, _ = h.repo.CountPendingVotes(user.ID)
	counts.UpcomingMeetings, _ = h.repo.CountUpcomingMeetings(user.ID, today)
	counts.PendingRefunds, _ = h.repo.CountPendingRefunds(user.ID)
	counts.OpenComplaints, _ = h.repo.CountOpenComplaints(user.ID)

	return rest.SuccessResponse(ctx, "success", counts)
}

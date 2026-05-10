package handlers

import (
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type PioneerBadgeHandler struct {
	db   *gorm.DB
	auth helper.Auth
}

func SetupPioneerBadgeRoutes(rh *rest.RestHandler) {
	h := &PioneerBadgeHandler{db: rh.DB, auth: rh.Auth}
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

	// milestone ที่ active หรือ rejected ในโปรเจกต์ของ pioneer
	h.db.Raw(`
		SELECT COUNT(m.id)
		FROM milestones m
		JOIN projects p ON p.id = m.project_id
		WHERE p.pioneer_user_id = ?
		  AND p.deleted_at IS NULL
		  AND m.status IN ('active', 'rejected')
		  AND m.deleted_at IS NULL
	`, user.ID).Scan(&counts.ActiveMilestones)

	// การประชุมที่ยังไม่ผ่านและยังไม่ถูกยกเลิก
	h.db.Raw(`
		SELECT COUNT(mt.id)
		FROM meetings mt
		JOIN milestones m ON m.id = mt.milestone_id
		JOIN projects p ON p.id = m.project_id
		WHERE p.pioneer_user_id = ?
		  AND p.deleted_at IS NULL
		  AND mt.status = 'open'
		  AND mt.deleted_at IS NULL
		  AND mt.date >= ?
	`, user.ID, today).Scan(&counts.UpcomingMeetings)

	// disbursements ที่รอยืนยัน
	h.db.Table("disbursements").
		Where("pioneer_user_id = ? AND status = ? AND deleted_at IS NULL", user.ID, "pending").
		Count(&counts.PendingPayouts)

	return rest.SuccessResponse(ctx, "success", counts)
}

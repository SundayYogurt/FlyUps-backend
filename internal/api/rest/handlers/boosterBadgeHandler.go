package handlers

import (
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type BoosterBadgeHandler struct {
	db   *gorm.DB
	auth helper.Auth
}

func SetupBoosterBadgeRoutes(rh *rest.RestHandler) {
	h := &BoosterBadgeHandler{db: rh.DB, auth: rh.Auth}
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

	// milestone ที่เปิด voting แล้ว แต่ booster ยังไม่ได้โหวต
	h.db.Raw(`
		SELECT COUNT(DISTINCT m.id)
		FROM milestones m
		JOIN investments i ON i.project_id = m.project_id
		WHERE i.booster_user_id = ?
		  AND i.status IN ('verified', 'paid')
		  AND i.deleted_at IS NULL
		  AND m.voting_open = true
		  AND m.deleted_at IS NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM milestone_votes mv
		    WHERE mv.milestone_id = m.id
		      AND mv.booster_user_id = ?
		  )
	`, user.ID, user.ID).Scan(&counts.PendingVotes)

	// การประชุมที่กำลังจะมาสำหรับโปรเจกต์ที่ booster ลงทุน
	h.db.Raw(`
		SELECT COUNT(DISTINCT mt.id)
		FROM meetings mt
		JOIN milestones m ON m.id = mt.milestone_id
		JOIN investments i ON i.project_id = m.project_id
		WHERE i.booster_user_id = ?
		  AND i.status IN ('verified', 'paid')
		  AND i.deleted_at IS NULL
		  AND mt.status = 'open'
		  AND mt.deleted_at IS NULL
		  AND mt.date >= ?
	`, user.ID, today).Scan(&counts.UpcomingMeetings)

	// การลงทุนที่ขอคืนเงินและรอการอนุมัติ
	h.db.Table("investments").
		Where("booster_user_id = ? AND status = ? AND deleted_at IS NULL", user.ID, "refund_pending").
		Count(&counts.PendingRefunds)

	// คำร้องเรียนที่ยังเปิดอยู่
	h.db.Table("complaints").
		Where("complainant_id = ? AND status = ? AND deleted_at IS NULL", user.ID, "open").
		Count(&counts.OpenComplaints)

	return rest.SuccessResponse(ctx, "success", counts)
}

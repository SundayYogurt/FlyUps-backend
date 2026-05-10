package handlers

import (
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type AdminBadgeHandler struct {
	db   *gorm.DB
	auth helper.Auth
}

func SetupAdminBadgeRoutes(rh *rest.RestHandler) {
	h := &AdminBadgeHandler{db: rh.DB, auth: rh.Auth}
	rh.App.Get("/admin/badges", rh.Middlewares.AuthorizeAdmin, h.GetBadgeCounts)
}

type BadgeCounts struct {
	PendingProjects      int64 `json:"pending_projects"`
	SubmittedMilestones  int64 `json:"submitted_milestones"`
	PendingCancelReqs    int64 `json:"pending_cancel_requests"`
	OpenComplaints       int64 `json:"open_complaints"`
	PendingRefunds       int64 `json:"pending_refunds"`
	PendingVerifications int64 `json:"pending_verifications"`
	PendingDisbursements int64 `json:"pending_disbursements"`
	PendingProfitPools   int64 `json:"pending_profit_pools"`
}

func (h *AdminBadgeHandler) GetBadgeCounts(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	var counts BadgeCounts

	h.db.Table("projects").Where("state = ? AND deleted_at IS NULL", "pending_review").Count(&counts.PendingProjects)
	h.db.Table("milestones").Where("status = ? AND deleted_at IS NULL", "submitted").Count(&counts.SubmittedMilestones)
	h.db.Table("projects").Where("state = ? AND deleted_at IS NULL", "pending_cancel").Count(&counts.PendingCancelReqs)
	h.db.Table("complaints").Where("status = ? AND deleted_at IS NULL", "open").Count(&counts.OpenComplaints)
	h.db.Table("investments").Where("status = ? AND deleted_at IS NULL", "refund_pending").Count(&counts.PendingRefunds)
	h.db.Table("disbursements").Where("status = ? AND deleted_at IS NULL", "pending").Count(&counts.PendingDisbursements)
	h.db.Table("profit_pools").Where("status = ? AND deleted_at IS NULL", "pending").Count(&counts.PendingProfitPools)

	var studentCard, idCard int64
	h.db.Table("student_card_verifications").Where("status = ? AND deleted_at IS NULL", "pending").Count(&studentCard)
	h.db.Table("id_card_verifications").Where("status = ? AND deleted_at IS NULL", "pending").Count(&idCard)
	counts.PendingVerifications = studentCard + idCard

	return rest.SuccessResponse(ctx, "success", counts)
}

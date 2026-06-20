package handler

import (
	"flyup/internal/api/rest"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/service"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

type AdminLogHandler struct {
	svc  service.AdminLogService
	auth helper.Auth
}

func SetupAdminLogRoutes(rh *rest.RestHandler) {
	h := &AdminLogHandler{
		svc:  rh.AdminLogSvc,
		auth: rh.Auth,
	}
	rh.App.Get("/admin/logs", rh.Middlewares.AuthorizeAdmin, h.ListLogs)
}

// ListLogs godoc
// @Summary      List admin action logs
// @Description  Admin ดึงประวัติ action ทั้งหมดที่ admin กระทำในระบบ พร้อม pagination และ filter
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        page        query  int    false  "Page number (default 1)"
// @Param        page_size   query  int    false  "Page size (default 20, max 100)"
// @Param        admin_id    query  int    false  "Filter by admin user ID"
// @Param        action      query  string false  "Filter by action name เช่น approve_project"
// @Param        target_type query  string false  "Filter by target type เช่น user, project"
// @Param        target_id   query  int    false  "Filter by target entity ID"
// @Param        from        query  string false  "Filter from date (YYYY-MM-DD)"
// @Param        to          query  string false  "Filter to date (YYYY-MM-DD)"
// @Success      200  {object}  object  "success"
// @Failure      401  {object}  object  "Unauthorized"
// @Failure      500  {object}  object  "Internal Server Error"
// @Router       /admin/logs [get]
func (h *AdminLogHandler) ListLogs(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	filter := dto.AdminLogFilter{}

	// page
	if p := ctx.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			filter.Page = v
		}
	}
	if filter.Page < 1 {
		filter.Page = 1
	}

	// page_size
	if ps := ctx.Query("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			filter.PageSize = v
		}
	}
	if filter.PageSize < 1 {
		filter.PageSize = 20
	}

	// admin_id
	if aid := ctx.Query("admin_id"); aid != "" {
		if v, err := strconv.ParseUint(aid, 10, 64); err == nil {
			uid := uint(v)
			filter.AdminID = &uid
		}
	}

	// target_id
	if tid := ctx.Query("target_id"); tid != "" {
		if v, err := strconv.ParseUint(tid, 10, 64); err == nil {
			uid := uint(v)
			filter.TargetID = &uid
		}
	}

	filter.Action = ctx.Query("action")
	filter.TargetType = ctx.Query("target_type")
	filter.From = ctx.Query("from")
	filter.To = ctx.Query("to")

	logs, total, err := h.svc.ListLogs(filter)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{
		"message": "success",
		"data":    logs,
		"meta": fiber.Map{
			"total":     total,
			"page":      filter.Page,
			"page_size": filter.PageSize,
		},
	})
}

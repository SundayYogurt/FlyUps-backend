package handler

import (
	"flyup/internal/api/rest"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"flyup/internal/services"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type ComplaintHandler struct {
	svc       services.ComplaintService
	validator *validator.Validate
	auth      helper.Auth
}

func SetupComplaintRoutes(rh *rest.RestHandler) {
	svc := services.NewComplaintService(
		repository.NewComplaintRepository(rh.DB),
		repository.NewProjectRepository(rh.DB),
	)

	h := &ComplaintHandler{
		svc:       svc,
		validator: rh.Validator,
		auth:      rh.Auth,
	}

	// Logged-in users
	user := rh.App.Group("/complaints", rh.Middlewares.Authorize)
	user.Post("/", h.Create)
	user.Get("/me", h.ListMine)

	// Admin
	admin := rh.App.Group("/admin/complaints", rh.Middlewares.AuthorizeAdmin)
	admin.Get("/", h.AdminList)
	admin.Get("/project-stats/:project_id", h.AdminProjectStats)
	admin.Get("/:id", h.AdminGet)
	admin.Patch("/:id/resolve", h.AdminResolve)
	admin.Patch("/:id/reject", h.AdminReject)
}

// Create godoc
// @Summary      File a complaint about a project
// @Description  Logged-in user files a complaint (max 1 per project per user)
// @Tags         Complaints
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.CreateComplaintRequest true "Complaint payload"
// @Success      200 {object} map[string]interface{}
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      409 {object} map[string]string "already filed"
// @Router       /complaints [post]
func (h *ComplaintHandler) Create(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	var req dto.CreateComplaintRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	c, err := h.svc.Create(user.ID, req)
	if err != nil {
		if err.Error() == "you have already filed a complaint for this project" {
			return ctx.Status(http.StatusConflict).JSON(fiber.Map{"message": err.Error()})
		}
		if err.Error() == "project not found" {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return rest.BadRequestError(ctx, err.Error())
	}
	return rest.SuccessResponse(ctx, "complaint filed", c)
}

// ListMine godoc
// @Summary      List my complaints
// @Tags         Complaints
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{}
// @Router       /complaints/me [get]
func (h *ComplaintHandler) ListMine(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}
	items, err := h.svc.ListMine(user.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", items)
}

// AdminList godoc
// @Summary      List all complaints (admin only)
// @Tags         Complaints
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "open|resolved|rejected"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/complaints [get]
func (h *ComplaintHandler) AdminList(ctx fiber.Ctx) error {
	var statusFilter *domain.ComplaintStatus
	if s := ctx.Query("status"); s != "" {
		st := domain.ComplaintStatus(s)
		if st != domain.ComplaintOpen && st != domain.ComplaintResolved && st != domain.ComplaintRejected {
			return rest.BadRequestError(ctx, "invalid status")
		}
		statusFilter = &st
	}
	items, err := h.svc.AdminList(statusFilter)
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", items)
}

// AdminGet godoc
// @Summary      Get a single complaint (admin only)
// @Tags         Complaints
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Complaint ID"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/complaints/{id} [get]
func (h *ComplaintHandler) AdminGet(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid complaint id")
	}
	item, err := h.svc.AdminGet(uint(id))
	if err != nil {
		return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
	}
	return rest.SuccessResponse(ctx, "success", item)
}

// AdminResolve godoc
// @Summary      Resolve a complaint (admin only)
// @Tags         Complaints
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "Complaint ID"
// @Param        body body dto.ResolveComplaintRequest true "Resolution note"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/complaints/{id}/resolve [patch]
func (h *ComplaintHandler) AdminResolve(ctx fiber.Ctx) error {
	return h.adminClose(ctx, true)
}

// AdminReject godoc
// @Summary      Reject a complaint (admin only)
// @Tags         Complaints
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "Complaint ID"
// @Param        body body dto.ResolveComplaintRequest true "Rejection note"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/complaints/{id}/reject [patch]
func (h *ComplaintHandler) AdminReject(ctx fiber.Ctx) error {
	return h.adminClose(ctx, false)
}

// AdminProjectStats godoc
// @Summary Get complaint stats for a project (admin only)
// @Description Admin gets complaint statistics (total, open, resolved, rejected) for a specific project
// @Tags Complaints
// @Produce json
// @Security BearerAuth
// @Param project_id path int true "Project ID"
// @Success 200 {object} map[string]interface{} "Complaint statistics"
// @Failure 400 {object} map[string]string "Invalid project ID"
// @Failure 404 {object} map[string]string "Project not found"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /admin/complaints/project-stats/{project_id} [get]
func (h *ComplaintHandler) AdminProjectStats(ctx fiber.Ctx) error {
	projectID, err := strconv.ParseUint(ctx.Params("project_id"), 10, 32)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid project id")
	}
	stats, err := h.svc.GetProjectStats(uint(projectID))
	if err != nil {
		if err.Error() == "project not found" {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", stats)
}

func (h *ComplaintHandler) adminClose(ctx fiber.Ctx, resolve bool) error {
	admin := h.auth.GetCurrentUser(ctx)
	if admin.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid complaint id")
	}
	var req dto.ResolveComplaintRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	var (
		c     *domain.Complaint
		opErr error
	)
	if resolve {
		c, opErr = h.svc.AdminResolve(uint(id), admin.ID, req.AdminNote)
	} else {
		c, opErr = h.svc.AdminReject(uint(id), admin.ID, req.AdminNote)
	}
	if opErr != nil {
		if opErr.Error() == "complaint not found" {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": opErr.Error()})
		}
		return rest.BadRequestError(ctx, opErr.Error())
	}

	msg := "complaint resolved"
	if !resolve {
		msg = "complaint rejected"
	}
	return rest.SuccessResponse(ctx, msg, c)
}

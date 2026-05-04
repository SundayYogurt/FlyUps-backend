package handlers

import (
	"flyup/internal/api/rest"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"flyup/internal/service"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type DisbursementHandler struct {
	svc       service.DisbursementService
	validator *validator.Validate
	auth      helper.Auth
}

func SetupDisbursementRoutes(rh *rest.RestHandler) {
	svc := service.NewDisbursementService(
		repository.NewDisbursementRepository(rh.DB),
		repository.NewProjectRepository(rh.DB),
		repository.NewUserRepository(rh.DB),
		rh.NotifSvc,
	)

	h := &DisbursementHandler{
		svc:       svc,
		validator: rh.Validator,
		auth:      rh.Auth,
	}

	admin := rh.App.Group("/admin/disbursements", rh.Middlewares.AuthorizeAdmin)
	admin.Get("/", h.ListAll)
	admin.Get("/pending", h.ListPending)
	admin.Patch("/:id/confirm", h.Confirm)

	pioneer := rh.App.Group("/pioneer/payouts", rh.Middlewares.AuthorizePioneer)
	pioneer.Get("/", h.ListMyPayouts)
}

// ListMyPayouts godoc
// @Summary List my payouts
// @Description Pioneer gets all their disbursement records (pending + confirmed)
// @Tags Disbursements
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "List of pioneer payouts"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /pioneer/payouts [get]
func (h *DisbursementHandler) ListMyPayouts(ctx fiber.Ctx) error {
	currentUser := h.auth.GetCurrentUser(ctx)
	if currentUser.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}
	items, err := h.svc.ListMyPayouts(currentUser.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", items)
}

// ListAll godoc
// @Summary      List all disbursements (admin only)
// @Description  Get all disbursement records (pending + confirmed) including pioneer bank account
// @Tags         Disbursements
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "list of disbursements"
// @Failure      401  {object}  map[string]string       "unauthorized"
// @Failure      500  {object}  map[string]string       "internal server error"
// @Router       /admin/disbursements [get]
func (h *DisbursementHandler) ListAll(ctx fiber.Ctx) error {
	items, err := h.svc.ListAll()
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", items)
}

// ListPending godoc
// @Summary      List pending disbursements (admin only)
// @Description  Get disbursements awaiting admin confirmation
// @Tags         Disbursements
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "list of pending disbursements"
// @Failure      401  {object}  map[string]string       "unauthorized"
// @Failure      500  {object}  map[string]string       "internal server error"
// @Router       /admin/disbursements/pending [get]
func (h *DisbursementHandler) ListPending(ctx fiber.Ctx) error {
	items, err := h.svc.ListPending()
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", items)
}

// Confirm godoc
// @Summary      Confirm disbursement transfer (admin only)
// @Description  Admin confirms that funds have been transferred to the pioneer
// @Tags         Disbursements
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                              true  "Disbursement ID"
// @Param        body  body      dto.ConfirmDisbursementRequest   true  "Confirmation payload"
// @Success      200   {object}  map[string]interface{}           "disbursement confirmed"
// @Failure      400   {object}  map[string]string                "invalid request"
// @Failure      401   {object}  map[string]string                "unauthorized"
// @Failure      404   {object}  map[string]string                "disbursement not found"
// @Router       /admin/disbursements/{id}/confirm [patch]
func (h *DisbursementHandler) Confirm(ctx fiber.Ctx) error {
	currentUser := h.auth.GetCurrentUser(ctx)
	if currentUser.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid disbursement id")
	}

	var req dto.ConfirmDisbursementRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	result, err := h.svc.Confirm(uint(id), currentUser.ID, req)
	if err != nil {
		if err.Error() == "disbursement not found" {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "disbursement confirmed", result)
}

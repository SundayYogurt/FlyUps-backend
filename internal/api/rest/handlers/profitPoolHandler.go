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

type ProfitPoolHandler struct {
	svc       service.ProfitPoolService
	validator *validator.Validate
	auth      helper.Auth
}

func SetupProfitPoolRoutes(rh *rest.RestHandler) {
	svc := service.NewProfitPoolService(
		repository.NewProfitPoolRepository(rh.DB),
		repository.NewProjectRepository(rh.DB),
		repository.NewInvestmentRepository(rh.DB),
		repository.NewUserRepository(rh.DB),
		rh.NotifSvc,
	)
	h := &ProfitPoolHandler{svc, rh.Validator, rh.Auth}

	admin := rh.App.Group("/admin/profit-pools", rh.Middlewares.AuthorizeAdmin)
	admin.Post("/", h.Create)
	admin.Get("/", h.List)
	admin.Get("/:id", h.GetDetail)
	admin.Patch("/:id/payouts/:payoutId/confirm", h.ConfirmPayout)
}

func (h *ProfitPoolHandler) Create(ctx fiber.Ctx) error {
	currentUser := h.auth.GetCurrentUser(ctx)
	if currentUser.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}
	var req dto.CreateProfitPoolRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}
	result, err := h.svc.Create(currentUser.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}
	return rest.SuccessResponse(ctx, "profit pool created", result)
}

func (h *ProfitPoolHandler) List(ctx fiber.Ctx) error {
	items, err := h.svc.List()
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", items)
}

func (h *ProfitPoolHandler) GetDetail(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid id")
	}
	detail, err := h.svc.GetDetail(uint(id))
	if err != nil {
		if err.Error() == "profit pool not found" {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", detail)
}

func (h *ProfitPoolHandler) ConfirmPayout(ctx fiber.Ctx) error {
	currentUser := h.auth.GetCurrentUser(ctx)
	if currentUser.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}
	poolID, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid pool id")
	}
	payoutID, err := strconv.ParseUint(ctx.Params("payoutId"), 10, 32)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid payout id")
	}
	var req dto.ConfirmInvestorPayoutRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}
	if err := h.svc.ConfirmPayout(uint(poolID), uint(payoutID), currentUser.ID, req); err != nil {
		if err.Error() == "profit pool not found" || err.Error() == "payout not found" {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return rest.BadRequestError(ctx, err.Error())
	}
	return rest.SuccessResponse(ctx, "payout confirmed", nil)
}

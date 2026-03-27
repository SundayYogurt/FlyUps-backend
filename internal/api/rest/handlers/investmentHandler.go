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

type InvestmentHandler struct {
	svc       service.InvestmentService
	validator *validator.Validate
	auth      helper.Auth
}

func SetupInvestmentRoutes(rh *rest.RestHandler) {
	svc := service.NewInvestmentService(
		repository.NewProjectRepository(rh.DB),
		repository.NewInvestmentRepository(rh.DB),
		repository.NewTransactionRepository(rh.DB),
		rh.Config.StripeSecretKey,
		rh.Config.StripeWebhookSecret,
	)

	h := &InvestmentHandler{
		svc:       svc,
		validator: rh.Validator,
		auth:      rh.Auth,
	}

	rh.App.Post("/stripe/webhook", h.StripeWebhook)

	priv := rh.App.Group("/investments", rh.Middlewares.Authorize)
	priv.Post("/", h.CreateInvestment)
	priv.Get("/", h.ListMyInvestments)
	priv.Get("/:id", h.GetInvestment)
}

func (h *InvestmentHandler) GetInvestment(ctx fiber.Ctx) error {
	currentUser := h.auth.GetCurrentUser(ctx)
	if currentUser.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid investment id")
	}

	investment, txn, err := h.svc.GetInvestment(currentUser.ID, uint(id))
	if err != nil {
		if err.Error() == "investment not found" {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", fiber.Map{
		"investment":  investment,
		"transaction": txn,
	})
}

func (h *InvestmentHandler) CreateInvestment(ctx fiber.Ctx) error {
	currentUser := h.auth.GetCurrentUser(ctx)
	if currentUser.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	var req dto.CreateInvestmentRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	result, err := h.svc.CreateInvestment(currentUser.ID, currentUser.Email, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return ctx.Status(http.StatusCreated).JSON(fiber.Map{
		"message": "investment created",
		"data":    result,
	})
}

func (h *InvestmentHandler) ListMyInvestments(ctx fiber.Ctx) error {
	currentUser := h.auth.GetCurrentUser(ctx)

	if currentUser.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	investments, err := h.svc.ListUserInvestments(currentUser.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", investments)
}

func (h *InvestmentHandler) StripeWebhook(ctx fiber.Ctx) error {
	sig := ctx.Get("Stripe-Signature")
	payload := ctx.Body()

	if err := h.svc.HandleStripeWebhook(payload, sig); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"received": true})
}

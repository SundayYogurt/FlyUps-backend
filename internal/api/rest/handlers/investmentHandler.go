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

// GetInvestment godoc
// @Summary      Get investment detail
// @Description  Get a specific investment and its transaction by investment ID (booster only)
// @Tags         Investments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Investment ID"
// @Success      200  {object}  map[string]interface{}  "investment and transaction detail"
// @Failure      400  {object}  map[string]string       "invalid investment id"
// @Failure      401  {object}  map[string]string       "unauthorized"
// @Failure      404  {object}  map[string]string       "investment not found"
// @Failure      500  {object}  map[string]string       "internal server error"
// @Router       /investments/{id} [get]
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

// CreateInvestment godoc
// @Summary      Create an investment
// @Description  Create a new investment for a project and get a PromptPay QR code
// @Tags         Investments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.CreateInvestmentRequest  true  "Investment request"
// @Success      201   {object}  dto.InvestmentResponse       "investment created with QR code"
// @Failure      400   {object}  map[string]string            "invalid request or business rule violation"
// @Failure      401   {object}  map[string]string            "unauthorized"
// @Router       /investments [post]
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

// ListMyInvestments godoc
// @Summary      List my investments
// @Description  Get all investments made by the current booster user
// @Tags         Investments
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "list of investments"
// @Failure      401  {object}  map[string]string       "unauthorized"
// @Failure      500  {object}  map[string]string       "internal server error"
// @Router       /investments [get]
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

// StripeWebhook godoc
// @Summary      Stripe webhook
// @Description  Endpoint for Stripe to send payment events (payment_intent.succeeded, payment_intent.payment_failed)
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Param        Stripe-Signature  header    string  true  "Stripe webhook signature"
// @Success      200               {object}  map[string]bool    "webhook received"
// @Failure      400               {object}  map[string]string  "invalid signature or payload"
// @Router       /stripe/webhook [post]
func (h *InvestmentHandler) StripeWebhook(ctx fiber.Ctx) error {
	sig := ctx.Get("Stripe-Signature")
	payload := ctx.Body()

	if err := h.svc.HandleStripeWebhook(payload, sig); err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{"message": err.Error()})
	}

	return ctx.Status(http.StatusOK).JSON(fiber.Map{"received": true})
}

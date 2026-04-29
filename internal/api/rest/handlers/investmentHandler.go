package handlers

import (
	"flyup/internal/api/rest"
	"flyup/internal/domain"
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
		repository.NewUserRepository(rh.DB),
		repository.NewDisbursementRepository(rh.DB),
		rh.Config.StripeSecretKey,
		rh.Config.StripeWebhookSecret,
		rh.NotifSvc,
		rh.Notification,
	)

	h := &InvestmentHandler{
		svc:       svc,
		validator: rh.Validator,
		auth:      rh.Auth,
	}

	rh.App.Post("/stripe/webhook", h.StripeWebhook)
	rh.App.Get("/investments/projects/:projectId/investors", h.GetProjectInvestors)

	priv := rh.App.Group("/investments", rh.Middlewares.Authorize)
	priv.Post("/", h.CreateInvestment)
	priv.Get("/", h.ListMyInvestments)
	priv.Get("/my-projects", h.ListMyInvestedProjects)
	priv.Get("/:id", h.GetInvestment)
	priv.Post("/:id/refund", h.RefundInvestment)
	priv.Post("/milestones/:milestone_id/vote", h.VoteMilestone)

	admin := rh.App.Group("/admin/investments", rh.Middlewares.AuthorizeAdmin)
	admin.Get("/refund-requests", h.ListRefundRequests)
	admin.Patch("/:id/approve-refund", h.ApproveRefund)
}

// VoteMilestone godoc
// @Summary Vote on milestone submission
// @Description Verified investor votes approve/reject on a submitted milestone
// @Tags Investments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param milestone_id path int true "Milestone ID"
// @Param request body dto.VoteMilestoneRequest true "Vote payload"
// @Success 200 {object} object "Vote saved"
// @Failure 400 {object} object "Invalid request"
// @Failure 401 {object} object "Unauthorized"
// @Router /investments/milestones/{milestone_id}/vote [post]
func (h *InvestmentHandler) VoteMilestone(ctx fiber.Ctx) error {
	currentUser := h.auth.GetCurrentUser(ctx)
	if currentUser.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	milestoneID, err := strconv.Atoi(ctx.Params("milestone_id"))
	if err != nil || milestoneID <= 0 {
		return rest.BadRequestError(ctx, "invalid milestone id")
	}

	var req dto.VoteMilestoneRequest
	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	choice := domain.MilestoneVoteChoice(req.Choice)
	v, err := h.svc.VoteMilestone(currentUser.ID, uint(milestoneID), choice)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "vote saved", v)
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

// ListRefundRequests godoc
// @Summary      List pending refund requests (admin only)
// @Description  Get all investments with refund_pending status including booster bank account
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "list of refund requests"
// @Failure      403  {object}  map[string]string       "access denied"
// @Router       /admin/investments/refund-requests [get]
func (h *InvestmentHandler) ListRefundRequests(ctx fiber.Ctx) error {
	result, err := h.svc.ListRefundRequests()
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", result)
}

// ApproveRefund godoc
// @Summary      Approve a refund request (admin only)
// @Description  Mark a refund_pending investment as refunded after manual bank transfer
// @Tags         Admin
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Investment ID"
// @Success      200  {object}  map[string]string  "refund approved"
// @Failure      400  {object}  map[string]string  "invalid id or business rule violation"
// @Failure      403  {object}  map[string]string  "access denied"
// @Router       /admin/investments/{id}/approve-refund [patch]
func (h *InvestmentHandler) ApproveRefund(ctx fiber.Ctx) error {
	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid investment id")
	}

	if err := h.svc.ApproveRefund(uint(id)); err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "refund approved successfully", nil)
}

// RefundInvestment godoc
// @Summary      Refund an investment
// @Description  Refund a verified investment while project is in funding state. Platform fee and VAT are non-refundable. Requires a note for admin approval.
// @Tags         Investments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int                          true  "Investment ID"
// @Param        body  body      dto.RefundInvestmentRequest  true  "Refund request with note"
// @Success      200   {object}  dto.RefundResponse           "refund processed"
// @Failure      400   {object}  map[string]string            "invalid id, missing note, or business rule violation"
// @Failure      401   {object}  map[string]string            "unauthorized"
// @Failure      404   {object}  map[string]string            "investment not found"
// @Router       /investments/{id}/refund [post]
func (h *InvestmentHandler) RefundInvestment(ctx fiber.Ctx) error {
	currentUser := h.auth.GetCurrentUser(ctx)
	if currentUser.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	id, err := strconv.ParseUint(ctx.Params("id"), 10, 32)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid investment id")
	}

	var req dto.RefundInvestmentRequest
	if err := ctx.Bind().JSON(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if req.Note == "" {
		return rest.BadRequestError(ctx, "note is required")
	}

	result, err := h.svc.RefundInvestment(currentUser.ID, uint(id), req.Note)
	if err != nil {
		if err.Error() == "investment not found" {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "refund processed successfully", result)
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

// GetProjectInvestors godoc
// @Summary Get project investors
// @Description Get all investors for a specific project (public endpoint)
// @Tags Investments
// @Produce json
// @Param projectId path int true "Project ID"
// @Success 200 {object} object "Project investors list"
// @Failure 400 {object} object "Invalid project ID"
// @Failure 404 {object} object "Project not found"
// @Failure 500 {object} object "Internal Server Error"
// @Router /investments/projects/{projectId}/investors [get]
func (h *InvestmentHandler) GetProjectInvestors(ctx fiber.Ctx) error {
	projectID, err := strconv.ParseUint(ctx.Params("projectId"), 10, 32)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid project id")
	}
	investors, err := h.svc.GetProjectInvestors(uint(projectID))
	if err != nil {
		if err.Error() == "project not found" {
			return ctx.Status(http.StatusNotFound).JSON(fiber.Map{"message": err.Error()})
		}
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", fiber.Map{
		"project_id": projectID,
		"total":      len(investors),
		"investors":  investors,
	})
}

// ListMyInvestedProjects godoc
// @Summary List my invested projects
// @Description Get all projects that the current booster user has invested in
// @Tags Investments
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object "List of invested projects"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal Server Error"
// @Router /investments/my-projects [get]
func (h *InvestmentHandler) ListMyInvestedProjects(ctx fiber.Ctx) error {
	currentUser := h.auth.GetCurrentUser(ctx)
	if currentUser.ID == 0 {
		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{"message": "unauthorized"})
	}

	projects, err := h.svc.ListInvestedProjects(currentUser.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", projects)
}

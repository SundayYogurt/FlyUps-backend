package handlers

import (
	"flyup/internal/api/rest"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"flyup/internal/service"
	"net/http"

	"github.com/gofiber/fiber/v3"
)
type ChatHandler struct {
	chatService service.ChatService
	auth        helper.Auth
}

func SetupChatRoutes(rh *rest.RestHandler) {
	chatRepo := repository.NewChatRepository(rh.DB)
	chatAIClient := service.NewChatAIClient(rh.Config.OpenAIAPIKey, rh.Config.OpenAIModel)

	investmentRepo := repository.NewInvestmentRepository(rh.DB)
	transactionRepo := repository.NewTransactionRepository(rh.DB)
	disbursementRepo := repository.NewDisbursementRepository(rh.DB)
	userRepo := repository.NewUserRepository(rh.DB)
	notifRepo := repository.NewNotificationRepository(rh.DB)
	notifSvc := service.NewNotificationService(notifRepo)
	projectRepo := repository.NewProjectRepository(rh.DB)

	investmentSvc := service.NewInvestmentService(
		projectRepo,
		investmentRepo,
		transactionRepo,
		userRepo,
		disbursementRepo,
		rh.Config.StripeSecretKey,
		rh.Config.StripeWebhookSecret,
		notifSvc,
		rh.Notification,
	)

	projectSvc := service.NewProjectService(
		projectRepo,
		repository.NewUserRepository(rh.DB),
		rh.Cloudinary,
		notifSvc,
		rh.Notification,
		investmentSvc,
	)

	disburseSvc := service.NewDisbursementService(
		disbursementRepo,
		projectRepo,
		userRepo,
		notifSvc,
	)

	complaintSvc := service.NewComplaintService(
		repository.NewComplaintRepository(rh.DB),
		projectRepo,
	)

	chatService := service.NewChatService(chatRepo, chatAIClient, investmentSvc, projectSvc, notifSvc, disburseSvc, complaintSvc)
	chatHandler := NewChatHandler(chatService, rh.Auth)

	chat := rh.App.Group("/chat", rh.Middlewares.Authorize)
	chat.Post("/messages", chatHandler.SendMessage)
	chat.Post("/actions/confirm", chatHandler.ConfirmAction)
}

func NewChatHandler(chatService service.ChatService, auth helper.Auth) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		auth:        auth,
	}
}

func (h *ChatHandler) SendMessage(c fiber.Ctx) error {
	user := h.auth.GetCurrentUser(c)
	if user.ID == 0 {
		return rest.ErrorMessage(c, http.StatusUnauthorized, fiber.NewError(fiber.StatusUnauthorized, "unauthorized"))
	}

	var req dto.SendChatMessageRequest
	if err := c.Bind().Body(&req); err != nil {
		return rest.BadRequestError(c, "invalid request body")
	}

	res, err := h.chatService.SendMessage(user.ID, req)
	if err != nil {
		return rest.InternalError(c, err)
	}

	return rest.SuccessResponse(c, "ok", res)
}

func (h *ChatHandler) ConfirmAction(c fiber.Ctx) error {
	user := h.auth.GetCurrentUser(c)
	if user.ID == 0 {
		return rest.ErrorMessage(c, http.StatusUnauthorized, fiber.NewError(fiber.StatusUnauthorized, "unauthorized"))
	}

	var req dto.ConfirmChatActionRequest
	if err := c.Bind().Body(&req); err != nil {
		return rest.BadRequestError(c, "invalid request body")
	}

	res, err := h.chatService.ConfirmAction(user.ID, req)
	if err != nil {
		return rest.InternalError(c, err)
	}

	return rest.SuccessResponse(c, "ok", res)
}


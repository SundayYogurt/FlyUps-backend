package handler

import (
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"flyup/internal/services"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

type FinancialHandler struct {
	svc  services.FinancialService
	auth helper.Auth
}

func SetupFinancialRoutes(rh *rest.RestHandler) {
	repo := repository.NewFinancialRepository(rh.DB)
	svc := services.NewFinancialService(repo, rh.Config.StripeSecretKey)
	h := &FinancialHandler{svc: svc, auth: rh.Auth}

	admin := rh.App.Group("/admin/financial", rh.Middlewares.AuthorizeAdmin)
	admin.Get("/summary", h.GetSummary)
}

func SetupProjectFinancialRoutes(rh *rest.RestHandler) {
	repo := repository.NewFinancialRepository(rh.DB)
	svc := services.NewFinancialService(repo, rh.Config.StripeSecretKey)
	h := &FinancialHandler{svc: svc, auth: rh.Auth}

	admin := rh.App.Group("/admin/financial", rh.Middlewares.AuthorizeAdmin)
	admin.Get("/projects", h.GetProjectsFinancial)
}

func (h *FinancialHandler) GetSummary(ctx fiber.Ctx) error {
	summary, err := h.svc.GetSummary()
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", summary)
}

func (h *FinancialHandler) GetProjectsFinancial(ctx fiber.Ctx) error {
	result, err := h.svc.GetProjectsFinancial()
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	if result == nil {
		result = []dto.ProjectFinancialItem{}
	}
	return ctx.Status(http.StatusOK).JSON(fiber.Map{"message": "success", "data": result})
}

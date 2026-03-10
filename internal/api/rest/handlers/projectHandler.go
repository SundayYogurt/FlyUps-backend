package handlers

import (
	"flyup/internal/api/rest"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"flyup/internal/service"
	"log"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type ProjectHandler struct {
	svc       service.ProjectService
	validator *validator.Validate
}

func SetupProjectRoutes(rh *rest.RestHandler) {

	app := rh.App

	svc := service.ProjectService{
		Repo: repository.NewProjectRepository(rh.DB),
		Auth: rh.Auth,
	}

	handler := ProjectHandler{
		svc:       svc,
		validator: validator.New(),
	}

	// private pioneer routes
	pioneerRoutes := app.Group("/pioneer", rh.Middlewares.AuthorizePioneer)

	pioneerRoutes.Post("/projects", handler.CreateProject)
	pioneerRoutes.Patch("/projects/:id", handler.UpdateDraft)
	pioneerRoutes.Post("/projects/:id/media", handler.AddMedia)
	pioneerRoutes.Get("/projects", handler.GetMyProjects)
	pioneerRoutes.Get("/projects/:id", handler.GetProject)
}

func (h *ProjectHandler) CreateProject(ctx fiber.Ctx) error {

	req := dto.CreateProjectRequest{}

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, "validation failed: "+err.Error())
	}

	user := h.svc.Auth.GetCurrentUser(ctx)
	log.Println(user)

	project, err := h.svc.CreateProject(user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "project created", project)
}

func (h *ProjectHandler) UpdateDraft(ctx fiber.Ctx) error {

	idStr := ctx.Params("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	req := dto.UpdateProjectDraftRequest{}

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	user := h.svc.Auth.GetCurrentUser(ctx)
	log.Println(user)

	err = h.svc.UpdateProjectDraft(uint(id), user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "project updated", nil)
}

func (h *ProjectHandler) AddMedia(ctx fiber.Ctx) error {

	idStr := ctx.Params("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	req := dto.AddProjectMediaRequest{}

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	if err := h.validator.Struct(req); err != nil {
		return rest.BadRequestError(ctx, "validation failed: "+err.Error())
	}

	user := h.svc.Auth.GetCurrentUser(ctx)
	log.Println(user)

	err = h.svc.AddProjectMedia(uint(id), user.ID, req)
	if err != nil {
		log.Println("add media error:", err)
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "media added", nil)
}

func (h *ProjectHandler) GetMyProjects(ctx fiber.Ctx) error {

	user := h.svc.Auth.GetCurrentUser(ctx)
	log.Println(user)

	projects, err := h.svc.GetProjectsByOwner(user.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "projects", projects)
}

func (h *ProjectHandler) GetProject(ctx fiber.Ctx) error {

	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid id")
	}

	user := h.svc.Auth.GetCurrentUser(ctx)
	log.Println(user)

	project, err := h.svc.GetProjectByID(uint(id), user.ID)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "project", project)
}

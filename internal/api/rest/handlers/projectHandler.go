package handlers

import (
	"flyup/internal/api/rest"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"flyup/internal/service"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type ProjectHandler struct {
	svc        service.ProjectService
	validator  *validator.Validate
	cloudinary *helper.CloudinaryService
}

func SetupProjectRoutes(rh *rest.RestHandler) {

	app := rh.App

	svc := service.ProjectService{
		Repo: repository.NewProjectRepository(rh.DB),
		Auth: rh.Auth,
	}

	handler := ProjectHandler{
		svc:        svc,
		validator:  validator.New(),
		cloudinary: rh.Cloudinary,
	}

	// private pioneer routes
	pioneerRoutes := app.Group("/pioneer", rh.Middlewares.AuthorizePioneer)

	pioneerRoutes.Post("/projects", handler.CreateProject)
	pioneerRoutes.Patch("/projects/:id", handler.UpdateDraft)
	pioneerRoutes.Post("/projects/:id/media", handler.AddMedia)
	pioneerRoutes.Post("/projects/:id/story", handler.AddStory)
	pioneerRoutes.Post("/projects/:id/risk", handler.AddRisk)
	pioneerRoutes.Post("/projects/:id/faq", handler.AddFAQ)

	pioneerRoutes.Post("/projects/:id/milestone", handler.CreateMilestone)

	pioneerRoutes.Post("/projects/:id/funding-policy", handler.SetFundingPolicy)
	pioneerRoutes.Post("/projects/:id/profit-policy", handler.SetProfitPolicy)

	pioneerRoutes.Post("/projects/:id/submit", handler.SubmitProject)

	pioneerRoutes.Get("/projects", handler.GetMyProjects)
	pioneerRoutes.Get("/projects/:id", handler.GetProjectDetail)

	pioneerRoutes.Post("/projects/:id/publish", handler.PublishProject)
	pioneerRoutes.Get("/projects/:id/validate", handler.ValidateProject)

}

func (h *ProjectHandler) CreateProject(ctx fiber.Ctx) error {

	user := h.svc.Auth.GetCurrentUser(ctx)
	log.Println(user)

	project, err := h.svc.CreateProject(user.ID)
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

	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return rest.BadRequestError(ctx, "file required")
	}

	const maxSize = 5 * 1024 * 1024
	if fileHeader.Size > maxSize {
		return rest.BadRequestError(ctx, "file too large (max 5MB)")
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return rest.BadRequestError(ctx, "file must be image")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	url, err := h.cloudinary.UploadImage(file)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	log.Println("UPLOADED URL:", url)

	req := dto.AddProjectMediaRequest{
		URL:  url,
		Type: "image",
	}

	user := h.svc.Auth.GetCurrentUser(ctx)

	err = h.svc.AddProjectMedia(uint(id), user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "media added", fiber.Map{
		"url": url,
	})
}

func (h *ProjectHandler) GetMyProjects(ctx fiber.Ctx) error {

	user := h.svc.Auth.GetCurrentUser(ctx)
	log.Println(user)

	projects, err := h.svc.GetProjectsByOwnerResponse(user.ID)
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

func (h *ProjectHandler) AddStory(ctx fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	req := dto.AddProjectStoryRequest{}

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	user := h.svc.Auth.GetCurrentUser(ctx)

	err := h.svc.AddProjectStory(uint(id), user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "story added", nil)
}

func (h *ProjectHandler) AddRisk(ctx fiber.Ctx) error {

	id, _ := strconv.Atoi(ctx.Params("id"))

	req := dto.AddProjectRiskRequest{}

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	user := h.svc.Auth.GetCurrentUser(ctx)

	err := h.svc.AddProjectRisk(uint(id), user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "risk added", nil)
}

func (h *ProjectHandler) AddFAQ(ctx fiber.Ctx) error {

	id, _ := strconv.Atoi(ctx.Params("id"))

	req := dto.AddProjectFAQRequest{}

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	user := h.svc.Auth.GetCurrentUser(ctx)

	err := h.svc.AddProjectFAQ(uint(id), user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "faq added", nil)
}
func (h *ProjectHandler) CreateMilestone(ctx fiber.Ctx) error {

	id, _ := strconv.Atoi(ctx.Params("id"))

	req := dto.CreateMilestoneRequest{}

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	user := h.svc.Auth.GetCurrentUser(ctx)

	err := h.svc.CreateMilestone(uint(id), user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "milestone created", nil)
}

func (h *ProjectHandler) SetFundingPolicy(ctx fiber.Ctx) error {

	id, _ := strconv.Atoi(ctx.Params("id"))

	req := dto.SetFundingPolicyRequest{}

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	user := h.svc.Auth.GetCurrentUser(ctx)

	err := h.svc.SetFundingPolicy(uint(id), user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "funding policy set", nil)
}
func (h *ProjectHandler) SetProfitPolicy(ctx fiber.Ctx) error {

	id, _ := strconv.Atoi(ctx.Params("id"))

	req := dto.SetProfitPolicyRequest{}

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	user := h.svc.Auth.GetCurrentUser(ctx)

	err := h.svc.SetProfitPolicy(uint(id), user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "profit policy set", nil)
}

func (h *ProjectHandler) SubmitProject(ctx fiber.Ctx) error {

	id, _ := strconv.Atoi(ctx.Params("id"))

	req := dto.SubmitProjectRequest{}

	if err := ctx.Bind().Body(&req); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	user := h.svc.Auth.GetCurrentUser(ctx)

	err := h.svc.SubmitProject(uint(id), user.ID, req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "project submitted", nil)
}

func (h *ProjectHandler) GetProjectDetail(ctx fiber.Ctx) error {

	id, _ := strconv.Atoi(ctx.Params("id"))

	user := h.svc.Auth.GetCurrentUser(ctx)

	project, err := h.svc.GetProjectDetail(uint(id), user.ID)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "project detail", project)
}

func (h *ProjectHandler) PublishProject(ctx fiber.Ctx) error {

	id, _ := strconv.Atoi(ctx.Params("id"))

	user := h.svc.Auth.GetCurrentUser(ctx)

	err := h.svc.PublishProject(uint(id), user.ID)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "project published", nil)
}

func (h *ProjectHandler) ValidateProject(ctx fiber.Ctx) error {

	idStr := ctx.Params("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	user := h.svc.Auth.GetCurrentUser(ctx)

	res, err := h.svc.ValidateProject(uint(id), user.ID)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, "validation result", res)
}

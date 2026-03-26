package handlers

import (
	"errors"
	"flyup/internal/api/rest"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"flyup/internal/repository"
	"flyup/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type ProjectHandler struct {
	svc       service.ProjectService
	validator *validator.Validate
	auth      helper.Auth
}

func SetupProjectRoutes(rh *rest.RestHandler) {

	app := rh.App

	svc := service.NewProjectService(
		repository.NewProjectRepository(rh.DB),
		rh.Cloudinary,
	)

	handler := ProjectHandler{
		svc:       svc,
		validator: rh.Validator,
		auth:      rh.Auth,
	}

	// public
	pub := app.Group("/projects")
	pub.Get("/:id", handler.GetPublicProjectByID)

	//Category Routes
	app.Get("/categories", handler.GetAllCategories)
	app.Get("/categories/:id", handler.GetCategoryByID)

	// private (pioneer)
	priv := app.Group("/pioneer/projects", rh.Middlewares.AuthorizePioneer)
	priv.Post("/", handler.CreateProject)
	priv.Get("/", handler.GetMyProjects)
	priv.Get("/:id", handler.GetMyProjectByID)
	priv.Patch("/:id", handler.UpdateProject)
	priv.Post("/:id/media", handler.AddProjectMedia)

	// Admin
	admin := app.Group("/admin/categories", rh.Middlewares.AuthorizeAdmin)
	admin.Post("/", handler.CreateCategory)
	admin.Put("/:id", handler.UpdateCategory)
	admin.Delete("/:id", handler.DeleteCategory)
}

func (h *ProjectHandler) AddProjectMedia(ctx fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("id"))

	// รับไฟล์ (ใช้ Key "file" แทน "image" เพื่อให้ครอบคลุมทั้งคู่)
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return rest.BadRequestError(ctx, "no file uploaded")
	}

	// จำกัดขนาดไฟล์ (เช่น วิดีโอห้ามเกิน 20MB)
	if fileHeader.Size > 20*1024*1024 {
		return rest.BadRequestError(ctx, "file size too large (max 20MB)")
	}

	file, _ := fileHeader.Open()
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	media := &domain.ProjectMedia{}

	// ส่ง fileHeader เข้าไปด้วย
	if err := h.svc.AddProjectMedia(uint(id), file, fileHeader, media); err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "uploaded successfully", media)
}

func (h *ProjectHandler) CreateCategory(ctx fiber.Ctx) error {
	// ใช้ Local Struct เพื่อรับเฉพาะค่าที่อนุญาตให้ส่งมา
	var body struct {
		Name string `json:"name" validate:"required,min=2,max=50"`
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}

	// ตรวจสอบเงื่อนไข (Validation)
	if err := h.validator.Struct(body); err != nil {
		return rest.BadRequestError(ctx, "category name must be 2-50 characters long")
	}

	// เรียก Service เพื่อสร้างข้อมูล
	category, err := h.svc.CreateCategory(body.Name)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "created category successfully", category)
}

func (h *ProjectHandler) UpdateCategory(ctx fiber.Ctx) error {
	// ดึง ID จาก URL
	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return rest.BadRequestError(ctx, "Invalid category id")
	}

	// รับข้อมูลที่ต้องการแก้ไข
	var body struct {
		Name string `json:"name" validate:"required,min=2,max=50"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "Invalid request body")
	}

	if err := h.validator.Struct(body); err != nil {
		return rest.BadRequestError(ctx, "ชcategory name must be 2-50 characters long")
	}

	// เรียก Service อัปเดต
	category, err := h.svc.UpdateCategory(uint(id), body.Name)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "updated category successfully", category)
}

func (h *ProjectHandler) DeleteCategory(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return rest.BadRequestError(ctx, "Invalid category id")
	}

	// เรียก Service ลบ
	if err := h.svc.DeleteCategory(uint(id)); err != nil {
		// หากลบไม่ได้เพราะถูกใช้งานอยู่ Service จะส่ง Error มา
		// เราสามารถเช็คข้อความ Error เพื่อส่ง Status 409 Conflict ได้
		return rest.ErrorMessage(ctx, http.StatusConflict, err)
	}

	return rest.SuccessResponse(ctx, "deleted successfully", nil)
}

func (h *ProjectHandler) GetAllCategories(ctx fiber.Ctx) error {
	categories, err := h.svc.GetAllCategories()
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", categories)
}

func (h *ProjectHandler) GetCategoryByID(ctx fiber.Ctx) error {
	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return rest.BadRequestError(ctx, "invalid category id")
	}

	category, err := h.svc.GetCategoryByID(uint(id))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusNotFound, errors.New("category not found"))
	}

	return rest.SuccessResponse(ctx, "success", category)
}

func (h *ProjectHandler) UpdateProject(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}
	// parse project ID
	idStr := ctx.Params("id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil || projectID <= 0 {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	// parse request body
	var body struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		CategoryID  *uint   `json:"category_id"`
		Visibility  *string `json:"visibility"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	updateData := &domain.Project{}
	if body.Title != nil {
		updateData.Title = *body.Title
	}
	if body.Description != nil {
		updateData.Description = body.Description
	}
	if body.CategoryID != nil {
		updateData.CategoryID = body.CategoryID
	}
	if body.Visibility != nil {
		switch *body.Visibility {
		case string(domain.VisibilityPrivate):
			updateData.Visibility = domain.VisibilityPrivate
		case string(domain.VisibilityPublic):
			updateData.Visibility = domain.VisibilityPublic
		case string(domain.VisibilityUnlisted):
			updateData.Visibility = domain.VisibilityUnlisted
		default:
			return rest.BadRequestError(ctx, "invalid visibility")
		}
	}

	// call service
	project, err := h.svc.UpdateProject(uint(projectID), updateData)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	// build response
	var category *string
	if project.Category != nil {
		c := project.Category.Name
		category = &c
	}

	response := dto.ProjectResponse{
		ID:          project.ID,
		OwnerUserID: project.OwnerUserID,
		Category:    category,
		Title:       project.Title,
		Description: project.Description,
		State:       string(project.State),
		Status:      string(project.Status),
		Visibility:  string(project.Visibility),
		FundingGoal: project.FundingGoal,
		CreatedAt:   project.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   project.UpdatedAt.Format(time.RFC3339),
	}

	return rest.SuccessResponse(ctx, "project updated", response)
}

func (h *ProjectHandler) CreateProject(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)

	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	proj, err := h.svc.CreateProject(user.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	var category *string

	if proj.Category != nil {
		c := proj.Category.Name
		category = &c
	}

	response := dto.ProjectResponse{
		ID:          proj.ID,
		OwnerUserID: proj.OwnerUserID,
		Category:    category,
		Title:       proj.Title,
		Description: proj.Description,

		State:      string(proj.State),
		Status:     string(proj.Status),
		Visibility: string(proj.Visibility),

		FundingGoal: proj.FundingGoal,
		CreatedAt:   proj.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   proj.UpdatedAt.Format(time.RFC3339),
	}

	return rest.SuccessResponse(ctx, "project created", response)
}

func (h *ProjectHandler) GetMyProjects(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)

	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	projects, err := h.svc.GetMyProjects(user.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	result := make([]dto.ProjectResponse, 0, len(projects))

	for _, proj := range projects {
		result = append(result, toProjectResponse(&proj))
	}

	return rest.SuccessResponse(ctx, "success", result)
}

func (h *ProjectHandler) GetMyProjectByID(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)

	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	proj, err := h.svc.GetOwnerProjectByID(uint(id), user.ID)
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusForbidden, err)
	}

	return rest.SuccessResponse(ctx, "success", toProjectResponse(proj))
}

func (h *ProjectHandler) GetPublicProjectByID(ctx fiber.Ctx) error {

	idStr := ctx.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	proj, err := h.svc.GetPublicProjectByID(uint(id))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusNotFound, err)
	}

	return rest.SuccessResponse(ctx, "success", toProjectResponse(proj))
}

func toProjectResponse(proj *domain.Project) dto.ProjectResponse {
	var category *string

	if proj.Category != nil {
		c := proj.Category.Name
		category = &c
	}

	return dto.ProjectResponse{
		ID:          proj.ID,
		OwnerUserID: proj.OwnerUserID,
		Category:    category,
		Title:       proj.Title,
		Description: proj.Description,
		State:       string(proj.State),
		Status:      string(proj.Status),
		Visibility:  string(proj.Visibility),
		FundingGoal: proj.FundingGoal,
		CreatedAt:   proj.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   proj.UpdatedAt.Format(time.RFC3339),
	}
}

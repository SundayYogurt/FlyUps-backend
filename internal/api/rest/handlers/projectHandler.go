package handlers

import (
	"errors"
	"flyup/internal/api/rest"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
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

	// Public Project
	pub := app.Group("/projects")
	pub.Get("/", handler.GetPublicProjects)
	pub.Get("/:id", handler.GetPublicProjectByID)
	pub.Get("/category/:category_id", handler.GetProjectsByCategory)
	pub.Get("/:id/updates", handler.GetProjectUpdates)
	pub.Get("/:id/faqs", handler.GetProjectFAQs)
	pub.Get("/:id/threads", handler.GetProjectThreads)
	pub.Get("/threads/:thread_id/messages", handler.GetProjectThreadMessages)

	// Categories
	app.Get("/categories", handler.GetAllCategories)
	app.Get("/categories/:id", handler.GetCategoryByID)

	// Pioneer (Private) Projects
	priv := app.Group("/pioneer/projects", rh.Middlewares.AuthorizePioneer)
	priv.Post("/", handler.CreateProject)
	priv.Get("/", handler.GetMyProjects)
	priv.Get("/:id", handler.GetMyProjectByID)
	priv.Patch("/:id", handler.UpdateProject)
	priv.Delete("/:id", handler.DeleteProject)
	priv.Patch("/:id/submit", handler.SubmitForReview)
	priv.Patch("/:id/close", handler.CloseProject)
	priv.Post("/:id/updates", handler.CreateProjectUpdate)
	priv.Patch("/updates/:update_id", handler.UpdateProjectUpdate)
	priv.Delete("/updates/:update_id", handler.DeleteProjectUpdate)
	priv.Post("/:id/faqs", handler.CreateProjectFAQ)
	priv.Patch("/faqs/:faq_id", handler.UpdateProjectFAQ)
	priv.Delete("/faqs/:faq_id", handler.DeleteProjectFAQ)
	priv.Post("/:id/threads", handler.CreateProjectThread)
	priv.Patch("/threads/:thread_id", handler.UpdateProjectThread)
	priv.Delete("/threads/:thread_id", handler.DeleteProjectThread)
	priv.Post("/threads/:thread_id/messages", handler.CreateProjectThreadMessage)
	priv.Patch("/messages/:message_id", handler.UpdateProjectThreadMessage)
	priv.Delete("/messages/:message_id", handler.DeleteProjectThreadMessage)

	// Media
	priv.Post("/:id/media", handler.AttachProjectMedia)
	priv.Get("/:id/media", handler.GetProjectMedia)
	priv.Patch("/media/:media_id", handler.UpdateProjectMedia)
	priv.Delete("/media/:media_id", handler.DeleteProjectMedia)

	// Milestones
	priv.Post("/:id/milestones", handler.AddProjectMilestone)
	priv.Get("/:id/milestones", handler.GetProjectMilestones)
	priv.Patch("/milestones/:milestone_id", handler.UpdateProjectMilestone)
	priv.Delete("/milestones/:milestone_id", handler.DeleteProjectMilestone)

	// Stories
	priv.Post("/:id/stories", handler.AddProjectStory)
	priv.Get("/:id/stories", handler.GetProjectStories)
	priv.Patch("/stories/:story_id", handler.UpdateProjectStory)
	priv.Delete("/stories/:story_id", handler.DeleteProjectStory)

	// Admin Category
	adminCat := app.Group("/admin/categories", rh.Middlewares.AuthorizeAdmin)
	adminCat.Post("/", handler.CreateCategory)
	adminCat.Put("/:id", handler.UpdateCategory)
	adminCat.Delete("/:id", handler.DeleteCategory)

	adminProj := app.Group("/admin/projects", rh.Middlewares.AuthorizeAdmin)
	adminProj.Patch("/:id/approve", handler.ApproveProject)
	adminProj.Patch("/:id/reject", handler.RejectProject)
	adminProj.Patch("/:id/status", handler.UpdateProjectStatus)
}

func (h *ProjectHandler) AttachProjectMedia(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil || id <= 0 {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	var body struct {
		URL  string           `json:"url" validate:"required"`
		Type domain.MediaType `json:"type" validate:"required"`
	}

	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	if err := h.validator.Struct(body); err != nil {
		return rest.BadRequestError(ctx, "invalid input")
	}

	media := &domain.ProjectMedia{
		Type: body.Type,
		URL:  body.URL,
	}

	err = h.svc.AttachProjectMedia(ctx.Context(), uint(id), body.URL, body.Type, user)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "media attached", media)
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
	categoryModel := &domain.ProjectCategory{Name: body.Name}
	category, err := h.svc.CreateCategory(categoryModel)
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
		return rest.BadRequestError(ctx, "category name must be 2-50 characters long")
	}

	// เรียก Service อัปเดต
	categoryModel := &domain.ProjectCategory{ID: uint(id), Name: body.Name}
	category, err := h.svc.UpdateCategory(categoryModel)
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
	var body dto.UpdateProjectRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	// call service
	_, err = h.svc.UpdateProject(uint(projectID), body, user)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	// Fetch updated project detail
	updatedProject, err := h.svc.GetProjectDetailByID(uint(projectID))
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "project updated", h.toProjectDetailResponse(updatedProject))
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

	// Fetch newly created project detail
	createdProject, err := h.svc.GetProjectDetailByID(proj.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "project created", h.toProjectDetailResponse(createdProject))
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
		result = append(result, h.toProjectResponse(&proj))
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

	return rest.SuccessResponse(ctx, "success", h.toProjectDetailResponse(proj))
}

func (h *ProjectHandler) GetPublicProjects(ctx fiber.Ctx) error {
	projects, err := h.svc.GetPublicProjects()
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	result := make([]dto.ProjectResponse, 0, len(projects))
	for _, proj := range projects {
		result = append(result, h.toProjectResponse(&proj))
	}

	return rest.SuccessResponse(ctx, "success", result)
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

	return rest.SuccessResponse(ctx, "success", h.toProjectDetailResponse(proj))
}

func (h *ProjectHandler) toProjectDetailResponse(proj *domain.Project) dto.ProjectDetailResponse {
	return dto.ProjectDetailResponse{
		ProjectResponse: h.toProjectResponse(proj),
		Media:           proj.Media,
		Milestones:      proj.Milestones,
		Stories:         proj.Stories,
		FAQs:            proj.FAQs,
	}
}

func (h *ProjectHandler) toProjectResponse(proj *domain.Project) dto.ProjectResponse {
	var category *string
	if proj.Category != nil {
		c := proj.Category.Name
		category = &c
	}

	var finalEndDate *time.Time
	if !proj.EndDate.IsZero() {
		finalEndDate = &proj.EndDate
	}
	var finalFundingAt *time.Time
	if !proj.FundingAt.IsZero() {
		finalFundingAt = &proj.FundingAt
	}

	var ownerProfile *dto.ProjectOwnerProfile
	if proj.Owner != nil {
		ownerProjects, _ := h.svc.GetMyProjects(proj.OwnerUserID)

		ownerProfile = &dto.ProjectOwnerProfile{
			FirstName:    proj.Owner.FirstName,
			LastName:     proj.Owner.LastName,
			VerifyStatus: proj.Owner.Status,
			ProjectCount: len(ownerProjects),
		}
		if proj.Owner.StudentProfile != nil {
			sp := proj.Owner.StudentProfile
			if sp.University != nil && sp.University.NameTH != nil {
				ownerProfile.University = *sp.University.NameTH
			}
			ownerProfile.Faculty = sp.Faculty
			ownerProfile.Major = sp.Major
			ownerProfile.Bio = sp.Bio
			ownerProfile.VerifyStatus = sp.VerifyStatus
		}
	}

	return dto.ProjectResponse{
		ID:              proj.ID,
		OwnerUserID:     proj.OwnerUserID,
		Category:        category,
		Title:           proj.Title,
		Description:     proj.Description,
		State:           string(proj.State),
		Status:          string(proj.Status),
		Visibility:      string(proj.Visibility),
		Risk:            proj.Risk,
		PlatformFee:     proj.PlatformFee,
		FundingGoal:     proj.FundingGoal,
		Softcap:         proj.Softcap,
		CurrentFunding:  proj.CurrentFunding,
		DurationDays:    proj.DurationDays,
		DurationMonths:  proj.DurationMonths,
		EndDate:         finalEndDate,
		FundingAt:       finalFundingAt,
		ProfitSharePct:  proj.ProfitSharePct,
		MinInvestAmount: proj.MinInvestAmount,
		MaxInvestAmount: proj.MaxInvestAmount,
		CreatedAt:       proj.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       proj.UpdatedAt.Format(time.RFC3339),
		OwnerProfile:    ownerProfile,
	}
}

func (h *ProjectHandler) AddProjectMilestone(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	idStr := ctx.Params("id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil || projectID <= 0 {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	var body dto.CreateMilestoneRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}

	milestone, err := h.svc.CreateMilestone(uint(projectID), body, user)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "milestone created successfully", milestone)
}

func (h *ProjectHandler) DeleteProjectMilestone(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	milestoneIDStr := ctx.Params("milestone_id")
	milestoneID, err := strconv.Atoi(milestoneIDStr)
	if err != nil || milestoneID <= 0 {
		return rest.BadRequestError(ctx, "invalid milestone id")
	}

	if err := h.svc.DeleteMilestone(uint(milestoneID), user); err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "milestone deleted successfully", nil)
}

func (h *ProjectHandler) AddProjectStory(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	idStr := ctx.Params("id")
	projectID, err := strconv.Atoi(idStr)
	if err != nil || projectID <= 0 {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	var body domain.StorySection
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}

	body.ProjectID = uint(projectID)

	// call service
	if err := h.svc.CreateStorySection(&body, user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	return rest.SuccessResponse(ctx, "story section created successfully", body)
}

func (h *ProjectHandler) UpdateProjectStory(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	storyIDStr := ctx.Params("story_id")
	storyID, err := strconv.Atoi(storyIDStr)
	if err != nil || storyID <= 0 {
		return rest.BadRequestError(ctx, "invalid story id")
	}

	var body domain.StorySection
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}

	body.ID = uint(storyID)

	// call service
	if err := h.svc.UpdateStorySection(&body, user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	return rest.SuccessResponse(ctx, "story section updated successfully", body)
}

func (h *ProjectHandler) DeleteProjectStory(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	storyIDStr := ctx.Params("story_id")
	storyID, err := strconv.Atoi(storyIDStr)
	if err != nil || storyID <= 0 {
		return rest.BadRequestError(ctx, "invalid story id")
	}

	// call service
	if err := h.svc.DeleteStorySection(uint(storyID), user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	return rest.SuccessResponse(ctx, "story section deleted successfully", nil)
}

func (h *ProjectHandler) DeleteProject(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	if err := h.svc.DeleteProject(uint(id), user); err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "project deleted successfully", nil)
}

func (h *ProjectHandler) GetProjectMedia(ctx fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	media, err := h.svc.GetProjectMedia(uint(id))
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", media)
}

func (h *ProjectHandler) UpdateProjectMedia(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}
	mediaID, _ := strconv.Atoi(ctx.Params("media_id"))
	var body domain.ProjectMedia
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.svc.UpdateProjectMedia(uint(mediaID), &body, user); err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "media updated successfully", body)
}

func (h *ProjectHandler) DeleteProjectMedia(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}
	mediaID, _ := strconv.Atoi(ctx.Params("media_id"))
	if err := h.svc.DeleteProjectMedia(uint(mediaID), user); err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "media deleted successfully", nil)
}

func (h *ProjectHandler) GetProjectMilestones(ctx fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	milestones, err := h.svc.GetProjectMilestones(uint(id))
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", milestones)
}

func (h *ProjectHandler) UpdateProjectMilestone(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}
	milestoneID, _ := strconv.Atoi(ctx.Params("milestone_id"))
	var body dto.UpdateMilestoneRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.svc.UpdateMilestone(uint(milestoneID), body, user); err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "milestone updated successfully", nil)
}

func (h *ProjectHandler) GetProjectStories(ctx fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	stories, err := h.svc.GetProjectStories(uint(id))
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", stories)
}

// --- NEW HANDLERS ---
func (h *ProjectHandler) GetProjectsByCategory(ctx fiber.Ctx) error {
	id, _ := strconv.Atoi(ctx.Params("category_id"))
	projects, err := h.svc.GetProjectsByCategory(uint(id))
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	result := make([]dto.ProjectResponse, 0, len(projects))
	for _, proj := range projects {
		result = append(result, h.toProjectResponse(&proj))
	}
	return rest.SuccessResponse(ctx, "success", result)
}

func (h *ProjectHandler) UpdateProjectStatus(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	var body struct {
		State  string `json:"state"`
		Status string `json:"status"`
	}
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.svc.UpdateProjectStatus(uint(id), domain.ProjectState(body.State), domain.ProjectStatus(body.Status), user); err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "status updated successfully", nil)
}

func (h *ProjectHandler) CreateProjectUpdate(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	var body dto.CreateProjectUpdateRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.svc.CreateProjectUpdate(uint(id), body, user); err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "update created successfully", nil)
}

func (h *ProjectHandler) UpdateProjectUpdate(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	updateID, _ := strconv.Atoi(ctx.Params("update_id"))

	var body dto.UpdateProjectUpdateRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}

	if err := h.svc.UpdateProjectUpdate(uint(updateID), body, user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	return rest.SuccessResponse(ctx, "update modified successfully", nil)
}

func (h *ProjectHandler) DeleteProjectUpdate(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	updateID, _ := strconv.Atoi(ctx.Params("update_id"))

	if err := h.svc.DeleteProjectUpdate(uint(updateID), user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	return rest.SuccessResponse(ctx, "update deleted successfully", nil)
}

func (h *ProjectHandler) GetProjectUpdates(ctx fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	updates, err := h.svc.GetProjectUpdates(uint(id))
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", updates)
}

func (h *ProjectHandler) GetProjectFAQs(ctx fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	faqs, err := h.svc.GetProjectFAQs(uint(id))
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", faqs)
}

func (h *ProjectHandler) CreateProjectFAQ(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	var body domain.ProjectFAQ
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	body.ProjectID = uint(id)
	if err := h.svc.CreateProjectFAQ(&body, user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "faq created successfully", body)
}

func (h *ProjectHandler) UpdateProjectFAQ(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	faqID, _ := strconv.Atoi(ctx.Params("faq_id"))
	var body domain.ProjectFAQ
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	body.ID = uint(faqID)
	if err := h.svc.UpdateProjectFAQ(&body, user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "faq updated successfully", body)
}

func (h *ProjectHandler) DeleteProjectFAQ(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	faqID, _ := strconv.Atoi(ctx.Params("faq_id"))
	if err := h.svc.DeleteProjectFAQ(uint(faqID), user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "faq deleted successfully", nil)
}

func (h *ProjectHandler) GetProjectThreads(ctx fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	threads, err := h.svc.GetProjectThreads(uint(id))
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", threads)
}

func (h *ProjectHandler) CreateProjectThread(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	var body domain.ProjectThread
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	body.ProjectID = uint(id)
	if err := h.svc.CreateProjectThread(&body, user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "thread created successfully", body)
}

func (h *ProjectHandler) UpdateProjectThread(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	threadID, _ := strconv.Atoi(ctx.Params("thread_id"))
	var body domain.ProjectThread
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	body.ID = uint(threadID)
	if err := h.svc.UpdateProjectThread(&body, user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "thread updated successfully", body)
}

func (h *ProjectHandler) DeleteProjectThread(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	threadID, _ := strconv.Atoi(ctx.Params("thread_id"))
	if err := h.svc.DeleteProjectThread(uint(threadID), user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "thread deleted successfully", nil)
}

func (h *ProjectHandler) GetProjectThreadMessages(ctx fiber.Ctx) error {
	threadID, _ := strconv.Atoi(ctx.Params("thread_id"))
	msgs, err := h.svc.GetProjectThreadMessages(uint(threadID))
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", msgs)
}

func (h *ProjectHandler) CreateProjectThreadMessage(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	threadID, _ := strconv.Atoi(ctx.Params("thread_id"))
	var body domain.ProjectThreadMessage
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	body.ThreadID = uint(threadID)
	if err := h.svc.CreateProjectThreadMessage(&body, user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "message created successfully", body)
}

func (h *ProjectHandler) UpdateProjectThreadMessage(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	msgID, _ := strconv.Atoi(ctx.Params("message_id"))
	var body domain.ProjectThreadMessage
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	body.ID = uint(msgID)
	if err := h.svc.UpdateProjectThreadMessage(&body, user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "message updated successfully", body)
}

func (h *ProjectHandler) DeleteProjectThreadMessage(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	msgID, _ := strconv.Atoi(ctx.Params("message_id"))
	if err := h.svc.DeleteProjectThreadMessage(uint(msgID), user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "message deleted successfully", nil)
}

func (h *ProjectHandler) SubmitForReview(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	if err := h.svc.SubmitForReview(uint(id), user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "project submitted for review", nil)
}

func (h *ProjectHandler) ApproveProject(ctx fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	if err := h.svc.ApproveProject(uint(id)); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "project approved", nil)
}

func (h *ProjectHandler) RejectProject(ctx fiber.Ctx) error {
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	if err := h.svc.RejectProject(uint(id)); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "project rejected", nil)
}

func (h *ProjectHandler) CloseProject(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}
	if err := h.svc.CloseProject(uint(id), user); err != nil {
		switch err.Error() {
		case "permission denied":
			return rest.ErrorMessage(ctx, http.StatusForbidden, err)
		case "project not found":
			return rest.ErrorMessage(ctx, http.StatusNotFound, err)
		default:
			return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
		}
	}
	return rest.SuccessResponse(ctx, "project closed successfully", nil)
}

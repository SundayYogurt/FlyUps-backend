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
		repository.NewUserRepository(rh.DB),
		rh.Cloudinary,
		rh.NotifSvc,
		rh.Notification,
	)

	handler := ProjectHandler{
		svc:       svc,
		validator: rh.Validator,
		auth:      rh.Auth,
	}

	// Public Project
	pub := app.Group("/projects")
	pub.Get("/", handler.GetPublicProjects)
	pub.Get("/recommend", handler.GetRecommendationProjects)
	pub.Get("/new", handler.GetNewProjects)
	pub.Get("/ending", handler.GetEndingSoonProjects)
	pub.Get("/:id<int>", handler.GetPublicProjectByID)
	pub.Get("/category/:category_id<int>", handler.GetProjectsByCategory)
	pub.Get("/:id<int>/updates", handler.GetProjectUpdates)
	pub.Get("/:id<int>/faqs", handler.GetProjectFAQs)
	pub.Get("/:id<int>/threads", handler.GetProjectThreads)
	pub.Get("/threads/:thread_id<int>/messages", handler.GetProjectThreadMessages)

	// Categories
	app.Get("/categories", handler.GetAllCategories)
	app.Get("/categories/:id<int>", handler.GetCategoryByID)

	// Pioneer (Private) Projects
	priv := app.Group("/pioneer/projects", rh.Middlewares.AuthorizePioneer)
	priv.Post("/", handler.CreateProject)
	priv.Get("/", handler.GetMyProjects)
	priv.Get("/:id<int>", handler.GetMyProjectByID)
	priv.Patch("/:id<int>", handler.UpdateProject)
	priv.Delete("/:id<int>", handler.DeleteProject)
	priv.Patch("/:id<int>/submit", handler.SubmitForReview)
	priv.Patch("/:id<int>/close", handler.CloseProject)
	priv.Post("/:id<int>/updates", handler.CreateProjectUpdate)
	priv.Patch("/updates/:update_id<int>", handler.UpdateProjectUpdate)
	priv.Delete("/updates/:update_id<int>", handler.DeleteProjectUpdate)
	priv.Post("/:id<int>/faqs", handler.CreateProjectFAQ)
	priv.Patch("/faqs/:faq_id<int>", handler.UpdateProjectFAQ)
	priv.Delete("/faqs/:faq_id<int>", handler.DeleteProjectFAQ)
	priv.Post("/:id<int>/threads", handler.CreateProjectThread)
	priv.Patch("/threads/:thread_id<int>", handler.UpdateProjectThread)
	priv.Delete("/threads/:thread_id<int>", handler.DeleteProjectThread)
	priv.Post("/threads/:thread_id<int>/messages", handler.CreateProjectThreadMessage)
	priv.Patch("/messages/:message_id<int>", handler.UpdateProjectThreadMessage)
	priv.Delete("/messages/:message_id<int>", handler.DeleteProjectThreadMessage)
	priv.Patch("/:id<int>/cancel", handler.CancelProject)

	// Media
	priv.Post("/:id<int>/media", handler.AttachProjectMedia)
	priv.Get("/:id<int>/media", handler.GetProjectMedia)
	priv.Patch("/media/:media_id<int>", handler.UpdateProjectMedia)
	priv.Delete("/media/:media_id<int>", handler.DeleteProjectMedia)

	// Milestones
	priv.Post("/:id<int>/milestones", handler.AddProjectMilestone)
	pub.Get("/:id<int>/milestones", handler.GetProjectMilestones)
	priv.Patch("/milestones/:milestone_id<int>", handler.UpdateProjectMilestone)
	priv.Patch("/milestones/:milestone_id<int>/submit", handler.SubmitProjectMilestone)
	priv.Patch("/milestones/:milestone_id<int>/open-vote", handler.OpenMilestoneVoting)
	priv.Delete("/milestones/:milestone_id<int>", handler.DeleteProjectMilestone)

	//meeting
	Me := app.Group("/me", rh.Middlewares.Authorize)

	Me.Get("/meetings", handler.GetMyMeetings)
	Me.Get("/meeting/:id", handler.GetMyMeeting)
	Me.Get("/projects/:id/meetings", handler.GetMyProjectMeetings)
	Me.Get("/milestones/:id/meetings", handler.GetMyMilestoneMeetings)

	priv.Post("/meeting", handler.Meeting)
	priv.Patch("/meeting/:id", handler.EditMeeting)
	priv.Patch("/cancel/meeting/:id", handler.CancelMeeting)

	// Stories
	priv.Post("/:id<int>/stories", handler.AddProjectStory)
	priv.Get("/:id<int>/stories", handler.GetProjectStories)
	priv.Patch("/stories/:story_id<int>", handler.UpdateProjectStory)
	priv.Delete("/stories/:story_id<int>", handler.DeleteProjectStory)

	// Admin Category
	adminCat := app.Group("/admin/categories", rh.Middlewares.AuthorizeAdmin)
	adminCat.Post("/", handler.CreateCategory)
	adminCat.Put("/:id<int>", handler.UpdateCategory)
	adminCat.Delete("/:id<int>", handler.DeleteCategory)

	adminProj := app.Group("/admin/projects", rh.Middlewares.AuthorizeAdmin)
	adminProj.Patch("/:id<int>/approve", handler.ApproveProject)
	adminProj.Patch("/:id<int>/reject", handler.RejectProject)
	adminProj.Patch("/:id<int>/status", handler.UpdateProjectStatus)
	adminProj.Get("/pending-review", handler.ProjectsPendingList)
	adminProj.Get("/:id<int>/detail/pending-review", handler.ProjectDetailReview)

	// Admin Milestone Submission Review
	adminProj.Get("/milestones/submitted", handler.AdminListSubmittedMilestones)
	adminProj.Patch("/milestones/:milestone_id<int>/approve", handler.AdminApproveMilestoneSubmission)
	adminProj.Patch("/milestones/:milestone_id<int>/reject", handler.AdminRejectMilestoneSubmission)
	adminProj.Get("/milestones/:milestone_id<int>", handler.AdminGetMilestoneDetail)
}

func (h *ProjectHandler) GetRecommendationProjects(ctx fiber.Ctx) error {
	projects, err := h.svc.GetProjectRecommendations()
	if err != nil {
		return err
	}
	return rest.SuccessResponse(ctx, "success", projects)
}

func (h *ProjectHandler) GetNewProjects(ctx fiber.Ctx) error {
	projects, err := h.svc.GetNewProjects()
	if err != nil {
		return err
	}
	return rest.SuccessResponse(ctx, "success", projects)
}

func (h *ProjectHandler) GetEndingSoonProjects(ctx fiber.Ctx) error {
	projects, err := h.svc.GetProjectEndingSoon()
	if err != nil {
		return err
	}
	return rest.SuccessResponse(ctx, "success", projects)
}

// CancelProject godoc
// @Summary Cancel Project
// @Description Pioneer cancels their project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Success 200 {object} object "Project cancelled"
// @Router /pioneer/projects/{id}/cancel [patch]
func (h *ProjectHandler) CancelProject(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil || id <= 0 {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	err = h.svc.CancelProject(uint(id), user)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "cancelled successfully", nil)
}

// AttachProjectMedia godoc
// @Summary Attach Media to Project
// @Description Pioneer uploads multiple media (image/video) URLs to project in one request
// @Tags Projects
// AttachProjectMedia godoc
// @Summary Attach Media Array to Project
// @Description Pioneer uploads array of media items (each with url and type array) to project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Param request body []dto.ProjectMediaItem true "Array of Media Items"
// @Success 200 {object} object "Media attached successfully"
// @Failure 400 {object} object "Invalid body format or missing required fields"
// @Failure 401 {object} object "Unauthorized"
// @Router /pioneer/projects/{id}/media [post]
func (h *ProjectHandler) AttachProjectMedia(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil || id <= 0 {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	type mediaItem struct {
		URL  string             `json:"url" validate:"required"`
		Type []domain.MediaType `json:"type" validate:"required"`
	}

	var items []mediaItem
	if err := ctx.Bind().Body(&items); err != nil {
		return rest.BadRequestError(ctx, "invalid body: expected array of {url, type[]}")
	}

	if len(items) == 0 {
		return rest.BadRequestError(ctx, "at least one media item is required")
	}

	var created []domain.ProjectMedia
	for _, item := range items {
		if err := h.validator.Struct(item); err != nil {
			return rest.BadRequestError(ctx, "invalid item: "+err.Error())
		}
		if len(item.Type) == 0 {
			return rest.BadRequestError(ctx, "type is required for each media item")
		}
		if err := h.svc.AttachProjectMedia(ctx.Context(), uint(id), item.URL, item.Type, user); err != nil {
			return rest.InternalError(ctx, err)
		}
		created = append(created, domain.ProjectMedia{
			ProjectID: uint(id),
			URL:       item.URL,
			Type:      item.Type,
		})
	}

	return rest.SuccessResponse(ctx, "media attached", created)
}

// CreateCategory godoc
// @Summary Create Category
// @Description Admin creates a new project category
// @Tags Categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body object true "Category Name"
// @Success 200 {object} object "Category created"
// @Router /admin/categories [post]
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

// UpdateCategory godoc
// @Summary Update Category
// @Description Admin updates an existing project category
// @Tags Categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Category ID"
// @Param request body object true "New Category Name"
// @Success 200 {object} object "Category updated"
// @Router /admin/categories/{id} [put]
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

// DeleteCategory godoc
// @Summary Delete Category
// @Description Admin deletes a project category
// @Tags Categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Category ID"
// @Success 200 {object} object "Category deleted"
// @Router /admin/categories/{id} [delete]
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

// GetAllCategories godoc
// @Summary Get All Categories
// @Description Get list of all project categories
// @Tags Categories
// @Accept json
// @Produce json
// @Success 200 {object} object "List of categories"
// @Router /categories [get]
func (h *ProjectHandler) GetAllCategories(ctx fiber.Ctx) error {
	categories, err := h.svc.GetAllCategories()
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", categories)
}

// GetCategoryByID godoc
// @Summary Get Category by ID
// @Description Get specific category by ID
// @Tags Categories
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} object "Category details"
// @Router /categories/{id} [get]
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

// UpdateProject godoc
// @Summary Update Project
// @Description Pioneer updates their existing project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Param request body dto.UpdateProjectRequest true "Update Project Request"
// @Success 200 {object} object "Project updated"
// @Router /pioneer/projects/{id} [patch]
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
	updatedProject, err := h.svc.GetOwnerProjectByID(uint(projectID), user.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "project updated", h.toProjectDetailResponse(updatedProject))
}

// CreateProject godoc
// @Summary Create Project
// @Description Pioneer creates a new project outline
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 201 {object} object "Project created"
// @Router /pioneer/projects [post]
func (h *ProjectHandler) CreateProject(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)

	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	proj, err := h.svc.CreateProject(user.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	// Fetch newly created project detail (owner view; do not apply public filters)
	createdProject, err := h.svc.GetOwnerProjectByID(proj.ID, user.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "project created", h.toProjectDetailResponse(createdProject))
}

// GetMyProjects godoc
// @Summary List My Projects
// @Description Pioneer gets list of their own projects
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object "List of projects"
// @Router /pioneer/projects [get]
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

// GetMyProjectByID godoc
// @Summary Get My Project by ID
// @Description Pioneer gets details of their specific project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Success 200 {object} object "Project details"
// @Router /pioneer/projects/{id} [get]
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

// GetPublicProjects godoc
// @Summary Get Public Projects
// @Description Get list of all public/approved projects
// @Tags Projects
// @Accept json
// @Produce json
// @Success 200 {object} object "List of public projects"
// @Router /projects [get]
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

// GetPublicProjectByID godoc
// @Summary Get Public Project by ID
// @Description Get specific public project details
// @Tags Projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {object} object "Project details"
// @Router /projects/{id} [get]
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

// AddProjectMilestone godoc
// @Summary Add Milestone to Project
// @Description Pioneer adds a new milestone to their project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Param request body dto.CreateMilestoneRequest true "Milestone Data"
// @Success 200 {object} object "Milestone created"
// @Router /pioneer/projects/{id}/milestones [post]
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

// DeleteProjectMilestone godoc
// @Summary Delete Project Milestone
// @Description Pioneer deletes a specific milestone
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param milestone_id path int true "Milestone ID"
// @Success 200 {object} object "Milestone deleted"
// @Router /pioneer/projects/milestones/{milestone_id} [delete]
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

// AddProjectStory godoc
// @Summary Add Story Section to Project
// @Description Pioneer adds a story section to their project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Param request body object true "Story Section Data"
// @Success 200 {object} object "Story created"
// @Router /pioneer/projects/{id}/stories [post]
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

// UpdateProjectStory godoc
// @Summary Update Story Section
// @Description Pioneer updates an existing story section
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param story_id path int true "Story ID"
// @Param request body object true "Story Section Data"
// @Success 200 {object} object "Story updated"
// @Router /pioneer/projects/stories/{story_id} [patch]
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

// DeleteProjectStory godoc
// @Summary Delete Story Section
// @Description Pioneer deletes an existing story section
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param story_id path int true "Story ID"
// @Success 200 {object} object "Story deleted"
// @Router /pioneer/projects/stories/{story_id} [delete]
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

// DeleteProject godoc
// @Summary Delete Project
// @Description Pioneer deletes their whole project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Success 200 {object} object "Project deleted"
// @Router /pioneer/projects/{id} [delete]
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

// GetProjectMedia godoc
// @Summary Get Project Media
// @Description Pioneer gets their project media items
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Success 200 {object} object "List of project media"
// @Router /pioneer/projects/{id}/media [get]
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

// UpdateProjectMedia godoc
// @Summary Update Project Media
// @Description Pioneer updates a project media item
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param media_id path int true "Media ID"
// @Param request body object true "Media Update Data"
// @Success 200 {object} object "Media updated"
// @Router /pioneer/projects/media/{media_id} [patch]
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

// DeleteProjectMedia godoc
// @Summary Delete Project Media
// @Description Pioneer deletes a project media item
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param media_id path int true "Media ID"
// @Success 200 {object} object "Media deleted"
// @Router /pioneer/projects/media/{media_id} [delete]
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

// GetProjectMilestones godoc
// @Summary Get Project Milestones
// @Description Pioneer gets all milestones of a project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Success 200 {object} object "List of milestones"
// @Router /pioneer/projects/{id}/milestones [get]
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

// UpdateProjectMilestone godoc
// @Summary Update Project Milestone
// @Description Pioneer updates a specific milestone
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param milestone_id path int true "Milestone ID"
// @Param request body dto.UpdateMilestoneRequest true "Milestone Data"
// @Success 200 {object} object "Milestone updated"
// @Router /pioneer/projects/milestones/{milestone_id} [patch]
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

// SubmitProjectMilestone godoc
// @Summary Submit Project Milestone
// @Description Pioneer submits milestone evidence (summary, criteria, attachments, external links)
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param milestone_id path int true "Milestone ID"
// @Param request body dto.SubmitMilestoneRequest true "Milestone submission payload"
// @Success 200 {object} object "Milestone submitted"
// @Router /pioneer/projects/milestones/{milestone_id}/submit [patch]
func (h *ProjectHandler) SubmitProjectMilestone(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}

	milestoneID, err := strconv.Atoi(ctx.Params("milestone_id"))
	if err != nil || milestoneID <= 0 {
		return rest.BadRequestError(ctx, "invalid milestone id")
	}

	var body dto.SubmitMilestoneRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid request body")
	}
	if err := h.validator.Struct(body); err != nil {
		return rest.BadRequestError(ctx, "invalid request: "+err.Error())
	}

	m, err := h.svc.SubmitMilestone(uint(milestoneID), body, user)
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	return rest.SuccessResponse(ctx, "milestone submitted successfully", m)
}

// AdminListSubmittedMilestones godoc
// @Summary Admin list submitted milestones
// @Description Admin gets milestones waiting review (status=submitted). Optional filter by project_id query.
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id query int false "Project ID filter"
// @Success 200 {array} dto.AdminMilestoneListResponse "Submitted milestones"
// @Router /admin/projects/milestones/submitted [get]
func (h *ProjectHandler) AdminListSubmittedMilestones(ctx fiber.Ctx) error {
	var projectIDPtr *uint
	if q := ctx.Query("project_id"); q != "" {
		v, err := strconv.Atoi(q)
		if err != nil || v <= 0 {
			return rest.BadRequestError(ctx, "invalid project_id")
		}
		uv := uint(v)
		projectIDPtr = &uv
	}

	items, err := h.svc.GetSubmittedMilestonesForAdmin(projectIDPtr)
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", items)
}

// AdminGetMilestoneDetail godoc
// @Summary Admin get milestone detail
// @Description Admin gets detailed view of a milestone, including project owner info and mapped evidences.
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param milestone_id path int true "Milestone ID"
// @Success 200 {object} dto.AdminMilestoneDetailResponse "Milestone details"
// @Router /admin/projects/milestones/{milestone_id} [get]
func (h *ProjectHandler) AdminGetMilestoneDetail(ctx fiber.Ctx) error {
	milestoneID, err := strconv.Atoi(ctx.Params("milestone_id"))
	if err != nil || milestoneID <= 0 {
		return rest.BadRequestError(ctx, "invalid milestone id")
	}

	detail, err := h.svc.GetAdminMilestoneDetail(uint(milestoneID))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusNotFound, err)
	}
	return rest.SuccessResponse(ctx, "success", detail)
}

// AdminApproveMilestoneSubmission godoc
// @Summary Admin approves milestone submission
// @Description Admin approves milestone submission and marks it as paid (no approved state)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param milestone_id path int true "Milestone ID"
// @Success 200 {object} object "Milestone marked as paid"
// @Router /admin/projects/milestones/{milestone_id}/approve [patch]
func (h *ProjectHandler) AdminApproveMilestoneSubmission(ctx fiber.Ctx) error {
	milestoneID, err := strconv.Atoi(ctx.Params("milestone_id"))
	if err != nil || milestoneID <= 0 {
		return rest.BadRequestError(ctx, "invalid milestone id")
	}

	m, err := h.svc.AdminApproveMilestoneSubmission(uint(milestoneID))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "milestone approved (ready for voting)", m)
}

// AdminRejectMilestoneSubmission godoc
// @Summary Admin rejects milestone submission
// @Description Admin rejects milestone submission (keeps rejected state for resubmission)
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param milestone_id path int true "Milestone ID"
// @Param request body object false "Optional reason"
// @Success 200 {object} object "Milestone rejected"
// @Router /admin/projects/milestones/{milestone_id}/reject [patch]
func (h *ProjectHandler) AdminRejectMilestoneSubmission(ctx fiber.Ctx) error {
	milestoneID, err := strconv.Atoi(ctx.Params("milestone_id"))
	if err != nil || milestoneID <= 0 {
		return rest.BadRequestError(ctx, "invalid milestone id")
	}

	var body struct {
		Reason *string `json:"reason,omitempty"`
	}
	_ = ctx.Bind().Body(&body) // optional

	m, err := h.svc.AdminRejectMilestoneSubmission(uint(milestoneID), body.Reason)
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "milestone rejected", m)
}

// OpenMilestoneVoting godoc
// @Summary Open milestone voting
// @Description Pioneer opens booster voting after admin approves milestone submission
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param milestone_id path int true "Milestone ID"
// @Success 200 {object} object "Voting opened"
// @Router /pioneer/projects/milestones/{milestone_id}/open-vote [patch]
func (h *ProjectHandler) OpenMilestoneVoting(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.ErrorMessage(ctx, http.StatusUnauthorized, errors.New("unauthorized"))
	}
	milestoneID, err := strconv.Atoi(ctx.Params("milestone_id"))
	if err != nil || milestoneID <= 0 {
		return rest.BadRequestError(ctx, "invalid milestone id")
	}

	m, err := h.svc.OpenMilestoneVoting(uint(milestoneID), user)
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "voting opened", m)
}

// GetProjectStories godoc
// @Summary Get Project Stories
// @Description Pioneer gets all story sections of a project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Success 200 {object} object "List of stories"
// @Router /pioneer/projects/{id}/stories [get]
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
// GetProjectsByCategory godoc
// @Summary Get Projects By Category
// @Description Get public projects filtered by category ID
// @Tags Projects
// @Accept json
// @Produce json
// @Param category_id path int true "Category ID"
// @Success 200 {object} object "List of projects"
// @Router /projects/category/{category_id} [get]
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

// UpdateProjectStatus godoc
// @Summary Update Project Status
// @Description Admin updates the status of a project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Param request body object true "Status Update Data"
// @Success 200 {object} object "Status updated"
// @Router /admin/projects/{id}/status [patch]
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
	if err := h.svc.UpdateProjectStatus(uint(id), domain.ProjectState(body.State), domain.ProjectStatus(body.Status)); err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "status updated successfully", nil)
}

// CreateProjectUpdate godoc
// @Summary Create Project Update
// @Description Pioneer creates a new project update
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Param request body dto.CreateProjectUpdateRequest true "Update Data"
// @Success 200 {object} object "Update created"
// @Router /pioneer/projects/{id}/updates [post]
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

// UpdateProjectUpdate godoc
// @Summary Modify Project Update
// @Description Pioneer modifies a specific project update
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param update_id path int true "Update ID"
// @Param request body dto.UpdateProjectUpdateRequest true "Update Data"
// @Success 200 {object} object "Update modified"
// @Router /pioneer/projects/updates/{update_id} [patch]
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

// DeleteProjectUpdate godoc
// @Summary Delete Project Update
// @Description Pioneer deletes a specific project update
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param update_id path int true "Update ID"
// @Success 200 {object} object "Update deleted"
// @Router /pioneer/projects/updates/{update_id} [delete]
func (h *ProjectHandler) DeleteProjectUpdate(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	updateID, _ := strconv.Atoi(ctx.Params("update_id"))

	if err := h.svc.DeleteProjectUpdate(uint(updateID), user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	return rest.SuccessResponse(ctx, "update deleted successfully", nil)
}

// GetProjectUpdates godoc
// @Summary Get Project Updates
// @Description Get public updates for a project
// @Tags Projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {object} object "List of updates"
// @Router /projects/{id}/updates [get]
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

// GetProjectFAQs godoc
// @Summary Get Project FAQs
// @Description Get public FAQs for a project
// @Tags Projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {object} object "List of FAQs"
// @Router /projects/{id}/faqs [get]
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

// CreateProjectFAQ godoc
// @Summary Create Project FAQ
// @Description Pioneer creates a new FAQ for their project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Param request body object true "FAQ Data"
// @Success 200 {object} object "FAQ created"
// @Router /pioneer/projects/{id}/faqs [post]
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

// UpdateProjectFAQ godoc
// @Summary Update Project FAQ
// @Description Pioneer updates an existing FAQ
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param faq_id path int true "FAQ ID"
// @Param request body object true "FAQ Data"
// @Success 200 {object} object "FAQ updated"
// @Router /pioneer/projects/faqs/{faq_id} [patch]
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

// DeleteProjectFAQ godoc
// @Summary Delete Project FAQ
// @Description Pioneer deletes an existing FAQ
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param faq_id path int true "FAQ ID"
// @Success 200 {object} object "FAQ deleted"
// @Router /pioneer/projects/faqs/{faq_id} [delete]
func (h *ProjectHandler) DeleteProjectFAQ(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	faqID, _ := strconv.Atoi(ctx.Params("faq_id"))
	if err := h.svc.DeleteProjectFAQ(uint(faqID), user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "faq deleted successfully", nil)
}

// GetProjectThreads godoc
// @Summary Get Project Threads
// @Description Get public discussion threads for a project
// @Tags Projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {object} object "List of threads"
// @Router /projects/{id}/threads [get]
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

// CreateProjectThread godoc
// @Summary Create Project Thread
// @Description User creates a new discussion thread for a project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Param request body object true "Thread Data"
// @Success 200 {object} object "Thread created"
// @Router /pioneer/projects/{id}/threads [post]
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

// UpdateProjectThread godoc
// @Summary Update Project Thread
// @Description Pioneer updates an existing project thread
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param thread_id path int true "Thread ID"
// @Param request body object true "Thread Data"
// @Success 200 {object} object "Thread updated"
// @Router /pioneer/projects/threads/{thread_id} [patch]
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

// DeleteProjectThread godoc
// @Summary Delete Project Thread
// @Description Pioneer deletes an existing project thread
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param thread_id path int true "Thread ID"
// @Success 200 {object} object "Thread deleted"
// @Router /pioneer/projects/threads/{thread_id} [delete]
func (h *ProjectHandler) DeleteProjectThread(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	threadID, _ := strconv.Atoi(ctx.Params("thread_id"))
	if err := h.svc.DeleteProjectThread(uint(threadID), user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "thread deleted successfully", nil)
}

// GetProjectThreadMessages godoc
// @Summary Get Project Thread Messages
// @Description Get public messages in a project thread
// @Tags Projects
// @Accept json
// @Produce json
// @Param thread_id path int true "Thread ID"
// @Success 200 {object} object "List of messages"
// @Router /projects/threads/{thread_id}/messages [get]
func (h *ProjectHandler) GetProjectThreadMessages(ctx fiber.Ctx) error {
	threadID, _ := strconv.Atoi(ctx.Params("thread_id"))
	msgs, err := h.svc.GetProjectThreadMessages(uint(threadID))
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	return rest.SuccessResponse(ctx, "success", msgs)
}

// CreateProjectThreadMessage godoc
// @Summary Create Project Thread Message
// @Description User replies to a project thread
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param thread_id path int true "Thread ID"
// @Param request body object true "Message Data"
// @Success 200 {object} object "Message created"
// @Router /pioneer/projects/threads/{thread_id}/messages [post]
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

// UpdateProjectThreadMessage godoc
// @Summary Update Project Thread Message
// @Description User updates their project thread message
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param message_id path int true "Message ID"
// @Param request body object true "Message Data"
// @Success 200 {object} object "Message updated"
// @Router /pioneer/projects/messages/{message_id} [patch]
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

// DeleteProjectThreadMessage godoc
// @Summary Delete Project Thread Message
// @Description User deletes their project thread message
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param message_id path int true "Message ID"
// @Success 200 {object} object "Message deleted"
// @Router /pioneer/projects/messages/{message_id} [delete]
func (h *ProjectHandler) DeleteProjectThreadMessage(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	msgID, _ := strconv.Atoi(ctx.Params("message_id"))
	if err := h.svc.DeleteProjectThreadMessage(uint(msgID), user); err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}
	return rest.SuccessResponse(ctx, "message deleted successfully", nil)
}

// SubmitForReview godoc
// @Summary Submit Project For Review
// @Description Pioneer submits project for admin review
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Success 200 {object} object "Project submitted"
// @Router /pioneer/projects/{id}/submit [patch]
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

// ApproveProject godoc
// @Summary Approve Project
// @Description Admin approves a project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Success 200 {object} object "Project approved"
// @Router /admin/projects/{id}/approve [patch]
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

// RejectProject godoc
// @Summary Reject Project
// @Description Admin rejects a project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Success 200 {object} object "Project rejected"
// @Router /admin/projects/{id}/reject [patch]
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

// CloseProject godoc
// @Summary Close Project
// @Description Pioneer or Admin closes a project
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Success 200 {object} object "Project closed"
// @Router /pioneer/projects/{id}/close [patch]
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

// ProjectsPendingList godoc
// @Summary Get projects pending review
// @Description Admin gets all projects that are pending review/approval
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} object "List of projects pending review"
// @Failure 401 {object} object "Unauthorized"
// @Failure 500 {object} object "Internal Server Error"
// @Router /admin/projects/pending-review [get]
func (h *ProjectHandler) ProjectsPendingList(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.BadRequestError(ctx, "unauthorized")
	}

	reqs, err := h.svc.GetAllProjectsRequest()
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "successfully", reqs)
}

// ProjectDetailReview godoc
// @Summary Get project detail for review
// @Description Admin gets full detail of a specific project for review
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Success 200 {object} object "Project detail for review"
// @Failure 400 {object} object "Invalid project ID or error"
// @Failure 401 {object} object "Unauthorized"
// @Router /admin/projects/{id}/detail/pending-review [get]
func (h *ProjectHandler) ProjectDetailReview(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}
	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}

	project, err := h.svc.GetProjectDetailRequest(uint(id))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	return rest.SuccessResponse(ctx, "project detail review success", project)
}

func (h *ProjectHandler) Meeting(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	var body dto.CreateMeetingRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	meeting, err := h.svc.Meeting(body, user.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "meeting created successfully", meeting)
}

func (h *ProjectHandler) EditMeeting(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.ErrorMessage(ctx, http.StatusBadRequest, errors.New("invalid project id"))
	}

	var body dto.UpdateMeetingRequest
	if err := ctx.Bind().Body(&body); err != nil {
		return rest.BadRequestError(ctx, "invalid body")
	}

	meeting, err := h.svc.EditMeeting(uint(id), body, user.ID)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "meeting updated successfully", meeting)
}

func (h *ProjectHandler) CancelMeeting(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	id, err := strconv.Atoi(ctx.Params("id"))
	if err != nil {
		return rest.BadRequestError(ctx, "invalid meeting id")
	}

	err = h.svc.CancelMeeting(uint(id), user)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "meeting canceled successfully", nil)
}

func (h *ProjectHandler) GetMyMeeting(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	idParam := ctx.Params("id")
	meetingID, err := strconv.Atoi(idParam)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid meeting id")
	}

	meeting, err := h.svc.GetMyMeeting(user.ID, uint(meetingID))
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", meeting)
}

func (h *ProjectHandler) GetMyMilestoneMeetings(ctx fiber.Ctx) error {

	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	idParam := ctx.Params("id")
	milestoneID, err := strconv.Atoi(idParam)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid milestone id")
	}

	filter := ctx.Query("filter", "all") // default = all

	// validate filter
	if filter != "upcoming" && filter != "past" && filter != "all" {
		return rest.BadRequestError(ctx, "invalid filter")
	}

	meetings, err := h.svc.GetMyMeetingsByMilestone(user.ID, uint(milestoneID), filter)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", meetings)
}

func (h *ProjectHandler) GetMyProjectMeetings(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	idParam := ctx.Params("id")
	projectID, err := strconv.Atoi(idParam)
	if err != nil {
		return rest.BadRequestError(ctx, "invalid project id")
	}

	filter := ctx.Query("filter", "all")

	// validate filter
	if filter != "upcoming" && filter != "past" && filter != "all" {
		return rest.BadRequestError(ctx, "invalid filter")
	}

	meetings, err := h.svc.GetMyMeetingsByProject(user.ID, uint(projectID), filter)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", meetings)
}

func (h *ProjectHandler) GetMyMeetings(ctx fiber.Ctx) error {
	user := h.auth.GetCurrentUser(ctx)
	if user.ID == 0 {
		return rest.UnauthorizedError(ctx, "unauthorized")
	}

	meetings, err := h.svc.GetMyMeetings(user.ID)

	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "success", meetings)
}

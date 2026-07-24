package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock: ProjectService

type MockProjectService struct {
	mock.Mock
}

func (m *MockProjectService) CreateProject(ownerID uint) (*domain.Project, error) {
	args := m.Called(ownerID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) GetProjectDetailByID(id uint) (*domain.Project, error) {
	args := m.Called(id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) UpdateProject(projectID uint, input dto.UpdateProjectRequest, user domain.User) (*domain.Project, error) {
	args := m.Called(projectID, input, user)
	if v := args.Get(0); v != nil {
		return v.(*domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) DeleteProject(projectID uint, user domain.User) error {
	return m.Called(projectID, user).Error(0)
}

func (m *MockProjectService) GetMyProjects(ownerID uint) ([]domain.Project, error) {
	args := m.Called(ownerID)
	return args.Get(0).([]domain.Project), args.Error(1)
}

func (m *MockProjectService) GetPublicProjects(filter dto.PublicProjectFilter) ([]domain.Project, error) {
	args := m.Called(filter)
	return args.Get(0).([]domain.Project), args.Error(1)
}

func (m *MockProjectService) GetPublicProjectByID(id uint) (*domain.Project, error) {
	args := m.Called(id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) GetOwnerProjectByID(id uint, ownerID uint) (*domain.Project, error) {
	args := m.Called(id, ownerID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) GetProjectsByCategory(categoryID uint) ([]domain.Project, error) {
	args := m.Called(categoryID)
	return args.Get(0).([]domain.Project), args.Error(1)
}

func (m *MockProjectService) UpdateProjectStatus(projectID uint, newState domain.ProjectState, newStatus domain.ProjectStatus) error {
	return m.Called(projectID, newState, newStatus).Error(0)
}

func (m *MockProjectService) GetProjectRecommendations() ([]domain.Project, error) {
	args := m.Called()
	return args.Get(0).([]domain.Project), args.Error(1)
}

func (m *MockProjectService) GetNewProjects() ([]domain.Project, error) {
	args := m.Called()
	return args.Get(0).([]domain.Project), args.Error(1)
}

func (m *MockProjectService) GetProjectEndingSoon() ([]domain.Project, error) {
	args := m.Called()
	return args.Get(0).([]domain.Project), args.Error(1)
}

func (m *MockProjectService) GetExecutingProjects() ([]domain.Project, error) {
	args := m.Called()
	return args.Get(0).([]domain.Project), args.Error(1)
}

func (m *MockProjectService) GetPublicProjectBySlug(slug string) (*domain.Project, error) {
	args := m.Called(slug)
	if v := args.Get(0); v != nil {
		return v.(*domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) AttachProjectMedia(ctx context.Context, projectID uint, url string, mediaTypes []domain.MediaType, user domain.User) error {
	return m.Called(ctx, projectID, url, mediaTypes, user).Error(0)
}

func (m *MockProjectService) GetProjectMedia(projectID uint) ([]domain.ProjectMedia, error) {
	args := m.Called(projectID)
	return args.Get(0).([]domain.ProjectMedia), args.Error(1)
}

func (m *MockProjectService) UpdateProjectMedia(mediaID uint, input *domain.ProjectMedia, user domain.User) error {
	return m.Called(mediaID, input, user).Error(0)
}

func (m *MockProjectService) DeleteProjectMedia(mediaID uint, user domain.User) error {
	return m.Called(mediaID, user).Error(0)
}

func (m *MockProjectService) CreateMilestone(projectID uint, input dto.CreateMilestoneRequest, user domain.User) (*domain.Milestone, error) {
	args := m.Called(projectID, input, user)
	if v := args.Get(0); v != nil {
		return v.(*domain.Milestone), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) UpdateMilestone(milestoneID uint, input dto.UpdateMilestoneRequest, user domain.User) error {
	return m.Called(milestoneID, input, user).Error(0)
}

func (m *MockProjectService) DeleteMilestone(milestoneID uint, user domain.User) error {
	return m.Called(milestoneID, user).Error(0)
}

func (m *MockProjectService) GetProjectMilestones(projectID uint) ([]domain.Milestone, error) {
	args := m.Called(projectID)
	return args.Get(0).([]domain.Milestone), args.Error(1)
}

func (m *MockProjectService) SubmitMilestone(milestoneID uint, input dto.SubmitMilestoneRequest, user domain.User) (*domain.Milestone, error) {
	args := m.Called(milestoneID, input, user)
	if v := args.Get(0); v != nil {
		return v.(*domain.Milestone), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) CancelSubmit(milestoneID uint, user domain.User) (*domain.Milestone, error) {
	args := m.Called(milestoneID, user)
	if v := args.Get(0); v != nil {
		return v.(*domain.Milestone), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) AdminApproveMilestoneSubmission(milestoneID uint) (*domain.Milestone, error) {
	args := m.Called(milestoneID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Milestone), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) AdminRejectMilestoneSubmission(milestoneID uint, reason *string) (*domain.Milestone, error) {
	args := m.Called(milestoneID, reason)
	if v := args.Get(0); v != nil {
		return v.(*domain.Milestone), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) OpenMilestoneVoting(milestoneID uint, user domain.User) (*domain.Milestone, error) {
	args := m.Called(milestoneID, user)
	if v := args.Get(0); v != nil {
		return v.(*domain.Milestone), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) GetSubmittedMilestonesForAdmin(projectID *uint) ([]dto.AdminMilestoneListResponse, error) {
	args := m.Called(projectID)
	return args.Get(0).([]dto.AdminMilestoneListResponse), args.Error(1)
}

func (m *MockProjectService) GetAdminMilestoneDetail(milestoneID uint) (*dto.AdminMilestoneDetailResponse, error) {
	args := m.Called(milestoneID)
	if v := args.Get(0); v != nil {
		return v.(*dto.AdminMilestoneDetailResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) CreateProjectUpdate(projectID uint, req dto.CreateProjectUpdateRequest, user domain.User) error {
	return m.Called(projectID, req, user).Error(0)
}

func (m *MockProjectService) GetProjectUpdates(projectID uint) ([]domain.ProjectUpdate, error) {
	args := m.Called(projectID)
	return args.Get(0).([]domain.ProjectUpdate), args.Error(1)
}

func (m *MockProjectService) UpdateProjectUpdate(updateID uint, input dto.UpdateProjectUpdateRequest, user domain.User) error {
	return m.Called(updateID, input, user).Error(0)
}

func (m *MockProjectService) DeleteProjectUpdate(updateID uint, user domain.User) error {
	return m.Called(updateID, user).Error(0)
}

func (m *MockProjectService) GetProjectStories(projectID uint) ([]domain.StorySection, error) {
	args := m.Called(projectID)
	return args.Get(0).([]domain.StorySection), args.Error(1)
}

func (m *MockProjectService) CreateStorySection(section *domain.StorySection, user domain.User) error {
	return m.Called(section, user).Error(0)
}

func (m *MockProjectService) UpdateStorySection(section *domain.StorySection, user domain.User) error {
	return m.Called(section, user).Error(0)
}

func (m *MockProjectService) DeleteStorySection(sectionID uint, user domain.User) error {
	return m.Called(sectionID, user).Error(0)
}

func (m *MockProjectService) GetCategoryByID(id uint) (*domain.ProjectCategory, error) {
	args := m.Called(id)
	if v := args.Get(0); v != nil {
		return v.(*domain.ProjectCategory), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) GetAllCategories() ([]domain.ProjectCategory, error) {
	args := m.Called()
	return args.Get(0).([]domain.ProjectCategory), args.Error(1)
}

func (m *MockProjectService) CreateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error) {
	args := m.Called(category)
	if v := args.Get(0); v != nil {
		return v.(*domain.ProjectCategory), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) UpdateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error) {
	args := m.Called(category)
	if v := args.Get(0); v != nil {
		return v.(*domain.ProjectCategory), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) DeleteCategory(categoryID uint) error {
	return m.Called(categoryID).Error(0)
}

func (m *MockProjectService) GetProjectFAQs(projectID uint) ([]domain.ProjectFAQ, error) {
	args := m.Called(projectID)
	return args.Get(0).([]domain.ProjectFAQ), args.Error(1)
}

func (m *MockProjectService) CreateProjectFAQ(faq *domain.ProjectFAQ, user domain.User) error {
	return m.Called(faq, user).Error(0)
}

func (m *MockProjectService) UpdateProjectFAQ(faq *domain.ProjectFAQ, user domain.User) error {
	return m.Called(faq, user).Error(0)
}

func (m *MockProjectService) DeleteProjectFAQ(faqID uint, user domain.User) error {
	return m.Called(faqID, user).Error(0)
}

func (m *MockProjectService) GetProjectThreads(projectID uint) ([]domain.ProjectThread, error) {
	args := m.Called(projectID)
	return args.Get(0).([]domain.ProjectThread), args.Error(1)
}

func (m *MockProjectService) GetProjectUpdateThreads(projectID, updateID uint) ([]domain.ProjectThread, error) {
	args := m.Called(projectID, updateID)
	return args.Get(0).([]domain.ProjectThread), args.Error(1)
}

func (m *MockProjectService) CreateProjectThread(thread *domain.ProjectThread, user domain.User) error {
	return m.Called(thread, user).Error(0)
}

func (m *MockProjectService) CreateBoosterThread(thread *domain.ProjectThread, user domain.User) error {
	return m.Called(thread, user).Error(0)
}

func (m *MockProjectService) CreateUpdateThread(thread *domain.ProjectThread, user domain.User) error {
	return m.Called(thread, user).Error(0)
}

func (m *MockProjectService) CreateBoosterUpdateThread(thread *domain.ProjectThread, user domain.User) error {
	return m.Called(thread, user).Error(0)
}

func (m *MockProjectService) UpdateProjectThread(thread *domain.ProjectThread, user domain.User) error {
	return m.Called(thread, user).Error(0)
}

func (m *MockProjectService) DeleteProjectThread(threadID uint, user domain.User) error {
	return m.Called(threadID, user).Error(0)
}

func (m *MockProjectService) GetProjectThreadMessages(threadID uint) ([]domain.ProjectThreadMessage, error) {
	args := m.Called(threadID)
	return args.Get(0).([]domain.ProjectThreadMessage), args.Error(1)
}

func (m *MockProjectService) CreateProjectThreadMessage(msg *domain.ProjectThreadMessage, user domain.User) error {
	return m.Called(msg, user).Error(0)
}

func (m *MockProjectService) UpdateProjectThreadMessage(msg *domain.ProjectThreadMessage, user domain.User) error {
	return m.Called(msg, user).Error(0)
}

func (m *MockProjectService) DeleteProjectThreadMessage(msgID uint, user domain.User) error {
	return m.Called(msgID, user).Error(0)
}

func (m *MockProjectService) SubmitForReview(projectID uint, user domain.User) error {
	return m.Called(projectID, user).Error(0)
}

func (m *MockProjectService) ApproveProject(projectID uint) error {
	return m.Called(projectID).Error(0)
}

func (m *MockProjectService) RejectProject(projectID uint) error {
	return m.Called(projectID).Error(0)
}

func (m *MockProjectService) CloseProject(projectID uint, user domain.User) error {
	return m.Called(projectID, user).Error(0)
}

func (m *MockProjectService) CancelProject(projectID uint, user domain.User) error {
	return m.Called(projectID, user).Error(0)
}

func (m *MockProjectService) GetAllProjectsRequest() ([]domain.Project, error) {
	args := m.Called()
	return args.Get(0).([]domain.Project), args.Error(1)
}

func (m *MockProjectService) GetProjectDetailRequest(projectID uint) (*domain.Project, error) {
	args := m.Called(projectID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) GetProjectDetailAny(projectID uint) (*domain.Project, error) {
	args := m.Called(projectID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) AutoProjectLifecycleTick(now time.Time) error {
	return m.Called(now).Error(0)
}

func (m *MockProjectService) SubmitCancelRequest(projectID uint, input dto.CancelProjectRequest, user domain.User) error {
	return m.Called(projectID, input, user).Error(0)
}

func (m *MockProjectService) ApproveCancelProject(projectID uint) error {
	return m.Called(projectID).Error(0)
}

func (m *MockProjectService) RejectCancelProject(projectID uint) error {
	return m.Called(projectID).Error(0)
}

func (m *MockProjectService) GetCancelRequest() ([]domain.Project, error) {
	args := m.Called()
	return args.Get(0).([]domain.Project), args.Error(1)
}

func (m *MockProjectService) GetCancelPreview(projectID uint) (*dto.CancelPreviewResponse, error) {
	args := m.Called(projectID)
	if v := args.Get(0); v != nil {
		return v.(*dto.CancelPreviewResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) AdminListProjects(filter dto.AdminProjectFilter) ([]domain.Project, error) {
	args := m.Called(filter)
	return args.Get(0).([]domain.Project), args.Error(1)
}

func (m *MockProjectService) GetPlatformStats() (*dto.PlatformStatsResponse, error) {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.(*dto.PlatformStatsResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) Meeting(input dto.CreateMeetingRequest, userID uint) (*domain.Meeting, error) {
	args := m.Called(input, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Meeting), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) EditMeeting(meetingID uint, input dto.UpdateMeetingRequest, userID uint) (*domain.Meeting, error) {
	args := m.Called(meetingID, input, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Meeting), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) CancelMeeting(meetingID uint, user domain.User) error {
	return m.Called(meetingID, user).Error(0)
}

func (m *MockProjectService) GetMyMeeting(userID uint, meetingID uint) (*domain.Meeting, error) {
	args := m.Called(userID, meetingID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Meeting), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockProjectService) GetMyMeetings(userID uint) ([]domain.Meeting, error) {
	args := m.Called(userID)
	return args.Get(0).([]domain.Meeting), args.Error(1)
}

func (m *MockProjectService) GetMyMeetingsByMilestone(userID uint, milestoneID uint, filter string) ([]domain.Meeting, error) {
	args := m.Called(userID, milestoneID, filter)
	return args.Get(0).([]domain.Meeting), args.Error(1)
}

func (m *MockProjectService) GetMyMeetingsByProject(userID uint, projectID uint, filter string) ([]domain.Meeting, error) {
	args := m.Called(userID, projectID, filter)
	return args.Get(0).([]domain.Meeting), args.Error(1)
}

func (m *MockProjectService) GetMyMeetingsAsBooster(userID uint) ([]dto.InvestorMeetingItem, error) {
	args := m.Called(userID)
	return args.Get(0).([]dto.InvestorMeetingItem), args.Error(1)
}

// Setup

func setupProjectTest(t *testing.T) (*fiber.App, *MockProjectService, *ProjectHandler) {
	app := fiber.New()
	mockSvc := new(MockProjectService)
	h := &ProjectHandler{
		svc:         mockSvc,
		validator:   validator.New(),
		auth:        helper.Auth{Secret: testSecret},
		adminLogSvc: &MockAdminLogService{},
	}
	return app, mockSvc, h
}

// GetPublicProjects

func TestProjectHandler_GetPublicProjects(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects", h.GetPublicProjects)

	mockSvc.On("GetPublicProjects", mock.Anything).Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetPublicProjectByID

func TestProjectHandler_GetPublicProjectByID(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/:id", h.GetPublicProjectByID)

	mockSvc.On("GetPublicProjectByID", uint(1)).Return(&domain.Project{ID: 1}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/1", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetPublicProjectByID_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/projects/:id", h.GetPublicProjectByID)

	req := httptest.NewRequest(http.MethodGet, "/projects/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestProjectHandler_GetPublicProjectByID_NotFound(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/:id", h.GetPublicProjectByID)

	mockSvc.On("GetPublicProjectByID", uint(999)).Return(nil, errors.New("not found"))
	req := httptest.NewRequest(http.MethodGet, "/projects/999", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// GetAllCategories

func TestProjectHandler_GetAllCategories(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/categories", h.GetAllCategories)

	mockSvc.On("GetAllCategories").Return([]domain.ProjectCategory{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/categories", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetCategoryByID

func TestProjectHandler_GetCategoryByID(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/categories/:id", h.GetCategoryByID)

	mockSvc.On("GetCategoryByID", uint(2)).Return(&domain.ProjectCategory{ID: 2, Name: "Tech"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/categories/2", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetCategoryByID_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/categories/:id", h.GetCategoryByID)

	req := httptest.NewRequest(http.MethodGet, "/categories/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// CreateCategory

func TestProjectHandler_CreateCategory(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Post("/admin/categories", h.CreateCategory)

	body := map[string]string{"name": "Technology"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("CreateCategory", mock.Anything).Return(&domain.ProjectCategory{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/categories", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_CreateCategory_ValidationFail(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Post("/admin/categories", h.CreateCategory)

	body := map[string]string{"name": "X"} // ชื่อสั้นเกินไป (< 2 chars)
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/admin/categories", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// UpdateCategory

func TestProjectHandler_UpdateCategory(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Put("/admin/categories/:id", h.UpdateCategory)

	body := map[string]string{"name": "New Category"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("UpdateCategory", mock.Anything).Return(&domain.ProjectCategory{}, nil)
	req := httptest.NewRequest(http.MethodPut, "/admin/categories/3", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_UpdateCategory_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Put("/admin/categories/:id", h.UpdateCategory)

	body := map[string]string{"name": "New Category"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/admin/categories/abc", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// DeleteCategory

func TestProjectHandler_DeleteCategory(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Delete("/admin/categories/:id", h.DeleteCategory)

	mockSvc.On("DeleteCategory", uint(3)).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/admin/categories/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_DeleteCategory_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Delete("/admin/categories/:id", h.DeleteCategory)

	req := httptest.NewRequest(http.MethodDelete, "/admin/categories/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// GetMyProjects

func TestProjectHandler_GetMyProjects(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/pioneer/projects", h.GetMyProjects)

	mockSvc.On("GetMyProjects", uint(1)).Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/pioneer/projects", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetMyProjects_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/pioneer/projects", h.GetMyProjects)

	req := httptest.NewRequest(http.MethodGet, "/pioneer/projects", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// GetMyProjectByID

func TestProjectHandler_GetMyProjectByID(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/pioneer/projects/:id", h.GetMyProjectByID)

	mockSvc.On("GetOwnerProjectByID", uint(5), uint(1)).Return(&domain.Project{ID: 5}, nil)
	req := httptest.NewRequest(http.MethodGet, "/pioneer/projects/5", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetMyProjectByID_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/pioneer/projects/:id", h.GetMyProjectByID)

	req := httptest.NewRequest(http.MethodGet, "/pioneer/projects/5", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_GetMyProjectByID_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/pioneer/projects/:id", h.GetMyProjectByID)

	req := httptest.NewRequest(http.MethodGet, "/pioneer/projects/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// CreateProject

func TestProjectHandler_CreateProject(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/projects", h.CreateProject)

	proj := &domain.Project{ID: 10, OwnerUserID: 1}
	mockSvc.On("CreateProject", uint(1)).Return(proj, nil)
	mockSvc.On("GetOwnerProjectByID", uint(10), uint(1)).Return(proj, nil)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_CreateProject_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Post("/pioneer/projects", h.CreateProject)

	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// DeleteProject

func TestProjectHandler_DeleteProject(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "pioneer@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Delete("/pioneer/projects/:id", h.DeleteProject)

	mockSvc.On("DeleteProject", uint(5), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/5", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_DeleteProject_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Delete("/pioneer/projects/:id", h.DeleteProject)

	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/5", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// GetPlatformStats

func TestProjectHandler_GetPlatformStats(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/stats", h.GetPlatformStats)

	mockSvc.On("GetPlatformStats").Return(&dto.PlatformStatsResponse{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetProjectMilestones

func TestProjectHandler_GetProjectMilestones(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/:id/milestones", h.GetProjectMilestones)

	mockSvc.On("GetProjectMilestones", uint(3)).Return([]domain.Milestone{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/3/milestones", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// AdminListProjects

func TestProjectHandler_AdminListProjects(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/admin/projects", h.AdminListProjects)

	mockSvc.On("AdminListProjects", mock.Anything).Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/projects", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetProjectUpdates

func TestProjectHandler_GetProjectUpdates(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/:id/updates", h.GetProjectUpdates)

	mockSvc.On("GetProjectUpdates", uint(3)).Return([]domain.ProjectUpdate{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/3/updates", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetProjectFAQs

func TestProjectHandler_GetProjectFAQs(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/:id/faqs", h.GetProjectFAQs)

	mockSvc.On("GetProjectFAQs", uint(3)).Return([]domain.ProjectFAQ{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/3/faqs", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetProjectThreads

func TestProjectHandler_GetProjectThreads(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/:id/threads", h.GetProjectThreads)

	mockSvc.On("GetProjectThreads", uint(3)).Return([]domain.ProjectThread{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/3/threads", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetRecommendationProjects

func TestProjectHandler_GetRecommendationProjects(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/recommend", h.GetRecommendationProjects)

	mockSvc.On("GetProjectRecommendations").Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/recommend", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetNewProjects

func TestProjectHandler_GetNewProjects(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/new", h.GetNewProjects)

	mockSvc.On("GetNewProjects").Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/new", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetEndingSoonProjects

func TestProjectHandler_GetEndingSoonProjects(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/ending", h.GetEndingSoonProjects)

	mockSvc.On("GetProjectEndingSoon").Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/ending", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetExecutingProjects

func TestProjectHandler_GetExecutingProjects(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/executing", h.GetExecutingProjects)

	mockSvc.On("GetExecutingProjects").Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/executing", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetPublicProjectBySlug

func TestProjectHandler_GetPublicProjectBySlug(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/slug/:slug", h.GetPublicProjectBySlug)

	mockSvc.On("GetPublicProjectBySlug", "my-slug").Return(&domain.Project{ID: 1}, nil)
	mockSvc.On("GetMyProjects", uint(0)).Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/slug/my-slug", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetPublicProjectBySlug_NotFound(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/slug/:slug", h.GetPublicProjectBySlug)

	mockSvc.On("GetPublicProjectBySlug", "bad-slug").Return(nil, errors.New("not found"))
	req := httptest.NewRequest(http.MethodGet, "/projects/slug/bad-slug", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// CancelProject

func TestProjectHandler_CancelProject(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/:id/cancel", h.CancelProject)

	mockSvc.On("CancelProject", uint(5), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/5/cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_CancelProject_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/pioneer/projects/:id/cancel", h.CancelProject)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/5/cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_CancelProject_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/:id/cancel", h.CancelProject)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/abc/cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// SubmitCancelProject

func TestProjectHandler_SubmitCancelProject(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/:id/submit-cancel", h.SubmitCancelProject)

	body := dto.CancelProjectRequest{Reason: "Need to cancel because of a valid reason here"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("SubmitCancelRequest", uint(5), mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/5/submit-cancel", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_SubmitCancelProject_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/pioneer/projects/:id/submit-cancel", h.SubmitCancelProject)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/5/submit-cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ApproveCancel

func TestProjectHandler_ApproveCancel(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/projects/:id/approve-cancel", h.ApproveCancel)

	mockSvc.On("ApproveCancelProject", uint(5)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/5/approve-cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_ApproveCancel_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/admin/projects/:id/approve-cancel", h.ApproveCancel)

	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/5/approve-cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_ApproveCancel_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/projects/:id/approve-cancel", h.ApproveCancel)

	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/abc/approve-cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// RejectCancel

func TestProjectHandler_RejectCancel(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/projects/:id/reject-cancel", h.RejectCancel)

	mockSvc.On("RejectCancelProject", uint(5)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/5/reject-cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_RejectCancel_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/admin/projects/:id/reject-cancel", h.RejectCancel)

	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/5/reject-cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// GetCancelPreview

func TestProjectHandler_GetCancelPreview(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/admin/projects/:id/cancel-preview", h.GetCancelPreview)

	mockSvc.On("GetCancelPreview", uint(5)).Return(&dto.CancelPreviewResponse{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/projects/5/cancel-preview", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetCancelPreview_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/admin/projects/:id/cancel-preview", h.GetCancelPreview)

	req := httptest.NewRequest(http.MethodGet, "/admin/projects/abc/cancel-preview", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// GetPendingCancel

func TestProjectHandler_GetPendingCancel(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/projects/cancel-request", h.GetPendingCancel)

	mockSvc.On("GetCancelRequest").Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/projects/cancel-request", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetPendingCancel_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/admin/projects/cancel-request", h.GetPendingCancel)

	req := httptest.NewRequest(http.MethodGet, "/admin/projects/cancel-request", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// UpdateProject

func TestProjectHandler_UpdateProject(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/:id", h.UpdateProject)

	title := "Updated Title"
	body := dto.UpdateProjectRequest{Title: &title}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("UpdateProject", uint(5), mock.Anything, mock.Anything).Return(&domain.Project{ID: 5}, nil)
	mockSvc.On("GetOwnerProjectByID", uint(5), uint(1)).Return(&domain.Project{ID: 5}, nil)
	mockSvc.On("GetMyProjects", uint(0)).Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/5", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_UpdateProject_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/pioneer/projects/:id", h.UpdateProject)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/5", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// AddProjectMilestone

func TestProjectHandler_AddProjectMilestone(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/projects/:id/milestones", h.AddProjectMilestone)

	body := dto.CreateMilestoneRequest{Title: "Milestone 1"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("CreateMilestone", uint(5), mock.Anything, mock.Anything).Return(&domain.Milestone{ID: 1}, nil)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/5/milestones", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_AddProjectMilestone_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Post("/pioneer/projects/:id/milestones", h.AddProjectMilestone)

	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/5/milestones", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_AddProjectMilestone_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/projects/:id/milestones", h.AddProjectMilestone)

	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/abc/milestones", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// DeleteProjectMilestone

func TestProjectHandler_DeleteProjectMilestone(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Delete("/pioneer/projects/milestones/:milestone_id", h.DeleteProjectMilestone)

	mockSvc.On("DeleteMilestone", uint(3), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/milestones/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_DeleteProjectMilestone_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Delete("/pioneer/projects/milestones/:milestone_id", h.DeleteProjectMilestone)

	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/milestones/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_DeleteProjectMilestone_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Delete("/pioneer/projects/milestones/:milestone_id", h.DeleteProjectMilestone)

	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/milestones/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// AddProjectStory

func TestProjectHandler_AddProjectStory(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/projects/:id/stories", h.AddProjectStory)

	body := domain.StorySection{Title: "Story Title", Body: "Story content"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("CreateStorySection", mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/5/stories", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_AddProjectStory_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Post("/pioneer/projects/:id/stories", h.AddProjectStory)

	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/5/stories", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_AddProjectStory_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/projects/:id/stories", h.AddProjectStory)

	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/abc/stories", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// UpdateProjectStory

func TestProjectHandler_UpdateProjectStory(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/stories/:story_id", h.UpdateProjectStory)

	body := domain.StorySection{Title: "Updated Title"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("UpdateStorySection", mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/stories/3", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_UpdateProjectStory_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/pioneer/projects/stories/:story_id", h.UpdateProjectStory)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/stories/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_UpdateProjectStory_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/stories/:story_id", h.UpdateProjectStory)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/stories/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// DeleteProjectStory

func TestProjectHandler_DeleteProjectStory(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Delete("/pioneer/projects/stories/:story_id", h.DeleteProjectStory)

	mockSvc.On("DeleteStorySection", uint(3), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/stories/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_DeleteProjectStory_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Delete("/pioneer/projects/stories/:story_id", h.DeleteProjectStory)

	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/stories/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// GetProjectMedia

func TestProjectHandler_GetProjectMedia(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/pioneer/projects/:id/media", h.GetProjectMedia)

	mockSvc.On("GetProjectMedia", uint(5)).Return([]domain.ProjectMedia{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/pioneer/projects/5/media", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// UpdateProjectMedia

func TestProjectHandler_UpdateProjectMedia(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/media/:media_id", h.UpdateProjectMedia)

	body := domain.ProjectMedia{URL: "https://example.com/img.jpg"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("UpdateProjectMedia", uint(2), mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/media/2", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_UpdateProjectMedia_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/pioneer/projects/media/:media_id", h.UpdateProjectMedia)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/media/2", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// DeleteProjectMedia

func TestProjectHandler_DeleteProjectMedia(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Delete("/pioneer/projects/media/:media_id", h.DeleteProjectMedia)

	mockSvc.On("DeleteProjectMedia", uint(2), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/media/2", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_DeleteProjectMedia_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Delete("/pioneer/projects/media/:media_id", h.DeleteProjectMedia)

	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/media/2", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// UpdateProjectMilestone

func TestProjectHandler_UpdateProjectMilestone(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/milestones/:milestone_id", h.UpdateProjectMilestone)

	title := "Updated Milestone"
	body := dto.UpdateMilestoneRequest{Title: &title}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("UpdateMilestone", uint(3), mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/milestones/3", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_UpdateProjectMilestone_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/pioneer/projects/milestones/:milestone_id", h.UpdateProjectMilestone)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/milestones/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// SubmitProjectMilestone

func TestProjectHandler_SubmitProjectMilestone(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/milestones/:milestone_id/submit", h.SubmitProjectMilestone)

	body := dto.SubmitMilestoneRequest{Summary: "ได้ดำเนินการตามแผนเรียบร้อยแล้ว ระบบทำงานได้ตามเป้าหมายที่กำหนดไว้ในเฟสนี้ทุกประการ"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("SubmitMilestone", uint(3), mock.Anything, mock.Anything).Return(&domain.Milestone{ID: 3}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/milestones/3/submit", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_SubmitProjectMilestone_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/pioneer/projects/milestones/:milestone_id/submit", h.SubmitProjectMilestone)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/milestones/3/submit", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_SubmitProjectMilestone_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/milestones/:milestone_id/submit", h.SubmitProjectMilestone)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/milestones/abc/submit", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// CancelProjectMilestone

func TestProjectHandler_CancelProjectMilestone(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/milestones/:milestone_id/cancel", h.CancelProjectMilestone)

	mockSvc.On("CancelSubmit", uint(3), mock.Anything).Return(&domain.Milestone{ID: 3}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/milestones/3/cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_CancelProjectMilestone_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/pioneer/projects/milestones/:milestone_id/cancel", h.CancelProjectMilestone)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/milestones/3/cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_CancelProjectMilestone_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/milestones/:milestone_id/cancel", h.CancelProjectMilestone)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/milestones/abc/cancel", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// AdminListSubmittedMilestones

func TestProjectHandler_AdminListSubmittedMilestones(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/admin/projects/milestones/submitted", h.AdminListSubmittedMilestones)

	mockSvc.On("GetSubmittedMilestonesForAdmin", (*uint)(nil)).Return([]dto.AdminMilestoneListResponse{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/projects/milestones/submitted", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// AdminGetMilestoneDetail

func TestProjectHandler_AdminGetMilestoneDetail(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/admin/projects/milestones/:milestone_id", h.AdminGetMilestoneDetail)

	mockSvc.On("GetAdminMilestoneDetail", uint(3)).Return(&dto.AdminMilestoneDetailResponse{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/projects/milestones/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_AdminGetMilestoneDetail_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/admin/projects/milestones/:milestone_id", h.AdminGetMilestoneDetail)

	req := httptest.NewRequest(http.MethodGet, "/admin/projects/milestones/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// AdminApproveMilestoneSubmission

func TestProjectHandler_AdminApproveMilestoneSubmission(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/projects/milestones/:milestone_id/approve", h.AdminApproveMilestoneSubmission)

	mockSvc.On("AdminApproveMilestoneSubmission", uint(3)).Return(&domain.Milestone{ID: 3}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/milestones/3/approve", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_AdminApproveMilestoneSubmission_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/admin/projects/milestones/:milestone_id/approve", h.AdminApproveMilestoneSubmission)

	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/milestones/abc/approve", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// AdminRejectMilestoneSubmission

func TestProjectHandler_AdminRejectMilestoneSubmission(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/projects/milestones/:milestone_id/reject", h.AdminRejectMilestoneSubmission)

	mockSvc.On("AdminRejectMilestoneSubmission", uint(3), mock.Anything).Return(&domain.Milestone{ID: 3}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/milestones/3/reject", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_AdminRejectMilestoneSubmission_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/admin/projects/milestones/:milestone_id/reject", h.AdminRejectMilestoneSubmission)

	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/milestones/abc/reject", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// OpenMilestoneVoting

func TestProjectHandler_OpenMilestoneVoting(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/milestones/:milestone_id/open-vote", h.OpenMilestoneVoting)

	mockSvc.On("OpenMilestoneVoting", uint(3), mock.Anything).Return(&domain.Milestone{ID: 3}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/milestones/3/open-vote", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_OpenMilestoneVoting_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/pioneer/projects/milestones/:milestone_id/open-vote", h.OpenMilestoneVoting)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/milestones/3/open-vote", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_OpenMilestoneVoting_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/milestones/:milestone_id/open-vote", h.OpenMilestoneVoting)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/milestones/abc/open-vote", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// GetProjectStories

func TestProjectHandler_GetProjectStories(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/pioneer/projects/:id/stories", h.GetProjectStories)

	mockSvc.On("GetProjectStories", uint(5)).Return([]domain.StorySection{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/pioneer/projects/5/stories", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetProjectsByCategory

func TestProjectHandler_GetProjectsByCategory(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/category/:category_id", h.GetProjectsByCategory)

	mockSvc.On("GetProjectsByCategory", uint(2)).Return([]domain.Project{}, nil)
	mockSvc.On("GetMyProjects", mock.Anything).Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/category/2", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// UpdateProjectStatus

func TestProjectHandler_UpdateProjectStatus(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/projects/:id/status", h.UpdateProjectStatus)

	body := dto.UpdateProjectStatusRequest{State: "funding", Status: "active"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("UpdateProjectStatus", uint(5), mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/5/status", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_UpdateProjectStatus_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/admin/projects/:id/status", h.UpdateProjectStatus)

	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/5/status", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// CreateProjectUpdate

func TestProjectHandler_CreateProjectUpdate(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/projects/:id/updates", h.CreateProjectUpdate)

	body := dto.CreateProjectUpdateRequest{Title: "Update 1", Content: "content here"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("CreateProjectUpdate", uint(5), mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/5/updates", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_CreateProjectUpdate_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Post("/pioneer/projects/:id/updates", h.CreateProjectUpdate)

	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/5/updates", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// UpdateProjectUpdate

func TestProjectHandler_UpdateProjectUpdate(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/updates/:update_id", h.UpdateProjectUpdate)

	title := "New Title"
	body := dto.UpdateProjectUpdateRequest{Title: &title}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("UpdateProjectUpdate", uint(3), mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/updates/3", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// DeleteProjectUpdate

func TestProjectHandler_DeleteProjectUpdate(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Delete("/pioneer/projects/updates/:update_id", h.DeleteProjectUpdate)

	mockSvc.On("DeleteProjectUpdate", uint(3), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/updates/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// CreateProjectFAQ

func TestProjectHandler_CreateProjectFAQ(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/projects/:id/faqs", h.CreateProjectFAQ)

	body := domain.ProjectFAQ{Question: "What is this?", Answer: "A project."}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("CreateProjectFAQ", mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/5/faqs", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// UpdateProjectFAQ

func TestProjectHandler_UpdateProjectFAQ(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/faqs/:faq_id", h.UpdateProjectFAQ)

	body := domain.ProjectFAQ{Answer: "Updated answer"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("UpdateProjectFAQ", mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/faqs/2", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// DeleteProjectFAQ

func TestProjectHandler_DeleteProjectFAQ(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Delete("/pioneer/projects/faqs/:faq_id", h.DeleteProjectFAQ)

	mockSvc.On("DeleteProjectFAQ", uint(2), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/faqs/2", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetProjectUpdateThreads

func TestProjectHandler_GetProjectUpdateThreads(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/:id/updates/:update_id/threads", h.GetProjectUpdateThreads)

	mockSvc.On("GetProjectUpdateThreads", uint(5), uint(3)).Return([]domain.ProjectThread{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/5/updates/3/threads", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// CreateProjectThread

func TestProjectHandler_CreateProjectThread(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/projects/:id/threads", h.CreateProjectThread)

	threadTitle := "Thread Title"
	body := domain.ProjectThread{Title: &threadTitle, Body: "thread body"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("CreateProjectThread", mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/5/threads", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// CreateBoosterProjectThread

func TestProjectHandler_CreateBoosterProjectThread(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 2, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/booster/projects/:id/threads", h.CreateBoosterProjectThread)

	boosterTitle := "Booster Thread"
	body := domain.ProjectThread{Title: &boosterTitle, Body: "booster thread body"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("CreateBoosterThread", mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/booster/projects/5/threads", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// CreateProjectUpdateThread

func TestProjectHandler_CreateProjectUpdateThread(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/projects/:id/updates/:update_id/threads", h.CreateProjectUpdateThread)

	updateTitle := "Update Thread"
	body := domain.ProjectThread{Title: &updateTitle, Body: "update thread body"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("CreateUpdateThread", mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/5/updates/3/threads", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// CreateBoosterProjectUpdateThread

func TestProjectHandler_CreateBoosterProjectUpdateThread(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 2, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Post("/booster/projects/:id/updates/:update_id/threads", h.CreateBoosterProjectUpdateThread)

	boosterUpdateTitle := "Booster Update Thread"
	body := domain.ProjectThread{Title: &boosterUpdateTitle, Body: "booster update thread body"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("CreateBoosterUpdateThread", mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/booster/projects/5/updates/3/threads", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// UpdateProjectThread

func TestProjectHandler_UpdateProjectThread(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/threads/:thread_id", h.UpdateProjectThread)

	updatedTitle := "Updated Thread"
	body := domain.ProjectThread{Title: &updatedTitle, Body: "updated thread body"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("UpdateProjectThread", mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/threads/3", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// DeleteProjectThread

func TestProjectHandler_DeleteProjectThread(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Delete("/pioneer/projects/threads/:thread_id", h.DeleteProjectThread)

	mockSvc.On("DeleteProjectThread", uint(3), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/threads/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// GetProjectThreadMessages

func TestProjectHandler_GetProjectThreadMessages(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Get("/projects/threads/:thread_id/messages", h.GetProjectThreadMessages)

	mockSvc.On("GetProjectThreadMessages", uint(3)).Return([]domain.ProjectThreadMessage{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/projects/threads/3/messages", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// CreateProjectThreadMessage

func TestProjectHandler_CreateProjectThreadMessage(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/projects/threads/:thread_id/messages", h.CreateProjectThreadMessage)

	body := domain.ProjectThreadMessage{Body: "Hello there"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("CreateProjectThreadMessage", mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/threads/3/messages", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// UpdateProjectThreadMessage

func TestProjectHandler_UpdateProjectThreadMessage(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/messages/:message_id", h.UpdateProjectThreadMessage)

	body := domain.ProjectThreadMessage{Body: "Updated content"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("UpdateProjectThreadMessage", mock.Anything, mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/messages/4", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// DeleteProjectThreadMessage

func TestProjectHandler_DeleteProjectThreadMessage(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Delete("/pioneer/projects/messages/:message_id", h.DeleteProjectThreadMessage)

	mockSvc.On("DeleteProjectThreadMessage", uint(4), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/pioneer/projects/messages/4", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// SubmitForReview

func TestProjectHandler_SubmitForReview(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/:id/submit", h.SubmitForReview)

	mockSvc.On("SubmitForReview", uint(5), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/5/submit", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ApproveProject

func TestProjectHandler_ApproveProject(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/projects/:id/approve", h.ApproveProject)

	mockSvc.On("ApproveProject", uint(5)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/5/approve", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_ApproveProject_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/admin/projects/:id/approve", h.ApproveProject)

	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/abc/approve", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// RejectProject

func TestProjectHandler_RejectProject(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/projects/:id/reject", h.RejectProject)

	mockSvc.On("RejectProject", uint(5)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/5/reject", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_RejectProject_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/admin/projects/:id/reject", h.RejectProject)

	req := httptest.NewRequest(http.MethodPatch, "/admin/projects/abc/reject", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// CloseProject

func TestProjectHandler_CloseProject(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/:id/close", h.CloseProject)

	mockSvc.On("CloseProject", uint(5), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/5/close", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ProjectsPendingList

func TestProjectHandler_ProjectsPendingList(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/projects/pending-review", h.ProjectsPendingList)

	mockSvc.On("GetAllProjectsRequest").Return([]domain.Project{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/projects/pending-review", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ProjectDetailReview

func TestProjectHandler_ProjectDetailReview(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/projects/:id/detail/pending-review", h.ProjectDetailReview)

	mockSvc.On("GetProjectDetailRequest", uint(5)).Return(&domain.Project{ID: 5}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/projects/5/detail/pending-review", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_ProjectDetailReview_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/admin/projects/:id/detail/pending-review", h.ProjectDetailReview)

	req := httptest.NewRequest(http.MethodGet, "/admin/projects/5/detail/pending-review", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_ProjectDetailReview_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/projects/:id/detail/pending-review", h.ProjectDetailReview)

	req := httptest.NewRequest(http.MethodGet, "/admin/projects/abc/detail/pending-review", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ProjectDetailAny

func TestProjectHandler_ProjectDetailAny(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/projects/:id/detail", h.ProjectDetailAny)

	mockSvc.On("GetProjectDetailAny", uint(5)).Return(&domain.Project{ID: 5}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/projects/5/detail", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_ProjectDetailAny_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/admin/projects/:id/detail", h.ProjectDetailAny)

	req := httptest.NewRequest(http.MethodGet, "/admin/projects/5/detail", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// Meeting

func TestProjectHandler_Meeting(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/pioneer/projects/meeting", h.Meeting)

	body := dto.CreateMeetingRequest{MilestoneID: 1, Date: "2026-01-01", Time: "10:00", MeetingType: "online"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("Meeting", mock.Anything, uint(1)).Return(&domain.Meeting{ID: 1}, nil)
	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/meeting", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_Meeting_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Post("/pioneer/projects/meeting", h.Meeting)

	req := httptest.NewRequest(http.MethodPost, "/pioneer/projects/meeting", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// EditMeeting

func TestProjectHandler_EditMeeting(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/meeting/:id", h.EditMeeting)

	body := dto.UpdateMeetingRequest{MilestoneID: 1, Date: "2026-01-02", Time: "11:00", MeetingType: "online"}
	bodyJSON, _ := json.Marshal(body)
	mockSvc.On("EditMeeting", uint(1), mock.Anything, uint(1)).Return(&domain.Meeting{ID: 1}, nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/meeting/1", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_EditMeeting_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/pioneer/projects/meeting/:id", h.EditMeeting)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/meeting/1", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// CancelMeeting

func TestProjectHandler_CancelMeeting(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/pioneer/projects/cancel/meeting/:id", h.CancelMeeting)

	mockSvc.On("CancelMeeting", uint(1), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/cancel/meeting/1", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_CancelMeeting_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Patch("/pioneer/projects/cancel/meeting/:id", h.CancelMeeting)

	req := httptest.NewRequest(http.MethodPatch, "/pioneer/projects/cancel/meeting/1", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// GetMyMeeting

func TestProjectHandler_GetMyMeeting(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/me/meeting/:id", h.GetMyMeeting)

	mockSvc.On("GetMyMeeting", uint(1), uint(2)).Return(&domain.Meeting{ID: 2}, nil)
	req := httptest.NewRequest(http.MethodGet, "/me/meeting/2", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetMyMeeting_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/me/meeting/:id", h.GetMyMeeting)

	req := httptest.NewRequest(http.MethodGet, "/me/meeting/2", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestProjectHandler_GetMyMeeting_InvalidID(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/me/meeting/:id", h.GetMyMeeting)

	req := httptest.NewRequest(http.MethodGet, "/me/meeting/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// GetMyMilestoneMeetings

func TestProjectHandler_GetMyMilestoneMeetings(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/me/milestones/:id/meetings", h.GetMyMilestoneMeetings)

	mockSvc.On("GetMyMeetingsByMilestone", uint(1), uint(3), "all").Return([]domain.Meeting{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/me/milestones/3/meetings", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetMyMilestoneMeetings_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/me/milestones/:id/meetings", h.GetMyMilestoneMeetings)

	req := httptest.NewRequest(http.MethodGet, "/me/milestones/3/meetings", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// GetMyProjectMeetings

func TestProjectHandler_GetMyProjectMeetings(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/me/projects/:id/meetings", h.GetMyProjectMeetings)

	mockSvc.On("GetMyMeetingsByProject", uint(1), uint(5), "all").Return([]domain.Meeting{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/me/projects/5/meetings", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetMyProjectMeetings_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/me/projects/:id/meetings", h.GetMyProjectMeetings)

	req := httptest.NewRequest(http.MethodGet, "/me/projects/5/meetings", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// GetMyMeetings

func TestProjectHandler_GetMyMeetings(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Get("/me/meetings", h.GetMyMeetings)

	mockSvc.On("GetMyMeetings", uint(1)).Return([]domain.Meeting{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/me/meetings", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetMyMeetings_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/me/meetings", h.GetMyMeetings)

	req := httptest.NewRequest(http.MethodGet, "/me/meetings", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// GetMyInvestorMeetings

func TestProjectHandler_GetMyInvestorMeetings(t *testing.T) {
	app, mockSvc, h := setupProjectTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 2, Email: "booster@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/me/investor-meetings", h.GetMyInvestorMeetings)

	mockSvc.On("GetMyMeetingsAsBooster", uint(2)).Return([]dto.InvestorMeetingItem{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/me/investor-meetings", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestProjectHandler_GetMyInvestorMeetings_Unauthorized(t *testing.T) {
	app, _, h := setupProjectTest(t)
	app.Get("/me/investor-meetings", h.GetMyInvestorMeetings)

	req := httptest.NewRequest(http.MethodGet, "/me/investor-meetings", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

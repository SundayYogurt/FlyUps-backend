package services

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateProject_Success(t *testing.T) {
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)

	// Create dummy cloudinary (if we pass nil some methods may panic, but CreateProject doesn't use it)
	svc := NewProjectService(projRepo, userRepo, &helper.CloudinaryService{}, nil, nil, nil, nil)

	ownerID := uint(1)

	// Mock Id Card verification:
	projRepo.On("GetApprovedIdCard", ownerID).Return(&domain.IdCardVerification{Status: "approved"}, nil)

	// Mock Student Card verification:
	projRepo.On("GetApprovedStudentCard", ownerID).Return(&domain.StudentCardVerification{Status: "approved"}, nil)

	// Mock Bank Account:
	userRepo.On("FindBankByUserId", ownerID).Return([]domain.BankAccount{{ID: 1, AccountNumber: "12345"}}, nil)

	// Mock Project Creation:
	expectedProject := &domain.Project{
		ID:          100,
		OwnerUserID: ownerID,
		State:       domain.StateDraft,
		Status:      domain.StatusActive,
		Visibility:  domain.VisibilityPrivate,
		Title:       "Untitled Project",
		PlatformFee: 5.0,
	}

	projRepo.On("CreateProject", mock.AnythingOfType("*domain.Project")).Return(expectedProject, nil)
	projRepo.On("UpdateProject", expectedProject).Return(expectedProject, nil)

	result, err := svc.CreateProject(ownerID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(100), result.ID)
	assert.Equal(t, ownerID, result.OwnerUserID)
	assert.Equal(t, domain.StateDraft, result.State)

	projRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestCreateProject_Fail_NoBank(t *testing.T) {
	projRepo := new(ProjectRepository)
	userRepo := new(mockUserRepository)
	svc := NewProjectService(projRepo, userRepo, &helper.CloudinaryService{}, nil, nil, nil, nil)

	ownerID := uint(1)

	projRepo.On("GetApprovedIdCard", ownerID).Return(&domain.IdCardVerification{Status: "approved"}, nil)
	projRepo.On("GetApprovedStudentCard", ownerID).Return(&domain.StudentCardVerification{Status: "approved"}, nil)

	// Return empty bank account
	userRepo.On("FindBankByUserId", ownerID).Return([]domain.BankAccount{}, nil)

	result, err := svc.CreateProject(ownerID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, "bank account needed. please add your bank account first", err.Error())

	projRepo.AssertExpectations(t)
	userRepo.AssertExpectations(t)
}

func TestDeleteProject_Success(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	user := domain.User{ID: 1}
	projectID := uint(10)

	// Mock finding project
	projRepo.On("FindProjectByID", projectID).Return(&domain.Project{
		ID:          projectID,
		OwnerUserID: user.ID,
		State:       domain.StateDraft,
	}, nil)

	// Mock executing delete
	projRepo.On("DeleteProject", projectID).Return(nil)

	err := svc.DeleteProject(projectID, user)

	assert.NoError(t, err)
	projRepo.AssertExpectations(t)
}

func TestUpdateProject_Success(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	user := domain.User{ID: 1}
	projectID := uint(10)

	existingProject := &domain.Project{
		ID:          projectID,
		OwnerUserID: user.ID,
		State:       domain.StateDraft, // Must be draft to update
	}
	projRepo.On("FindProjectByID", projectID).Return(existingProject, nil)

	newTitle := "New Title Awesome"
	input := dto.UpdateProjectRequest{
		Title: &newTitle,
	}

	// Make sure UpdateProject is configured to accept mock
	projRepo.On("UpdateProject", mock.AnythingOfType("*domain.Project")).Return(existingProject, nil)

	res, err := svc.UpdateProject(projectID, input, user)

	assert.NoError(t, err)
	assert.Equal(t, "New Title Awesome", res.Title) // Object is modified in memory

	projRepo.AssertExpectations(t)
}

func TestGetMyProjects_Success(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	ownerID := uint(1)
	projects := []domain.Project{
		{ID: 1, Title: "Project 1"},
		{ID: 2, Title: "Project 2"},
	}

	projRepo.On("FindProjectsByOwnerID", ownerID).Return(projects, nil)

	res, err := svc.GetMyProjects(ownerID)

	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, "Project 1", res[0].Title)

	projRepo.AssertExpectations(t)
}

func TestGetProjectDetailByID_Success(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	projectID := uint(100)
	expectedProject := &domain.Project{ID: projectID, Title: "Detail View", State: domain.StateFunding}

	statusPtr := domain.StatusActive
	visibilityPtr := domain.VisibilityPublic
	projRepo.On("FindProjectDetailByID", projectID, (*domain.ProjectState)(nil), &statusPtr, &visibilityPtr).Return(expectedProject, nil)

	res, err := svc.GetProjectDetailByID(projectID)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "Detail View", res.Title)

	projRepo.AssertExpectations(t)
}

func TestCreateCategory_Success(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	inputCategory := &domain.ProjectCategory{Name: "Tech"}
	createdCategory := &domain.ProjectCategory{ID: 1, Name: "Tech"}

	projRepo.On("CreateCategory", inputCategory).Return(createdCategory, nil)

	res, err := svc.CreateCategory(inputCategory)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, uint(1), res.ID)

	projRepo.AssertExpectations(t)
}

func TestCreateMilestone_Success(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	user := domain.User{ID: 5}
	projectID := uint(10)

	existingProject := &domain.Project{ID: projectID, OwnerUserID: user.ID}

	projRepo.On("FindProjectByID", projectID).Return(existingProject, nil)
	projRepo.On("FindMilestonesByProjectID", projectID).Return([]domain.Milestone{}, nil) // 0 existing => phase 1
	projRepo.On("CreateMilestone", mock.AnythingOfType("*domain.Milestone")).Return(nil)

	input := dto.CreateMilestoneRequest{} // Usually passed with title, desc, etc but core sets defaults
	res, err := svc.CreateMilestone(projectID, input, user)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 1, res.PhaseNo)
	assert.Equal(t, domain.MilestoneDraft, res.Status)

	projRepo.AssertExpectations(t)
}

func TestGetAllPendingStateProjects(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	// mock data
	expected := []domain.Project{
		{ID: 1},
		{ID: 2},
	}

	// ต้อง match "pending"
	projRepo.On("FindProjectsState", string(domain.StatePendingReview)).
		Return(expected, nil)

	// call
	result, err := svc.GetAllProjectsRequest()

	// assert
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	projRepo.AssertExpectations(t)
}

func TestGetAllPendingStateProjectRequests_Error(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	projRepo.On("FindProjectsState", string(domain.StatePendingReview)).
		Return(nil, errors.New("db error"))

	result, err := svc.GetAllProjectsRequest()

	assert.Error(t, err)
	assert.Nil(t, result)

	projRepo.AssertExpectations(t)
}

func TestGetProjectPendingDetail(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	projectID := uint(100)

	expected := &domain.Project{
		ID: projectID,
	}

	projRepo.On("FindProjectsPendingDetail", projectID, string(domain.StatePendingReview)).
		Return(expected, nil)

	result, err := svc.GetProjectDetailRequest(projectID)

	// assert
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	projRepo.AssertExpectations(t)
}

func TestGetProjectPendingDetail_fail_invalidID(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	result, err := svc.GetProjectDetailRequest(0)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestAutoProjectLifecycleTick_FundingExpireToDraftFailed(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	state := domain.StateFunding
	status := domain.StatusActive
	now := time.Now().UTC()
	expired := domain.Project{
		ID:             99,
		State:          domain.StateFunding,
		Status:         domain.StatusActive,
		FundingGoal:    100000,
		Softcap:        70000,
		CurrentFunding: 10000,
		EndDate:        now.Add(-1 * time.Hour),
	}

	executing := domain.StateExecuting
	projRepo.On("FindProjects", &state, &status, (*domain.ProjectVisibility)(nil), (*uint)(nil), "").Return([]domain.Project{expired}, nil)
	projRepo.On("FindProjects", &executing, &status, (*domain.ProjectVisibility)(nil), (*uint)(nil), "").Return([]domain.Project{}, nil)
	projRepo.On("UpdateProject", mock.AnythingOfType("*domain.Project")).Return(&expired, nil)

	err := svc.AutoProjectLifecycleTick(now)
	assert.NoError(t, err)
	projRepo.AssertExpectations(t)
}

func TestAutoProjectLifecycleTick_ExecutionExpireToClosedFailed(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	funding := domain.StateFunding
	executing := domain.StateExecuting
	status := domain.StatusActive
	now := time.Now().UTC()
	executionEnd := now.Add(-1 * time.Hour)
	executingProject := domain.Project{
		ID:             88,
		State:          domain.StateExecuting,
		Status:         domain.StatusActive,
		ExecutionEndAt: &executionEnd,
	}
	milestones := []domain.Milestone{
		{ID: 1, ProjectID: 88, Status: domain.MilestonePaid},
		{ID: 2, ProjectID: 88, Status: domain.MilestoneRejected},
	}

	projRepo.On("FindProjects", &funding, &status, (*domain.ProjectVisibility)(nil), (*uint)(nil), "").Return([]domain.Project{}, nil)
	projRepo.On("FindProjects", &executing, &status, (*domain.ProjectVisibility)(nil), (*uint)(nil), "").Return([]domain.Project{executingProject}, nil)
	projRepo.On("FindMilestonesByProjectID", uint(88)).Return(milestones, nil)
	projRepo.On("UpdateProject", mock.AnythingOfType("*domain.Project")).Return(&executingProject, nil)

	err := svc.AutoProjectLifecycleTick(now)
	assert.NoError(t, err)
	projRepo.AssertExpectations(t)
}

func TestAutoProjectLifecycleTick_MilestoneOverdue_SuspendProjectFailed(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	funding := domain.StateFunding
	executing := domain.StateExecuting
	status := domain.StatusActive
	now := time.Now().UTC()
	due := now.Add(-1 * time.Hour)

	executingProject := domain.Project{
		ID:     77,
		State:  domain.StateExecuting,
		Status: domain.StatusActive,
	}
	milestones := []domain.Milestone{
		{ID: 1, ProjectID: 77, PhaseNo: 1, Status: domain.MilestoneActive, DueDate: &due},
	}

	projRepo.On("FindProjects", &funding, &status, (*domain.ProjectVisibility)(nil), (*uint)(nil), "").Return([]domain.Project{}, nil)
	projRepo.On("FindProjects", &executing, &status, (*domain.ProjectVisibility)(nil), (*uint)(nil), "").Return([]domain.Project{executingProject}, nil)
	projRepo.On("FindMilestonesByProjectID", uint(77)).Return(milestones, nil)
	projRepo.On("UpdateMilestone", mock.AnythingOfType("*domain.Milestone")).Return(nil)
	projRepo.On("UpdateProject", mock.AnythingOfType("*domain.Project")).Return(&executingProject, nil)

	err := svc.AutoProjectLifecycleTick(now)
	assert.NoError(t, err)
	projRepo.AssertExpectations(t)
}

package service

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"fmt"

	"gorm.io/gorm"
)

type ProjectService struct {
	Repo repository.ProjectRepository
	Auth helper.Auth
}

func (s ProjectService) CreateProject(ownerID uint, input dto.CreateProjectRequest) (*domain.Project, error) {

	// check category exists
	_, err := s.Repo.GetCategoryByID(input.CategoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, errors.New("internal server error")
	}

	slug, err := s.generateUniqueSlug(input.Title)
	if err != nil {
		return nil, errors.New("failed to generate slug")
	}

	project := domain.Project{
		OwnerUserID: ownerID,
		Title:       input.Title,
		Slug:        slug,
		CategoryID:  &input.CategoryID,
		State:       domain.ProjectStateDraft,
		Status:      domain.ProjectStatusActive,
		Visibility:  domain.ProjectVisibilityPrivate,
	}

	err = s.Repo.Create(&project)
	if err != nil {
		return nil, errors.New("failed to create project")
	}

	return &project, nil
}

func (s ProjectService) UpdateProjectDraft(
	projectID uint,
	ownerID uint,
	input dto.UpdateProjectDraftRequest,
) error {

	project, err := s.Repo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("project not found")
		}
		return errors.New("internal server error")
	}

	// check owner
	if project.OwnerUserID != ownerID {
		return errors.New("permission denied")
	}

	// allow edit only draft
	if project.State != domain.ProjectStateDraft {
		return errors.New("project cannot be edited")
	}

	if input.Title != "" {
		project.Title = input.Title
	}

	if input.Description != "" {
		project.Description = input.Description
	}

	if input.FundingGoal > 0 {
		project.FundingGoal = input.FundingGoal
	}

	err = s.Repo.Update(project)
	if err != nil {
		return errors.New("failed to update project")
	}

	return nil
}

func (s ProjectService) AddProjectMedia(
	projectID uint,
	ownerID uint,
	input dto.AddProjectMediaRequest,
) error {

	project, err := s.Repo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("project not found")
		}
		return errors.New("internal server error")
	}

	if project.OwnerUserID != ownerID {
		return errors.New("permission denied")
	}

	media := domain.ProjectMedia{
		ProjectID: projectID,
		URL:       input.URL,
		Type:      input.Type,
	}

	err = s.Repo.CreateMedia(&media)
	if err != nil {
		return errors.New("failed to add media")
	}

	return nil
}

func (s ProjectService) GetProjectsByOwner(ownerID uint) ([]domain.Project, error) {

	projects, err := s.Repo.GetByOwner(ownerID)
	if err != nil {
		return nil, err
	}

	return projects, nil
}
func (s ProjectService) GetProjectByID(id uint, ownerID uint) (*domain.Project, error) {

	project, err := s.Repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if project.OwnerUserID != ownerID {
		return nil, errors.New("access denied")
	}

	return project, nil
}

func (s ProjectService) generateUniqueSlug(title string) (string, error) {

	base := helper.GenerateSlug(title)
	slug := base
	i := 1

	for {
		exists, err := s.Repo.SlugExists(slug)
		if err != nil {
			return "", err
		}

		if !exists {
			break
		}

		i++
		slug = fmt.Sprintf("%s-%d", base, i)
	}

	return slug, nil
}

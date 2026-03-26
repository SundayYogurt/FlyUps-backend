package service

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/repository"
)

type ProjectService interface {
	CreateProject(ownerID uint) (*domain.Project, error)
	UpdateProject(projectID uint, data *domain.Project) (*domain.Project, error)
	GetMyProjects(id uint) ([]domain.Project, error)
	GetPublicProjectByID(id uint) (*domain.Project, error)
	GetOwnerProjectByID(id uint, ownerID uint) (*domain.Project, error)
	AddProjectMedia(projectID uint, media *domain.ProjectMedia) error
	UpdateStorySection(sectionID uint, data *domain.StorySection) error
	AddProjectFAQ(projectID uint, faq *domain.ProjectFAQ) error
	UpdateMilestone(milestoneID uint, data *domain.Milestone) error
	CreateProjectUpdate(projectID uint, update *domain.ProjectUpdate) error
	UpdateProjectStatus(projectID uint, newState domain.ProjectState, newStatus domain.ProjectStatus) error

	//categories
	GetAllCategories() ([]domain.ProjectCategory, error)
	GetCategoryByID(id uint) (*domain.ProjectCategory, error)
	CreateCategory(name string) (*domain.ProjectCategory, error)
	UpdateCategory(id uint, name string) (*domain.ProjectCategory, error)
	DeleteCategory(id uint) error
}

type projectService struct {
	projectRepo repository.ProjectRepository
}

func NewProjectService(projectRepo repository.ProjectRepository) ProjectService {
	return &projectService{
		projectRepo: projectRepo,
	}
}

// GetAllCategories ดึงหมวดหมู่ทั้งหมด
func (s *projectService) GetAllCategories() ([]domain.ProjectCategory, error) {
	return s.projectRepo.FindAllCategories()
}

// GetCategoryByID ดึงหมวดหมู่ตาม ID
func (s *projectService) GetCategoryByID(id uint) (*domain.ProjectCategory, error) {
	if id == 0 {
		return nil, errors.New("category ID is required")
	}
	return s.projectRepo.FindCategoryByID(id)
}

// CreateCategory สร้างหมวดหมู่ใหม่
func (s *projectService) CreateCategory(name string) (*domain.ProjectCategory, error) {
	if name == "" {
		return nil, errors.New("category name is required")
	}

	// สร้าง Object category
	newCategory := &domain.ProjectCategory{
		Name: name,
	}

	return s.projectRepo.CreateCategory(newCategory)
}

// UpdateCategory แก้ไขชื่อหมวดหมู่
func (s *projectService) UpdateCategory(id uint, name string) (*domain.ProjectCategory, error) {
	if id == 0 || name == "" {
		return nil, errors.New("invalid input")
	}

	// หา Category เดิมก่อน
	category, err := s.projectRepo.FindCategoryByID(id)
	if err != nil {
		return nil, err
	}

	// อัปเดตฟิลด์
	category.Name = name

	return s.projectRepo.UpdateCategory(category)
}

// DeleteCategory ลบหมวดหมู่
func (s *projectService) DeleteCategory(id uint) error {
	if id == 0 {
		return errors.New("category ID is required")
	}

	// เช็คว่ามี Project ใช้งานอยู่ไหม
	count, err := s.projectRepo.CountProjectsByCategoryID(id)
	if err != nil {
		return err
	}

	if count > 0 {
		// ถ้ามี project ใช้งานอยู่ ห้ามลบ
		return errors.New("cannot delete: category is currently in use by projects")
	}

	// ถ้าไม่มีคนใช้ ก็ดึงข้อมูลมาลบตามปกติ
	category, err := s.projectRepo.FindCategoryByID(id)
	if err != nil {
		return err
	}

	return s.projectRepo.DeleteCategory(category)
}

func (p *projectService) UpdateProject(projectID uint, data *domain.Project) (*domain.Project, error) {
	if projectID == 0 {
		return nil, errors.New("projectID is required")
	}

	project, err := p.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return nil, err
	}

	// update fields
	if data.Title != "" {
		project.Title = data.Title
	}
	if data.Description != nil {
		project.Description = data.Description
	}
	if data.Category != nil {
		project.CategoryID = data.CategoryID
		project.Category = nil
	}

	if data.Visibility != "" {
		project.Visibility = data.Visibility
	}

	return p.projectRepo.UpdateProject(project)
}

func (p *projectService) AddProjectMedia(projectID uint, media *domain.ProjectMedia) error {
	//TODO implement me
	panic("implement me")
}

func (p *projectService) UpdateStorySection(sectionID uint, data *domain.StorySection) error {
	//TODO implement me
	panic("implement me")
}

func (p *projectService) AddProjectFAQ(projectID uint, faq *domain.ProjectFAQ) error {
	//TODO implement me
	panic("implement me")
}

func (p *projectService) UpdateMilestone(milestoneID uint, data *domain.Milestone) error {
	//TODO implement me
	panic("implement me")
}

func (p *projectService) CreateProjectUpdate(projectID uint, update *domain.ProjectUpdate) error {
	//TODO implement me
	panic("implement me")
}

func (p *projectService) UpdateProjectStatus(projectID uint, newState domain.ProjectState, newStatus domain.ProjectStatus) error {
	//TODO implement me
	panic("implement me")
}

func (p *projectService) CreateProject(ownerID uint) (*domain.Project, error) {
	if ownerID == 0 {
		return nil, errors.New("owner is required")
	}

	project := &domain.Project{
		OwnerUserID: ownerID,

		// default สำหรับ draft
		State:      domain.StateDraft,
		Status:     domain.StatusActive,
		Visibility: domain.VisibilityPrivate,
		Title:      "Untitled Project", // กัน null
	}

	return p.projectRepo.CreateProject(project)
}

func (p *projectService) GetMyProjects(ownerID uint) ([]domain.Project, error) {
	if ownerID == 0 {
		return nil, errors.New("owner is required")
	}

	return p.projectRepo.FindProjectsByOwnerID(ownerID)
}

func (p *projectService) GetPublicProjectByID(id uint) (*domain.Project, error) {
	project, err := p.projectRepo.FindProjectByID(id)
	if err != nil {
		return nil, err
	}

	if project.Visibility != domain.VisibilityPublic {
		return nil, errors.New("project is not public")
	}

	return project, nil
}
func (p *projectService) GetOwnerProjectByID(id uint, ownerID uint) (*domain.Project, error) {
	if id == 0 || ownerID == 0 {
		return nil, errors.New("invalid input")
	}

	project, err := p.projectRepo.FindProjectByID(id)
	if err != nil {
		return nil, err
	}

	if project.OwnerUserID != ownerID {
		return nil, errors.New("forbidden")
	}

	return project, nil
}

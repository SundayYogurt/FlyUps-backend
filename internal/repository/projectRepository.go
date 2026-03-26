package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type ProjectRepository interface {
	CreateProject(project *domain.Project) (*domain.Project, error)
	FindProjectByID(id uint) (*domain.Project, error)
	FindProjectsByOwnerID(ownerID uint) ([]domain.Project, error)
	UpdateProject(project *domain.Project) (*domain.Project, error)

	// Category Management
	FindCategoryByID(id uint) (*domain.ProjectCategory, error)
	FindAllCategories() ([]domain.ProjectCategory, error)
	CreateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error)
	UpdateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error)
	DeleteCategory(category *domain.ProjectCategory) error
	CountProjectsByCategoryID(categoryID uint) (int64, error)
	CreateProjectMedia(media *domain.ProjectMedia) error
}

type projectRepository struct {
	db *gorm.DB
}

func (p *projectRepository) CreateProjectMedia(media *domain.ProjectMedia) error {
	var lastSortOrder int
	p.db.Model(&domain.ProjectMedia{}).Where("project_id = ?", media.ProjectID).Select("COALESCE(MAX(project_id), 0)").Scan(&lastSortOrder)

	if media.SortOrder == 0 {
		media.SortOrder = lastSortOrder + 1
	}

	if err := p.db.Create(media).Error; err != nil {
		return err
	}
	return nil
}

func (p *projectRepository) CountProjectsByCategoryID(categoryID uint) (int64, error) {
	var count int64
	// นับจำนวน project ที่มี category_id ตรงกับที่ส่งมา
	err := p.db.Model(&domain.Project{}).Where("category_id = ?", categoryID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (p *projectRepository) FindCategoryByID(id uint) (*domain.ProjectCategory, error) {
	var category domain.ProjectCategory

	err := p.db.Where("id = ?", id).First(&category).Error
	if err != nil {
		return nil, err
	}

	return &category, nil
}

func (p *projectRepository) FindAllCategories() ([]domain.ProjectCategory, error) {
	var categories []domain.ProjectCategory

	err := p.db.Order("id DESC").Find(&categories).Error
	if err != nil {
		return nil, err
	}
	return categories, nil
}

func (p *projectRepository) CreateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error) {
	if err := p.db.Create(category).Error; err != nil {
		return nil, err
	}
	return category, nil
}

func (p *projectRepository) UpdateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error) {
	if err := p.db.Save(category).Error; err != nil {
		return nil, err
	}
	return category, nil
}

func (p *projectRepository) DeleteCategory(category *domain.ProjectCategory) error {
	if err := p.db.Delete(category).Error; err != nil {
		return err
	}
	return nil
}

func (p *projectRepository) UpdateProject(project *domain.Project) (*domain.Project, error) {
	if err := p.db.Save(project).Error; err != nil {
		return nil, err
	}
	return project, nil
}

func (p *projectRepository) FindProjectByID(id uint) (*domain.Project, error) {
	var project domain.Project

	if err := p.db.First(&project, id).Error; err != nil {
		return nil, err
	}

	return &project, nil
}

func (p *projectRepository) CreateProject(project *domain.Project) (*domain.Project, error) {
	if err := p.db.Create(project).Error; err != nil {
		return nil, err
	}

	return project, nil
}

func (p *projectRepository) FindProjectsByOwnerID(ownerID uint) ([]domain.Project, error) {
	var projects []domain.Project

	err := p.db.
		Where("owner_user_id = ?", ownerID).
		Order("created_at DESC").
		Find(&projects).Error

	if err != nil {
		return nil, err
	}

	return projects, nil
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

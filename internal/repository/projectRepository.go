package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type ProjectRepository interface {
	Create(project *domain.Project) error
	GetByID(id uint) (*domain.Project, error)
	Update(project *domain.Project) error
	GetCategoryByID(id uint) (*domain.ProjectCategory, error)
	GetByOwner(ownerID uint) ([]domain.Project, error)
	CreateMedia(media *domain.ProjectMedia) error
	SlugExists(slug string) (bool, error)
}

type projectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) Create(project *domain.Project) error {
	return r.db.Create(project).Error
}

func (r *projectRepository) GetByID(id uint) (*domain.Project, error) {

	var project domain.Project

	err := r.db.First(&project, id).Error
	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (r *projectRepository) GetByOwner(ownerID uint) ([]domain.Project, error) {

	var projects []domain.Project

	err := r.db.
		Where("owner_user_id = ?", ownerID).
		Order("created_at desc").
		Find(&projects).Error

	if err != nil {
		return nil, err
	}

	return projects, nil
}

func (r *projectRepository) Update(project *domain.Project) error {
	return r.db.Save(project).Error
}

func (r *projectRepository) GetCategoryByID(id uint) (*domain.ProjectCategory, error) {
	var category domain.ProjectCategory
	err := r.db.First(&category, id).Error
	return &category, err
}

func (r *projectRepository) CreateMedia(media *domain.ProjectMedia) error {
	return r.db.Create(media).Error
}

func (r *projectRepository) SlugExists(slug string) (bool, error) {
	var count int64

	err := r.db.Model(&domain.Project{}).
		Where("slug = ?", slug).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

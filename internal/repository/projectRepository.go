package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type ProjectRepository interface {
	Create(project *domain.Project) error
	CreateStory(story *domain.ProjectStorySection) error
	GetByID(id uint) (*domain.Project, error)
	Update(project *domain.Project) error
	GetCategoryByID(id uint) (*domain.ProjectCategory, error)
	GetByOwner(ownerID uint) ([]domain.Project, error)
	CreateMedia(media *domain.ProjectMedia) error
	SlugExists(slug string) (bool, error)
	CreateRisk(risk *domain.ProjectRisk) error
	CreateFAQ(faq *domain.ProjectFAQ) error
	CreateFundingPolicy(policy *domain.ProjectFundingPolicy) error
	CreateMilestone(milestone *domain.Milestone) error
	GetFullProject(id uint) (*domain.Project, error)
	CreateProfitPolicy(policy *domain.ProjectProfitPolicy) error
	GetProjectsForReview() ([]domain.Project, error)
	GetMilestonesByProject(projectID uint) ([]domain.Milestone, error)
	CountMediaByProject(projectID uint) (int64, error)
	MediaSortExists(projectID uint, sortOrder int) (bool, error)
	GetMaxRiskSort(projectID uint) (int, error)
	GetMaxFAQSort(projectID uint) (int, error)
	GetMaxMediaSort(projectID uint) (int, error)
	GetMaxStorySectionSort(projectID uint) (int, error)
	GetMaxMilestoneSort(projectID uint) (int, error)
	GetFundingPolicyByProject(projectID uint) (*domain.ProjectFundingPolicy, error)
	GetProfitPolicyByProject(projectID uint) (*domain.ProjectProfitPolicy, error)
}

type projectRepository struct {
	db *gorm.DB
}

func (r *projectRepository) CreateProfitPolicy(policy *domain.ProjectProfitPolicy) error {
	return r.db.Create(policy).Error
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

func (r *projectRepository) GetFundingPolicyByProject(projectID uint) (*domain.ProjectFundingPolicy, error) {

	var policy domain.ProjectFundingPolicy

	err := r.db.
		Where("project_id = ?", projectID).
		First(&policy).Error

	return &policy, err
}

func (r *projectRepository) GetProfitPolicyByProject(projectID uint) (*domain.ProjectProfitPolicy, error) {

	var policy domain.ProjectProfitPolicy

	err := r.db.
		Where("project_id = ?", projectID).
		First(&policy).Error

	return &policy, err
}

func (r *projectRepository) GetMaxFAQSort(projectID uint) (int, error) {

	var maxSort int

	err := r.db.
		Model(&domain.ProjectFAQ{}).
		Where("project_id = ?", projectID).
		Select("COALESCE(MAX(sort_order), -1)").
		Scan(&maxSort).Error

	return maxSort, err
}

func (r *projectRepository) GetMaxRiskSort(projectID uint) (int, error) {

	var maxSort int

	err := r.db.
		Model(&domain.ProjectRisk{}).
		Where("project_id = ?", projectID).
		Select("COALESCE(MAX(sort_order), -1)").
		Scan(&maxSort).Error

	return maxSort, err
}

func (r *projectRepository) GetMaxMilestoneSort(projectID uint) (int, error) {
	var maxSort int

	err := r.db.
		Model(&domain.Milestone{}).
		Where("project_id = ?", projectID).
		Select("COALESCE(MAX(sort_order), -1)").
		Scan(&maxSort).Error

	return maxSort, err
}

func (r *projectRepository) GetMaxStorySectionSort(projectID uint) (int, error) {
	var maxSort int

	err := r.db.
		Model(&domain.ProjectStorySection{}).
		Where("project_id = ?", projectID).
		Select("COALESCE(MAX(sort_order), -1)").
		Scan(&maxSort).Error

	return maxSort, err
}

func (r *projectRepository) GetMaxMediaSort(projectID uint) (int, error) {
	var maxSort int

	err := r.db.
		Model(&domain.ProjectMedia{}).
		Where("project_id = ?", projectID).
		Select("COALESCE(MAX(sort_order), -1)").
		Scan(&maxSort).Error

	return maxSort, err
}

func (r *projectRepository) GetFullProject(id uint) (*domain.Project, error) {

	var project domain.Project

	err := r.db.
		Preload("Media", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc")
		}).
		Preload("StorySections", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc")
		}).
		Preload("Risks", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc")
		}).
		Preload("FAQ", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order asc")
		}).
		Preload("Milestones", func(db *gorm.DB) *gorm.DB {
			return db.Order("phase_no asc")
		}).
		Preload("FundingPolicy").
		Preload("ProfitPolicy").
		First(&project, id).Error

	return &project, err
}

func (r *projectRepository) CreateRisk(risk *domain.ProjectRisk) error {
	return r.db.Create(risk).Error
}

func (r *projectRepository) GetMilestonesByProject(projectID uint) ([]domain.Milestone, error) {

	var milestones []domain.Milestone

	err := r.db.
		Where("project_id = ?", projectID).
		Order("phase_no asc").
		Find(&milestones).Error

	if err != nil {
		return nil, err
	}

	return milestones, nil
}

func (r *projectRepository) CreateFAQ(faq *domain.ProjectFAQ) error {
	return r.db.Create(faq).Error
}

func (r *projectRepository) CreateFundingPolicy(policy *domain.ProjectFundingPolicy) error {
	return r.db.Create(policy).Error
}

func (r *projectRepository) CreateMilestone(m *domain.Milestone) error {
	return r.db.Create(m).Error
}

func (r *projectRepository) CreateStory(story *domain.ProjectStorySection) error {
	return r.db.Create(story).Error
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

func (r *projectRepository) GetProjectsForReview() ([]domain.Project, error) {

	var projects []domain.Project

	err := r.db.
		Where("state = ?", domain.ProjectStateReview).
		Order("created_at asc").
		Find(&projects).Error

	if err != nil {
		return nil, err
	}

	return projects, nil
}

func (r *projectRepository) CountMediaByProject(projectID uint) (int64, error) {

	var count int64

	err := r.db.
		Model(&domain.ProjectMedia{}).
		Where("project_id = ?", projectID).
		Count(&count).Error

	return count, err
}

func (r *projectRepository) MediaSortExists(projectID uint, sortOrder int) (bool, error) {

	var count int64

	err := r.db.
		Model(&domain.ProjectMedia{}).
		Where("project_id = ? AND sort_order = ?", projectID, sortOrder).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

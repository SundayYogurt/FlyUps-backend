package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type ProjectRepository interface {
	CreateProject(project *domain.Project) (*domain.Project, error)
	FindProjectByID(id uint) (*domain.Project, error)
	FindProjectDetailByID(id uint, state *domain.ProjectState, status *domain.ProjectStatus, visibility *domain.ProjectVisibility) (*domain.Project, error)
	FindProjectByIDAndOwner(id uint, ownerID uint) (*domain.Project, error)
	FindProjectsByOwnerID(ownerID uint) ([]domain.Project, error)
	FindProjects(state *domain.ProjectState, status *domain.ProjectStatus, visibility *domain.ProjectVisibility) ([]domain.Project, error)
	FindProjectsByCategory(categoryID uint) ([]domain.Project, error)
	UpdateProject(project *domain.Project) (*domain.Project, error)
	DeleteProject(projectId uint) error
	CreateProjectUpdate(update *domain.ProjectUpdate) error
	FindUpdatesByProjectID(projectID uint) ([]domain.ProjectUpdate, error)
	FindProjectUpdateByID(updateID uint) (*domain.ProjectUpdate, error)
	UpdateProjectUpdate(update *domain.ProjectUpdate) error
	DeleteProjectUpdate(updateID uint) error

	FindMediaByProjectID(projectID uint) ([]domain.ProjectMedia, error)
	FindMediaByID(id uint) (*domain.ProjectMedia, error)
	CreateProjectMedia(media *domain.ProjectMedia) error
	UpdateProjectMedia(media *domain.ProjectMedia) error
	DeleteProjectMedia(mediaID uint) error

	FindCategoryByID(id uint) (*domain.ProjectCategory, error)
	FindAllCategories() ([]domain.ProjectCategory, error)
	CreateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error)
	UpdateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error)
	DeleteCategory(categoryId uint) error
	CountProjectsByCategoryID(categoryID uint) (int64, error)

	// milestone
	FindMilestoneByID(id uint) (*domain.Milestone, error)
	FindMilestonesByProjectID(projectID uint) ([]domain.Milestone, error)
	CreateMilestone(m *domain.Milestone) error
	UpdateMilestone(m *domain.Milestone) error
	DeleteMilestone(id uint) error

	// story
	FindStoriesByProjectID(projectID uint) ([]domain.StorySection, error)
	FindStorySectionByID(sectionID uint) (*domain.StorySection, error)
	CreateStorySection(section *domain.StorySection) error
	UpdateStorySection(section *domain.StorySection) error
	DeleteStorySection(sectionID uint) error

	// Threads
	FindThreadsByProjectID(projectID uint) ([]domain.ProjectThread, error)
	FindThreadByID(threadID uint) (*domain.ProjectThread, error)
	CreateThread(thread *domain.ProjectThread) error
	UpdateThread(thread *domain.ProjectThread) error
	DeleteThread(threadID uint) error

	// Thread Messages
	FindMessagesByThreadID(threadID uint) ([]domain.ProjectThreadMessage, error)
	FindThreadMessageByID(msgID uint) (*domain.ProjectThreadMessage, error)
	CreateThreadMessage(msg *domain.ProjectThreadMessage) error
	UpdateThreadMessage(msg *domain.ProjectThreadMessage) error
	DeleteThreadMessage(msgID uint) error

	// FAQ
	FindFAQsByProjectID(projectID uint) ([]domain.ProjectFAQ, error)
	FindFAQByID(faqID uint) (*domain.ProjectFAQ, error)
	CreateFAQ(faq *domain.ProjectFAQ) error
	UpdateFAQ(faq *domain.ProjectFAQ) error
	DeleteFAQ(faqID uint) error
}

type projectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}
func (p *projectRepository) FindProjectByIDAndOwner(id uint, ownerID uint) (*domain.Project, error) {
	var project domain.Project

	err := p.db.
		Preload("Category").
		Preload("Owner.StudentProfile.University").
		Preload("Media").
		Where("id = ? AND owner_user_id = ?", id, ownerID).
		First(&project).Error

	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (r *projectRepository) FindFAQsByProjectID(projectID uint) ([]domain.ProjectFAQ, error) {
	var faqs []domain.ProjectFAQ
	err := r.db.Where("project_id = ?", projectID).Order("sort_order ASC").Find(&faqs).Error
	return faqs, err
}

func (r *projectRepository) FindFAQByID(faqID uint) (*domain.ProjectFAQ, error) {
	var faq domain.ProjectFAQ
	err := r.db.Where("id = ?", faqID).First(&faq).Error
	return &faq, err
}

func (r *projectRepository) CreateFAQ(faq *domain.ProjectFAQ) error {
	return r.db.Create(faq).Error
}

func (r *projectRepository) UpdateFAQ(faq *domain.ProjectFAQ) error {
	return r.db.Model(&domain.ProjectFAQ{ID: faq.ID}).Updates(faq).Error
}

func (r *projectRepository) DeleteFAQ(faqID uint) error {
	return r.db.Delete(&domain.ProjectFAQ{}, faqID).Error
}

func (r *projectRepository) FindThreadsByProjectID(projectID uint) ([]domain.ProjectThread, error) {
	var threads []domain.ProjectThread
	err := r.db.Where("project_id = ?", projectID).Find(&threads).Error
	return threads, err
}

func (r *projectRepository) FindThreadByID(threadID uint) (*domain.ProjectThread, error) {
	var thread domain.ProjectThread
	err := r.db.Where("id = ?", threadID).First(&thread).Error
	return &thread, err
}

func (r *projectRepository) CreateThread(thread *domain.ProjectThread) error {
	return r.db.Create(thread).Error
}

func (r *projectRepository) UpdateThread(thread *domain.ProjectThread) error {
	return r.db.Model(&domain.ProjectThread{ID: thread.ID}).Updates(thread).Error
}

func (r *projectRepository) DeleteThread(threadID uint) error {
	return r.db.Delete(&domain.ProjectThread{}, threadID).Error
}

func (r *projectRepository) FindMessagesByThreadID(threadID uint) ([]domain.ProjectThreadMessage, error) {
	var msgs []domain.ProjectThreadMessage
	err := r.db.Where("thread_id = ?", threadID).Find(&msgs).Error
	return msgs, err
}

func (r *projectRepository) FindThreadMessageByID(msgID uint) (*domain.ProjectThreadMessage, error) {
	var msg domain.ProjectThreadMessage
	err := r.db.Where("id = ?", msgID).First(&msg).Error
	return &msg, err
}

func (r *projectRepository) CreateThreadMessage(msg *domain.ProjectThreadMessage) error {
	return r.db.Create(msg).Error
}

func (r *projectRepository) UpdateThreadMessage(msg *domain.ProjectThreadMessage) error {
	return r.db.Model(&domain.ProjectThreadMessage{ID: msg.ID}).Updates(msg).Error
}

func (r *projectRepository) DeleteThreadMessage(msgID uint) error {
	return r.db.Delete(&domain.ProjectThreadMessage{}, msgID).Error
}

func (p *projectRepository) CreateProjectUpdate(update *domain.ProjectUpdate) error {
	return p.db.Create(update).Error
}

func (p *projectRepository) FindProjectUpdateByID(updateID uint) (*domain.ProjectUpdate, error) {
	var update domain.ProjectUpdate
	err := p.db.Where("id = ?", updateID).First(&update).Error
	return &update, err
}

func (p *projectRepository) UpdateProjectUpdate(update *domain.ProjectUpdate) error {
	return p.db.Model(&domain.ProjectUpdate{ID: update.ID}).Updates(update).Error
}

func (p *projectRepository) DeleteProjectUpdate(updateID uint) error {
	return p.db.Delete(&domain.ProjectUpdate{}, updateID).Error
}

func (p *projectRepository) FindStoriesByProjectID(projectID uint) ([]domain.StorySection, error) {
	var stories []domain.StorySection
	err := p.db.
		Where("project_id = ?", projectID).
		Order("sort_order ASC").
		Find(&stories).Error
	return stories, err
}

func (p *projectRepository) FindStorySectionByID(sectionID uint) (*domain.StorySection, error) {
	var section domain.StorySection
	err := p.db.Where("id = ?", sectionID).First(&section).Error
	return &section, err
}

func (p *projectRepository) CreateStorySection(section *domain.StorySection) error {
	return p.db.Create(section).Error
}

func (p *projectRepository) UpdateStorySection(section *domain.StorySection) error {
	return p.db.Model(&domain.StorySection{ID: section.ID}).Updates(section).Error
}

func (p *projectRepository) DeleteStorySection(sectionID uint) error {
	return p.db.Delete(&domain.StorySection{}, sectionID).Error
}

func (r *projectRepository) FindUpdatesByProjectID(projectID uint) ([]domain.ProjectUpdate, error) {
	var updates []domain.ProjectUpdate
	err := r.db.Where("project_id = ?", projectID).
		Order("created_at DESC").
		Find(&updates).Error

	return updates, err
}

func (p *projectRepository) FindMilestoneByID(id uint) (*domain.Milestone, error) {
	var m domain.Milestone
	if err := p.db.First(&m, id).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func (p *projectRepository) FindMilestonesByProjectID(projectID uint) ([]domain.Milestone, error) {
	var list []domain.Milestone
	err := p.db.
		Where("project_id = ?", projectID).
		Order("phase_no ASC").
		Find(&list).Error
	return list, err
}

func (p *projectRepository) CreateMilestone(m *domain.Milestone) error {
	return p.db.Create(m).Error
}

func (p *projectRepository) UpdateMilestone(m *domain.Milestone) error {
	return p.db.Save(m).Error
}

func (p *projectRepository) DeleteMilestone(id uint) error {
	return p.db.Delete(&domain.Milestone{}, id).Error
}

func (p *projectRepository) CreateProject(project *domain.Project) (*domain.Project, error) {
	if err := p.db.Create(project).Error; err != nil {
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

func (p *projectRepository) FindProjectDetailByID(id uint, state *domain.ProjectState, status *domain.ProjectStatus, visibility *domain.ProjectVisibility) (*domain.Project, error) {
	var project domain.Project

	query := p.db.Model(&domain.Project{}).
		Preload("Category").
		Preload("Owner.StudentProfile.University").
		Preload("Media", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Preload("Milestones", func(db *gorm.DB) *gorm.DB {
			return db.Order("phase_no ASC")
		}).
		Preload("Stories", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Preload("FAQs", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Where("id = ?", id)

	// filter
	if state != nil {
		query = query.Where("state = ?", *state)
	}

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if visibility != nil {
		query = query.Where("visibility = ?", *visibility)
	}

	if err := query.First(&project).Error; err != nil {
		return nil, err
	}

	return &project, nil
}

func (p *projectRepository) FindProjects(state *domain.ProjectState, status *domain.ProjectStatus, visibility *domain.ProjectVisibility,
) ([]domain.Project, error) {
	var projects []domain.Project

	query := p.db.Model(&domain.Project{}).
		Preload("Category").
		Preload("Owner.StudentProfile.University").
		Preload("Media").
		Order("id DESC")

	if state != nil {
		query = query.Where("state = ?", *state)
	}

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if visibility != nil {
		query = query.Where("visibility = ?", *visibility)
	}

	err := query.Find(&projects).Error
	return projects, err
}

func (p *projectRepository) FindProjectsByOwnerID(ownerID uint) ([]domain.Project, error) {
	var projects []domain.Project
	err := p.db.
		Where("owner_user_id = ?", ownerID).
		Order("created_at DESC").
		Find(&projects).Error
	return projects, err
}

func (p *projectRepository) FindProjectsByCategory(categoryID uint) ([]domain.Project, error) {
	var projects []domain.Project
	err := p.db.
		Where("category_id = ?", categoryID).
		Order("created_at DESC").
		Find(&projects).Error
	return projects, err
}

func (p *projectRepository) UpdateProject(project *domain.Project) (*domain.Project, error) {
	if err := p.db.Save(project).Error; err != nil {
		return nil, err
	}
	return project, nil
}

func (p *projectRepository) DeleteProject(id uint) error {
	return p.db.Delete(&domain.Project{}, id).Error
}

// media

func (p *projectRepository) FindMediaByProjectID(projectID uint) ([]domain.ProjectMedia, error) {
	var media []domain.ProjectMedia
	err := p.db.
		Where("project_id = ?", projectID).
		Order("sort_order ASC").
		Find(&media).Error
	return media, err
}

func (p *projectRepository) FindMediaByID(id uint) (*domain.ProjectMedia, error) {
	var media domain.ProjectMedia
	if err := p.db.First(&media, id).Error; err != nil {
		return nil, err
	}
	return &media, nil
}

func (p *projectRepository) CreateProjectMedia(media *domain.ProjectMedia) error {
	var lastSortOrder int

	err := p.db.
		Model(&domain.ProjectMedia{}).
		Where("project_id = ?", media.ProjectID).
		Select("COALESCE(MAX(sort_order), 0)").
		Scan(&lastSortOrder).Error
	if err != nil {
		return err
	}

	if media.SortOrder == 0 {
		media.SortOrder = lastSortOrder + 1
	}

	return p.db.Create(media).Error
}

func (p *projectRepository) UpdateProjectMedia(media *domain.ProjectMedia) error {
	return p.db.Save(media).Error
}

func (p *projectRepository) DeleteProjectMedia(mediaID uint) error {
	return p.db.Delete(&domain.ProjectMedia{}, mediaID).Error
}

// category
func (p *projectRepository) FindCategoryByID(id uint) (*domain.ProjectCategory, error) {
	var category domain.ProjectCategory
	if err := p.db.First(&category, id).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (p *projectRepository) FindAllCategories() ([]domain.ProjectCategory, error) {
	var categories []domain.ProjectCategory
	err := p.db.Order("id DESC").Find(&categories).Error
	return categories, err
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

func (p *projectRepository) DeleteCategory(categoryId uint) error {
	return p.db.Delete(&domain.ProjectCategory{}, categoryId).Error
}

func (p *projectRepository) CountProjectsByCategoryID(categoryID uint) (int64, error) {
	var count int64
	err := p.db.
		Model(&domain.Project{}).
		Where("category_id = ?", categoryID).
		Count(&count).Error
	return count, err
}

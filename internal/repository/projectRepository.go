package repository

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"time"

	"github.com/gofiber/utils/v2/strings"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProjectRepository interface {
	CreateProject(project *domain.Project) (*domain.Project, error)
	FindProjectByID(id uint) (*domain.Project, error)
	FindProjectDetailByID(id uint, state *domain.ProjectState, status *domain.ProjectStatus, visibility *domain.ProjectVisibility) (*domain.Project, error)
	FindProjectByIDAndOwner(id uint, ownerID uint) (*domain.Project, error)
	FindProjectsByOwnerID(ownerID uint) ([]domain.Project, error)
	FindProjects(state *domain.ProjectState, status *domain.ProjectStatus, visibility *domain.ProjectVisibility) ([]domain.Project, error)
	FindPublicProjects(filter dto.PublicProjectFilter) ([]domain.Project, error)
	FindProjectsByCategory(categoryID uint) ([]domain.Project, error)
	UpdateProject(project *domain.Project) (*domain.Project, error)
	DeleteProject(projectId uint) error
	CreateProjectUpdate(update *domain.ProjectUpdate) error
	FindUpdatesByProjectID(projectID uint) ([]domain.ProjectUpdate, error)
	FindProjectUpdateByID(updateID uint) (*domain.ProjectUpdate, error)
	UpdateProjectUpdate(update *domain.ProjectUpdate) error
	DeleteProjectUpdate(updateID uint) error
	GetApprovedIdCard(userID uint) (*domain.IdCardVerification, error)
	GetApprovedStudentCard(userID uint) (*domain.StudentCardVerification, error)
	FindProjectsState(state string) ([]domain.Project, error)
	FindProjectsPendingDetail(projectID uint, state string) (*domain.Project, error)
	FindProjectRecommendations() ([]domain.Project, error)
	FindNewProjects() ([]domain.Project, error)
	FindProjectEndingSoon() ([]domain.Project, error)
	SaveMeeting(meeting *domain.Meeting) error
	FindInvestorsEmailByProjectID(projectID uint) ([]string, error)
	FindInvestorIDsByProjectID(projectID uint) ([]uint, error)
	ExistsProjectByOwnerAndStates(userID uint, states []domain.ProjectState) (bool, error)
	FindProjectsCancelRequest() ([]domain.Project, error)

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
	FindMilestonesByStatus(status domain.MilestoneStatus) ([]domain.Milestone, error)
	FindMilestonesByProjectIDAndStatus(projectID uint, status domain.MilestoneStatus) ([]domain.Milestone, error)
	CreateMilestone(m *domain.Milestone) error
	UpdateMilestone(m *domain.Milestone) error
	DeleteMilestone(id uint) error

	// milestone votes (booster)
	UpsertMilestoneVote(vote *domain.MilestoneVote) error
	CountVerifiedBoostersByProjectID(projectID uint) (int64, error)
	CountMilestoneVotes(milestoneID uint, choice domain.MilestoneVoteChoice) (int64, error)
	HasVerifiedInvestment(projectID uint, boosterUserID uint) (bool, error)
	CloseMeetingsByMilestoneID(milestoneID uint) error
	FindMeetingByID(id uint) (*domain.Meeting, error)
	FindMeetingsByMilestone(milestoneID uint, filter string) ([]domain.Meeting, error)
	FindMeetingsByProject(projectID uint, filter string) ([]domain.Meeting, error)
	UpdateMeeting(meeting *domain.Meeting) (*domain.Meeting, error)
	FindMeetingsByOwnerID(ownerID uint) ([]domain.Meeting, error)
	FindVote(milestoneID uint, boosterUserID uint) (*domain.MilestoneVote, error)
	FindMeetingByMilestoneID(milestoneID uint) (*domain.Meeting, error)

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

func (p *projectRepository) FindProjectsCancelRequest() ([]domain.Project, error) {
	var projects []domain.Project

	err := p.db.Model(&domain.Project{}).Where("state = ?", domain.StatePendingCancel).Find(&projects).Error
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func (p *projectRepository) ExistsProjectByOwnerAndStates(userID uint, states []domain.ProjectState) (bool, error) {
	var count int64

	err := p.db.Model(&domain.Project{}).
		Where("owner_user_id = ? AND state IN ?", userID, states).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}
func (p *projectRepository) FindMeetingByMilestoneID(milestoneID uint) (*domain.Meeting, error) {
	var meeting domain.Meeting
	err := p.db.Where("milestone_id = ?", milestoneID).First(&meeting).Error

	if err != nil {
		return nil, err
	}

	return &meeting, nil
}

func (p *projectRepository) FindVote(milestoneID uint, boosterUserID uint) (*domain.MilestoneVote, error) {
	var vote domain.MilestoneVote

	err := p.db.
		Where("milestone_id = ? AND booster_user_id = ?", milestoneID, boosterUserID).
		First(&vote).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &vote, nil
}

func (p *projectRepository) FindMeetingsByOwnerID(ownerID uint) ([]domain.Meeting, error) {
	var meetings []domain.Meeting

	err := p.db.
		Joins("JOIN milestones ON milestones.id = meetings.milestone_id").
		Joins("JOIN projects ON projects.id = milestones.project_id").
		Where("projects.owner_user_id = ?", ownerID).
		Find(&meetings).Error

	if err != nil {
		return nil, err
	}

	return meetings, nil
}

func (p *projectRepository) UpdateMeeting(meeting *domain.Meeting) (*domain.Meeting, error) {
	err := p.db.Model(&domain.Meeting{}).Where("id = ?", meeting.ID).Updates(meeting).Error

	if err != nil {
		return nil, err
	}

	return meeting, err
}

func (p *projectRepository) SaveMeeting(meeting *domain.Meeting) error {
	return p.db.Create(meeting).Error
}

func (p *projectRepository) FindMeetingByID(id uint) (*domain.Meeting, error) {
	var m domain.Meeting
	err := p.db.First(&m, id).Error
	return &m, err
}

func (p *projectRepository) FindMeetingsByMilestone(milestoneID uint, filter string) ([]domain.Meeting, error) {
	var meetings []domain.Meeting

	now := time.Now()

	db := p.db.Model(&domain.Meeting{}).
		Where("milestone_id = ?", milestoneID)

	switch filter {
	case "upcoming":
		db = db.Where("meetings.time >= ?", now).Order("meetings.time ASC")

	case "past":
		db = db.Where("meetings.time < ?", now).Order("meetings.time DESC")

	default: // all
		db = db.Order("date ASC, time ASC")
	}

	err := db.Find(&meetings).Error
	return meetings, err
}

func (p *projectRepository) FindMeetingsByProject(projectID uint, filter string) ([]domain.Meeting, error) {
	var meetings []domain.Meeting

	now := time.Now().UTC()
	filter = strings.ToLower(filter)

	db := p.db.Model(&domain.Meeting{}).
		Joins("JOIN milestones ON milestones.id = meetings.milestone_id").
		Where("milestones.project_id = ?", projectID).
		Distinct("meetings.*")

	switch filter {
	case "upcoming":
		db = db.Where("meetings.time >= ?", now).
			Order("meetings.time ASC")

	case "past":
		db = db.Where("meetings.time < ?", now).
			Order("meetings.time DESC")

	default: // all
		db = db.Order("meetings.time ASC")
	}

	err := db.Find(&meetings).Error
	return meetings, err
}

func (p *projectRepository) CloseMeetingsByMilestoneID(milestoneID uint) error {
	return p.db.Model(&domain.Meeting{}).
		Where("milestone_id = ? AND status != ?", milestoneID, domain.MeetingClosed).
		Update("status", domain.MeetingClosed).Error
}

func (p *projectRepository) FindInvestorsEmailByProjectID(projectID uint) ([]string, error) {
	var emails []string

	err := p.db.Model(&domain.Investment{}).
		Select("DISTINCT users.email").
		Joins("JOIN users ON users.id = investments.booster_user_id").
		Where("investments.project_id = ? AND investments.status = ?", projectID, domain.InvestmentVerified).
		Pluck("users.email", &emails).Error

	return emails, err
}

func (p *projectRepository) FindProjectRecommendations() ([]domain.Project, error) {
	var projects []domain.Project
	err := p.db.
		Where("state = ? AND visibility = ?", domain.StateFunding, domain.VisibilityPublic).
		Order(`
		current_funding / 
		GREATEST(EXTRACT(EPOCH FROM (NOW() - created_at)), 3600) DESC
	`).
		Find(&projects).Error

	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (p *projectRepository) FindNewProjects() ([]domain.Project, error) {
	var projects []domain.Project
	// เรียงตามวันที่เปิดให้ระดมทุน (funding_at) ล่าสุด
	err := p.db.Where("state = ? AND visibility = ?", domain.StateFunding, domain.VisibilityPublic).
		Order("funding_at DESC").
		Find(&projects).Error

	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (p *projectRepository) FindProjectEndingSoon() ([]domain.Project, error) {
	var projects []domain.Project
	// ดึงโปรเจกต์ที่ยังไม่หมดเวลา แต่ใกล้จะถึงวัน EndDate ที่สุด
	err := p.db.Where("state = ? AND visibility = ? AND end_date > ?", domain.StateFunding, domain.VisibilityPublic, time.Now()).
		Order("end_date ASC").
		Find(&projects).Error

	if err != nil {
		return nil, err
	}
	return projects, nil
}

func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

func (p *projectRepository) FindProjectsPendingDetail(projectID uint, state string) (*domain.Project, error) {
	var project domain.Project

	err := p.db.
		Preload("Owner").
		Preload("Category").
		Preload("Owner.StudentProfile.University").
		Preload("Owner.IdCardVerification").
		Preload("Owner.StudentCardVerification").
		Preload("Media").
		Preload("Milestones").
		Preload("Stories").
		Preload("FAQs").
		Where("id = ? AND state = ?", projectID, state).
		First(&project).Error

	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (p *projectRepository) FindProjectsState(state string) ([]domain.Project, error) {
	var projects []domain.Project
	err := p.db.
		Preload("Owner.StudentProfile.University").
		Preload("Owner.IdCardVerification").
		Preload("Owner.StudentCardVerification").
		Where("state = ?", state).
		Order("created_at DESC").
		Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func (p *projectRepository) GetApprovedStudentCard(userID uint) (*domain.StudentCardVerification, error) {
	var v domain.StudentCardVerification

	err := p.db.Where("user_id = ? AND status = ?", userID, "approved").First(&v).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &v, nil
}

func (p *projectRepository) GetApprovedIdCard(userID uint) (*domain.IdCardVerification, error) {
	var v domain.IdCardVerification

	err := p.db.Where("user_id = ? AND status = ?", userID, "approved").First(&v).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &v, nil
}

func (p *projectRepository) FindProjectByIDAndOwner(id uint, ownerID uint) (*domain.Project, error) {
	var project domain.Project

	err := p.db.Model(&domain.Project{}).
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
		Where("id = ? AND owner_user_id = ?", id, ownerID).
		First(&project).Error

	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (p *projectRepository) FindFAQsByProjectID(projectID uint) ([]domain.ProjectFAQ, error) {
	var faqs []domain.ProjectFAQ
	err := p.db.Where("project_id = ?", projectID).Order("sort_order ASC").Find(&faqs).Error
	return faqs, err
}

func (p *projectRepository) FindFAQByID(faqID uint) (*domain.ProjectFAQ, error) {
	var faq domain.ProjectFAQ
	err := p.db.Where("id = ?", faqID).First(&faq).Error
	return &faq, err
}

func (p *projectRepository) CreateFAQ(faq *domain.ProjectFAQ) error {
	return p.db.Create(faq).Error
}

func (p *projectRepository) UpdateFAQ(faq *domain.ProjectFAQ) error {
	return p.db.Model(&domain.ProjectFAQ{ID: faq.ID}).Updates(faq).Error
}

func (p *projectRepository) DeleteFAQ(faqID uint) error {
	return p.db.Delete(&domain.ProjectFAQ{}, faqID).Error
}

func (p *projectRepository) FindThreadsByProjectID(projectID uint) ([]domain.ProjectThread, error) {
	var threads []domain.ProjectThread
	err := p.db.Where("project_id = ?", projectID).Find(&threads).Error
	return threads, err
}

func (p *projectRepository) FindThreadByID(threadID uint) (*domain.ProjectThread, error) {
	var thread domain.ProjectThread
	err := p.db.Where("id = ?", threadID).First(&thread).Error
	return &thread, err
}

func (p *projectRepository) CreateThread(thread *domain.ProjectThread) error {
	return p.db.Create(thread).Error
}

func (p *projectRepository) UpdateThread(thread *domain.ProjectThread) error {
	return p.db.Model(&domain.ProjectThread{ID: thread.ID}).Updates(thread).Error
}

func (p *projectRepository) DeleteThread(threadID uint) error {
	return p.db.Delete(&domain.ProjectThread{}, threadID).Error
}

func (p *projectRepository) FindMessagesByThreadID(threadID uint) ([]domain.ProjectThreadMessage, error) {
	var msgs []domain.ProjectThreadMessage
	err := p.db.Where("thread_id = ?", threadID).Find(&msgs).Error
	return msgs, err
}

func (p *projectRepository) FindThreadMessageByID(msgID uint) (*domain.ProjectThreadMessage, error) {
	var msg domain.ProjectThreadMessage
	err := p.db.Where("id = ?", msgID).First(&msg).Error
	return &msg, err
}

func (p *projectRepository) CreateThreadMessage(msg *domain.ProjectThreadMessage) error {
	return p.db.Create(msg).Error
}

func (p *projectRepository) UpdateThreadMessage(msg *domain.ProjectThreadMessage) error {
	return p.db.Model(&domain.ProjectThreadMessage{ID: msg.ID}).Updates(msg).Error
}

func (p *projectRepository) DeleteThreadMessage(msgID uint) error {
	return p.db.Delete(&domain.ProjectThreadMessage{}, msgID).Error
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

func (p *projectRepository) FindUpdatesByProjectID(projectID uint) ([]domain.ProjectUpdate, error) {
	var updates []domain.ProjectUpdate
	err := p.db.Where("project_id = ?", projectID).
		Order("created_at DESC").
		Find(&updates).Error

	return updates, err
}

func (p *projectRepository) FindMilestoneByID(id uint) (*domain.Milestone, error) {
	var m domain.Milestone
	if err := p.db.Preload("Meetings").First(&m, id).Error; err != nil {
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

func (p *projectRepository) FindMilestonesByStatus(status domain.MilestoneStatus) ([]domain.Milestone, error) {
	var list []domain.Milestone
	err := p.db.
		Where("status = ?", status).
		Order("updated_at DESC").
		Find(&list).Error
	return list, err
}

func (p *projectRepository) FindMilestonesByProjectIDAndStatus(projectID uint, status domain.MilestoneStatus) ([]domain.Milestone, error) {
	var list []domain.Milestone
	err := p.db.
		Where("project_id = ? AND status = ?", projectID, status).
		Order("updated_at DESC").
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

func (p *projectRepository) UpsertMilestoneVote(vote *domain.MilestoneVote) error {
	// Postgres upsert by unique (milestone_id, booster_user_id)
	return p.db.
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "milestone_id"}, {Name: "booster_user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"choice", "updated_at"}),
		}).
		Create(vote).Error
}

func (p *projectRepository) CountVerifiedBoostersByProjectID(projectID uint) (int64, error) {
	var count int64
	err := p.db.Model(&domain.Investment{}).
		Where("project_id = ? AND status = ?", projectID, string(domain.InvestmentVerified)).
		Distinct("booster_user_id").
		Count(&count).Error
	return count, err
}

func (p *projectRepository) CountMilestoneVotes(milestoneID uint, choice domain.MilestoneVoteChoice) (int64, error) {
	var count int64
	err := p.db.Model(&domain.MilestoneVote{}).
		Where("milestone_id = ? AND choice = ?", milestoneID, choice).
		Count(&count).Error
	return count, err
}

func (p *projectRepository) HasVerifiedInvestment(projectID uint, boosterUserID uint) (bool, error) {
	var count int64
	err := p.db.Model(&domain.Investment{}).
		Where("project_id = ? AND booster_user_id = ? AND status = ?", projectID, boosterUserID, string(domain.InvestmentVerified)).
		Count(&count).Error
	return count > 0, err
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

// FindPublicProjects ดึงโปรเจกต์สาธารณะพร้อม filter และ sort
func (p *projectRepository) FindPublicProjects(filter dto.PublicProjectFilter) ([]domain.Project, error) {
	var projects []domain.Project

	query := p.db.Model(&domain.Project{}).
		Preload("Category").
		Preload("Owner.StudentProfile.University").
		Preload("Media").
		Where("state IN ? AND visibility = ? AND status = ?",
			[]domain.ProjectState{domain.StateFunding, domain.StateExecuting},
			domain.VisibilityPublic,
			domain.StatusActive,
		)

	// search by title
	if filter.Search != "" {
		query = query.Where("LOWER(title) LIKE ?", "%"+strings.ToLower(filter.Search)+"%")
	}

	// filter by category
	if filter.CategoryID != nil {
		query = query.Where("category_id = ?", *filter.CategoryID)
	}

	// filter by funding goal range
	if filter.MinGoal > 0 {
		query = query.Where("funding_goal >= ?", filter.MinGoal)
	}
	if filter.MaxGoal > 0 {
		query = query.Where("funding_goal <= ?", filter.MaxGoal)
	}

	// sort
	switch filter.Sort {
	case "ending_soon":
		query = query.Where("end_date > ?", time.Now()).Order("end_date ASC")
	case "popular":
		query = query.Order("current_funding DESC")
	default: // newest
		query = query.Order("funding_at DESC")
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
	// ลบ related records ก่อน (เพื่อหลีกเลี่ยง foreign key constraint)
	// ลบ project_media
	p.db.Unscoped().Where("project_id = ?", id).Delete(&domain.ProjectMedia{})

	// ลบ milestones
	p.db.Unscoped().Where("project_id = ?", id).Delete(&domain.Milestone{})

	// ลบ stories
	p.db.Unscoped().Where("project_id = ?", id).Delete(&domain.StorySection{})

	// ลบ FAQs
	p.db.Unscoped().Where("project_id = ?", id).Delete(&domain.ProjectFAQ{})

	// ลบ project updates
	p.db.Unscoped().Where("project_id = ?", id).Delete(&domain.ProjectUpdate{})

	// ลบ threads และ messages
	p.db.Unscoped().Where("project_id = ?", id).Delete(&domain.ProjectThreadMessage{})
	p.db.Unscoped().Where("project_id = ?", id).Delete(&domain.ProjectThread{})

	// ลบ investments (ถ้ามี)
	p.db.Unscoped().Where("project_id = ?", id).Delete(&domain.ProjectInvestment{})

	// สุดท้ายลบ project
	return p.db.Unscoped().Delete(&domain.Project{}, id).Error
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

func (p *projectRepository) FindInvestorIDsByProjectID(projectID uint) ([]uint, error) {
	var ids []uint

	err := p.db.Model(&domain.Investment{}).
		Select("DISTINCT investments.booster_user_id").
		Where("investments.project_id = ? AND investments.status = ?", projectID, domain.InvestmentVerified).
		Pluck("investments.booster_user_id", &ids).Error

	return ids, err
}

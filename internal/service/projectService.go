package service

import (
	"context"
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"fmt"
	"net/url"
	"strings"
	"time"
)

type ProjectService interface {
	// PROJECT CORE
	CreateProject(ownerID uint) (*domain.Project, error)
	GetProjectDetailByID(id uint) (*domain.Project, error)
	UpdateProject(projectID uint, input dto.UpdateProjectRequest, user domain.User) (*domain.Project, error)
	DeleteProject(projectID uint, user domain.User) error
	GetMyProjects(ownerID uint) ([]domain.Project, error)
	GetPublicProjects() ([]domain.Project, error)
	GetPublicProjectByID(id uint) (*domain.Project, error)
	GetOwnerProjectByID(id uint, ownerID uint) (*domain.Project, error)
	GetProjectsByCategory(categoryID uint) ([]domain.Project, error)
	UpdateProjectStatus(projectID uint, newState domain.ProjectState, newStatus domain.ProjectStatus) error

	// MEDIA
	AttachProjectMedia(ctx context.Context, projectID uint, url string, mediaTypes []domain.MediaType, user domain.User) error
	GetProjectMedia(projectID uint) ([]domain.ProjectMedia, error)
	UpdateProjectMedia(mediaID uint, input *domain.ProjectMedia, user domain.User) error
	DeleteProjectMedia(mediaID uint, user domain.User) error

	// MILESTONE
	CreateMilestone(projectID uint, input dto.CreateMilestoneRequest, user domain.User) (*domain.Milestone, error)
	UpdateMilestone(milestoneID uint, input dto.UpdateMilestoneRequest, user domain.User) error
	DeleteMilestone(milestoneID uint, user domain.User) error
	GetProjectMilestones(projectID uint) ([]domain.Milestone, error)
	SubmitMilestone(milestoneID uint, input dto.SubmitMilestoneRequest, user domain.User) (*domain.Milestone, error)
	AdminApproveMilestoneSubmission(milestoneID uint) (*domain.Milestone, error)
	AdminRejectMilestoneSubmission(milestoneID uint, reason *string) (*domain.Milestone, error)
	OpenMilestoneVoting(milestoneID uint, user domain.User) (*domain.Milestone, error)
	GetSubmittedMilestonesForAdmin(projectID *uint) ([]dto.AdminMilestoneListResponse, error)
	GetAdminMilestoneDetail(milestoneID uint) (*dto.AdminMilestoneDetailResponse, error)
	// PROJECT UPDATE
	CreateProjectUpdate(projectID uint, req dto.CreateProjectUpdateRequest, user domain.User) error
	GetProjectUpdates(projectID uint) ([]domain.ProjectUpdate, error)
	UpdateProjectUpdate(updateID uint, input dto.UpdateProjectUpdateRequest, user domain.User) error
	DeleteProjectUpdate(updateID uint, user domain.User) error

	// STORY SECTION
	GetProjectStories(projectID uint) ([]domain.StorySection, error)
	CreateStorySection(section *domain.StorySection, user domain.User) error
	UpdateStorySection(section *domain.StorySection, user domain.User) error
	DeleteStorySection(sectionID uint, user domain.User) error

	// CATEGORY
	GetCategoryByID(id uint) (*domain.ProjectCategory, error)
	GetAllCategories() ([]domain.ProjectCategory, error)
	CreateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error)
	UpdateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error)
	DeleteCategory(categoryID uint) error

	// FAQ
	GetProjectFAQs(projectID uint) ([]domain.ProjectFAQ, error)
	CreateProjectFAQ(faq *domain.ProjectFAQ, user domain.User) error
	UpdateProjectFAQ(faq *domain.ProjectFAQ, user domain.User) error
	DeleteProjectFAQ(faqID uint, user domain.User) error

	// THREADS & MESSAGES
	GetProjectThreads(projectID uint) ([]domain.ProjectThread, error)
	CreateProjectThread(thread *domain.ProjectThread, user domain.User) error
	UpdateProjectThread(thread *domain.ProjectThread, user domain.User) error
	DeleteProjectThread(threadID uint, user domain.User) error
	GetProjectThreadMessages(threadID uint) ([]domain.ProjectThreadMessage, error)
	CreateProjectThreadMessage(msg *domain.ProjectThreadMessage, user domain.User) error
	UpdateProjectThreadMessage(msg *domain.ProjectThreadMessage, user domain.User) error
	DeleteProjectThreadMessage(msgID uint, user domain.User) error

	// PROJECT LIFECYCLE
	SubmitForReview(projectID uint, user domain.User) error
	ApproveProject(projectID uint) error
	RejectProject(projectID uint) error
	CloseProject(projectID uint, user domain.User) error
	CancelProject(projectID uint, user domain.User) error
	GetAllProjectsRequest() ([]domain.Project, error)
	GetProjectDetailRequest(projectID uint) (*domain.Project, error)
	AutoProjectLifecycleTick(now time.Time) error
}

type projectService struct {
	projectRepo repository.ProjectRepository
	userRepo    repository.UserRepository
	cld         *helper.CloudinaryService
	notifSvc    NotificationService
}

func NewProjectService(projectRepo repository.ProjectRepository, userRepo repository.UserRepository, cld *helper.CloudinaryService, notifSvc NotificationService) ProjectService {
	return &projectService{
		projectRepo: projectRepo,
		userRepo:    userRepo,
		cld:         cld,
		notifSvc:    notifSvc,
	}
}

// PROJECT CORE
func (s *projectService) CreateProject(ownerID uint) (*domain.Project, error) {
	if ownerID == 0 {
		return nil, errors.New("owner is required")
	}

	// เช็ค Id Card
	idCard, err := s.projectRepo.GetApprovedIdCard(ownerID)
	if err != nil {
		return nil, err
	}
	if idCard == nil {
		return nil, errors.New("id card not verified")
	}

	// เช็ค Student Card
	studentCard, err := s.projectRepo.GetApprovedStudentCard(ownerID)
	if err != nil {
		return nil, err
	}
	if studentCard == nil {
		return nil, errors.New("student card not verified")
	}

	bank, err := s.userRepo.FindBankByUserId(ownerID)
	if err != nil {
		return nil, err
	}

	if len(bank) == 0 {
		return nil, errors.New("bank account needed. please add your bank account first")
	}

	project := &domain.Project{
		OwnerUserID: ownerID,
		State:       domain.StateDraft,
		Status:      domain.StatusActive,
		Visibility:  domain.VisibilityPrivate,
		Title:       "Untitled Project",
		PlatformFee: 5.0,
	}

	return s.projectRepo.CreateProject(project)
}

func (s *projectService) UpdateProject(projectID uint, input dto.UpdateProjectRequest, user domain.User) (*domain.Project, error) {
	project, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return nil, err
	}

	if project.OwnerUserID != user.ID {
		return nil, errors.New("forbidden")
	}

	// เช็คสถานะ: หากโปรเจกต์ถูกอนุมัติหรือไม่อยู่ใน Draft แล้ว จะไม่อนุญาตให้แก้ไขข้อมูลหลัก
	if project.State != domain.StateDraft {
		return nil, errors.New("cannot update project: only projects in draft state can be edited")
	}

	if input.Title != nil {
		project.Title = *input.Title
	}
	if input.Description != nil {
		project.Description = input.Description
	}
	if input.CategoryID != nil {
		// เช็คว่า Category ที่ระบุมามีอยู่จริงในแอปหรือไม่ ก่อนอัปเดต
		_, err := s.projectRepo.FindCategoryByID(*input.CategoryID)
		if err != nil {
			return nil, errors.New("invalid category id: category does not exist")
		}
		project.CategoryID = input.CategoryID
	}
	if input.Visibility != nil {
		project.Visibility = domain.ProjectVisibility(*input.Visibility)
	}
	if input.FundingGoal != nil {
		project.FundingGoal = *input.FundingGoal
		// บังคับให้ลงทุนขั้นต่ำเป็น 1% ของยอดระดมทุน (Fix)
		project.MinInvestAmount = (*input.FundingGoal) * 0.01
	}
	if input.Softcap != nil {
		minSoftcap := project.FundingGoal * 0.7

		if *input.Softcap < minSoftcap {
			return nil, errors.New("softcap must be at least 70% of funding goal")
		}

		if *input.Softcap > project.FundingGoal {
			return nil, errors.New("softcap must not exceed 100% of the target fundraising")
		}
		project.Softcap = *input.Softcap
	}
	if input.DurationDays != nil {
		project.DurationDays = *input.DurationDays
	}
	if input.DurationMonths != nil {
		if *input.DurationMonths > 48 {
			return nil, errors.New("duration months must be less than 48 months")
		}
		project.DurationMonths = *input.DurationMonths
	}
	if input.DurationDays != nil {
		project.DurationDays = *input.DurationDays
	}
	if input.DurationMonths != nil {
		project.DurationMonths = *input.DurationMonths
	}

	if input.DurationDays != nil {
		if *input.DurationDays <= 0 || *input.DurationDays > 60 {
			return nil, errors.New("fundraising duration must be between 1 and 60 days")
		}

		if project.State == domain.StateFunding && !project.FundingAt.IsZero() {
			// ถ้าอยู่ในสถานะ funding แล้ว ให้ขยับวันจบนับจากวันที่เริ่ม funding (ระดมทุน)
			project.EndDate = project.FundingAt.AddDate(0, 0, project.DurationDays)
		} else if project.State == domain.StateDraft || project.State == domain.StatePendingReview {
			// ล้างค่า EndDate หากยังไม่ได้รับ approve
			project.EndDate = time.Time{}
		}
	}

	if input.DurationMonths != nil {
		if *input.DurationMonths <= 0 {
			return nil, errors.New("project duration must be greater than 0 months")
		}
	}

	if input.ProfitSharePct != nil {
		project.ProfitSharePct = *input.ProfitSharePct
	}
	// ไม่ต้องดึงค่า MinInvestAmount จาก input อีกต่อไป เพราะคำนวณจาก FundingGoal
	if input.MaxInvestAmount != nil {
		project.MaxInvestAmount = *input.MaxInvestAmount
	}
	if input.PlatformFee != nil {
		project.PlatformFee = *input.PlatformFee
	}

	if input.Risk != nil {
		project.Risk = input.Risk
	}

	return s.projectRepo.UpdateProject(project)
}

func (s *projectService) DeleteProject(projectID uint, user domain.User) error {
	project, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return err
	}

	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}

	if project.State != domain.StateDraft {
		return errors.New("cannot delete project: only projects in draft state can be edited")
	}

	return s.projectRepo.DeleteProject(projectID)
}

func (s *projectService) GetMyProjects(ownerID uint) ([]domain.Project, error) {
	projects, err := s.projectRepo.FindProjectsByOwnerID(ownerID)
	if err != nil {
		return nil, errors.New("failed to retrieve projects")
	}
	return projects, nil
}

func (s *projectService) GetPublicProjects() ([]domain.Project, error) {
	status := domain.StatusActive
	visibility := domain.VisibilityPublic
	projects, err := s.projectRepo.FindProjects(nil, &status, &visibility)

	if err != nil {
		return nil, err
	}

	// public projects include both fundraising and post-fundraising execution phases
	filtered := make([]domain.Project, 0, len(projects))
	for _, p := range projects {
		if p.State == domain.StateFunding {
			filtered = append(filtered, p)
		}
	}

	return filtered, nil
}

func (s *projectService) GetProjectDetailByID(id uint) (*domain.Project, error) {
	status := domain.StatusActive
	visibility := domain.VisibilityPublic
	project, err := s.projectRepo.FindProjectDetailByID(id, nil, &status, &visibility)
	if err != nil {
		return nil, errors.New("project not found")
	}
	if project.State != domain.StateFunding && project.State != domain.StateExecuting {
		return nil, errors.New("project not found")
	}
	return project, nil
}

func (s *projectService) GetPublicProjectByID(id uint) (*domain.Project, error) {
	status := domain.StatusActive
	visibility := domain.VisibilityPublic
	project, err := s.projectRepo.FindProjectDetailByID(id, nil, &status, &visibility)
	if err != nil {
		return nil, err
	}

	if project.Visibility != domain.VisibilityPublic {
		return nil, errors.New("not public")
	}
	if project.State != domain.StateFunding && project.State != domain.StateExecuting {
		return nil, errors.New("not public")
	}

	return project, nil
}

func (s *projectService) GetOwnerProjectByID(id uint, ownerID uint) (*domain.Project, error) {
	project, err := s.projectRepo.FindProjectByIDAndOwner(id, ownerID)
	if err != nil {
		return nil, err
	}

	if project.OwnerUserID != ownerID {
		return nil, errors.New("forbidden")
	}

	return project, nil
}

func (s *projectService) GetProjectsByCategory(categoryID uint) ([]domain.Project, error) {
	projects, err := s.projectRepo.FindProjectsByCategory(categoryID)
	if err != nil {
		return nil, errors.New("failed to retrieve projects for this category")
	}
	return projects, nil
}

func (s *projectService) UpdateProjectStatus(projectID uint, newState domain.ProjectState, newStatus domain.ProjectStatus) error {
	project, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return err
	}

	if !helper.IsValidStateTransition(project.State, newState) {
		return errors.New("invalid state transition")
	}

	project.State = newState
	project.Status = newStatus

	_, err = s.projectRepo.UpdateProject(project)
	return err
}

// MEDIA
func (s *projectService) AttachProjectMedia(ctx context.Context, projectID uint, url string, mediaTypes []domain.MediaType, user domain.User) error {
	project, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return err
	}

	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}

	media := &domain.ProjectMedia{
		ProjectID: projectID,
		URL:       url,
		Type:      mediaTypes,
	}

	return s.projectRepo.CreateProjectMedia(media)
}

func (s *projectService) GetProjectMedia(projectID uint) ([]domain.ProjectMedia, error) {
	if _, err := s.projectRepo.FindProjectByID(projectID); err != nil {
		return nil, errors.New("project not found")
	}
	media, err := s.projectRepo.FindMediaByProjectID(projectID)
	if err != nil {
		return nil, errors.New("failed to retrieve media")
	}
	return media, nil
}

func (s *projectService) UpdateProjectMedia(mediaID uint, input *domain.ProjectMedia, user domain.User) error {
	media, err := s.projectRepo.FindMediaByID(mediaID)
	if err != nil {
		return err
	}

	project, err := s.projectRepo.FindProjectByID(media.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}

	if input.URL != "" {
		media.URL = input.URL
	}
	if len(input.Type) > 0 {
		media.Type = input.Type
	}
	if input.SortOrder != 0 {
		media.SortOrder = input.SortOrder
	}

	return s.projectRepo.UpdateProjectMedia(media)
}

func (s *projectService) DeleteProjectMedia(mediaID uint, user domain.User) error {
	media, err := s.projectRepo.FindMediaByID(mediaID)
	if err != nil {
		return err
	}

	project, err := s.projectRepo.FindProjectByID(media.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}

	return s.projectRepo.DeleteProjectMedia(mediaID)
}

// MILESTONE
func (s *projectService) CreateMilestone(projectID uint, input dto.CreateMilestoneRequest, user domain.User) (*domain.Milestone, error) {
	project, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return nil, err
	}
	if project.OwnerUserID != user.ID {
		return nil, errors.New("permission denied")
	}

	existing, err := s.projectRepo.FindMilestonesByProjectID(projectID)
	if err != nil {
		return nil, err
	}

	if len(existing) >= 4 {
		return nil, errors.New("maximum 4 milestones allowed")
	}

	phaseNo := len(existing) + 1
	percent, _ := helper.GetMilestonePercent(phaseNo)

	milestone := &domain.Milestone{
		ProjectID:      projectID,
		PhaseNo:        phaseNo,
		SortOrder:      phaseNo,
		PercentRelease: percent,
		Status:         domain.MilestoneDraft,
		DueDate:        input.DueDate,
	}

	return milestone, s.projectRepo.CreateMilestone(milestone)
}

func (s *projectService) UpdateMilestone(milestoneID uint, input dto.UpdateMilestoneRequest, user domain.User) error {
	m, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil {
		return err
	}

	project, err := s.projectRepo.FindProjectByID(m.ProjectID)
	if err != nil {
		return err
	}

	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}

	if input.Title != nil {
		m.Title = *input.Title
	}

	if input.Description != nil {
		m.Description = input.Description
	}

	if input.PhaseNo != nil {
		if *input.PhaseNo < 1 || *input.PhaseNo > 4 {
			return errors.New("phase must be between 1 and 4")
		}

		m.PhaseNo = *input.PhaseNo

		percent, err := helper.GetMilestonePercent(*input.PhaseNo)
		if err != nil {
			return err
		}
		m.PercentRelease = percent
	}

	if input.Duration != nil {
		m.Duration = input.Duration
	}

	if input.DueDate != nil {
		m.DueDate = input.DueDate
	}

	if input.AcceptanceCriteria != nil {
		m.AcceptanceCriteria = input.AcceptanceCriteria
	}

	if input.URLs != nil {
		if len(input.URLs) == 0 {
			m.URLs = []string{}
		} else {
			var validURLs []string
			for _, rawURL := range input.URLs {
				raw := strings.TrimSpace(rawURL)
				if raw == "" {
					continue
				}
				u, err := url.Parse(raw)
				if err != nil {
					return errors.New("invalid url")
				}

				// ป้องกัน fake URL (รองรับแต่ cloudinary)
				if !strings.Contains(u.Host, "res.cloudinary.com") {
					return errors.New("invalid file source")
				}

				validURLs = append(validURLs, raw)
			}
			m.URLs = validURLs
		}
	}

	if len(input.Type) > 0 {
		for _, t := range input.Type {
			if t != domain.MediaTypeRaw && t != domain.MediaTypeImage && t != domain.MediaTypeVideo {
				return errors.New("unsupported media type: " + string(t))
			}
		}
		m.Type = input.Type
	}

	if input.SortOrder != nil {
		m.SortOrder = *input.SortOrder
	}

	if input.Status != nil {
		validStatuses := map[domain.MilestoneStatus]bool{
			domain.MilestoneDraft:     true,
			domain.MilestoneWaiting:   true,
			domain.MilestoneActive:    true,
			domain.MilestoneSubmitted: true,
			domain.MilestoneApproved:  true,
			domain.MilestoneRejected:  true,
			domain.MilestoneFailed:    true,
			domain.MilestonePaid:      true,
		}

		if !validStatuses[*input.Status] {
			return errors.New("invalid milestone status")
		}

		m.Status = *input.Status
	}

	return s.projectRepo.UpdateMilestone(m)
}

func (s *projectService) AdminApproveMilestoneSubmission(milestoneID uint) (*domain.Milestone, error) {
	m, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil {
		return nil, errors.New("milestone not found")
	}
	if m.Status != domain.MilestoneSubmitted {
		return nil, errors.New("milestone is not submitted")
	}

	// step 1: admin approves submission -> approved (vote can be opened by pioneer)
	m.Status = domain.MilestoneApproved
	m.VotingOpen = false
	m.VotingOpenedAt = nil
	m.VotingClosedAt = nil
	if err := s.projectRepo.UpdateMilestone(m); err != nil {
		return nil, err
	}

	return m, nil
}

func (s *projectService) AdminRejectMilestoneSubmission(milestoneID uint, reason *string) (*domain.Milestone, error) {
	m, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil {
		return nil, errors.New("milestone not found")
	}
	if m.Status != domain.MilestoneSubmitted {
		return nil, errors.New("milestone is not submitted")
	}

	// store reason in submission summary tail if provided (minimal change without new column)
	if reason != nil {
		r := strings.TrimSpace(*reason)
		if r != "" {
			if m.SubmissionSummary == nil {
				m.SubmissionSummary = &r
			} else {
				merged := strings.TrimSpace(*m.SubmissionSummary) + "\n\nReject reason: " + r
				m.SubmissionSummary = &merged
			}
		}
	}

	m.Status = domain.MilestoneRejected
	m.VotingOpen = false
	now := time.Now().UTC()
	m.VotingClosedAt = &now
	if err := s.projectRepo.UpdateMilestone(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *projectService) OpenMilestoneVoting(milestoneID uint, user domain.User) (*domain.Milestone, error) {
	m, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil {
		return nil, errors.New("milestone not found")
	}
	project, err := s.projectRepo.FindProjectByID(m.ProjectID)
	if err != nil {
		return nil, err
	}
	if project.OwnerUserID != user.ID {
		return nil, errors.New("permission denied")
	}
	if m.Status != domain.MilestoneApproved {
		return nil, errors.New("milestone is not approved for voting")
	}
	if m.VotingOpen {
		return nil, errors.New("voting is already open")
	}

	now := time.Now().UTC()
	m.VotingOpen = true
	m.VotingOpenedAt = &now
	m.VotingClosedAt = nil
	if err := s.projectRepo.UpdateMilestone(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (s *projectService) GetSubmittedMilestonesForAdmin(projectID *uint) ([]dto.AdminMilestoneListResponse, error) {
	var milestones []domain.Milestone
	var err error
	if projectID != nil {
		milestones, err = s.projectRepo.FindMilestonesByProjectIDAndStatus(*projectID, domain.MilestoneSubmitted)
	} else {
		milestones, err = s.projectRepo.FindMilestonesByStatus(domain.MilestoneSubmitted)
	}
	if err != nil {
		return nil, err
	}

	var response []dto.AdminMilestoneListResponse
	for _, m := range milestones {
		p, err := s.projectRepo.FindProjectByID(m.ProjectID)
		if err != nil {
			continue // skip if project deleted
		}
		var ownerProfile *dto.ProjectOwnerProfile
		ownerUser, err := s.userRepo.FindUserById(p.OwnerUserID)
		if err == nil {
			ownerProfile = &dto.ProjectOwnerProfile{
				FirstName: ownerUser.FirstName,
				LastName:  ownerUser.LastName,
			}
			if ownerUser.StudentProfile != nil {
				if ownerUser.StudentProfile.University != nil && ownerUser.StudentProfile.University.NameTH != nil {
					ownerProfile.University = *ownerUser.StudentProfile.University.NameTH
				}
				ownerProfile.Faculty = ownerUser.StudentProfile.Faculty
				ownerProfile.Major = ownerUser.StudentProfile.Major
			}
		}

		response = append(response, dto.AdminMilestoneListResponse{
			Milestone:    m,
			ProjectTitle: p.Title,
			Owner:        ownerProfile,
		})
	}
	return response, nil
}

func (s *projectService) GetAdminMilestoneDetail(milestoneID uint) (*dto.AdminMilestoneDetailResponse, error) {
	m, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil {
		return nil, errors.New("milestone not found")
	}

	p, err := s.projectRepo.FindProjectByID(m.ProjectID)
	if err != nil {
		return nil, errors.New("project not found")
	}

	var ownerProfile *dto.ProjectOwnerProfile
	ownerUser, err := s.userRepo.FindUserById(p.OwnerUserID)
	if err == nil {
		ownerProfile = &dto.ProjectOwnerProfile{
			FirstName: ownerUser.FirstName,
			LastName:  ownerUser.LastName,
		}
		if ownerUser.StudentProfile != nil {
			if ownerUser.StudentProfile.University != nil && ownerUser.StudentProfile.University.NameTH != nil {
				ownerProfile.University = *ownerUser.StudentProfile.University.NameTH
			}
			ownerProfile.Faculty = ownerUser.StudentProfile.Faculty
			ownerProfile.Major = ownerUser.StudentProfile.Major
		}
	}

	// Calculate Funding Goal
	fundingGoal := float64(0)
	if m.PercentRelease > 0 {
		fundingGoal = (p.FundingGoal * float64(m.PercentRelease)) / 100.0
	}

	// Map EvidenceFiles
	var evidenceFiles []dto.EvidenceFile
	for i, attachment := range m.SubmissionAttachments {
		evidenceFiles = append(evidenceFiles, dto.EvidenceFile{
			ID:       fmt.Sprintf("file-%d", i),
			URL:      attachment,
			FileName: fmt.Sprintf("Attachment %d", i+1),
		})
	}

	// Map EvidenceLinks
	var evidenceLinks []dto.EvidenceLink
	for i, link := range m.SubmissionLinks {
		evidenceLinks = append(evidenceLinks, dto.EvidenceLink{
			Name: fmt.Sprintf("Link %d", i+1),
			URL:  link,
		})
	}

	// Checked Criteria
	var checkedCriteria []bool
	if m.AcceptanceCriteria != nil && *m.AcceptanceCriteria != "" {
		lines := strings.Split(*m.AcceptanceCriteria, "\n")
		// Clean lines
		var criteriaList []string
		for _, l := range lines {
			t := strings.TrimSpace(l)
			// Remove hyphen or numbering if you want, or just check literally:
			if t != "" {
				criteriaList = append(criteriaList, t)
			}
		}

		for _, req := range criteriaList {
			matched := false
			for _, sub := range m.SubmissionCriteria {
				if strings.Contains(strings.ToLower(sub), strings.ToLower(req)) || req == sub {
					matched = true
					break
				}
			}
			checkedCriteria = append(checkedCriteria, matched)
		}
	}

	return &dto.AdminMilestoneDetailResponse{
		Milestone:       *m,
		ProjectTitle:    p.Title,
		Owner:           ownerProfile,
		EvidenceFiles:   evidenceFiles,
		EvidenceLinks:   evidenceLinks,
		CheckedCriteria: checkedCriteria,
		FundingGoal:     fundingGoal,
		EndDate:         m.DueDate,
		ProgressPct:     m.PercentRelease,
	}, nil
}

func (s *projectService) DeleteMilestone(milestoneID uint, user domain.User) error {
	m, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil {
		return errors.New("milestone not found")
	}
	project, err := s.projectRepo.FindProjectByID(m.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}
	return s.projectRepo.DeleteMilestone(milestoneID)
}

func (s *projectService) GetProjectMilestones(projectID uint) ([]domain.Milestone, error) {
	if _, err := s.projectRepo.FindProjectByID(projectID); err != nil {
		return nil, errors.New("project not found")
	}
	milestones, err := s.projectRepo.FindMilestonesByProjectID(projectID)
	if err != nil {
		return nil, errors.New("failed to retrieve milestones")
	}
	return milestones, nil
}

func (s *projectService) SubmitMilestone(milestoneID uint, input dto.SubmitMilestoneRequest, user domain.User) (*domain.Milestone, error) {
	m, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil {
		return nil, errors.New("milestone not found")
	}

	project, err := s.projectRepo.FindProjectByID(m.ProjectID)
	if err != nil {
		return nil, err
	}
	if project.OwnerUserID != user.ID {
		return nil, errors.New("permission denied")
	}

	// only allow submit when active (and allow resubmission when rejected)
	switch m.Status {
	case domain.MilestoneActive, domain.MilestoneRejected:
		// ok
	default:
		return nil, errors.New("milestone cannot be submitted in current status")
	}

	// sequential guard: phase > 1 requires previous phase to be paid
	if m.PhaseNo > 1 {
		list, err := s.projectRepo.FindMilestonesByProjectID(m.ProjectID)
		if err != nil {
			return nil, err
		}
		prevPhase := m.PhaseNo - 1
		foundPrev := false
		for i := range list {
			if list[i].PhaseNo == prevPhase {
				foundPrev = true
				if list[i].Status != domain.MilestonePaid {
					return nil, errors.New("previous milestone must be paid before submitting this phase")
				}
				break
			}
		}
		if !foundPrev {
			return nil, errors.New("previous milestone not found")
		}
	}

	summary := strings.TrimSpace(input.Summary)
	if summary == "" {
		return nil, errors.New("summary is required")
	}

	// validate attachments: must be cloudinary URLs (same rule as existing milestone media)
	attachments := make([]string, 0, len(input.Attachments))
	for _, rawURL := range input.Attachments {
		raw := strings.TrimSpace(rawURL)
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil {
			return nil, errors.New("invalid attachment url")
		}
		if !strings.Contains(u.Host, "res.cloudinary.com") {
			return nil, errors.New("invalid attachment source")
		}
		attachments = append(attachments, raw)
	}

	// validate external links: require http/https
	links := make([]string, 0, len(input.Links))
	for _, rawURL := range input.Links {
		raw := strings.TrimSpace(rawURL)
		if raw == "" {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil {
			return nil, errors.New("invalid link url")
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, errors.New("invalid link scheme")
		}
		if strings.TrimSpace(u.Host) == "" {
			return nil, errors.New("invalid link host")
		}
		links = append(links, raw)
	}

	criteria := make([]string, 0, len(input.Criteria))
	for _, c := range input.Criteria {
		v := strings.TrimSpace(c)
		if v == "" {
			continue
		}
		criteria = append(criteria, v)
	}

	now := time.Now().UTC()
	m.SubmissionSummary = &summary
	m.SubmissionCriteria = criteria
	m.SubmissionAttachments = attachments
	m.SubmissionLinks = links
	m.SubmittedAt = &now
	m.Status = domain.MilestoneSubmitted

	if err := s.projectRepo.UpdateMilestone(m); err != nil {
		return nil, err
	}
	return m, nil
}

// PROJECT UPDATE
func (s *projectService) CreateProjectUpdate(projectID uint, input dto.CreateProjectUpdateRequest, user domain.User) error {
	project, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}

	update := &domain.ProjectUpdate{
		ProjectID:  projectID,
		PostedBy:   user.ID,
		Title:      input.Title,
		Body:       input.Content,
		Visibility: domain.VisibilityPublic,
	}
	return s.projectRepo.CreateProjectUpdate(update)
}

func (s *projectService) UpdateProjectUpdate(updateID uint, input dto.UpdateProjectUpdateRequest, user domain.User) error {
	update, err := s.projectRepo.FindProjectUpdateByID(updateID)
	if err != nil {
		return errors.New("update not found")
	}
	project, err := s.projectRepo.FindProjectByID(update.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}

	if input.Title != nil {
		update.Title = *input.Title
	}
	if input.Content != nil {
		update.Body = *input.Content
	}
	return s.projectRepo.UpdateProjectUpdate(update)
}

func (s *projectService) DeleteProjectUpdate(updateID uint, user domain.User) error {
	update, err := s.projectRepo.FindProjectUpdateByID(updateID)
	if err != nil {
		return errors.New("update not found")
	}
	project, err := s.projectRepo.FindProjectByID(update.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}

	return s.projectRepo.DeleteProjectUpdate(updateID)
}

func (s *projectService) GetProjectUpdates(projectID uint) ([]domain.ProjectUpdate, error) {
	if _, err := s.projectRepo.FindProjectByID(projectID); err != nil {
		return nil, errors.New("project not found")
	}
	updates, err := s.projectRepo.FindUpdatesByProjectID(projectID)
	if err != nil {
		return nil, errors.New("failed to retrieve project updates")
	}
	return updates, nil
}

// STORY SECTION
func (s *projectService) GetProjectStories(projectID uint) ([]domain.StorySection, error) {
	if _, err := s.projectRepo.FindProjectByID(projectID); err != nil {
		return nil, errors.New("project not found")
	}
	stories, err := s.projectRepo.FindStoriesByProjectID(projectID)
	if err != nil {
		return nil, errors.New("failed to retrieve stories")
	}
	return stories, nil
}

func (s *projectService) CreateStorySection(section *domain.StorySection, user domain.User) error {
	project, err := s.projectRepo.FindProjectByID(section.ProjectID)
	if err != nil {
		return errors.New("project not found")
	}
	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}
	// เช็คสถานะ
	if project.State != domain.StateDraft {
		return errors.New("cannot edit stories unless project is in draft state")
	}
	return s.projectRepo.CreateStorySection(section)
}

func (s *projectService) UpdateStorySection(section *domain.StorySection, user domain.User) error {
	existing, err := s.projectRepo.FindStorySectionByID(section.ID)
	if err != nil {
		return errors.New("story not found")
	}
	project, err := s.projectRepo.FindProjectByID(existing.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}
	if project.State != domain.StateDraft {
		return errors.New("cannot edit stories unless project is in draft state")
	}
	return s.projectRepo.UpdateStorySection(section)
}

func (s *projectService) DeleteStorySection(sectionID uint, user domain.User) error {
	existing, err := s.projectRepo.FindStorySectionByID(sectionID)
	if err != nil {
		return errors.New("story not found")
	}
	project, err := s.projectRepo.FindProjectByID(existing.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}
	if project.State != domain.StateDraft {
		return errors.New("cannot delete stories unless project is in draft state")
	}
	return s.projectRepo.DeleteStorySection(sectionID)
}

// CATEGORY
func (s *projectService) GetCategoryByID(id uint) (*domain.ProjectCategory, error) {
	category, err := s.projectRepo.FindCategoryByID(id)
	if err != nil {
		return nil, errors.New("category does not exist")
	}
	return category, nil
}

func (s *projectService) GetAllCategories() ([]domain.ProjectCategory, error) {
	categories, err := s.projectRepo.FindAllCategories()
	if err != nil {
		return nil, errors.New("categories do not exist or failed to retrieve")
	}
	return categories, nil
}

func (s *projectService) CreateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error) {
	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" {
		return nil, errors.New("category name is required")
	}
	return s.projectRepo.CreateCategory(category)
}

func (s *projectService) UpdateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error) {
	if _, err := s.projectRepo.FindCategoryByID(category.ID); err != nil {
		return nil, errors.New("category not found")
	}
	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" {
		return nil, errors.New("category name is required")
	}
	return s.projectRepo.UpdateCategory(category)
}

func (s *projectService) DeleteCategory(categoryID uint) error {
	if _, err := s.projectRepo.FindCategoryByID(categoryID); err != nil {
		return errors.New("category not found")
	}
	count, err := s.projectRepo.CountProjectsByCategoryID(categoryID)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("cannot delete a category that is currently assigned to one or more projects")
	}
	return s.projectRepo.DeleteCategory(categoryID)
}

// FAQ
func (s *projectService) GetProjectFAQs(projectID uint) ([]domain.ProjectFAQ, error) {
	if _, err := s.projectRepo.FindProjectByID(projectID); err != nil {
		return nil, errors.New("project not found")
	}
	faqs, err := s.projectRepo.FindFAQsByProjectID(projectID)
	if err != nil {
		return nil, errors.New("failed to retrieve FAQs")
	}
	return faqs, nil
}

func (s *projectService) CreateProjectFAQ(faq *domain.ProjectFAQ, user domain.User) error {
	project, err := s.projectRepo.FindProjectByID(faq.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("forbidden")
	}
	if project.State != domain.StateDraft {
		return errors.New("cannot edit FAQs unless project is in draft state")
	}
	return s.projectRepo.CreateFAQ(faq)
}

func (s *projectService) UpdateProjectFAQ(faq *domain.ProjectFAQ, user domain.User) error {
	existing, err := s.projectRepo.FindFAQByID(faq.ID)
	if err != nil {
		return errors.New("faq not found")
	}
	project, err := s.projectRepo.FindProjectByID(existing.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("forbidden")
	}
	if project.State != domain.StateDraft {
		return errors.New("cannot edit FAQs unless project is in draft state")
	}
	return s.projectRepo.UpdateFAQ(faq)
}

func (s *projectService) DeleteProjectFAQ(faqID uint, user domain.User) error {
	existing, err := s.projectRepo.FindFAQByID(faqID)
	if err != nil {
		return errors.New("faq not found")
	}
	project, err := s.projectRepo.FindProjectByID(existing.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("forbidden")
	}
	if project.State != domain.StateDraft {
		return errors.New("cannot delete FAQs unless project is in draft state")
	}
	return s.projectRepo.DeleteFAQ(faqID)
}

func (s *projectService) GetProjectThreads(projectID uint) ([]domain.ProjectThread, error) {
	if _, err := s.projectRepo.FindProjectByID(projectID); err != nil {
		return nil, errors.New("project not found")
	}
	threads, err := s.projectRepo.FindThreadsByProjectID(projectID)
	if err != nil {
		return nil, errors.New("failed to retrieve threads")
	}
	return threads, nil
}

func (s *projectService) CreateProjectThread(thread *domain.ProjectThread, user domain.User) error {
	project, err := s.projectRepo.FindProjectByID(thread.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("forbidden")
	}
	return s.projectRepo.CreateThread(thread)
}

func (s *projectService) UpdateProjectThread(thread *domain.ProjectThread, user domain.User) error {
	existing, err := s.projectRepo.FindThreadByID(thread.ID)
	if err != nil {
		return errors.New("thread not found")
	}
	project, err := s.projectRepo.FindProjectByID(existing.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("forbidden")
	}
	return s.projectRepo.UpdateThread(thread)
}

func (s *projectService) DeleteProjectThread(threadID uint, user domain.User) error {
	existing, err := s.projectRepo.FindThreadByID(threadID)
	if err != nil {
		return errors.New("thread not found")
	}
	project, err := s.projectRepo.FindProjectByID(existing.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("forbidden")
	}
	return s.projectRepo.DeleteThread(threadID)
}

func (s *projectService) GetProjectThreadMessages(threadID uint) ([]domain.ProjectThreadMessage, error) {
	if _, err := s.projectRepo.FindThreadByID(threadID); err != nil {
		return nil, errors.New("thread not found")
	}
	msgs, err := s.projectRepo.FindMessagesByThreadID(threadID)
	if err != nil {
		return nil, errors.New("failed to retrieve messages")
	}
	return msgs, nil
}

func (s *projectService) CreateProjectThreadMessage(msg *domain.ProjectThreadMessage, user domain.User) error {
	thread, err := s.projectRepo.FindThreadByID(msg.ThreadID)
	if err != nil {
		return errors.New("thread not found")
	}
	project, err := s.projectRepo.FindProjectByID(thread.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("forbidden")
	}
	return s.projectRepo.CreateThreadMessage(msg)
}

func (s *projectService) UpdateProjectThreadMessage(msg *domain.ProjectThreadMessage, user domain.User) error {
	existing, err := s.projectRepo.FindThreadMessageByID(msg.ID)
	if err != nil {
		return errors.New("message not found")
	}
	thread, err := s.projectRepo.FindThreadByID(existing.ThreadID)
	if err != nil {
		return err
	}
	project, err := s.projectRepo.FindProjectByID(thread.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("forbidden")
	}
	return s.projectRepo.UpdateThreadMessage(msg)
}

func (s *projectService) DeleteProjectThreadMessage(msgID uint, user domain.User) error {
	existing, err := s.projectRepo.FindThreadMessageByID(msgID)
	if err != nil {
		return errors.New("message not found")
	}
	thread, err := s.projectRepo.FindThreadByID(existing.ThreadID)
	if err != nil {
		return err
	}
	project, err := s.projectRepo.FindProjectByID(thread.ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerUserID != user.ID {
		return errors.New("forbidden")
	}
	return s.projectRepo.DeleteThreadMessage(msgID)
}

// PROJECT LIFECYCLE
func (s *projectService) validateProjectForSubmit(project *domain.Project) error {
	if project.Title == "" {
		return errors.New("project title is required")
	}
	if project.Description == nil || *project.Description == "" {
		return errors.New("project description is required")
	}
	if project.CategoryID == nil {
		return errors.New("project category is required")
	}
	if project.FundingGoal <= 0 {
		return errors.New("funding goal is required")
	}
	if project.Softcap <= 0 {
		return errors.New("softcap is required")
	}
	if project.Softcap > project.FundingGoal {
		return errors.New("softcap cannot be greater than funding goal")
	}
	if project.DurationDays <= 0 {
		return errors.New("fundraising duration days is required and must be greater than 0")
	}
	if project.DurationMonths <= 0 {
		return errors.New("project duration months is required and must be greater than 0")
	}
	media, err := s.projectRepo.FindMediaByProjectID(project.ID)
	if err != nil {
		return err
	}
	if len(media) == 0 {
		return errors.New("at least one media required")
	}
	milestones, err := s.projectRepo.FindMilestonesByProjectID(project.ID)
	if err != nil {
		return err
	}
	if len(milestones) == 0 {
		return errors.New("at least one milestone required")
	}
	if len(milestones) > 4 {
		return errors.New("maximum 4 milestones allowed")
	}
	stories, err := s.projectRepo.FindStoriesByProjectID(project.ID)
	if err != nil {
		return err
	}
	if len(stories) == 0 {
		return errors.New("at least one story section required")
	}
	if project.Risk == nil || *project.Risk == "" {
		return errors.New("project risk information is required")
	}
	return nil
}

func (s *projectService) SubmitForReview(projectID uint, user domain.User) error {
	p, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return err
	}
	if p.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}
	// เช็คสถานะ: โครงการต้องอยู่ใน Draft ถึงจะ submit ได้
	if p.State != domain.StateDraft {
		return errors.New("project must be in draft state to submit for review")
	}
	if err := s.validateProjectForSubmit(p); err != nil {
		return err
	}
	p.State = domain.StatePendingReview
	_, err = s.projectRepo.UpdateProject(p)
	if err != nil {
		return err
	}

	// notify admin ทุกคนว่ามีโปรเจกต์รอ review
	if s.notifSvc != nil {
		admins, err := s.userRepo.FindAllByRole("admin")
		if err == nil {
			relatedID := p.ID
			relatedType := "project"
			for _, admin := range admins {
				body := fmt.Sprintf("โปรเจกต์ \"%s\" รอการอนุมัติ", p.Title)
				s.notifSvc.CreateAndPush(admin.ID, domain.NotifProjectStatus, "โปรเจกต์ใหม่รอการอนุมัติ", body, &relatedID, &relatedType)
			}
		}
	}
	return nil
}

func (s *projectService) ApproveProject(projectID uint) error {
	p, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return err
	}

	if p.State == domain.StateFunding {
		return errors.New("project is in funding state")
	}
	// เช็คสถานะ: โครงการต้องอยู่ใน pending_review ถึงจะอนุมัติได้
	if p.State != domain.StatePendingReview {
		return errors.New("project must be in pending review state to be approved")
	}
	// re-validate
	if err := s.validateProjectForSubmit(p); err != nil {
		return err
	}
	p.FundingAt = time.Now().UTC()
	if p.DurationDays > 0 {
		p.EndDate = p.FundingAt.AddDate(0, 0, p.DurationDays)
	}
	p.State = domain.StateFunding
	p.Status = domain.StatusActive
	p.Visibility = domain.VisibilityPublic
	_, err = s.projectRepo.UpdateProject(p)
	if err != nil {
		return err
	}

	// เมื่อแอดมิน approve โปรเจกต์: เปลี่ยน milestone จาก draft -> waiting
	milestones, err := s.projectRepo.FindMilestonesByProjectID(p.ID)
	if err == nil {
		for i := range milestones {
			if milestones[i].Status == domain.MilestoneDraft {
				milestones[i].Status = domain.MilestoneWaiting
				_ = s.projectRepo.UpdateMilestone(&milestones[i])
			}
		}
	}

	// notify pioneer ว่าโปรเจกต์ได้รับการอนุมัติ
	if s.notifSvc != nil {
		relatedID := p.ID
		relatedType := "project"
		body := fmt.Sprintf("โปรเจกต์ \"%s\" ได้รับการอนุมัติและเริ่มระดมทุนแล้ว", p.Title)
		s.notifSvc.CreateAndPush(p.OwnerUserID, domain.NotifProjectStatus, "โปรเจกต์ได้รับการอนุมัติ", body, &relatedID, &relatedType)
	}
	return nil
}

func (s *projectService) RejectProject(projectID uint) error {
	p, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return err
	}
	// เช็คสถานะ: โครงการต้องอยู่ใน pending_review ถึงจะปฏิเสธได้
	if p.State != domain.StatePendingReview {
		return errors.New("project must be in pending review state to be rejected")
	}
	// กลับสู่สถานะ Draft ให้ไปแก้ไขใหม่ และตั้ง Status เป็น Rejected
	p.State = domain.StateDraft
	p.Status = domain.StatusRejected
	_, err = s.projectRepo.UpdateProject(p)
	if err != nil {
		return err
	}
	// notify pioneer ว่าโปรเจกต์ถูกปฏิเสธ
	if s.notifSvc != nil {
		relatedID := p.ID
		relatedType := "project"
		body := fmt.Sprintf("โปรเจกต์ \"%s\" ถูกปฏิเสธ กรุณาแก้ไขและส่งใหม่อีกครั้ง", p.Title)
		s.notifSvc.CreateAndPush(p.OwnerUserID, domain.NotifProjectStatus, "โปรเจกต์ถูกปฏิเสธ", body, &relatedID, &relatedType)
	}
	return nil
}

func (s *projectService) CloseProject(projectID uint, user domain.User) error {
	p, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return err
	}

	if p.OwnerUserID != user.ID {
		return errors.New("permission denied")
	}

	if p.State != domain.StateFunding && p.State != domain.StateExecuting {
		return errors.New("project must be in funding or executing state")
	}

	now := time.Now()

	// CASE 0: ถ้าอยู่ executing แล้วให้ปิดได้เมื่อ milestone จบแล้วทั้งหมด
	if p.State == domain.StateExecuting {
		milestones, err := s.projectRepo.FindMilestonesByProjectID(p.ID)
		if err != nil {
			return err
		}
		if len(milestones) == 0 {
			return errors.New("cannot close project: milestones are missing")
		}
		for _, m := range milestones {
			if m.Status != domain.MilestonePaid {
				return errors.New("cannot close project: all milestones must be paid")
			}
		}
		p.State = domain.StateClosed
		p.Status = domain.StatusCompleted
		_, err = s.projectRepo.UpdateProject(p)
		return err
	}

	// CASE 1: เงินเต็ม → เข้าสู่ execution
	if p.CurrentFunding >= p.FundingGoal {
		startExecutionAt := time.Now().UTC()
		p.State = domain.StateExecuting
		p.Status = domain.StatusActive
		if p.DurationMonths > 0 {
			executionEnd := startExecutionAt.AddDate(0, p.DurationMonths, 0)
			p.ExecutionEndAt = &executionEnd
		}
		_, err = s.projectRepo.UpdateProject(p)
		if err != nil {
			return err
		}
		// ระดมทุนสำเร็จ: เปลี่ยน milestone จาก waiting -> active (เริ่มที่ phase 1)
		milestones, err := s.projectRepo.FindMilestonesByProjectID(p.ID)
		if err == nil {
			for i := range milestones {
				if milestones[i].Status == domain.MilestoneWaiting && milestones[i].PhaseNo == 1 {
					milestones[i].Status = domain.MilestoneActive
					_ = s.projectRepo.UpdateMilestone(&milestones[i])
					break
				}
			}
		}
		return nil
	}

	// CASE 2: ยังไม่หมดเวลา funding → ห้ามปิด
	if now.Before(p.EndDate) {
		return errors.New("cannot close project before end date unless funding goal is reached")
	}

	// CASE 3: หมดเวลา funding แล้ว → ตัดสินผล
	if p.CurrentFunding >= p.Softcap {
		startExecutionAt := time.Now().UTC()
		p.State = domain.StateExecuting
		p.Status = domain.StatusActive
		if p.DurationMonths > 0 {
			executionEnd := startExecutionAt.AddDate(0, p.DurationMonths, 0)
			p.ExecutionEndAt = &executionEnd
		}
	} else {
		p.State = domain.StateClosed
		p.Status = domain.StatusFailed
	}

	_, err = s.projectRepo.UpdateProject(p)
	if err != nil {
		return err
	}

	// ระดมทุนสำเร็จ (ถึง softcap): เปลี่ยน milestone จาก waiting -> active (เริ่มที่ phase 1)
	if p.State == domain.StateExecuting {
		milestones, err := s.projectRepo.FindMilestonesByProjectID(p.ID)
		if err == nil {
			for i := range milestones {
				if milestones[i].Status == domain.MilestoneWaiting && milestones[i].PhaseNo == 1 {
					milestones[i].Status = domain.MilestoneActive
					_ = s.projectRepo.UpdateMilestone(&milestones[i])
					break
				}
			}
		}
	}

	return nil
}

func (s *projectService) CancelProject(projectID uint, user domain.User) error {
	project, err := s.projectRepo.FindProjectByID(projectID)

	if err != nil {
		return err
	}

	if project.OwnerUserID != user.ID {
		return errors.New("you are not authorized to cancel this project")
	}

	if project.State == domain.StateCancelled || project.State == domain.StateDraft {
		return errors.New("project is already cancelled or state is draft")
	}

	if project.CurrentFunding > 0 {
		return errors.New("cannot cancel project with existing investments")
	}

	// update state
	project.State = domain.StateDraft

	project.Status = domain.StatusActive

	project.Visibility = domain.VisibilityPrivate

	_, err = s.projectRepo.UpdateProject(project)
	if err != nil {
		return err
	}

	return nil

}

func (s *projectService) GetAllProjectsRequest() ([]domain.Project, error) {
	state := domain.StatePendingReview
	return s.projectRepo.FindProjectsState(string(state))
}

func (s *projectService) GetProjectDetailRequest(projectID uint) (*domain.Project, error) {
	if projectID == 0 {
		return nil, errors.New("invalid project ID")
	}

	project, err := s.projectRepo.FindProjectsPendingDetail(
		projectID,
		string(domain.StatePendingReview),
	)
	if err != nil {
		return nil, err
	}

	if project == nil {
		return nil, errors.New("project not found")
	}

	return project, nil
}

func (s *projectService) AutoProjectLifecycleTick(now time.Time) error {
	state := domain.StateFunding
	status := domain.StatusActive
	projects, err := s.projectRepo.FindProjects(&state, &status, nil)
	if err != nil {
		return err
	}

	for i := range projects {
		p := &projects[i]

		// 1. Success condition: Reached Hardcap OR (time expired AND reached Softcap)
		isHardcapReached := p.CurrentFunding >= p.FundingGoal
		isExpiredAndSoftcap := !p.EndDate.IsZero() && !now.Before(p.EndDate) && p.CurrentFunding >= p.Softcap

		if isHardcapReached || isExpiredAndSoftcap {
			startExecutionAt := time.Now().UTC()
			p.State = domain.StateExecuting
			p.Status = domain.StatusActive
			if p.DurationMonths > 0 {
				executionEnd := startExecutionAt.AddDate(0, p.DurationMonths, 0)
				p.ExecutionEndAt = &executionEnd
			}

			if _, err := s.projectRepo.UpdateProject(p); err != nil {
				return err
			}

			// เริ่ม Milestone 1 (เปลี่ยนจาก waiting -> active)
			milestones, err := s.projectRepo.FindMilestonesByProjectID(p.ID)
			if err == nil {
				for j := range milestones {
					if milestones[j].Status == domain.MilestoneWaiting && milestones[j].PhaseNo == 1 {
						milestones[j].Status = domain.MilestoneActive
						_ = s.projectRepo.UpdateMilestone(&milestones[j])
						break
					}
				}
			}
			continue
		}

		// 2. Failure condition: Time expired and < Softcap
		if p.EndDate.IsZero() {
			continue
		}
		if now.Before(p.EndDate) {
			continue
		}

		// User requirement: if funding time expired and < softcap -> draft + failed
		p.State = domain.StateDraft
		p.Status = domain.StatusFailed
		if _, err := s.projectRepo.UpdateProject(p); err != nil {
			return err
		}
	}

	// execution timeout: when execution period ends, close project automatically.
	executing := domain.StateExecuting
	executingProjects, err := s.projectRepo.FindProjects(&executing, &status, nil)
	if err != nil {
		return err
	}
	for i := range executingProjects {
		p := &executingProjects[i]

		// milestone due date timeout:
		// if an active milestone is overdue and not submitted -> mark milestone failed and suspend project.
		milestones, err := s.projectRepo.FindMilestonesByProjectID(p.ID)
		if err != nil {
			return err
		}
		overdue := false
		for j := range milestones {
			m := &milestones[j]
			if m.Status != domain.MilestoneActive {
				continue
			}
			if m.DueDate == nil {
				continue
			}
			if now.Before((*m.DueDate).UTC()) {
				continue
			}

			// overdue while still active => fail
			m.Status = domain.MilestoneFailed
			m.VotingOpen = false
			m.VotingOpenedAt = nil
			m.VotingClosedAt = nil
			_ = s.projectRepo.UpdateMilestone(m)
			overdue = true
			break
		}
		if overdue {
			p.State = domain.StateClosed
			p.Status = domain.StatusFailed
			if _, err := s.projectRepo.UpdateProject(p); err != nil {
				return err
			}
			continue
		}

		if p.ExecutionEndAt == nil {
			continue
		}
		if now.Before(*p.ExecutionEndAt) {
			continue
		}
		allPaid := len(milestones) > 0
		for _, m := range milestones {
			if m.Status != domain.MilestonePaid {
				allPaid = false
				break
			}
		}

		p.State = domain.StateClosed
		if allPaid {
			p.Status = domain.StatusCompleted
		} else {
			p.Status = domain.StatusFailed
		}

		if _, err := s.projectRepo.UpdateProject(p); err != nil {
			return err
		}
	}

	return nil
}

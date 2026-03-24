package service

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
)

type ProjectService interface {
	CreateProject(ownerID uint) (*domain.Project, error)
	UpdateProjectDraft(projectID uint, ownerID uint, input dto.UpdateProjectDraftRequest) error
	AddProjectMedia(projectID uint, ownerID uint, input dto.AddProjectMediaRequest) error
	GetProjectsByOwnerResponse(ownerID uint) ([]dto.ProjectResponse, error)
	GetProjectByID(id uint, ownerID uint) (*domain.Project, error)
	AddProjectStory(projectID uint, ownerID uint, input dto.AddProjectStoryRequest) error
	AddProjectRisk(projectID uint, ownerID uint, input dto.AddProjectRiskRequest) error
	AddProjectFAQ(projectID uint, ownerID uint, input dto.AddProjectFAQRequest) error
	SetFundingPolicy(projectID uint, ownerID uint, input dto.SetFundingPolicyRequest) error
	SetProfitPolicy(projectID uint, ownerID uint, input dto.SetProfitPolicyRequest) error
	CreateMilestone(projectID uint, ownerID uint, input dto.CreateMilestoneRequest) error
	SubmitProject(projectID uint, ownerID uint, input dto.SubmitProjectRequest) error
	GetProjectDetail(id uint, ownerID uint) (*dto.ProjectDetailResponse, error)
	PublishProject(projectID uint, ownerID uint) error
	ApproveProject(projectID uint) error
	RejectProject(projectID uint, reason string) error
	ListProjectsForReview() ([]domain.Project, error)
	ValidateProject(projectID uint, ownerID uint) (*dto.ProjectValidateResponse, error)
}

type projectService struct {
	Repo repository.ProjectRepository
	Auth helper.Auth
}

func NewProjectServiceForTest(repo repository.ProjectRepository) ProjectService {
	return &projectService{
		Repo: repo,
	}
}

func NewProjectService(
	repo repository.ProjectRepository,
	auth helper.Auth,
) ProjectService {
	return &projectService{
		Repo: repo,
		Auth: auth,
	}
}

func (s *projectService) CreateProject(ownerID uint) (*domain.Project, error) {

	//slug, err := s.generateUniqueSlug(input.Title)
	//if err != nil {
	//	return nil, errors.New("failed to generate slug")
	//}

	project := domain.Project{
		OwnerUserID: ownerID,
		//Slug:        slug,
		State:      domain.ProjectStateDraft,
		Status:     domain.ProjectStatusActive,
		Visibility: domain.ProjectVisibilityPrivate,
	}

	err := s.Repo.Create(&project)
	if err != nil {
		return nil, errors.New("failed to create project")
	}

	return &project, nil
}

func (s *projectService) UpdateProjectDraft(
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

func (s *projectService) AddProjectMedia(
	projectID uint,
	ownerID uint,
	input dto.AddProjectMediaRequest,
) error {

	project, err := s.getEditableProject(projectID, ownerID)
	if err != nil {
		return err
	}

	mediaCount, err := s.Repo.CountMediaByProject(project.ID)
	if err != nil {
		return err
	}

	maxSort, err := s.Repo.GetMaxMediaSort(project.ID)
	if err != nil {
		return err
	}

	if mediaCount >= 10 {
		return errors.New("media limit reached")
	}

	if input.Type != "image" && input.Type != "video" {
		return errors.New("invalid media type")
	}

	media := domain.ProjectMedia{
		ProjectID: project.ID,
		URL:       input.URL,
		Type:      input.Type,
		SortOrder: maxSort + 1,
	}
	log.Println("SERVICE URL:", input.URL)
	return s.Repo.CreateMedia(&media)
}

func (s *projectService) GetProjectsByOwnerResponse(ownerID uint) ([]dto.ProjectResponse, error) {

	projects, err := s.Repo.GetByOwner(ownerID)
	if err != nil {
		return nil, errors.New("internal server error")
	}

	var res []dto.ProjectResponse

	for _, p := range projects {
		res = append(res, dto.ProjectResponse{
			ID:          p.ID,
			Title:       p.Title,
			Description: p.Description,
			FundingGoal: p.FundingGoal,
			State:       string(p.State),
			Status:      string(p.Status),
			Visibility:  string(p.Visibility),
		})
	}

	return res, nil
}

func (s *projectService) GetProjectByID(id uint, ownerID uint) (*domain.Project, error) {

	project, err := s.Repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if project.OwnerUserID != ownerID {
		return nil, errors.New("access denied")
	}

	return project, nil
}

func (s *projectService) generateUniqueSlug(title string) (string, error) {

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

func (s *projectService) getEditableProject(projectID uint, ownerID uint) (*domain.Project, error) {

	project, err := s.Repo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("project not found")
		}
		return nil, errors.New("internal server error")
	}

	if project.OwnerUserID != ownerID {
		return nil, errors.New("permission denied")
	}

	if project.State != domain.ProjectStateDraft {
		return nil, errors.New("project cannot be edited")
	}

	return project, nil
}

func (s *projectService) AddProjectStory(projectID uint, ownerID uint, input dto.AddProjectStoryRequest) error {

	project, err := s.getEditableProject(projectID, ownerID)
	if err != nil {
		return err
	}

	maxSort, err := s.Repo.GetMaxStorySectionSort(project.ID)
	if err != nil {
		return err
	}

	story := domain.ProjectStorySection{
		ProjectID: project.ID,
		Title:     input.Title,
		Body:      input.Body,
		SortOrder: maxSort + 1,
	}

	return s.Repo.CreateStory(&story)
}

func (s *projectService) AddProjectRisk(
	projectID uint,
	ownerID uint,
	input dto.AddProjectRiskRequest,
) error {

	project, err := s.getEditableProject(projectID, ownerID)
	if err != nil {
		return err
	}

	maxSort, err := s.Repo.GetMaxRiskSort(project.ID)
	if err != nil {
		return err
	}

	risk := domain.ProjectRisk{
		ProjectID:  project.ID,
		Title:      input.Title,
		Detail:     input.Detail,
		Severity:   input.Severity,
		Mitigation: input.Mitigation,
		SortOrder:  maxSort + 1,
	}

	return s.Repo.CreateRisk(&risk)
}

func (s *projectService) AddProjectFAQ(projectID uint, ownerID uint, input dto.AddProjectFAQRequest) error {

	project, err := s.getEditableProject(projectID, ownerID)
	if err != nil {
		return err
	}

	maxSort, err := s.Repo.GetMaxFAQSort(project.ID)
	if err != nil {
		return err
	}

	faq := domain.ProjectFAQ{
		ProjectID: project.ID,
		Question:  input.Question,
		Answer:    input.Answer,
		SortOrder: maxSort + 1,
	}

	err = s.Repo.CreateFAQ(&faq)
	if err != nil {
		return errors.New("failed to add project faq")
	}

	return nil
}

func (s *projectService) SetFundingPolicy(
	projectID uint,
	ownerID uint,
	input dto.SetFundingPolicyRequest,
) error {

	project, err := s.getEditableProject(projectID, ownerID)
	if err != nil {
		return err
	}

	existing, err := s.Repo.GetFundingPolicyByProject(project.ID)
	if err == nil && existing.ID != 0 {
		return errors.New("funding policy already exists")
	}

	if input.SoftCapAmount > input.HardCapAmount {
		return errors.New("soft cap cannot exceed hard cap")
	}

	if input.HardCapAmount > project.FundingGoal {
		return errors.New("hard cap cannot exceed funding goal")
	}

	minFunding := helper.CalculateMinInvest(project.FundingGoal)
	maxFunding := project.FundingGoal
	policy := domain.ProjectFundingPolicy{
		ProjectID:       project.ID,
		FundingModel:    domain.FundingModel(input.FundingModel),
		SoftCapAmount:   &input.SoftCapAmount,
		HardCapAmount:   &input.HardCapAmount,
		MinInvestAmount: &minFunding,
		MaxInvestAmount: &maxFunding,
	}

	err = s.Repo.CreateFundingPolicy(&policy)
	if err != nil {
		return errors.New("failed to set funding policy")
	}

	return nil
}

func (s *projectService) SetProfitPolicy(projectID uint, ownerID uint, input dto.SetProfitPolicyRequest) error {

	project, err := s.getEditableProject(projectID, ownerID)
	if err != nil {
		return err
	}

	var startDate *time.Time

	if input.ExpectedStartDate != "" {
		t, err := time.Parse("2006-01-02", input.ExpectedStartDate)
		if err != nil {
			return errors.New("invalid expected_start_date")
		}
		startDate = &t
	}

	policy := domain.ProjectProfitPolicy{
		ProjectID:             project.ID,
		DistributionFrequency: input.DistributionFrequency,
		MinimumDurationMonths: input.MinimumDurationMonths,
		TotalQuarters:         input.TotalQuarters,
		ExpectedStartDate:     startDate,
		Note:                  input.Note,
	}

	err = s.Repo.CreateProfitPolicy(&policy)
	if err != nil {
		return errors.New("failed to set profit policy")
	}

	return nil
}

func (s *projectService) validateMilestones(milestones []domain.Milestone) error {

	if len(milestones) != 4 {
		return errors.New("project must have exactly 4 milestones")
	}

	totalPercent := 0
	phaseMap := make(map[int]bool)

	for _, m := range milestones {

		if m.PhaseNo < 1 || m.PhaseNo > 4 {
			return errors.New("invalid milestone phase")
		}

		if m.PercentRelease <= 0 || m.PercentRelease > 100 {
			return errors.New("invalid milestone percent")
		}

		if phaseMap[m.PhaseNo] {
			return errors.New("duplicate milestone phase")
		}

		phaseMap[m.PhaseNo] = true
		totalPercent += m.PercentRelease
	}

	if totalPercent != 100 {
		return errors.New("milestone percent must equal 100")
	}

	return nil
}

func (s *projectService) CreateMilestone(projectID uint, ownerID uint, input dto.CreateMilestoneRequest) error {

	project, err := s.getEditableProject(projectID, ownerID)
	if err != nil {
		return err
	}

	existing, err := s.Repo.GetMilestonesByProject(project.ID)
	if err != nil {
		return err
	}

	if len(existing) >= 4 {
		return errors.New("milestone limit reached")
	}

	milestone := domain.Milestone{
		ProjectID:      project.ID,
		PhaseNo:        input.PhaseNo,
		Title:          input.Title,
		Description:    input.Description,
		PercentRelease: helper.GetMilestonePercent(input.PhaseNo),
		Status:         domain.MilestonePending,
	}

	all := append(existing, milestone)

	// validate เฉพาะตอนครบ 4 milestone
	if len(all) == 4 {
		err = s.validateMilestones(all)
		if err != nil {
			return err
		}
	}

	return s.Repo.CreateMilestone(&milestone)
}

func (s *projectService) validateProjectBeforeSubmit(projectID uint) error {

	project, err := s.Repo.GetFullProject(projectID)
	if err != nil {
		return err
	}

	if project.Title == "" {
		return errors.New("title is required")
	}

	if project.Description == "" {
		return errors.New("description is required")
	}

	if project.FundingGoal <= 0 {
		return errors.New("funding goal is required")
	}

	if len(project.Media) < 3 {
		return errors.New("project must have at least 3 media files")
	}

	if len(project.StorySections) < 3 {
		return errors.New("project must have at least 3 story sections")
	}

	if len(project.FAQ) < 3 {
		return errors.New("project must have at least 3 faq")
	}

	if len(project.Risks) == 0 {
		return errors.New("project must declare risks")
	}

	if project.FundingPolicy == nil {
		return errors.New("funding policy is required")
	}

	if project.ProfitPolicy == nil {
		return errors.New("profit policy is required")
	}

	if len(project.Milestones) != 4 {
		return errors.New("project must have 4 milestones")
	}

	total := 0
	for _, m := range project.Milestones {
		total += m.PercentRelease
	}

	if total != 100 {
		return errors.New("milestone percent must equal 100")
	}

	return nil
}

func (s *projectService) SubmitProject(projectID uint, ownerID uint, input dto.SubmitProjectRequest) error {

	if !input.Confirm {
		return errors.New("confirmation required")
	}

	project, err := s.getEditableProject(projectID, ownerID)
	if err != nil {
		return err
	}

	err = s.validateProjectBeforeSubmit(project.ID)
	if err != nil {
		return err
	}

	slug, err := s.generateUniqueSlug(project.Title)
	if err != nil {
		return errors.New("failed to generate slug")
	}

	project.Slug = &slug
	project.State = domain.ProjectStateReview

	err = s.Repo.Update(project)
	if err != nil {
		return errors.New("failed to submit project")
	}

	return nil
}

func (s *projectService) GetProjectDetail(id uint, ownerID uint) (*dto.ProjectDetailResponse, error) {

	project, err := s.Repo.GetFullProject(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("project not found")
		}
		return nil, errors.New("internal server error")
	}

	if project.OwnerUserID != ownerID {
		return nil, errors.New("access denied")
	}

	res := dto.ProjectDetailResponse{
		ID:          project.ID,
		Title:       project.Title,
		Description: project.Description,
		FundingGoal: project.FundingGoal,
		State:       string(project.State),
	}

	for _, m := range project.Media {
		res.Media = append(res.Media, dto.ProjectMediaResponse{
			ID:   m.ID,
			URL:  m.URL,
			Type: m.Type,
		})
	}

	for _, sct := range project.StorySections {
		res.Story = append(res.Story, dto.StoryResponse{
			ID:    sct.ID,
			Title: sct.Title,
			Body:  sct.Body,
		})
	}

	for _, r := range project.Risks {
		res.Risks = append(res.Risks, dto.RiskResponse{
			ID:       r.ID,
			Title:    r.Title,
			Detail:   r.Detail,
			Severity: r.Severity,
		})
	}

	for _, f := range project.FAQ {
		res.FAQ = append(res.FAQ, dto.FAQResponse{
			ID:       f.ID,
			Question: f.Question,
			Answer:   f.Answer,
		})
	}

	for _, ms := range project.Milestones {
		res.Milestones = append(res.Milestones, dto.MilestoneResponse{
			ID:             ms.ID,
			PhaseNo:        ms.PhaseNo,
			Title:          ms.Title,
			Description:    ms.Description,
			PercentRelease: float64(ms.PercentRelease),
		})
	}
	if project.FundingPolicy != nil {

		var softCap float64
		var hardCap float64
		var minInvest float64
		var maxInvest float64

		if project.FundingPolicy.SoftCapAmount != nil {
			softCap = *project.FundingPolicy.SoftCapAmount
		}

		if project.FundingPolicy.HardCapAmount != nil {
			hardCap = *project.FundingPolicy.HardCapAmount
		}

		if project.FundingPolicy.MinInvestAmount != nil {
			minInvest = *project.FundingPolicy.MinInvestAmount
		}

		if project.FundingPolicy.MaxInvestAmount != nil {
			maxInvest = *project.FundingPolicy.MaxInvestAmount
		}

		res.FundingPolicy = dto.FundingPolicyResponse{
			SoftCapAmount:   softCap,
			HardCapAmount:   hardCap,
			MinInvestAmount: minInvest,
			MaxInvestAmount: maxInvest,
		}
	}

	if project.ProfitPolicy != nil {

		res.ProfitPolicy = &dto.ProfitPolicyResponse{
			DistributionFrequency: project.ProfitPolicy.DistributionFrequency,
			MinimumDurationMonths: project.ProfitPolicy.MinimumDurationMonths,
			TotalQuarters:         project.ProfitPolicy.TotalQuarters,
			ExpectedStartDate:     project.ProfitPolicy.ExpectedStartDate,
			Note:                  project.ProfitPolicy.Note,
		}
	}

	return &res, nil
}

func (s *projectService) PublishProject(projectID uint, ownerID uint) error {

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

	if project.State != domain.ProjectStateApproved {
		return errors.New("project not approved")
	}

	project.Visibility = domain.ProjectVisibilityPublic
	project.State = domain.ProjectStateFunding

	err = s.Repo.Update(project)
	if err != nil {
		return errors.New("failed to publish project")
	}

	return nil
}

func (s *projectService) ApproveProject(projectID uint) error {

	project, err := s.Repo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("project not found")
		}
		return errors.New("internal server error")
	}

	if project.State != domain.ProjectStateReview {
		return errors.New("project not in review state")
	}

	project.State = domain.ProjectStateApproved

	err = s.Repo.Update(project)
	if err != nil {
		return errors.New("failed to approve project")
	}

	return nil
}

func (s *projectService) RejectProject(projectID uint, reason string) error {

	project, err := s.Repo.GetByID(projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("project not found")
		}
		return errors.New("internal server error")
	}

	if project.State != domain.ProjectStateReview {
		return errors.New("project not in review state")
	}

	project.Status = domain.ProjectStatusRejected
	project.State = domain.ProjectStateCancelled

	err = s.Repo.Update(project)
	if err != nil {
		return errors.New("failed to reject project")
	}

	return nil
}

func (s *projectService) ListProjectsForReview() ([]domain.Project, error) {

	projects, err := s.Repo.GetProjectsForReview()
	if err != nil {
		return nil, errors.New("failed to fetch projects")
	}

	return projects, nil
}

func (s *projectService) ValidateProject(projectID uint, ownerID uint) (*dto.ProjectValidateResponse, error) {

	project, err := s.Repo.GetFullProject(projectID)
	if err != nil {
		return nil, err
	}

	if project.OwnerUserID != ownerID {
		return nil, errors.New("access denied")
	}

	var missing []string

	if project.Title == "" {
		missing = append(missing, "title")
	}

	if project.Description == "" {
		missing = append(missing, "description")
	}

	if project.FundingGoal <= 0 {
		missing = append(missing, "funding_goal")
	}

	if len(project.Media) == 0 {
		missing = append(missing, "media")
	}

	if project.ProfitPolicy == nil {
		missing = append(missing, "profit_policy")
	}

	if len(project.StorySections) == 0 {
		missing = append(missing, "story")
	}

	if project.FundingPolicy == nil {
		missing = append(missing, "funding_policy")
	}

	if len(project.Milestones) != 4 {
		missing = append(missing, "milestones")
	}

	res := dto.ProjectValidateResponse{
		Ready:   len(missing) == 0,
		Missing: missing,
	}

	return &res, nil
}

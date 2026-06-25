package services

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestAdminApproveMilestoneSubmission_SetApproved(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	milestoneID := uint(10)
	m := &domain.Milestone{
		ID:     milestoneID,
		Status: domain.MilestoneSubmitted,
	}

	projRepo.On("FindMilestoneByID", milestoneID).Return(m, nil)
	projRepo.On("UpdateMilestone", mock.AnythingOfType("*domain.Milestone")).Return(nil)

	res, err := svc.AdminApproveMilestoneSubmission(milestoneID)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, domain.MilestoneApproved, res.Status)
	assert.False(t, res.VotingOpen)
}

func TestOpenMilestoneVoting_Success(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	user := domain.User{ID: 7}
	milestoneID := uint(11)
	projectID := uint(99)
	m := &domain.Milestone{
		ID:        milestoneID,
		ProjectID: projectID,
		Status:    domain.MilestoneApproved,
	}
	p := &domain.Project{ID: projectID, OwnerUserID: user.ID}

	projRepo.On("FindMilestoneByID", milestoneID).Return(m, nil)
	projRepo.On("FindProjectByID", projectID).Return(p, nil)
	projRepo.On("FindMeetingByMilestoneID", milestoneID).Return(&domain.Meeting{ID: 1, MilestoneID: milestoneID}, nil)
	projRepo.On("UpdateMilestone", mock.AnythingOfType("*domain.Milestone")).Return(nil)
	projRepo.On("DeleteVotesByMilestoneID", uint(11)).Return(nil)

	res, err := svc.OpenMilestoneVoting(milestoneID, user)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.True(t, res.VotingOpen)
	assert.NotNil(t, res.VotingOpenedAt)
}

func TestCloseProject_FundingToExecuting_AndActivatePhaseOne(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	user := domain.User{ID: 1}
	projectID := uint(200)
	p := &domain.Project{
		ID:             projectID,
		OwnerUserID:    user.ID,
		State:          domain.StateFunding,
		Status:         domain.StatusActive,
		FundingGoal:    1000,
		CurrentFunding: 1000,
	}
	milestones := []domain.Milestone{
		{ID: 1, ProjectID: projectID, PhaseNo: 1, Status: domain.MilestoneWaiting},
	}

	projRepo.On("FindProjectByID", projectID).Return(p, nil)
	projRepo.On("UpdateProject", mock.AnythingOfType("*domain.Project")).Return(p, nil)
	projRepo.On("FindMilestonesByProjectID", projectID).Return(milestones, nil)
	projRepo.On("UpdateMilestone", mock.AnythingOfType("*domain.Milestone")).Return(nil)

	err := svc.CloseProject(projectID, user)
	assert.NoError(t, err)
	assert.Equal(t, domain.StateExecuting, p.State)
	assert.Equal(t, domain.StatusActive, p.Status)
}

func TestCloseProject_ExecutingToClosed_WhenAllPaid(t *testing.T) {
	projRepo := new(ProjectRepository)
	svc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	user := domain.User{ID: 1}
	projectID := uint(201)
	p := &domain.Project{
		ID:          projectID,
		OwnerUserID: user.ID,
		State:       domain.StateExecuting,
		Status:      domain.StatusActive,
	}
	milestones := []domain.Milestone{
		{ID: 1, ProjectID: projectID, PhaseNo: 1, Status: domain.MilestonePaid},
		{ID: 2, ProjectID: projectID, PhaseNo: 2, Status: domain.MilestonePaid},
	}

	projRepo.On("FindProjectByID", projectID).Return(p, nil)
	projRepo.On("FindMilestonesByProjectID", projectID).Return(milestones, nil)
	projRepo.On("UpdateProject", mock.AnythingOfType("*domain.Project")).Return(p, nil)

	err := svc.CloseProject(projectID, user)
	assert.NoError(t, err)
	assert.Equal(t, domain.StateClosed, p.State)
	assert.Equal(t, domain.StatusCompleted, p.Status)
}

func TestVoteMilestone_AutoPaidOnMajorityApprove(t *testing.T) {
	projRepo := new(ProjectRepository)
	disbRepo := new(mockDisbursementRepo)
	investRepo := new(mockInvestmentRepo)
	svc := NewInvestmentService(projRepo, investRepo, nil, nil, disbRepo, "", "", nil, nil)

	milestoneID := uint(301)
	projectID := uint(401)
	boosterID := uint(501)
	m := &domain.Milestone{
		ID:         milestoneID,
		ProjectID:  projectID,
		Status:     domain.MilestoneApproved,
		VotingOpen: true,
	}
	project := &domain.Project{ID: projectID, OwnerUserID: 1, CurrentFunding: 10000}

	projRepo.On("FindMilestoneByID", milestoneID).Return(m, nil)
	projRepo.On("HasVerifiedInvestment", projectID, boosterID).Return(true, nil)
	projRepo.On("FindVote", milestoneID, boosterID).Return((*domain.MilestoneVote)(nil), errors.New("not found"))
	projRepo.On("UpsertMilestoneVote", mock.AnythingOfType("*domain.MilestoneVote")).Return(nil)
	projRepo.On("SumVerifiedInvestmentByProjectID", projectID).Return(float64(10000), nil)
	projRepo.On("SumMilestoneVotes", milestoneID, domain.MilestoneVoteApprove).Return(float64(6000), nil)
	projRepo.On("UpdateMilestone", mock.AnythingOfType("*domain.Milestone")).Return(nil)
	projRepo.On("CloseMeetingsByMilestoneID", milestoneID).Return(nil)
	// reject count still queried if majority approve not met, but here it is met (6000*2 > 10000)
	//projRepo.On("SumMilestoneVotes", milestoneID, domain.MilestoneVoteReject).Return(float64(0), nil)

	// disbursement creation path invoked after majority approve
	disbRepo.On("FindByMilestoneID", milestoneID).Return(nil, gorm.ErrRecordNotFound)
	projRepo.On("FindProjectByID", projectID).Return(project, nil)
	investRepo.On("FindVerifiedByProjectID", projectID).Return([]domain.Investment{}, nil)
	disbRepo.On("Create", mock.AnythingOfType("*domain.Disbursement")).Return(nil)
	// activate next phase
	projRepo.On("FindMilestonesByProjectID", projectID).Return([]domain.Milestone{}, nil)

	vote, err := svc.VoteMilestone(boosterID, milestoneID, domain.MilestoneVoteApprove)
	assert.NoError(t, err)
	assert.NotNil(t, vote)
	assert.Equal(t, domain.MilestonePaid, m.Status)
	assert.False(t, m.VotingOpen)
	assert.NotNil(t, m.VotingClosedAt)
}

func TestSubmitMilestone_SetsSubmittedAndData(t *testing.T) {
	projRepo := new(ProjectRepository)
	projectSvc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	user := domain.User{ID: 1}
	projectID := uint(601)
	milestoneID := uint(602)
	m := &domain.Milestone{
		ID:        milestoneID,
		ProjectID: projectID,
		PhaseNo:   1,
		Status:    domain.MilestoneActive,
	}
	p := &domain.Project{ID: projectID, OwnerUserID: user.ID}

	projRepo.On("FindMilestoneByID", milestoneID).Return(m, nil)
	projRepo.On("FindProjectByID", projectID).Return(p, nil)
	projRepo.On("UpdateMilestone", mock.AnythingOfType("*domain.Milestone")).Return(nil)

	req := dto.SubmitMilestoneRequest{
		Criteria:    []string{"A", "B"},
		Attachments: []string{"https://res.cloudinary.com/demo/raw/upload/v1/report.pdf"},
		Links:       []string{"https://github.com/example/repo/pull/1"},
	}

	res, err := projectSvc.SubmitMilestone(milestoneID, req, user)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, domain.MilestoneSubmitted, res.Status)
	assert.NotNil(t, res.SubmittedAt)
	assert.WithinDuration(t, time.Now().UTC(), *res.SubmittedAt, 5*time.Second)
}

func TestSubmitMilestone_Phase2_RequiresPrevPaid(t *testing.T) {
	projRepo := new(ProjectRepository)
	disbRepo := new(mockDisbursementRepo)
	projectSvc := NewProjectService(projRepo, nil, nil, nil, nil, nil, disbRepo)

	user := domain.User{ID: 1}
	projectID := uint(700)
	milestoneID := uint(701)
	m := &domain.Milestone{
		ID:        milestoneID,
		ProjectID: projectID,
		PhaseNo:   2,
		Status:    domain.MilestoneActive,
	}
	p := &domain.Project{ID: projectID, OwnerUserID: user.ID}

	projRepo.On("FindMilestoneByID", milestoneID).Return(m, nil)
	projRepo.On("FindProjectByID", projectID).Return(p, nil)
	projRepo.On("FindMilestonesByProjectID", projectID).Return([]domain.Milestone{
		{ID: 11, ProjectID: projectID, PhaseNo: 1, Status: domain.MilestonePaid},
		{ID: 12, ProjectID: projectID, PhaseNo: 2, Status: domain.MilestoneActive},
	}, nil)
	disbRepo.On("FindByMilestoneID", uint(11)).Return(&domain.Disbursement{Status: domain.DisbursementConfirmed}, nil)
	projRepo.On("UpdateMilestone", mock.AnythingOfType("*domain.Milestone")).Return(nil)

	req := dto.SubmitMilestoneRequest{}
	res, err := projectSvc.SubmitMilestone(milestoneID, req, user)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, domain.MilestoneSubmitted, res.Status)
}

func TestSubmitMilestone_Phase2_FailsWhenPrevNotPaid(t *testing.T) {
	projRepo := new(ProjectRepository)
	projectSvc := NewProjectService(projRepo, nil, nil, nil, nil, nil, nil)

	user := domain.User{ID: 1}
	projectID := uint(710)
	milestoneID := uint(711)
	m := &domain.Milestone{
		ID:        milestoneID,
		ProjectID: projectID,
		PhaseNo:   2,
		Status:    domain.MilestoneActive,
	}
	p := &domain.Project{ID: projectID, OwnerUserID: user.ID}

	projRepo.On("FindMilestoneByID", milestoneID).Return(m, nil)
	projRepo.On("FindProjectByID", projectID).Return(p, nil)
	projRepo.On("FindMilestonesByProjectID", projectID).Return([]domain.Milestone{
		{ID: 21, ProjectID: projectID, PhaseNo: 1, Status: domain.MilestoneRejected},
		{ID: 22, ProjectID: projectID, PhaseNo: 2, Status: domain.MilestoneActive},
	}, nil)

	req := dto.SubmitMilestoneRequest{}
	res, err := projectSvc.SubmitMilestone(milestoneID, req, user)
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, "previous milestone must be paid before submitting this phase", err.Error())
}

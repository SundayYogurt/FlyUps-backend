package services

import (
	"flyup/internal/domain"
	"flyup/internal/dto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestMilestoneStatusCannotBePatched(t *testing.T) {
	for _, status := range []domain.MilestoneStatus{domain.MilestonePaid, domain.MilestoneApproved, domain.MilestoneActive, domain.MilestoneDraft} {
		t.Run(string(status), func(t *testing.T) {
			repo := new(ProjectRepository)
			svc := NewProjectService(repo, nil, nil, nil, nil, nil, nil, nil)
			require.Error(t, svc.UpdateMilestone(1, dto.UpdateMilestoneRequest{Status: &status}, domain.User{ID: 2, Role: "pioneer"}))
			repo.AssertNotCalled(t, "UpdateMilestone", mock.Anything)
		})
	}
}
func TestMilestoneCreatePreservesContent(t *testing.T) {
	repo := new(ProjectRepository)
	svc := NewProjectService(repo, nil, nil, nil, nil, nil, nil, nil)
	repo.On("FindProjectByID", uint(1)).Return(&domain.Project{ID: 1, OwnerUserID: 2}, nil)
	repo.On("FindMilestonesByProjectID", uint(1)).Return([]domain.Milestone{}, nil)
	input := dto.CreateMilestoneRequest{Title: "Valid title", Description: ptrValidation("description"), Duration: ptrValidation(10), AcceptanceCriteria: ptrValidation("criteria"), URLs: []string{"https://res.cloudinary.com/demo/image/upload/proof.png"}, Type: []domain.MediaType{domain.MediaTypeImage}}
	repo.On("CreateMilestone", mock.MatchedBy(func(m *domain.Milestone) bool {
		return m.Title == input.Title && m.Description != nil && *m.Description == *input.Description && m.Duration != nil && *m.Duration == 10 && m.AcceptanceCriteria != nil && *m.AcceptanceCriteria == "criteria" && len(m.URLs) == 1 && len(m.Type) == 1 && m.Status == domain.MilestoneDraft
	})).Return(nil).Once()
	_, err := svc.CreateMilestone(1, input, domain.User{ID: 2})
	require.NoError(t, err)
	repo.AssertExpectations(t)
}
func TestMilestoneNormalEditStillWorks(t *testing.T) {
	repo := new(ProjectRepository)
	svc := NewProjectService(repo, nil, nil, nil, nil, nil, nil, nil)
	repo.On("FindMilestoneByID", uint(1)).Return(&domain.Milestone{ID: 1, ProjectID: 1, PhaseNo: 1, Status: domain.MilestoneDraft}, nil)
	repo.On("FindProjectByID", uint(1)).Return(&domain.Project{ID: 1, OwnerUserID: 2}, nil)
	repo.On("UpdateMilestone", mock.MatchedBy(func(m *domain.Milestone) bool { return m.Title == "Edited" && m.Status == domain.MilestoneDraft })).Return(nil).Once()
	require.NoError(t, svc.UpdateMilestone(1, dto.UpdateMilestoneRequest{Title: ptrValidation("Edited")}, domain.User{ID: 2}))
	repo.AssertExpectations(t)
}
func TestPasswordUnicodeBoundaries(t *testing.T) {
	for _, tc := range []struct {
		p     string
		valid bool
	}{
		{"Aa1!กก", false}, {"Aa1!กกก", false}, {"Aa1!กกกก", true},
		{"Aa1!" + strings.Repeat("a", 68), true}, {"Aa1!" + strings.Repeat("a", 69), false},
		{"Aa1!" + strings.Repeat("ก", 22) + "aa", true}, {"Aa1!" + strings.Repeat("ก", 23), false},
	} {
		require.Equal(t, tc.valid, validatePassword(tc.p) == nil)
	}
}

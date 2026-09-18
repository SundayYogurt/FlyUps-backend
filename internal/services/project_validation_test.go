package services

import (
	"flyup/internal/domain"
	"flyup/internal/dto"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestProjectValidationBeforeRepository(t *testing.T) {
	for _, input := range []dto.UpdateProjectRequest{
		{Title: ptrValidation(strings.Repeat("a", 51))},
		{FundingGoal: ptrValidation(math.Inf(1))},
		{FundingGoal: ptrValidation(math.NaN())},
		{FundingGoal: ptrValidation(1_000_000_001.0)},
		{ProfitSharePct: ptrValidation(101.0)},
		{DurationDays: ptrValidation(61)},
	} {
		repo := new(ProjectRepository)
		svc := NewProjectService(repo, nil, nil, nil, nil, nil, nil, nil)
		_, err := svc.UpdateProject(1, input, domain.User{ID: 1})
		require.Error(t, err)
		repo.AssertNotCalled(t, "FindProjectByID", mock.Anything)
	}
}

func TestMilestoneLimitDoesNotWrite(t *testing.T) {
	repo := new(ProjectRepository)
	svc := NewProjectService(repo, nil, nil, nil, nil, nil, nil, nil)
	repo.On("FindProjectByID", uint(1)).Return(&domain.Project{ID: 1, OwnerUserID: 2}, nil)
	repo.On("FindMilestonesByProjectID", uint(1)).Return(make([]domain.Milestone, 4), nil)
	_, err := svc.CreateMilestone(1, dto.CreateMilestoneRequest{}, domain.User{ID: 2})
	require.ErrorContains(t, err, "maximum 4")
	repo.AssertNotCalled(t, "CreateMilestone", mock.Anything)
}

func TestMediaBatchValidationBeforeWrite(t *testing.T) {
	repo := new(ProjectRepository)
	svc := NewProjectService(repo, nil, nil, nil, nil, nil, nil, nil)
	items := make([]domain.ProjectMedia, 6)
	for i := range items {
		items[i] = domain.ProjectMedia{URL: "https://example.com/photo.png", Type: []domain.MediaType{domain.MediaTypeImage}}
	}
	require.Error(t, svc.AttachProjectMediaBatch(1, items, domain.User{ID: 2}))
	repo.AssertNotCalled(t, "CreateProjectMediaBatch", mock.Anything)

	items = items[:5]
	repo.On("FindProjectByID", uint(1)).Return(&domain.Project{ID: 1, OwnerUserID: 2}, nil)
	repo.On("CreateProjectMediaBatch", mock.MatchedBy(func(media []domain.ProjectMedia) bool { return len(media) == 5 && media[0].ProjectID == 1 })).Return(nil).Once()
	require.NoError(t, svc.AttachProjectMediaBatch(1, items, domain.User{ID: 2}))
	repo.AssertExpectations(t)
}

func ptrValidation[T any](v T) *T { return &v }

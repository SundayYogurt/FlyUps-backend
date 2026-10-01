package services

import (
	"context"
	"flyup/internal/domain"
	"flyup/internal/repository"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Embed the repository interface so these read-only tests only stub the queries they use.
type publicDetailRepository struct {
	repository.ProjectRepository
	project *domain.Project
}

func (r *publicDetailRepository) FindProjectDetailByID(id uint, state *domain.ProjectState, status *domain.ProjectStatus, visibility *domain.ProjectVisibility) (*domain.Project, error) {
	if status != nil {
		panic("public detail must allow completed status")
	}
	return r.project, nil
}
func (r *publicDetailRepository) FindProjectBySlug(slug string) (*domain.Project, error) {
	return r.project, nil
}
func TestPublicCompletedProjectVisibility(t *testing.T) {
	for _, tc := range []struct {
		name       string
		state      domain.ProjectState
		status     domain.ProjectStatus
		visibility domain.ProjectVisibility
		allowed    bool
	}{
		{"completed", domain.StateClosed, domain.StatusCompleted, domain.VisibilityPublic, true},
		{"active", domain.StateFunding, domain.StatusActive, domain.VisibilityPublic, true},
		{"private", domain.StateClosed, domain.StatusCompleted, domain.VisibilityPrivate, false},
		{"unlisted", domain.StateClosed, domain.StatusCompleted, domain.VisibilityUnlisted, false},
		{"draft", domain.StateDraft, domain.StatusActive, domain.VisibilityPublic, false},
		{"failed", domain.StateClosed, domain.StatusFailed, domain.VisibilityPublic, false},
		{"cancelled", domain.StateCancelled, domain.StatusCancelled, domain.VisibilityPublic, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &domain.Project{ID: 35, State: tc.state, Status: tc.status, Visibility: tc.visibility}
			svc := NewProjectService(&publicDetailRepository{project: p}, nil, nil, nil, nil, nil, nil, nil)
			byID, idErr := svc.GetPublicProjectByID(context.Background(), 35)
			bySlug, slugErr := svc.GetPublicProjectBySlug(context.Background(), "project-35")
			if tc.allowed {
				assert.NoError(t, idErr)
				assert.NoError(t, slugErr)
				assert.Equal(t, p, byID)
				assert.Equal(t, p, bySlug)
			} else {
				assert.Error(t, idErr)
				assert.Error(t, slugErr)
				assert.Nil(t, byID)
				assert.Nil(t, bySlug)
			}
		})
	}
}

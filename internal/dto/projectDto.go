package dto

import (
	"flyup/internal/domain"
	"time"
)

type ProjectResponse struct {
	ID              uint                 `json:"id"`
	OwnerUserID     uint                 `json:"owner_user_id"`
	Category        *string              `json:"category"`
	Title           string               `json:"title"`
	Description     *string              `json:"description"`
	State           string               `json:"state"`
	Status          string               `json:"status"`
	Visibility      string               `json:"visibility"`
	Risk            *string              `json:"risk"`
	FundingGoal     float64              `json:"funding_goal"`
	Softcap         float64              `json:"softcap"`
	CurrentFunding  float64              `json:"current_funding"`
	DurationDays    int                  `json:"duration_days"`
	DurationMonths  int                  `json:"duration_months"`
	EndDate         *time.Time           `json:"end_date"`
	FundingAt       *time.Time           `json:"funding_at"`
	CreatedAt       string               `json:"created_at"`
	UpdatedAt       string               `json:"updated_at"`
	ProfitSharePct  float64              `json:"profit_share_pct"`
	MinInvestAmount float64              `json:"min_invest_amount"`
	MaxInvestAmount float64              `json:"max_invest_amount"`
	PlatformFee     float64              `json:"platform_fee"`
	OwnerProfile    *ProjectOwnerProfile `json:"owner_profile,omitempty"`
}

type ProjectOwnerProfile struct {
	FirstName    string  `json:"first_name"`
	LastName     string  `json:"last_name"`
	University   string  `json:"university"`
	Faculty      *string `json:"faculty"`
	Major        *string `json:"major"`
	Bio          *string `json:"bio"`
	ProjectCount int     `json:"project_count"`
}

type ProjectDetailResponse struct {
	ProjectResponse
	Media      []domain.ProjectMedia `json:"media"`
	Milestones []domain.Milestone    `json:"milestones"`
	Stories    []domain.StorySection `json:"stories"`
	FAQs       []domain.ProjectFAQ   `json:"faqs"`
}

type UpdateProjectRequest struct {
	CategoryID      *uint    `json:"category_id"`
	Title           *string  `json:"title"`
	Description     *string  `json:"description"`
	Visibility      *string  `json:"visibility"`
	Risk            *string  `json:"risk"`
	FundingGoal     *float64 `json:"funding_goal"`
	Softcap         *float64 `json:"softcap"`
	DurationDays    *int     `json:"duration_days"`
	DurationMonths  *int     `json:"duration_months"`
	ProfitSharePct  *float64 `json:"profit_share_pct"`
	MinInvestAmount *float64 `json:"min_invest_amount"`
	MaxInvestAmount *float64 `json:"max_invest_amount"`
	PlatformFee     *float64 `json:"platform_fee"`
}

type UpdateMilestoneRequest struct {
	Title              *string                 `json:"title"`
	Description        *string                 `json:"description"`
	Duration           *int                    `json:"duration,omitempty"`
	AcceptanceCriteria *string                 `json:"acceptance_criteria"`
	PhaseNo            *int                    `json:"phase_no"`
	Status             *domain.MilestoneStatus `json:"status,omitempty"`
	URLs               []string                `json:"urls,omitempty"`
	Type               *domain.MediaType       `json:"type,omitempty"`
	SortOrder          *int                    `json:"sort_order,omitempty"`
}

type CreateMilestoneRequest struct {
	Title              string                  `json:"title"`
	Description        *string                 `json:"description,omitempty"`
	Duration           *int                    `json:"duration,omitempty"`
	AcceptanceCriteria *string                 `json:"acceptance_criteria,omitempty"`
	Status             *domain.MilestoneStatus `json:"status,omitempty"`
	PhaseNo            int                     `json:"phase_no"`
	URLs               []string                `json:"urls,omitempty"`
	Type               domain.MediaType
	SortOrder          int
}

type CreateProjectUpdateRequest struct {
	Title       string                   `json:"title"`
	Content     string                   `json:"content"`
	Visibility  domain.ProjectVisibility `json:"visibility"`
	MilestoneID *uint                    `json:"milestone_id,omitempty"`
}

type UpdateProjectUpdateRequest struct {
	Title   *string `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
}

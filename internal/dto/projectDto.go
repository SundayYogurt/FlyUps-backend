package dto

import (
	"flyup/internal/domain"
	"time"
)

type ProjectResponse struct {
	ID              uint      `json:"id"`
	OwnerUserID     uint      `json:"owner_user_id"`
	Category        *string   `json:"category"`
	Title           string    `json:"title"`
	Description     *string   `json:"description"`
	State           string    `json:"state"`
	Status          string    `json:"status"`
	Visibility      string    `json:"visibility"`
	Risk            *string   `json:"risk"`
	FundingGoal     float64   `json:"funding_goal"`
	Softcap         float64   `json:"softcap"`
	CurrentFunding  float64   `json:"current_funding"`
	EndDate         time.Time `json:"end_date"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
	ProfitSharePct  float64   `json:"profit_share_pct"`
	MinInvestAmount float64   `json:"min_invest_amount"`
	MaxInvestAmount float64   `json:"max_invest_amount"`
	PlatformFee     float64   `json:"platform_fee"`
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
	ProfitSharePct  *float64 `json:"profit_share_pct"`
	MinInvestAmount *float64 `json:"min_invest_amount"`
	MaxInvestAmount *float64 `json:"max_invest_amount"`
	PlatformFee     *float64 `json:"platform_fee"`
}

type UpdateMilestoneRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	PhaseNo     *int    `json:"phase_no"`
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

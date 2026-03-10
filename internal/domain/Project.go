package domain

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID uint `json:"id"`

	OwnerUserID  uint  `json:"owner_user_id"`
	UniversityID *uint `json:"university_id,omitempty"`
	CategoryID   *uint `json:"category_id,omitempty"`

	Title       string  `json:"title"`
	Slug        *string `json:"slug,omitempty"`
	Description string  `json:"description"`

	FundingGoal float64 `json:"funding_goal"`

	FundingStartAt *time.Time `json:"funding_start_at,omitempty"`
	FundingEndAt   *time.Time `json:"funding_end_at,omitempty"`

	State      ProjectState      `json:"state"`
	Status     ProjectStatus     `json:"status"`
	Visibility ProjectVisibility `json:"visibility"`

	CoverImageURL string `json:"cover_image_url"`
	VideoURL      string `json:"video_url"`

	gorm.Model
}

type ProjectCategory struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`

	gorm.Model
}

type ProjectMedia struct {
	ID uint `json:"id"`

	ProjectID uint `json:"project_id"`

	Type      string `json:"type"`
	URL       string `json:"url"`
	SortOrder int    `json:"sort_order"`

	gorm.Model
}

type ProjectStorySection struct {
	ID uint `json:"id"`

	ProjectID uint `json:"project_id"`

	Title     string `json:"title"`
	Body      string `json:"body"`
	SortOrder int    `json:"sort_order"`

	gorm.Model
}

type ProjectRisk struct {
	ID uint `json:"id"`

	ProjectID uint `json:"project_id"`

	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Severity   string `json:"severity"`
	Mitigation string `json:"mitigation"`

	SortOrder int `json:"sort_order"`

	gorm.Model
}

type ProjectFAQ struct {
	ID uint `json:"id"`

	ProjectID uint `json:"project_id"`

	Question string `json:"question"`
	Answer   string `json:"answer"`

	SortOrder int `json:"sort_order"`

	gorm.Model
}

type ProjectFundingPolicy struct {
	ID uint `json:"id"`

	ProjectID uint `json:"project_id"`

	FundingModel string `json:"funding_model"`

	SoftCapAmount *float64 `json:"soft_cap_amount,omitempty"`
	HardCapAmount *float64 `json:"hard_cap_amount,omitempty"`

	MinInvestAmount *float64 `json:"min_invest_amount,omitempty"`
	MaxInvestAmount *float64 `json:"max_invest_amount,omitempty"`

	gorm.Model
}

type ProjectProfitPolicy struct {
	ID uint `json:"id"`

	ProjectID uint `json:"project_id"`

	DistributionFrequency string `json:"distribution_frequency"`

	MinimumDurationMonths int `json:"minimum_duration_months"`
	TotalQuarters         int `json:"total_quarters"`

	ExpectedStartDate *time.Time `json:"expected_start_date,omitempty"`
	Note              string     `json:"note"`

	gorm.Model
}

type Milestone struct {
	ID uint `json:"id"`

	ProjectID uint `json:"project_id"`

	PhaseNo int `json:"phase_no"`

	Title       string `json:"title"`
	Description string `json:"description"`

	PercentRelease int    `json:"percent_release"`
	Status         string `json:"status"`

	gorm.Model
}

type MilestoneSubmission struct {
	ID uint `json:"id"`

	MilestoneID uint `json:"milestone_id"`
	SubmittedBy uint `json:"submitted_by"`

	Note   string `json:"note"`
	Status string `json:"status"`

	SubmittedAt *time.Time `json:"submitted_at,omitempty"`

	gorm.Model
}

type MilestoneEvidence struct {
	ID uint `json:"id"`

	SubmissionID uint `json:"submission_id"`

	Type string `json:"type"`
	URL  string `json:"url"`

	gorm.Model
}

type ProjectState string

const (
	ProjectStateDraft     ProjectState = "draft"
	ProjectStateReview    ProjectState = "review"
	ProjectStateApproved  ProjectState = "approved"
	ProjectStateFunding   ProjectState = "funding"
	ProjectStateClosed    ProjectState = "closed"
	ProjectStateCancelled ProjectState = "cancelled"
)

type ProjectStatus string

const (
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusFunded    ProjectStatus = "funded"
	ProjectStatusFailed    ProjectStatus = "failed"
	ProjectStatusRejected  ProjectStatus = "rejected"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusCancelled ProjectStatus = "cancelled"
)

type ProjectVisibility string

const (
	ProjectVisibilityPrivate  ProjectVisibility = "private"
	ProjectVisibilityPublic   ProjectVisibility = "public"
	ProjectVisibilityUnlisted ProjectVisibility = "unlisted"
)

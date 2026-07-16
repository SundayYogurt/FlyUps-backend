package domain

import (
	"time"

	"gorm.io/gorm"
)

type ProjectState string
type ProjectStatus string
type ProjectVisibility string

const (
	StateDraft         ProjectState = "draft"
	StatePendingReview ProjectState = "pending_review"
	StateFunding       ProjectState = "funding"
	StateExecuting     ProjectState = "executing"
	StateClosed        ProjectState = "closed"
	StateCancelled     ProjectState = "cancelled"
	StatePendingCancel ProjectState = "pending_cancel"
	StateSuspended     ProjectState = "suspended"
)

const (
	StatusActive    ProjectStatus = "active"
	StatusFunded    ProjectStatus = "funded"
	StatusFailed    ProjectStatus = "failed"
	StatusRejected  ProjectStatus = "rejected"
	StatusCompleted ProjectStatus = "completed"
	StatusCancelled ProjectStatus = "cancelled"
	StatusSuspended ProjectStatus = "suspended"
)

const (
	VisibilityPrivate  ProjectVisibility = "private"
	VisibilityPublic   ProjectVisibility = "public"
	VisibilityUnlisted ProjectVisibility = "unlisted"
)

type Project struct {
	ID                uint              `json:"id"`
	OwnerUserID       uint              `json:"owner_user_id"`
	Owner             *User             `json:"owner,omitempty" gorm:"foreignKey:OwnerUserID"`
	CategoryID        *uint             `json:"category_id,omitempty"`
	Category          *ProjectCategory  `json:"category,omitempty"`
	CoverImage        *string           `json:"cover_image,omitempty"`
	Title             string            `json:"title"`
	Description       *string           `json:"description,omitempty" gorm:"size:40"`
	Risk              *string           `json:"risk,omitempty"`
	State             ProjectState      `json:"state"`
	Status            ProjectStatus     `json:"status"`
	PreviousState     *ProjectState     `json:"previous_state,omitempty"`
	PreviousStatus    *ProjectStatus    `json:"previous_status,omitempty"`
	Visibility        ProjectVisibility `json:"visibility"`
	FundingGoal       float64           `json:"funding_goal"`
	Softcap           float64           `json:"softcap"`
	CurrentFunding    float64           `json:"current_funding"`
	DurationDays      int               `json:"duration_days"`
	DurationMonths    int               `json:"duration_months"`
	EndDate           time.Time         `json:"end_date"`
	ExecutionEndAt    *time.Time        `json:"execution_end_at,omitempty"`
	ProfitSharePct    float64           `json:"profit_share_pct"`
	MinInvestAmount   float64           `json:"min_invest_amount"`
	MaxInvestAmount   float64           `json:"max_invest_amount"`
	PlatformFee       float64           `json:"platform_fee"`
	FundingAt         time.Time         `json:"funding_at"`
	Media             []ProjectMedia    `json:"media" gorm:"foreignKey:ProjectID"`
	Milestones        []Milestone       `json:"milestones" gorm:"foreignKey:ProjectID"`
	Stories           []StorySection    `json:"stories" gorm:"foreignKey:ProjectID"`
	FAQs              []ProjectFAQ      `json:"faqs" gorm:"foreignKey:ProjectID"`
	CancelReason      string            `json:"cancel_reason"`
	CancelDescription string            `json:"cancel_description"`
	Slug              string            `json:"slug" gorm:"uniqueIndex;not null;default:''"`
	gorm.Model
}

type ProjectCategory struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	gorm.Model
}

type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
	MediaTypeRaw   MediaType = "raw"
)

type ProjectMedia struct {
	ID        uint        `json:"id"`
	ProjectID uint        `json:"project_id"`
	Type      []MediaType `json:"type" gorm:"type:json;serializer:json"`
	URL       string      `json:"url"`
	SortOrder int         `json:"sort_order"`
	gorm.Model
}

type StorySection struct {
	ID        uint   `json:"id"`
	ProjectID uint   `json:"project_id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	SortOrder int    `json:"sort_order"`
	gorm.Model
}

type ProjectFAQ struct {
	ID        uint   `json:"id"`
	ProjectID uint   `json:"project_id"`
	Question  string `json:"question"`
	Answer    string `json:"answer"`
	SortOrder int    `json:"sort_order"`
	gorm.Model
}

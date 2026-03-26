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
	StateClosed        ProjectState = "closed"
	StateCancelled     ProjectState = "cancelled"
)

const (
	StatusActive    ProjectStatus = "active"
	StatusFunded    ProjectStatus = "funded"
	StatusFailed    ProjectStatus = "failed"
	StatusRejected  ProjectStatus = "rejected"
	StatusCompleted ProjectStatus = "completed"
	StatusCancelled ProjectStatus = "cancelled"
)

const (
	VisibilityPrivate  ProjectVisibility = "private"
	VisibilityPublic   ProjectVisibility = "public"
	VisibilityUnlisted ProjectVisibility = "unlisted"
)

type Project struct {
	ID              uint              `json:"id"`
	OwnerUserID     uint              `json:"owner_user_id"`
	CategoryID      *uint             `json:"category_id,omitempty"`
	Category        *ProjectCategory  `json:"category,omitempty"`
	Title           string            `json:"title"`
	Description     *string           `json:"description,omitempty"`
	State           ProjectState      `json:"state"`
	Status          ProjectStatus     `json:"status"`
	Visibility      ProjectVisibility `json:"visibility"`
	FundingGoal     float64           `json:"funding_goal"`
	ProfitSharePct  float64           `json:"profit_share_pct"`
	MinInvestAmount float64           `json:"min_invest_amount"`
	MaxInvestAmount float64           `json:"max_invest_amount"`
	PlatformFee     float64           `json:"platform_fee"`
	gorm.Model
}

type ProjectCategory struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
)

type ProjectMedia struct {
	ID        uint      `json:"id"`
	ProjectID uint      `json:"project_id"`
	Type      MediaType `json:"type"`
	URL       string    `json:"url"`
	SortOrder int       `json:"sort_order"`
}

type StorySection struct {
	ID        uint   `json:"id"`
	ProjectID uint   `json:"project_id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	SortOrder int    `json:"sort_order"`
}

type ProjectFAQ struct {
	ID        uint   `json:"id"`
	ProjectID uint   `json:"project_id"`
	Question  string `json:"question"`
	Answer    string `json:"answer"`
	SortOrder int    `json:"sort_order"`
}

type MilestoneStatus string

const (
	MilestoneDraft     MilestoneStatus = "draft"
	MilestoneActive    MilestoneStatus = "active"
	MilestoneSubmitted MilestoneStatus = "submitted"
	MilestoneApproved  MilestoneStatus = "approved"
	MilestoneRejected  MilestoneStatus = "rejected"
	MilestonePaid      MilestoneStatus = "paid"
)

type Milestone struct {
	ID             uint            `json:"id"`
	ProjectID      uint            `json:"project_id"`
	PhaseNo        int             `json:"phase_no"`
	Title          string          `json:"title"`
	Description    *string         `json:"description,omitempty"`
	PercentRelease int             `json:"percent_release"`
	Status         MilestoneStatus `json:"status"`
}

type InvestmentStatus string

const (
	InvestmentPending  InvestmentStatus = "pending_payment"
	InvestmentVerified InvestmentStatus = "verified"
	InvestmentRejected InvestmentStatus = "rejected"
	InvestmentRefunded InvestmentStatus = "refunded"
)

type ProjectInvestment struct {
	ID              uint             `json:"id"`
	ProjectID       uint             `json:"project_id"`
	BoosterUserID   uint             `json:"booster_user_id"`
	PrincipalAmount float64          `json:"principal_amount"`
	TotalAmount     float64          `json:"total_amount"`
	Status          InvestmentStatus `json:"status"`
	CreatedAt       time.Time        `json:"created_at"`
}

type ProjectUpdate struct {
	ID          uint              `json:"id"`
	ProjectID   uint              `json:"project_id"`
	MilestoneID *uint             `json:"milestone_id,omitempty"`
	PostedBy    uint              `json:"posted_by"`
	Title       string            `json:"title"`
	Body        string            `json:"body"`
	Visibility  ProjectVisibility `json:"visibility"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type ThreadStatus string

const (
	ThreadOpen     ThreadStatus = "open"
	ThreadAnswered ThreadStatus = "answered"
	ThreadHidden   ThreadStatus = "hidden"
	ThreadLocked   ThreadStatus = "locked"
)

type ProjectThread struct {
	ID        uint         `json:"id"`
	ProjectID uint         `json:"project_id"`
	Type      string       `json:"type"`
	CreatedBy uint         `json:"created_by"`
	Title     *string      `json:"title,omitempty"`
	Status    ThreadStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type MessageType string

const (
	MessageText MessageType = "text"
	MessageFile MessageType = "file"
	MessageLink MessageType = "link"
)

type ProjectThreadMessage struct {
	ID        uint         `json:"id"`
	ProjectID uint         `json:"project_id"`
	Type      string       `json:"type"`
	CreatedBy uint         `json:"created_by"`
	Title     *string      `json:"title,omitempty"`
	Status    ThreadStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

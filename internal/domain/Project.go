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
	Owner           *User             `json:"owner,omitempty" gorm:"foreignKey:OwnerUserID"`
	CategoryID      *uint             `json:"category_id,omitempty"`
	Category        *ProjectCategory  `json:"category,omitempty"`
	Title           string            `json:"title"`
	Description     *string           `json:"description,omitempty"`
	Risk            *string           `json:"risk,omitempty"`
	State           ProjectState      `json:"state"`
	Status          ProjectStatus     `json:"status"`
	Visibility      ProjectVisibility `json:"visibility"`
	FundingGoal     float64           `json:"funding_goal"`
	Softcap         float64           `json:"softcap"`
	CurrentFunding  float64           `json:"current_funding"`
	DurationDays    int               `json:"duration_days"`
	DurationMonths  int               `json:"duration_months"`
	EndDate         time.Time         `json:"end_date"`
	ExecutionEndAt  *time.Time        `json:"execution_end_at,omitempty"`
	ProfitSharePct  float64           `json:"profit_share_pct"`
	MinInvestAmount float64           `json:"min_invest_amount"`
	MaxInvestAmount float64           `json:"max_invest_amount"`
	PlatformFee     float64           `json:"platform_fee"`
	FundingAt       time.Time         `json:"funding_at"`
	Media           []ProjectMedia    `json:"media" gorm:"foreignKey:ProjectID"`
	Milestones      []Milestone       `json:"milestones" gorm:"foreignKey:ProjectID"`
	Stories         []StorySection    `json:"stories" gorm:"foreignKey:ProjectID"`
	FAQs            []ProjectFAQ      `json:"faqs" gorm:"foreignKey:ProjectID"`
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

type MilestoneStatus string

const (
	MilestoneDraft     MilestoneStatus = "draft"
	MilestoneWaiting   MilestoneStatus = "waiting"
	MilestoneActive    MilestoneStatus = "active"
	MilestoneSubmitted MilestoneStatus = "submitted"
	MilestoneApproved  MilestoneStatus = "approved"
	MilestoneRejected  MilestoneStatus = "rejected"
	MilestoneFailed    MilestoneStatus = "failed"
	MilestonePaid      MilestoneStatus = "paid"
)

type MilestoneVoteChoice string

const (
	MilestoneVoteApprove MilestoneVoteChoice = "approve"
	MilestoneVoteReject  MilestoneVoteChoice = "reject"
)

type Milestone struct {
	ID                 uint        `json:"id"`
	ProjectID          uint        `json:"project_id"`
	PhaseNo            int         `json:"phase_no"`
	Title              string      `json:"title"`
	Description        *string     `json:"description"`
	Duration           *int        `json:"duration,omitempty"`
	DueDate            *time.Time  `json:"due_date,omitempty"`
	AcceptanceCriteria *string     `json:"acceptance_criteria"`
	Type               []MediaType `json:"type" gorm:"type:json;serializer:json"`
	URLs               []string    `json:"urls" gorm:"type:json;serializer:json"`
	// Submission (what was done in this phase + evidence)
	SubmissionSummary     *string    `json:"submission_summary,omitempty"`
	SubmissionCriteria    []string   `json:"submission_criteria,omitempty" gorm:"type:json;serializer:json"`
	SubmissionAttachments []string   `json:"submission_attachments,omitempty" gorm:"type:json;serializer:json"`
	SubmissionLinks       []string   `json:"submission_links,omitempty" gorm:"type:json;serializer:json"`
	SubmittedAt           *time.Time `json:"submitted_at,omitempty"`
	// Booster voting gate (opened by pioneer after admin approves submission)
	VotingOpen     bool            `json:"voting_open" gorm:"default:false"`
	VotingOpenedAt *time.Time      `json:"voting_opened_at,omitempty"`
	VotingClosedAt *time.Time      `json:"voting_closed_at,omitempty"`
	SortOrder      int             `json:"sort_order"`
	PercentRelease int             `json:"percent_release"`
	Status         MilestoneStatus `json:"status"`
	Meetings       []Meeting       `gorm:"foreignKey:MilestoneID"`
	gorm.Model
}

// MilestoneVote represents booster voting on milestone submission.
// One booster can vote once per milestone.
type MilestoneVote struct {
	ID            uint                `json:"id"`
	MilestoneID   uint                `json:"milestone_id" gorm:"index;uniqueIndex:uniq_milestone_booster"`
	ProjectID     uint                `json:"project_id" gorm:"index"`
	BoosterUserID uint                `json:"booster_user_id" gorm:"index;uniqueIndex:uniq_milestone_booster"`
	Choice        MilestoneVoteChoice `json:"choice" gorm:"type:varchar(20);not null"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

type InvestmentStatus string

const (
	InvestmentPending       InvestmentStatus = "pending_payment"
	InvestmentVerified      InvestmentStatus = "verified"
	InvestmentRejected      InvestmentStatus = "rejected"
	InvestmentRefundPending InvestmentStatus = "refund_pending"
	InvestmentRefunded      InvestmentStatus = "refunded"
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
	ID          uint              `json:"id" gorm:"primaryKey"`
	ProjectID   uint              `json:"project_id" gorm:"index;not null"`
	MilestoneID *uint             `json:"milestone_id,omitempty" gorm:"index"`
	PostedBy    uint              `json:"posted_by" gorm:"not null"`
	Title       string            `json:"title" gorm:"type:varchar(255);not null"`
	Body        string            `json:"body" gorm:"type:text;not null"`
	Visibility  ProjectVisibility `json:"visibility" gorm:"type:varchar(20);default:'public'"`
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
	Body      string       `json:"body"`
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
	ThreadID  uint         `json:"thread_id"`
	Type      string       `json:"type"`
	CreatedBy uint         `json:"created_by"`
	Body      string       `json:"body"`
	Status    ThreadStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}
type MeetingType string

const (
	Online MeetingType = "online"
	Onsite MeetingType = "onsite"
	Hybrid MeetingType = "hybrid"
)

type MeetingStatus string

const (
	MeetingOpen      MeetingStatus = "open"
	MeetingClosed    MeetingStatus = "closed"
	MeetingCancelled MeetingStatus = "cancelled"
)

type Meeting struct {
	ID          uint          `json:"id"`
	MilestoneID uint          `json:"milestone_id" gorm:"index"`
	Milestone   Milestone     `gorm:"foreignKey:MilestoneID"`
	Date        time.Time     `json:"date"`
	Time        time.Time     `json:"time"`
	MeetingType MeetingType   `json:"meeting_type"`           // "online" หรือ "onsite"
	Link        *string       `json:"link,omitempty"`         // อนุญาตให้เป็น null
	Place       *string       `json:"place,omitempty"`        // อนุญาตให้เป็น null
	About       string        `json:"about" gorm:"type:text"` // ใช้ type text เพราะวาระการประชุมอาจจะยาว
	Status      MeetingStatus `json:"status"`

	gorm.Model
}

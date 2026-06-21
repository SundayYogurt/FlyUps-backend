package domain

import (
	"time"

	"gorm.io/gorm"
)

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
	MilestoneCancelled MilestoneStatus = "cancelled"
)

type MilestoneVoteChoice string

const (
	MilestoneVoteApprove MilestoneVoteChoice = "approve"
	MilestoneVoteReject  MilestoneVoteChoice = "reject"
)

type Milestone struct {
	ID                    uint            `json:"id"`
	ProjectID             uint            `json:"project_id"`
	PhaseNo               int             `json:"phase_no"`
	Title                 string          `json:"title"`
	Description           *string         `json:"description"`
	Duration              *int            `json:"duration,omitempty"`
	OriginalDueDate       *time.Time      `json:"original_due_date,omitempty"`
	DueDate               *time.Time      `json:"due_date,omitempty"`
	RetryCount            int             `json:"retry_count" gorm:"default:0"`
	RetryDeadline         *time.Time      `json:"retry_deadline,omitempty"`
	AcceptanceCriteria    *string         `json:"acceptance_criteria"`
	Type                  []MediaType     `json:"type" gorm:"type:json;serializer:json"`
	URLs                  []string        `json:"urls" gorm:"type:json;serializer:json"`
	AdminNote             *string         `json:"admin_note,omitempty"`
	SubmissionSummary     *string         `json:"submission_summary,omitempty"`
	SubmissionCriteria    []string        `json:"submission_criteria,omitempty" gorm:"type:json;serializer:json"`
	SubmissionAttachments []string        `json:"submission_attachments,omitempty" gorm:"type:json;serializer:json"`
	SubmissionLinks       []string        `json:"submission_links,omitempty" gorm:"type:json;serializer:json"`
	SubmittedAt           *time.Time      `json:"submitted_at,omitempty"`
	VotingOpen            bool            `json:"voting_open" gorm:"default:false"`
	VotingOpenedAt        *time.Time      `json:"voting_opened_at,omitempty"`
	VotingClosedAt        *time.Time      `json:"voting_closed_at,omitempty"`
	SortOrder             int             `json:"sort_order"`
	PercentRelease        int             `json:"percent_release"`
	Status                MilestoneStatus `json:"status"`
	Meetings              []Meeting       `gorm:"foreignKey:MilestoneID"`
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

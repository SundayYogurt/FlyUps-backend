package domain

import (
	"time"

	"gorm.io/gorm"
)

type DisbursementStatus string

const (
	DisbursementPending   DisbursementStatus = "pending"
	DisbursementConfirmed DisbursementStatus = "confirmed"
)

type Disbursement struct {
	ID             uint               `json:"id"`
	MilestoneID    uint               `json:"milestone_id" gorm:"uniqueIndex"`
	ProjectID      uint               `json:"project_id" gorm:"index"`
	PioneerUserID  uint               `json:"pioneer_user_id" gorm:"index"`
	Amount         float64            `json:"amount"`
	PhaseNo        int                `json:"phase_no"`
	PercentRelease int                `json:"percent_release"`
	Status         DisbursementStatus `json:"status" gorm:"default:'pending'"`
	TransferRef    string             `json:"transfer_ref"`
	ConfirmedAt    *time.Time         `json:"confirmed_at,omitempty"`
	ConfirmedBy    *uint              `json:"confirmed_by,omitempty"`
	AdminNote      string             `json:"admin_note"`
	gorm.Model
}

package domain

import (
	"time"

	"gorm.io/gorm"
)

type ComplaintStatus string

const (
	ComplaintOpen     ComplaintStatus = "open"
	ComplaintResolved ComplaintStatus = "resolved"
	ComplaintRejected ComplaintStatus = "rejected"
)

// Complaint = booster/pioneer ร้องเรียนโปรเจกต์ — 1 user ร้องเรียนได้ 1 ครั้งต่อ project
type Complaint struct {
	ID            uint            `json:"id"`
	ComplainantID uint            `json:"complainant_id" gorm:"uniqueIndex:idx_complaint_user_project;not null"`
	ProjectID     uint            `json:"project_id"     gorm:"uniqueIndex:idx_complaint_user_project;not null"`
	Subject       string          `json:"subject"        gorm:"not null"`
	Body          string          `json:"body"           gorm:"type:text;not null"`
	Status        ComplaintStatus `json:"status"         gorm:"default:'open';index"`
	AdminNote     string          `json:"admin_note"     gorm:"type:text"`
	ResolvedBy    *uint           `json:"resolved_by,omitempty"`
	ResolvedAt    *time.Time      `json:"resolved_at,omitempty"`
	Complainant   *User           `json:"complainant,omitempty" gorm:"foreignKey:ComplainantID"`
	Project       *Project        `json:"project,omitempty"     gorm:"foreignKey:ProjectID"`
	gorm.Model
}

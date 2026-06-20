package domain

import (
	"time"

	"gorm.io/gorm"
)

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
	MeetingType MeetingType   `json:"meeting_type"`
	Link        *string       `json:"link,omitempty"`
	Place       *string       `json:"place,omitempty"`
	Description *string       `json:"description,omitempty"`
	About       string        `json:"about" gorm:"type:text"`
	Status      MeetingStatus `json:"status"`
	gorm.Model
}

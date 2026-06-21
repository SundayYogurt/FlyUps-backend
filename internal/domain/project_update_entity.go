package domain

import "time"

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

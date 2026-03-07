package domain

import (
	"time"

	"gorm.io/gorm"
)

// StudentProfile represents extended details for pioneer users (students)
type StudentProfile struct {
	ID             uint       `json:"id"`
	UserID         uint       `json:"user_id"`
	UniversityID   uint       `json:"university_id"`
	StudentCode    *string    `json:"student_code,omitempty"`
	Faculty        *string    `json:"faculty,omitempty"`
	Major          *string    `json:"major,omitempty"`
	Bio            *string    `json:"bio,omitempty"`
	Portfolio      *string    `json:"portfolio,omitempty"`
	Skills         *string    `json:"skills,omitempty"`
	ReviewedBy     *uint      `json:"reviewed_by,omitempty"`
	StudentCardURL *string    `json:"student_card_url,omitempty"`
	VerifyStatus   string     `json:"verify_status"` // pending|approved|rejected
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
	gorm.Model
}

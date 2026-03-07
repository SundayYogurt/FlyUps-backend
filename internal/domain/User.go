package domain

import (
	"time"

	"gorm.io/gorm"
)

// User represents the core user entity
type User struct {
	ID                         uint       `json:"id"`
	Email                      string     `json:"email"`
	PasswordHash               string     `json:"-"`                    // ซ่อนไว้ไม่ให้ return ออกไปทาง API
	GoogleSub                  *string    `json:"google_sub,omitempty"` //omitempty ละเว้นถ้ามันว่างเปล่า * pointer ทำให้เก็บค่าเป็น NULL ได้
	FirstName                  string     `json:"first_name"`
	LastName                   string     `json:"last_name"`
	Phone                      string     `json:"phone"`
	Status                     string     `json:"status"` // active|suspended|deleted
	Role                       string     `json:"role"`
	EmailVerifiedAt            *time.Time `json:"email_verified_at,omitempty"`
	VerificationToken          *string    `json:"-"`
	VerificationTokenExpiresAt *time.Time `json:"-"`
	ResetTokenHash             *string    `json:"-"`
	ResetTokenExpiresAt        *time.Time `json:"-"`
	gorm.Model
}

// UserConsent represents the agreement records for a user
type UserConsent struct {
	ID          uint      `json:"id"`
	UserID      uint      `json:"user_id"`
	ConsentCode string    `json:"consent_code"`
	Accepted    bool      `json:"accepted"`
	AcceptedAt  time.Time `json:"accepted_at"`
	gorm.Model
}

package domain

import (
	"time"

	"gorm.io/gorm"
)

// User represents the core user entity
type User struct {
	ID                         uint            `json:"id"`
	Email                      string          `json:"email"`
	PasswordHash               string          `json:"-"`                    // ซ่อนไว้ไม่ให้ return ออกไปทาง API
	GoogleSub                  *string         `json:"google_sub,omitempty"` //omitempty ละเว้นถ้ามันว่างเปล่า * pointer ทำให้เก็บค่าเป็น NULL ได้
	FirstName                  string          `json:"first_name"`
	LastName                   string          `json:"last_name"`
	Phone                      string          `json:"phone"`
	Address                    *string         `json:"address,omitempty"`
	Status                     string          `json:"status"` // active|suspended|deleted
	Role                       string          `json:"role"`
	EmailVerifiedAt            *time.Time      `json:"email_verified_at,omitempty"`
	VerificationToken          *string         `json:"-"`
	VerificationTokenExpiresAt *time.Time      `json:"-"`
	ResetTokenHash             *string         `json:"-"`
	ResetTokenExpiresAt        *time.Time      `json:"-"`
	StudentProfile             *StudentProfile `json:"student_profile,omitempty" gorm:"foreignKey:UserID"`
	BankAccount                *BankAccount    `gorm:"foreignKey:UserID"`
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

const (
	ConsentTerm        = "TERM"
	ConsentAcceptTrue  = true
	ConsentPioneerTerm = "PIONEER_TERM"
)

type BankAccount struct {
	ID            uint       `json:"id"`
	UserID        uint       `json:"user_id"`
	BankName      string     `json:"bank_name"`
	AccountName   string     `json:"account_name"`
	AccountNumber string     `json:"account_number"` // เข้ารหัสก่อนเก็บ
	VerifyStatus  string     `json:"verify_status"`  // pending|approved|rejected
	VerifiedAt    *time.Time `json:"verified_at,omitempty"`
	ReviewedBy    *uint      `json:"reviewed_by,omitempty"` // admin ID
	ProofURL      *string    `json:"proof_url,omitempty"`   // ไฟล์ bank proof
	gorm.Model
}

type IdentityVerification struct {
	ID         uint
	UserID     uint
	Type       string // "student_card" หรือ "id_card"
	Document   string // URL
	Status     string // pending|approved|rejected
	VerifiedAt *time.Time
	ReviewedBy *uint
	gorm.Model
}

type BankVerification struct {
	ID            uint
	UserID        uint
	BankName      string
	AccountName   string
	AccountNumber string
	Proof         string // URL
	Status        string // pending|approved|rejected
	VerifiedAt    *time.Time
	ReviewedBy    *uint
	gorm.Model
}

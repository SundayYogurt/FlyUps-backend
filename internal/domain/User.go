package domain

import (
	"time"

	"gorm.io/gorm"
)

type status string

const (
	ACTIVE    status = "active"
	SUSPENDED status = "suspended"
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
	Status                     string          `json:"status"` // active|suspended
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
	ConsentTerm         = "TERM"
	ConsentPioneerTerm  = "PIONEER_TERM"
	ConsentDeclareTruth = "DECLARE_TRUTH"
)

type BankAccount struct {
	ID            uint   `json:"id"`
	UserID        uint   `json:"user_id"`
	BankName      string `json:"bank_name"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	gorm.Model
}

type VerifyStatus string

const (
	VerifyStatusPending  VerifyStatus = "pending"
	VerifyStatusApproved VerifyStatus = "approved"
	VerifyStatusRejected VerifyStatus = "rejected"
)

type IdCardVerification struct {
	ID         uint         `json:"id"`
	UserID     uint         `json:"user_id"`
	Document   string       `json:"document"`   // URL รูปบัตรประชาชน
	SelfieURL  *string      `json:"selfie_url"` // URL รูปเซลฟี่ (หน้าคู่บัตร)
	Status     VerifyStatus `json:"status"`     // pending|approved|rejected
	VerifiedAt *time.Time   `json:"verified_at,omitempty"`
	ReviewedBy *uint        `json:"reviewed_by,omitempty"`
	OcrPayload *string      `json:"ocr_payload,omitempty"`
	FaceScore  *float64     `json:"face_score,omitempty"`
	gorm.Model
}

type StudentCardVerification struct {
	ID         uint         `json:"id"`
	UserID     uint         `json:"user_id"`
	Document   string       `json:"document"` // URL รูปบัตรนักศึกษา
	Status     VerifyStatus `json:"status"`   // pending|approved|rejected
	VerifiedAt *time.Time   `json:"verified_at,omitempty"`
	ReviewedBy *uint        `json:"reviewed_by,omitempty"` // แอดมินเป็นคนตรวจ
	gorm.Model
}

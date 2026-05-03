package domain

import (
	"time"

	"gorm.io/gorm"
)

type UserStatus string

const (
	ACTIVE    UserStatus = "active"
	SUSPENDED UserStatus = "suspended"
)

// User represents the core user entity
type User struct {
	ID                         uint                     `json:"id"`
	Email                      string                   `json:"email"`
	PasswordHash               string                   `json:"-"`                    // ซ่อนไว้ไม่ให้ return ออกไปทาง API
	GoogleSub                  *string                  `json:"google_sub,omitempty"` //omitempty ละเว้นถ้ามันว่างเปล่า * pointer ทำให้เก็บค่าเป็น NULL ได้
	HasPassword                *bool                    `json:"has_password" gorm:"-"` // true=มี password, false=ไม่มี (Google only)
	FirstName                  string                   `json:"first_name"`
	LastName                   string                   `json:"last_name"`
	Phone                      string                   `json:"phone"`
	Address                    *string                  `json:"address,omitempty"`
	Status                     UserStatus               `json:"status"` // active|suspended
	Role                       string                   `json:"role"`
	Picture                    *string                  `json:"picture,omitempty"`
	EmailVerifiedAt            *time.Time               `json:"email_verified_at,omitempty"`
	VerificationToken          *string                  `json:"-"`
	VerificationTokenExpiresAt *time.Time               `json:"-"`
	ResetTokenHash             *string                  `json:"-"`
	ResetTokenExpiresAt        *time.Time               `json:"-"`
	StudentProfile             *StudentProfile          `json:"student_profile,omitempty" gorm:"foreignKey:UserID"`
	BankAccount                *BankAccount             `json:"bank_account,omitempty" gorm:"foreignKey:UserID"`
	StudentCardVerification    *StudentCardVerification `json:"student_card_verification,omitempty" gorm:"foreignKey:UserID"`
	IdCardVerification         *IdCardVerification      `json:"id_card_verification,omitempty" gorm:"foreignKey:UserID"`
	SuspendReason              *string                  `json:"suspend_reason,omitempty"`
	SuspendedBy                *uint                    `json:"suspended_by,omitempty"`
	SuspendedAt                *time.Time               `json:"suspended_at,omitempty"`
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
	User       User         `gorm:"foreignKey:UserID"`
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
	User       User         `gorm:"foreignKey:UserID"`
	VerifiedAt *time.Time   `json:"verified_at,omitempty"`
	ReviewedBy *uint        `json:"reviewed_by,omitempty"` // แอดมินเป็นคนตรวจ
	gorm.Model
}

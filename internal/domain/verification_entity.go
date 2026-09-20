package domain

import (
	"time"

	"gorm.io/gorm"
)

type VerifyStatus string

const (
	VerifyStatusPending  VerifyStatus = "pending"
	VerifyStatusApproved VerifyStatus = "approved"
	VerifyStatusRejected VerifyStatus = "rejected"
	VerifyStatusExpired  VerifyStatus = "expired"
)

type IdCardVerification struct {
	ID         uint         `json:"id"`
	UserID     uint         `json:"user_id"`
	Document   string       `json:"document"`
	SelfieURL  string       `json:"selfie_url"`
	Status     VerifyStatus `json:"status"`
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
	Document   string       `json:"document"`
	Status     VerifyStatus `json:"status"`
	User       User         `gorm:"foreignKey:UserID"`
	VerifiedAt *time.Time   `json:"verified_at,omitempty"`
	ReviewedBy *uint        `json:"reviewed_by,omitempty"`
	gorm.Model
}

type KYCUploadSession struct {
	ID        uint         `json:"primaryKey"`
	Token     string       `json:"type:varchar(100);uniqueIndex;not null"` // Token แบบสุ่ม (เช่น UUID)
	UserID    uint         `json:"not null;index"`                         // ผูกกับ User ที่กดขอ QR Code
	Status    VerifyStatus `json:"type:varchar(20);default:'pending'"`     // pending, approved, expired
	ExpiresAt time.Time    `json:"not null"`                               // เวลาหมดอายุ
	CreatedAt time.Time
	UpdatedAt time.Time
}

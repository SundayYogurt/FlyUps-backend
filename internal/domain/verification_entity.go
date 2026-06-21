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
)

type IdCardVerification struct {
	ID         uint         `json:"id"`
	UserID     uint         `json:"user_id"`
	Document   string       `json:"document"`
	SelfieURL  *string      `json:"selfie_url"`
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

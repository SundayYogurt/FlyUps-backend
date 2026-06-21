package domain

import (
	"time"

	"gorm.io/gorm"
)

type NotificationPrefsMap map[string]bool

type UserStatus string

const (
	ACTIVE    UserStatus = "active"
	SUSPENDED UserStatus = "suspended"
)

type User struct {
	ID                         uint                     `json:"id"`
	Email                      string                   `json:"email"`
	PasswordHash               string                   `json:"-"`
	GoogleSub                  *string                  `json:"google_sub,omitempty"`
	HasPassword                *bool                    `json:"has_password" gorm:"-"`
	FirstName                  string                   `json:"first_name"`
	LastName                   string                   `json:"last_name"`
	Phone                      string                   `json:"phone"`
	Address                    *string                  `json:"address,omitempty"`
	Status                     UserStatus               `json:"status"`
	Role                       string                   `json:"role"`
	Picture                    *string                  `json:"picture,omitempty"`
	EmailVerifiedAt            *time.Time               `json:"email_verified_at,omitempty"`
	VerificationToken          *string                  `json:"-"`
	VerificationTokenExpiresAt *time.Time               `json:"-"`
	ResetTokenHash             *string                  `json:"-"`
	ResetTokenExpiresAt        *time.Time               `json:"-"`
	StudentProfile             *StudentProfile          `json:"student_profile,omitempty" gorm:"foreignKey:UserID"`
	BankAccounts               []BankAccount            `json:"bank_accounts,omitempty" gorm:"foreignKey:UserID"`
	StudentCardVerification    *StudentCardVerification `json:"student_card_verification,omitempty" gorm:"foreignKey:UserID"`
	IdCardVerification         *IdCardVerification      `json:"id_card_verification,omitempty" gorm:"foreignKey:UserID"`
	SuspendReason              *string                  `json:"suspend_reason,omitempty"`
	SuspendedBy                *uint                    `json:"suspended_by,omitempty"`
	SuspendedAt                *time.Time               `json:"suspended_at,omitempty"`
	NotificationPreferences    NotificationPrefsMap     `json:"notification_preferences" gorm:"serializer:json;type:json;default:'{}'"`
	gorm.Model
}

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

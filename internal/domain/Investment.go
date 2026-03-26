package domain

import (
	"time"

	"gorm.io/gorm"
)

type Investment struct {
	ID              uint   `json:"id"`
	ReferenceNumber string `json:"reference_number" gorm:"uniqueIndex"`
	UserID          uint   `json:"user_id"`
	User            User   `json:"user" gorm:"foreignKey:UserID"`
	ProjectID       uint   `json:"project_id"`
	// Project         Project    `json:"project" gorm:"foreignKey:ProjectID"`
	Amount         float64    `json:"amount"`
	PlatformFee    float64    `json:"platform_fee"`
	VATAmount      float64    `json:"vat_amount"`
	NetAmount      float64    `json:"net_amount"`
	ProfitSharePct float64    `json:"profit_share_pct"`
	Status         string     `json:"status" form:"default:'pending'"`
	PaidAt         *time.Time `json:"paid_at,omitempty"`
	gorm.Model
}

type Transaction struct {
	ID                    uint       `json:"id"`
	InvestmentID          uint       `json:"investment_id"`
	Investment            Investment `json:"investment" gorm:"foreignKey:InvestmentID"`
	StripePaymentIntentID string     `json:"stripe_payment_intent_id"`
	StripeClientSecret    string     `json:"-"`
	QRCodeImageURL        string     `json:"qr_code_image_url"`
	ExpiresAt             time.Time  `json:"expires_at"`
	Status                string     `json:"status" gorm:"default:'pending'"`
	gorm.Model
}

const (
	// Investment Status
	InvestmentStatusPending   = "pending"
	InvestmentStatusPaid      = "paidd"
	InvestmentStatusFailed    = "failed"
	InvestmentStatusCancelled = "cancelled"

	// Transaction Status
	TransactionStatusPending   = "pending"
	TransactionStatusSucceeded = "succeeded"
	TransactionStatusFailed    = "failed"
	TransactionStatusExpired   = "expired"
)

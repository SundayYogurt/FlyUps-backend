package domain

import (
	"time"

	"gorm.io/gorm"
)

type TransactionStatus string

const (
	TransactionPending   TransactionStatus = "pending"
	TransactionSucceeded TransactionStatus = "succeeded"
	TransactionFailed    TransactionStatus = "failed"
	TransactionExpired   TransactionStatus = "expired"
)

type Transaction struct {
	ID                    uint              `json:"id"`
	InvestmentID          uint              `json:"investment_id"`
	Investment            Investment        `json:"investment,omitempty" gorm:"foreignKey:InvestmentID"`
	StripePaymentIntentID string            `json:"stripe_payment_intent_id"`
	StripeClientSecret    string            `json:"-"`
	QRCodeImageURL        string            `json:"qr_code_image_url"`
	ExpiresAt             time.Time         `json:"expires_at"`
	Status                TransactionStatus `json:"status" gorm:"default:'pending'"`
	gorm.Model
}

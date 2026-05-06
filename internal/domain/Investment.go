package domain

import (
	"time"

	"gorm.io/gorm"
)

type Investment struct {
	ID              uint             `json:"id"`
	ReferenceNumber string           `json:"reference_number" gorm:"uniqueIndex"`
	ProjectID       uint             `json:"project_id"`
	Project         *Project         `json:"project,omitempty" gorm:"foreignKey:ProjectID"`
	BoosterUserID   uint             `json:"booster_user_id"`
	TotalAmount     float64          `json:"total_amount"`
	PlatformFee     float64          `json:"platform_fee"`
	VATAmount       float64          `json:"vat_amount"`
	PrincipalAmount float64          `json:"principal_amount"`
	ProfitSharePct  float64          `json:"profit_share_pct"`
	Status          InvestmentStatus `json:"status" gorm:"default:'pending_payment'"`
	PaidAt          *time.Time       `json:"paid_at,omitempty"`
	RefundAmount    float64          `json:"refund_amount"`
	RefundNote      string           `json:"refund_note"`
	RefundedAt      *time.Time       `json:"refunded_at,omitempty"`
	gorm.Model
}

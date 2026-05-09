package domain

import (
	"time"

	"gorm.io/gorm"
)

type ProfitPoolStatus string
type InvestorPayoutStatus string

const (
	ProfitPoolPending   ProfitPoolStatus = "pending"
	ProfitPoolCompleted ProfitPoolStatus = "completed"
)

const (
	InvestorPayoutPending   InvestorPayoutStatus = "pending"
	InvestorPayoutConfirmed InvestorPayoutStatus = "confirmed"
)

// ProfitPool records a profit transfer from a pioneer for a project
type ProfitPool struct {
	ID            uint             `json:"id"`
	ProjectID     uint             `json:"project_id" gorm:"index"`
	PioneerUserID uint             `json:"pioneer_user_id" gorm:"index"`
	TotalAmount   float64          `json:"total_amount"`
	TransferRef   string           `json:"transfer_ref"`
	Status        ProfitPoolStatus `json:"status" gorm:"default:'pending'"`
	AdminNote     string           `json:"admin_note"`
	QuarterNo     int              `json:"quarter_no" gorm:"default:0"` // 0 = unspecified, 1-4 = quarterly payment
	gorm.Model
}

// InvestorProfitPayout records a per-investor payout from a profit pool
type InvestorProfitPayout struct {
	ID            uint                 `json:"id"`
	ProfitPoolID  uint                 `json:"profit_pool_id" gorm:"index"`
	ProjectID     uint                 `json:"project_id" gorm:"index"`
	BoosterUserID uint                 `json:"booster_user_id" gorm:"index"`
	Amount        float64              `json:"amount"`
	SharePct      float64              `json:"share_pct"`
	Status        InvestorPayoutStatus `json:"status" gorm:"default:'pending'"`
	TransferRef   string               `json:"transfer_ref"`
	AdminNote     string               `json:"admin_note"`
	ConfirmedAt   *time.Time           `json:"confirmed_at,omitempty"`
	ConfirmedBy   *uint                `json:"confirmed_by,omitempty"`
	gorm.Model
}

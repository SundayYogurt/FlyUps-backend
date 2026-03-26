package domain

import (
	"time"

	"gorm.io/gorm"
)

type ProjectFundingConfig struct {
	ID              uint    `json:"id"`
	ProjectID       uint    `json:"project_id" gorm:"uniqueIndex"` // 1 project ต่อ 1 config เท่านั้น
	ProfitSharePct  float64 `json:"profit_share_pct"`              // % กำไรที่ผู้ลงทุนได้รับ เช่น 15 = 15%
	MinInvestAmount float64 `json:"min_invest_amount"`             // จำนวนเงินลงทุนขั้นต่ำ เช่น 1000
	MaxInvestAmount float64 `json:"max_invest_amount"`             // จำนวนเงินลงทุนขั้นสูง เช่น 14000
	PlatformFeePct  float64 `json:"platform_fee_pct"`              // ค่าธรรมเนียมแพลตฟอร์ม เช่น 5 = 5%
	gorm.Model
}

type Investment struct {
	ID              uint             `json:"id"`
	ReferenceNumber string           `json:"reference_number" gorm:"uniqueIndex"`
	ProjectID       uint             `json:"project_id"`
	BoosterUserID   uint             `json:"booster_user_id"`
	TotalAmount     float64          `json:"total_amount"`
	PlatformFee     float64          `json:"platform_fee"`
	VATAmount       float64          `json:"vat_amount"`
	PrincipalAmount float64          `json:"principal_amount"`
	ProfitSharePct  float64          `json:"profit_share_pct"`
	Status          InvestmentStatus `json:"status" gorm:"default:'pending_payment'"`
	PaidAt          *time.Time       `json:"paid_at,omitempty"`
	gorm.Model
}

package dto

import "time"

type CreateInvestmentRequest struct {
	ProjectID uint    `json:"project_id" validate:"required"`
	Amount    float64 `json:"amount" validate:"required,gt=0"`
}

type RefundInvestmentRequest struct {
	Note string `json:"note" validate:"required"`
}

type InvestmentTermsResponse struct {
	ProjectID       uint    `json:"project_id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	ProfitSharePct  float64 `json:"profit_share_pct"`
	MinInvestAmount float64 `json:"min_invest_amount"`
	MaxInvestAmount float64 `json:"max_invest_amount"`
	PlatformFeePct  float64 `json:"platform_fee_pct"`
	FundingGoal     float64 `json:"funding_goal"`
}

type InvestmentSummary struct {
	ProjectID       uint    `json:"project_id"`
	Title           string  `json:"title"`
	TotalAmount     float64 `json:"total_amount"`
	PlatformFee     float64 `json:"platform_fee"`
	VATAmount       float64 `json:"vat_amount"`
	PrincipalAmount float64 `json:"principal_amount"`
	ProfitSharePct  float64 `json:"profit_share_pct"`
}

type InvestmentResponse struct {
	InvestmentID    uint    `json:"investment_id"`
	ReferenceNumber string  `json:"reference_number"`
	QRCodeImageURL  string  `json:"qr_code_image_url"`
	ExpiresAt       string  `json:"expires_at"`
	TotalAmount     float64 `json:"total_amount"`
	Title           string  `json:"title"`
}

type InvestmentDetailResponse struct {
	Investment  interface{} `json:"investment"`
	Transaction interface{} `json:"transaction"`
}

type RefundResponse struct {
	InvestmentID    uint    `json:"investment_id"`
	ReferenceNumber string  `json:"reference_number"`
	RefundAmount    float64 `json:"refund_amount"`
	TotalPaid       float64 `json:"total_paid"`
	FeesDeducted    float64 `json:"fees_deducted"`
}

type RefundBankAccount struct {
	BankName      string `json:"bank_name"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
}

type RefundRequestItem struct {
	InvestmentID    uint               `json:"investment_id"`
	ReferenceNumber string             `json:"reference_number"`
	BoosterUserID   uint               `json:"booster_user_id"`
	BoosterName     string             `json:"booster_name"`
	RefundAmount    float64            `json:"refund_amount"`
	TotalPaid       float64            `json:"total_paid"`
	RequestedAt     string             `json:"requested_at"`
	BankAccount     *RefundBankAccount `json:"bank_account"`
}

type ProjectInvestorItem struct {
	UserID          uint       `json:"user_id"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	Email           string     `json:"-"`
	Picture         *string    `json:"picture,omitempty"`
	PrincipalAmount float64    `json:"principal_amount"`
	TotalAmount     float64    `json:"total_amount"`
	InvestmentCount int        `json:"investment_count"`
	FirstInvestedAt *time.Time `json:"first_invested_at"`
}

type InvestedProjectItem struct {
	ProjectID       uint       `json:"project_id"`
	Title           string     `json:"title"`
	Status          string     `json:"state"`
	CoverImage      *string    `json:"cover_image"`
	ProfitSharePct  float64    `json:"profit_share_pct"`
	TotalAmount     float64    `json:"total_amount"`
	PrincipalAmount float64    `json:"principal_amount"`
	InvestmentCount int        `json:"investment_count"`
	FirstInvestedAt *time.Time `json:"first_invested_at"`
}

type VoteMilestoneRequest struct {
	Choice string `json:"choice" validate:"required,oneof=approve reject"`
}

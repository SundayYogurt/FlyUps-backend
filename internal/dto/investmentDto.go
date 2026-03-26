package dto

type CreateInvestmentRequest struct {
	ProjectID uint    `json:"project_id" validate:"required"`
	Amount    float64 `json:"amount" validate:"required,gt=0"`
}

type InvestmentTermsResponse struct {
	ProjectID       uint    `json:"project_id"`
	ProjectName     string  `json:"project_name"`
	Description     string  `json:"description"`
	ProfitSharePct  float64 `json:"profit_share_pct"`
	MinInvestAmount float64 `json:"min_invest_amount"`
	MaxInvestAmount float64 `json:"max_invest_amount"`
	PlatformFeePct  float64 `json:"platform_fee_pct"`
	GoalAmount      float64 `json:"goal_amount"`
	FundedAmount    float64 `json:"funded_amount"`
}

type InvestmentSummary struct {
	ProjectID      uint    `json:"project_id"`
	ProjectName    string  `json:"project_name"`
	Amount         float64 `json:"amount"`
	PlatformFee    float64 `json:"platform_fee"`
	VATAmount      float64 `json:"vat_amount"`
	NetAmount      float64 `json:"net_amount"`
	ProfitSharePct float64 `json:"profit_share_pct"`
}

type InvestmentResponse struct {
	InvestmentID    uint    `json:"investment_id"`
	ReferenceNumber string  `json:"reference_number"`
	QRCodeImageURL  string  `json:"qr_code_image_url"`
	ExpiresAt       string  `json:"expires_at"`
	Amount          float64 `json:"amount"`
	ProjectName     string  `json:"project_name"`
}

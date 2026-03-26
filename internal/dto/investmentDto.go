package dto

type CreateInvestmentRequest struct {
	ProjectID uint    `json:"project_id" validate:"required"`
	Amount    float64 `json:"amount" validate:"required,gt=0"`
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

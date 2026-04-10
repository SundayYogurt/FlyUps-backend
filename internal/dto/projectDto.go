package dto

type ProjectResponse struct {
	ID              uint    `json:"id"`
	OwnerUserID     uint    `json:"owner_user_id"`
	Category        *string `json:"category"`
	Title           string  `json:"title"`
	Description     *string `json:"description"`
	State           string  `json:"state"`
	Status          string  `json:"status"`
	Visibility      string  `json:"visibility"`
	FundingGoal     float64 `json:"funding_goal"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
	ProfitSharePct  float64 `json:"profit_share_pct"`
	MinInvestAmount float64 `json:"min_invest_amount"`
	MaxInvestAmount float64 `json:"max_invest_amount"`
	PlatformFee     float64 `json:"platform_fee"`
}

package dto

type FinancialStripeBalance struct {
	Available float64 `json:"available"`
	Pending   float64 `json:"pending"`
}

type FinancialDisbursementStat struct {
	TotalConfirmed float64 `json:"total_confirmed"`
	TotalPending   float64 `json:"total_pending"`
	CountConfirmed int64   `json:"count_confirmed"`
	CountPending   int64   `json:"count_pending"`
}

type FinancialRefundStat struct {
	TotalPending  float64 `json:"total_pending"`
	TotalRefunded float64 `json:"total_refunded"`
	CountPending  int64   `json:"count_pending"`
	CountRefunded int64   `json:"count_refunded"`
}

type FinancialPlatformStat struct {
	TotalFees    float64 `json:"total_fees"`
	TotalRevenue float64 `json:"total_revenue"`
}

type FinancialSummary struct {
	Stripe       FinancialStripeBalance    `json:"stripe"`
	Disbursement FinancialDisbursementStat `json:"disbursement"`
	Refund       FinancialRefundStat       `json:"refund"`
	Platform     FinancialPlatformStat     `json:"platform"`
}

type PhaseFinancialItem struct {
	PhaseNo        int     `json:"phase_no"`
	Title          string  `json:"title"`
	PercentRelease int     `json:"percent_release"`
	Amount         float64 `json:"amount"`
	Status         string  `json:"status"`
	ConfirmedAt    *string `json:"confirmed_at"`
}

type ProjectFinancialItem struct {
	ProjectID    uint                 `json:"project_id"`
	ProjectTitle string               `json:"project_title"`
	State        string               `json:"state"`
	FundingGoal  float64              `json:"funding_goal"`
	CurrentFund  float64              `json:"current_funding"`
	Phases       []PhaseFinancialItem `json:"phases"`
}

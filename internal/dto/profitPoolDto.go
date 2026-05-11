package dto

import "time"

type CreateProfitPoolRequest struct {
	ProjectID   uint    `json:"project_id" validate:"required"`
	TotalAmount float64 `json:"total_amount" validate:"required,gt=0"`
	TransferRef string  `json:"transfer_ref" validate:"required"`
	AdminNote   string  `json:"admin_note"`
	QuarterNo   int     `json:"quarter_no" validate:"min=0,max=4"`
}

type PioneerSubmitProfitRequest struct {
	TotalAmount float64 `json:"total_amount" validate:"required,gt=0"`
	TransferRef string  `json:"transfer_ref" validate:"required"`
	QuarterNo   int     `json:"quarter_no" validate:"required,min=1,max=4"`
	SlipImage   string  `json:"slip_image"`
}

type ConfirmInvestorPayoutRequest struct {
	TransferRef string `json:"transfer_ref" validate:"required"`
	Note        string `json:"note"`
}

type InvestorPayoutDetail struct {
	ID              uint                     `json:"id"`
	BoosterUserID   uint                     `json:"booster_user_id"`
	FirstName       string                   `json:"first_name"`
	LastName        string                   `json:"last_name"`
	Email           string                   `json:"email"`
	PrincipalAmount float64                  `json:"principal_amount"`
	Amount          float64                  `json:"amount"`
	SharePct        float64                  `json:"share_pct"`
	Status          string                   `json:"status"`
	TransferRef     string                   `json:"transfer_ref"`
	AdminNote       string                   `json:"admin_note"`
	ConfirmedAt     *time.Time               `json:"confirmed_at,omitempty"`
	BankAccount     *DisbursementBankAccount `json:"bank_account,omitempty"`
}

type ProfitPoolDetail struct {
	ID            uint                   `json:"id"`
	ProjectID     uint                   `json:"project_id"`
	ProjectTitle  string                 `json:"project_title"`
	PioneerUserID uint                   `json:"pioneer_user_id"`
	PioneerName   string                 `json:"pioneer_name"`
	TotalAmount   float64                `json:"total_amount"`
	TransferRef   string                 `json:"transfer_ref"`
	Status        string                 `json:"status"`
	AdminNote     string                 `json:"admin_note"`
	QuarterNo     int                    `json:"quarter_no"`
	CreatedAt     time.Time              `json:"created_at"`
	Payouts       []InvestorPayoutDetail `json:"payouts"`
}

type MyProfitPayoutItem struct {
	ID           uint       `json:"id"`
	ProjectID    uint       `json:"project_id"`
	ProjectTitle string     `json:"project_title"`
	CoverImage   string     `json:"cover_image"`
	QuarterNo    int        `json:"quarter_no"`
	Amount       float64    `json:"amount"`
	SharePct     float64    `json:"share_pct"`
	Status       string     `json:"status"`
	TransferRef  string     `json:"transfer_ref"`
	ConfirmedAt  *time.Time `json:"confirmed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ProfitPoolListItem struct {
	ID             uint      `json:"id"`
	ProjectID      uint      `json:"project_id"`
	ProjectTitle   string    `json:"project_title"`
	PioneerName    string    `json:"pioneer_name"`
	TotalAmount    float64   `json:"total_amount"`
	Status         string    `json:"status"`
	InvestorCount  int       `json:"investor_count"`
	ConfirmedCount int       `json:"confirmed_count"`
	QuarterNo      int       `json:"quarter_no"`
	CreatedAt      time.Time `json:"created_at"`
}

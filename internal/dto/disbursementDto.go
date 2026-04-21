package dto

import "time"

type DisbursementBankAccount struct {
	BankName      string `json:"bank_name"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
}

type DisbursementItem struct {
	ID             uint                     `json:"id"`
	MilestoneID    uint                     `json:"milestone_id"`
	ProjectID      uint                     `json:"project_id"`
	ProjectTitle   string                   `json:"project_title"`
	PioneerUserID  uint                     `json:"pioneer_user_id"`
	PioneerName    string                   `json:"pioneer_name"`
	PioneerEmail   string                   `json:"pioneer_email"`
	PhaseNo        int                      `json:"phase_no"`
	PercentRelease int                      `json:"percent_release"`
	Amount         float64                  `json:"amount"`
	Status         string                   `json:"status"`
	TransferRef    string                   `json:"transfer_ref"`
	AdminNote      string                   `json:"admin_note"`
	CreatedAt      time.Time                `json:"created_at"`
	ConfirmedAt    *time.Time               `json:"confirmed_at,omitempty"`
	BankAccount    *DisbursementBankAccount `json:"bank_account"`
}

type ConfirmDisbursementRequest struct {
	TransferRef string `json:"transfer_ref" validate:"required"`
	Note        string `json:"note"`
}

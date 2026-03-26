package dto

type ProjectResponse struct {
	ID          uint    `json:"id"`
	OwnerUserID uint    `json:"owner_user_id"`
	Category    *string `json:"category"`
	Title       string  `json:"title"`
	Description *string `json:"description"`

	State      string `json:"state"`
	Status     string `json:"status"`
	Visibility string `json:"visibility"`

	FundingGoal float64 `json:"funding_goal"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

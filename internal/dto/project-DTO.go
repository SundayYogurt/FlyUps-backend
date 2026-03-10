package dto

type UpdateProjectDraftRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	FundingGoal float64 `json:"funding_goal"`
}

type AddProjectMediaRequest struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}

package dto

import "time"

type UpdateProjectDraftRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	FundingGoal float64 `json:"funding_goal"`
}

type AddProjectMediaRequest struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}

type AddProjectStoryRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type AddProjectRiskRequest struct {
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Severity   string `json:"severity"`
	Mitigation string `json:"mitigation"`
}

type AddProjectFAQRequest struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type SetFundingPolicyRequest struct {
	FundingModel  string  `json:"funding_model"`
	SoftCapAmount float64 `json:"soft_cap_amount"`
	HardCapAmount float64 `json:"hard_cap_amount"`
}

type SetProfitPolicyRequest struct {
	DistributionFrequency string `json:"distribution_frequency"`
	MinimumDurationMonths int    `json:"minimum_duration_months"`
	TotalQuarters         int    `json:"total_quarters"`
	ExpectedStartDate     string `json:"expected_start_date"`
	Note                  string `json:"note"`
}

type CreateMilestoneRequest struct {
	PhaseNo        int    `json:"phase_no"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	PercentRelease int    `json:"percent_release"`
}

type SubmitProjectRequest struct {
	Confirm bool `json:"confirm"`
}

type ProjectResponse struct {
	ID          uint    `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	FundingGoal float64 `json:"funding_goal"`
	State       string  `json:"state"`
	Status      string  `json:"status"`
	Visibility  string  `json:"visibility"`
}

type ProjectDetailResponse struct {
	ID            uint                   `json:"id"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description"`
	FundingGoal   float64                `json:"funding_goal"`
	State         string                 `json:"state"`
	Media         []ProjectMediaResponse `json:"media"`
	Story         []StoryResponse        `json:"story"`
	Risks         []RiskResponse         `json:"risks"`
	FAQ           []FAQResponse          `json:"faq"`
	Milestones    []MilestoneResponse    `json:"milestones"`
	FundingPolicy FundingPolicyResponse  `json:"funding_policy"`
	ProfitPolicy  *ProfitPolicyResponse  `json:"profit_policy"`
}

type ProfitPolicyResponse struct {
	DistributionFrequency string     `json:"distribution_frequency"`
	MinimumDurationMonths int        `json:"minimum_duration_months"`
	TotalQuarters         int        `json:"total_quarters"`
	ExpectedStartDate     *time.Time `json:"expected_start_date"`
	Note                  string     `json:"note"`
}

type ProjectMediaResponse struct {
	ID   uint   `json:"id"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

type RiskResponse struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
}

type StoryResponse struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type FAQResponse struct {
	ID       uint   `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type MilestoneResponse struct {
	ID             uint    `json:"id"`
	PhaseNo        int     `json:"phase_no"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	PercentRelease float64 `json:"percent_release"`
}

type FundingPolicyResponse struct {
	SoftCapAmount   float64 `json:"soft_cap_amount"`
	HardCapAmount   float64 `json:"hard_cap_amount"`
	MinInvestAmount float64 `json:"min_invest_amount"`
	MaxInvestAmount float64 `json:"max_invest_amount"`
}

type ProjectValidateResponse struct {
	Ready   bool     `json:"ready"`
	Missing []string `json:"missing"`
}

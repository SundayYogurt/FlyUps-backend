package dto

import "time"

type CreateComplaintRequest struct {
	ProjectID uint   `json:"project_id" validate:"required"`
	Subject   string `json:"subject"    validate:"required,min=3,max=200"`
	Evidence  string `json:"evidence"   validate:"omitempty,url"`
	Body      string `json:"body"       validate:"required,min=10,max=5000"`
}

type ResolveComplaintRequest struct {
	AdminNote string `json:"admin_note" validate:"required,min=3,max=2000"`
}

type ComplaintComplainant struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type ComplaintProject struct {
	ID    uint   `json:"id"`
	Title string `json:"title"`
	State string `json:"state"`
}

type ComplaintItem struct {
	ID              uint                  `json:"id"`
	ComplainantID   uint                  `json:"complainant_id"`
	ProjectID       uint                  `json:"project_id"`
	Subject         string                `json:"subject"`
	Body            string                `json:"body"`
	Status          string                `json:"status"`
	AdminNote       string                `json:"admin_note"`
	ResolvedAt      *time.Time            `json:"resolved_at,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	Complainant     *ComplaintComplainant `json:"complainant,omitempty"`
	Project         *ComplaintProject     `json:"project,omitempty"`
	TotalReports    int64                 `json:"total_reports"`
	ResolvedReports int64                 `json:"resolved_reports"`
}

type ProjectComplaintStats struct {
	ProjectID       uint   `json:"project_id"`
	ProjectTitle    string `json:"project_title"`
	TotalReports    int64  `json:"total_reports"`
	ResolvedReports int64  `json:"resolved_reports"`
	Threshold       int    `json:"threshold"`
}

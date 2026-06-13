package domain

import "gorm.io/gorm"

type NotificationType string

const (
	NotifNewInvestment          NotificationType = "new_investment"
	NotifPaymentFailed          NotificationType = "payment_failed"
	NotifMilestone              NotificationType = "milestone"
	NotifMilestoneSubmitted     NotificationType = "milestone_submitted"
	NotifMilestoneRejected      NotificationType = "milestone_rejected"
	NotifVote                   NotificationType = "vote"
	NotifProjectStatus          NotificationType = "project_status"
	NotifProfit                 NotificationType = "profit"
	NotifMeeting                NotificationType = "meeting"
	NotifVerificationApproved   NotificationType = "verification_approved"
	NotifVerificationRejected   NotificationType = "verification_rejected"
)

type Notification struct {
	ID          uint             `json:"id"`
	UserID      uint             `json:"user_id"`
	Type        NotificationType `json:"type"`
	Title       string           `json:"title"`
	Body        string           `json:"body"`
	IsRead      bool             `json:"is_read" gorm:"default:false"`
	RelatedID   *uint            `json:"related_id,omitempty"`
	RelatedType *string          `json:"related_type,omitempty"`
	gorm.Model
}

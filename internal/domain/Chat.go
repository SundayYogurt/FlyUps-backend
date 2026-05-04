package domain

import "gorm.io/gorm"

type ChatSessionStatus string

const (
	ChatSessionStatusActive ChatSessionStatus = "active"
	ChatSessionStatusClosed ChatSessionStatus = "closed"
)

type ChatMessageRole string

const (
	ChatMessageRoleUser      ChatMessageRole = "user"
	ChatMessageRoleAssistant ChatMessageRole = "assistant"
	ChatMessageRoleSystem    ChatMessageRole = "system"
)

type ChatIntent string

const (
	ChatIntentUnknown           ChatIntent = "unknown"
	ChatIntentRefundPolicy      ChatIntent = "refund_policy"
	ChatIntentRefundRequest     ChatIntent = "refund_request"
	ChatIntentCancelTransaction ChatIntent = "cancel_transaction"
	ChatIntentTransactionStatus ChatIntent = "transaction_status"
	ChatIntentLatestTransaction ChatIntent = "latest_transaction"
)

type ChatSession struct {
	ID     uint              `json:"id"`
	UserID uint              `json:"user_id"`
	Status ChatSessionStatus `json:"status" gorm:"type:varchar(32);default:'active'"`
	gorm.Model
}

type ChatMessage struct {
	ID        uint            `json:"id"`
	SessionID uint            `json:"session_id"`
	Session   ChatSession     `json:"session" gorm:"foreignKey:SessionID"`
	Role      ChatMessageRole `json:"role" gorm:"type:varchar(32)"`
	Content   string          `json:"content" gorm:"type:text"`
	Intent    ChatIntent      `json:"intent" gorm:"type:varchar(64);default:'unknown'"`
	gorm.Model
}

type ChatActionType string

const (
	// Query actions — execute ทันที ไม่ต้อง confirm
	ChatActionTypeGetInvestments   ChatActionType = "get_my_investments"
	ChatActionTypeGetProjects      ChatActionType = "get_recommended_projects"
	ChatActionTypeGetProjectDetail ChatActionType = "get_project_detail"
	ChatActionTypeGetEndingSoon    ChatActionType = "get_projects_ending_soon"
	ChatActionTypeGetNewProjects   ChatActionType = "get_new_projects"
	ChatActionTypeGetNotifications ChatActionType = "get_my_notifications"
	ChatActionTypeGetDisbursements ChatActionType = "get_my_disbursements"

	// Action tools — ต้อง confirm ก่อน
	ChatActionTypeRefundTransaction ChatActionType = "refund_transaction"
	ChatActionTypeCancelTransaction ChatActionType = "cancel_transaction"
	ChatActionTypeVoteMilestone     ChatActionType = "vote_milestone"
	ChatActionTypeFileComplaint     ChatActionType = "file_complaint"
	ChatActionTypeMarkNotifRead     ChatActionType = "mark_notifications_read"
)

type ChatActionStatus string

const (
	ChatActionStatusPending   ChatActionStatus = "pending"
	ChatActionStatusConfirmed ChatActionStatus = "confirmed"
	ChatActionStatusRejected  ChatActionStatus = "rejected"
	ChatActionStatusExpired   ChatActionStatus = "expired"
	ChatActionStatusCompleted ChatActionStatus = "completed"
	ChatActionStatusFailed    ChatActionStatus = "failed"
)

type ChatAction struct {
	ID        uint `json:"id"`
	SessionID uint `json:"session_id"`
	UserID    uint `json:"user_id"`

	Session ChatSession `json:"session" gorm:"foreignKey:SessionID"`

	Type   ChatActionType   `json:"type" gorm:"type:varchar(64)"`
	Status ChatActionStatus `json:"status" gorm:"type:varchar(32);default:'pending'"`

	// InvestmentID คือ investment ที่จะ refund/cancel
	// ตั้งชื่อ column ว่า transaction_id เพื่อ backward compat กับ DB ที่สร้างไปแล้ว
	InvestmentID *uint   `json:"investment_id" gorm:"column:transaction_id"`
	Payload      *string `json:"payload,omitempty" gorm:"type:text"`

	gorm.Model
}

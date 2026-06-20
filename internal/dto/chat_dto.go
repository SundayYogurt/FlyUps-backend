package dto

import "flyup/internal/domain"

type CreateChatSessionRequest struct {
	Title *string `json:"title,omitempty"`
}

type ChatSessionResponse struct {
	ID     uint                     `json:"id"`
	UserID uint                     `json:"user_id"`
	Status domain.ChatSessionStatus `json:"status"`
}

// SendChatMessageRequest ใช้ตอน user ส่งข้อความเข้า
type SendChatMessageRequest struct {
	SessionID *uint  `json:"session_id,omitempty"`
	Message   string `json:"message" validate:"required"`
}

// ChatMessageResponse ใช้ตอบกลับ user พร้อมข้อมูลครบ
type ChatMessageResponse struct {
	ID        uint                   `json:"id"`
	SessionID uint                   `json:"session_id"`
	Role      domain.ChatMessageRole `json:"role"`
	Content   string                 `json:"content"`
	Intent    domain.ChatIntent      `json:"intent"`
}

// ChatActionResponse
type ChatActionResponse struct {
	ID           uint                    `json:"id"`
	SessionID    uint                    `json:"session_id"`
	Type         domain.ChatActionType   `json:"type"`
	Status       domain.ChatActionStatus `json:"status"`
	InvestmentID *uint                   `json:"investment_id,omitempty"`
}

type SendChatMessageResponse struct {
	Session ChatSessionResponse `json:"session"`
	Message ChatMessageResponse `json:"message"`
	Action  *ChatActionResponse `json:"action,omitempty"`
	Reply   ChatMessageResponse `json:"reply"`
}

// ConfirmChatActionRequest ใช้ตอน user กดยืนยัน/ปฏิเสธ action
type ConfirmChatActionRequest struct {
	ActionID uint `json:"action_id" validate:"required"`
	Confirm  bool `json:"confirm" validate:"required"`
}

type ConfirmChatActionResponse struct {
	Action ChatActionResponse  `json:"action"`
	Reply  ChatMessageResponse `json:"reply"`
}

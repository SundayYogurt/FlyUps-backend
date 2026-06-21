package domain

import "time"

type ThreadStatus string

const (
	ThreadOpen     ThreadStatus = "open"
	ThreadAnswered ThreadStatus = "answered"
	ThreadHidden   ThreadStatus = "hidden"
	ThreadLocked   ThreadStatus = "locked"
)

type MessageType string

const (
	MessageText MessageType = "text"
	MessageFile MessageType = "file"
	MessageLink MessageType = "link"
)

type ProjectThread struct {
	ID         uint         `json:"id"`
	ProjectID  uint         `json:"project_id"`
	UpdateID   *uint        `json:"update_id,omitempty"`
	Type       string       `json:"type"`
	CreatedBy  uint         `json:"created_by"`
	Title      *string      `json:"title,omitempty"`
	Body       string       `json:"body"`
	Status     ThreadStatus `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
	UserName   string       `json:"user_name" gorm:"column:user_name;<-:false"`
	UserAvatar *string      `json:"user_avatar,omitempty" gorm:"column:user_avatar;<-:false"`
}

type ProjectThreadMessage struct {
	ID         uint         `json:"id"`
	ProjectID  uint         `json:"project_id"`
	ThreadID   uint         `json:"thread_id"`
	Type       string       `json:"type"`
	CreatedBy  uint         `json:"created_by"`
	Body       string       `json:"body"`
	Status     ThreadStatus `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
	UserName   string       `json:"user_name" gorm:"column:user_name;<-:false"`
	UserAvatar *string      `json:"user_avatar,omitempty" gorm:"column:user_avatar;<-:false"`
}

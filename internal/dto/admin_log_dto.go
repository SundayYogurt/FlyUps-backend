package dto

import "time"

// AdminLogFilter query params สำหรับ list admin logs
type AdminLogFilter struct {
	Page       int    `query:"page"`
	PageSize   int    `query:"page_size"`
	AdminID    *uint  `query:"admin_id"`
	Action     string `query:"action"`
	TargetType string `query:"target_type"`
	TargetID   *uint  `query:"target_id"`
	From       string `query:"from"` // ISO 8601 date string เช่น "2026-01-01"
	To         string `query:"to"`   // ISO 8601 date string เช่น "2026-12-31"
}

// AdminLogAdmin ข้อมูล admin ที่แสดงใน log
type AdminLogAdmin struct {
	ID        uint   `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// AdminLogItem DTO สำหรับ response แต่ละ log entry
type AdminLogItem struct {
	ID         uint           `json:"id"`
	AdminID    uint           `json:"admin_id"`
	Admin      *AdminLogAdmin `json:"admin,omitempty"`
	Action     string         `json:"action"`
	TargetType string         `json:"target_type"`
	TargetID   *uint          `json:"target_id,omitempty"`
	Note       *string        `json:"note,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

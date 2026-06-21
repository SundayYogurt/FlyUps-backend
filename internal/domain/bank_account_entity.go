package domain

import "gorm.io/gorm"

type BankAccount struct {
	ID            uint   `json:"id"`
	UserID        uint   `json:"user_id"`
	BankName      string `json:"bank_name"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	IsDefault     bool   `json:"is_default" gorm:"default:false"`
	gorm.Model
}

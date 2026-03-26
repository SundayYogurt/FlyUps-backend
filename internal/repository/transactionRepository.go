package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type TransactionRepository interface {
	Create(txn *domain.Transaction) error
}

type transactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db}
}

func (r *transactionRepository) Create(txn *domain.Transaction) error {
	return r.db.Create(txn).Error
}

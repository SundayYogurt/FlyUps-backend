package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type TransactionRepository interface {
	Create(txn *domain.Transaction) error
	FindByPaymentIntentID(intentID string) (*domain.Transaction, error)
	FindByInvestmentID(investmentID uint) (*domain.Transaction, error)
	UpdateStatus(id uint, status domain.TransactionStatus) error
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

func (r *transactionRepository) FindByPaymentIntentID(intentID string) (*domain.Transaction, error) {
	txn := domain.Transaction{}

	err := r.db.Preload("Investment").Where("stripe_payment_intent_id = ?", intentID).First(&txn).Error

	if err != nil {
		return nil, err
	}

	return &txn, nil
}

func (r *transactionRepository) FindByInvestmentID(investmentID uint) (*domain.Transaction, error) {
	txn := domain.Transaction{}

	err := r.db.Where("investment_id = ?", investmentID).First(&txn).Error

	if err != nil {
		return nil, err
	}

	return &txn, nil
}

func (r *transactionRepository) UpdateStatus(id uint, status domain.TransactionStatus) error {
	return r.db.Model(&domain.Transaction{}).Where("id = ?", id).Update("status", status).Error
}

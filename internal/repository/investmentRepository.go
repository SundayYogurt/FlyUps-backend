package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

// interface

type InvestmentRepository interface {
	Create(investment *domain.Investment) error
}

type TransactionRepository interface {
	Create(tx *domain.Transaction) error
}

// struct

type investmentRepository struct {
	db *gorm.DB
}

type transactionRepository struct {
	db *gorm.DB
}

// constructors

func NewInvestmentRepository(db *gorm.DB) InvestmentRepository {
	return &investmentRepository{db}
}

func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{db}
}

// investmentrepository methods

func (r *investmentRepository) Create(investment *domain.Investment) error {
	return r.db.Create(investment).Error
}

// transactionrepository methods

func (r *transactionRepository) Create(tx *domain.Transaction) error {
	return r.db.Create(tx).Error
}

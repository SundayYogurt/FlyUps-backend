package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

// interface

type InvestmentRepository interface {
	Create(investment *domain.Investment) error
}

// struct

type investmentRepository struct {
	db *gorm.DB
}

// constructors

func NewInvestmentRepository(db *gorm.DB) InvestmentRepository {
	return &investmentRepository{db}
}

// investmentrepository methods

func (r *investmentRepository) Create(investment *domain.Investment) error {
	return r.db.Create(investment).Error
}

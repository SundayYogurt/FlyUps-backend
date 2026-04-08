package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type InvestmentRepository interface {
	Create(investment *domain.Investment) error
	FindByID(id uint) (*domain.Investment, error)
	FindByReferenceNumber(ref string) (*domain.Investment, error)
	UpdateStatus(id uint, status domain.InvestmentStatus) error
	UpdatePaid(investment *domain.Investment) error
	UpdateRefunded(investment *domain.Investment) error
	ListByBoosterUserID(boosterUserID uint) ([]domain.Investment, error)
	ListRefundPending() ([]domain.Investment, error)
	SumActiveByProjectID(projectID uint) (float64, error)
	IncrementProjectFunding(projectID uint, amount float64) error
}

type investmentRepository struct {
	db *gorm.DB
}

func NewInvestmentRepository(db *gorm.DB) InvestmentRepository {
	return &investmentRepository{db}
}

func (r *investmentRepository) Create(investment *domain.Investment) error {
	return r.db.Create(investment).Error
}

func (r *investmentRepository) FindByID(id uint) (*domain.Investment, error) {
	inv := &domain.Investment{}

	err := r.db.First(inv, id).Error
	if err != nil {
		return nil, err
	}
	return inv, nil
}

func (r *investmentRepository) FindByReferenceNumber(ref string) (*domain.Investment, error) {
	inv := &domain.Investment{}

	err := r.db.Where("reference_number = ?", ref).First(inv).Error

	if err != nil {
		return nil, err
	}

	return inv, nil
}

func (r *investmentRepository) UpdateStatus(id uint, status domain.InvestmentStatus) error {
	return r.db.Model(&domain.Investment{}).Where("id = ?", id).Update("status", status).Error
}

func (r *investmentRepository) UpdatePaid(investment *domain.Investment) error {
	return r.db.Save(investment).Error
}

func (r *investmentRepository) UpdateRefunded(investment *domain.Investment) error {
	return r.db.Save(investment).Error
}

func (r *investmentRepository) ListByBoosterUserID(boosterUserID uint) ([]domain.Investment, error) {
	inv := []domain.Investment{}

	err := r.db.Where("booster_user_id = ?", boosterUserID).Order("created_at desc").Find(&inv).Error

	return inv, err
}

func (r *investmentRepository) ListRefundPending() ([]domain.Investment, error) {
	var investments []domain.Investment
	err := r.db.Where("status = ?", domain.InvestmentRefundPending).Order("refunded_at ASC").Find(&investments).Error
	return investments, err
}

func (r *investmentRepository) IncrementProjectFunding(projectID uint, amount float64) error {
	return r.db.Model(&domain.Project{}).Where("id = ?", projectID).UpdateColumn("current_funding", gorm.Expr("current_funding + ?", amount)).Error
}

func (r *investmentRepository) SumActiveByProjectID(projectID uint) (float64, error) {
	var total float64
	err := r.db.Model(&domain.Investment{}).
		Where("project_id = ? AND status = ?", projectID, string(domain.InvestmentVerified)).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&total).Error
	return total, err
}

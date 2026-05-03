package repository

import (
	"flyup/internal/domain"
	"flyup/internal/dto"

	"gorm.io/gorm"
)

type InvestmentRepository interface {
	Create(investment *domain.Investment) error
	FindByID(id uint) (*domain.Investment, error)
	FindByReferenceNumber(ref string) (*domain.Investment, error)
	FindVerifiedByProjectID(projectID uint) ([]domain.Investment, error)
	UpdateStatus(id uint, status domain.InvestmentStatus) error
	UpdatePaid(investment *domain.Investment) error
	UpdateRefunded(investment *domain.Investment) error
	ListByBoosterUserID(boosterUserID uint) ([]domain.Investment, error)
	ListRefundPending() ([]domain.Investment, error)
	SumActiveByProjectID(projectID uint) (float64, error)
	IncrementProjectFunding(projectID uint, amount float64) error
	ListInvestorsByProjectID(projectID uint) ([]dto.ProjectInvestorItem, error)
	ListInvestedProjectsByUserID(boosterUserID uint) ([]dto.InvestedProjectItem, error)
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

func (r *investmentRepository) FindVerifiedByProjectID(projectID uint) ([]domain.Investment, error) {
	var investments []domain.Investment
	err := r.db.
		Where("project_id = ? AND status = ?", projectID, string(domain.InvestmentVerified)).
		Find(&investments).Error
	return investments, err
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

func (r *investmentRepository) ListInvestorsByProjectID(projectID uint) ([]dto.ProjectInvestorItem, error) {
	var result []dto.ProjectInvestorItem

	err := r.db.Model(&domain.Investment{}).
		Select(`users.id as user_id, users.first_name, users.last_name, users.picture,
            SUM(investments.principal_amount) as principal_amount,
            SUM(investments.total_amount) as total_amount,
            COUNT(investments.id) as investment_count,
            MIN(investments.paid_at) as first_invested_at`).
		Joins("JOIN users on users.id = investments.booster_user_id").
		Where("investments.project_id = ? AND investments.status = ?", projectID, string(domain.InvestmentVerified)).
		Group("users.id, users.first_name, users.last_name, users.picture").
		Order("first_invested_at ASC").
		Scan(&result).Error
	return result, err
}

func (r *investmentRepository) ListInvestedProjectsByUserID(boosterUserID uint) ([]dto.InvestedProjectItem, error) {
	var result []dto.InvestedProjectItem
	err := r.db.Model(&domain.Investment{}).
		Select(`projects.id as project_id, projects.title, projects.state,
                projects.profit_share_pct,
                SUM(investments.total_amount) as total_amount,
                SUM(investments.principal_amount) as principal_amount,
                COUNT(investments.id) as investment_count,
                MIN(investments.paid_at) as first_invested_at`).
		Joins("JOIN projects on projects.id = investments.project_id").
		Where("investments.booster_user_id = ? AND investments.status = ?", boosterUserID, string(domain.InvestmentVerified)).
		Group("projects.id, projects.title, projects.state, projects.profit_share_pct").
		Order("first_invested_at DESC").
		Scan(&result).Error
	return result, err
}

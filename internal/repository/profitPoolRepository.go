package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type ProfitPoolRepository interface {
	Create(pool *domain.ProfitPool) error
	FindByID(id uint) (*domain.ProfitPool, error)
	ListAll() ([]domain.ProfitPool, error)
	Update(pool *domain.ProfitPool) error

	CreatePayout(p *domain.InvestorProfitPayout) error
	FindPayoutByID(id uint) (*domain.InvestorProfitPayout, error)
	ListPayoutsByPoolID(poolID uint) ([]domain.InvestorProfitPayout, error)
	UpdatePayout(p *domain.InvestorProfitPayout) error
}

type profitPoolRepository struct {
	db *gorm.DB
}

func NewProfitPoolRepository(db *gorm.DB) ProfitPoolRepository {
	return &profitPoolRepository{db}
}

func (r *profitPoolRepository) Create(pool *domain.ProfitPool) error {
	return r.db.Create(pool).Error
}

func (r *profitPoolRepository) FindByID(id uint) (*domain.ProfitPool, error) {
	var pool domain.ProfitPool
	if err := r.db.First(&pool, id).Error; err != nil {
		return nil, err
	}
	return &pool, nil
}

func (r *profitPoolRepository) ListAll() ([]domain.ProfitPool, error) {
	var list []domain.ProfitPool
	if err := r.db.Order("created_at desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *profitPoolRepository) Update(pool *domain.ProfitPool) error {
	return r.db.Save(pool).Error
}

func (r *profitPoolRepository) CreatePayout(p *domain.InvestorProfitPayout) error {
	return r.db.Create(p).Error
}

func (r *profitPoolRepository) FindPayoutByID(id uint) (*domain.InvestorProfitPayout, error) {
	var p domain.InvestorProfitPayout
	if err := r.db.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *profitPoolRepository) ListPayoutsByPoolID(poolID uint) ([]domain.InvestorProfitPayout, error) {
	var list []domain.InvestorProfitPayout
	if err := r.db.Where("profit_pool_id = ?", poolID).Order("booster_user_id asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *profitPoolRepository) UpdatePayout(p *domain.InvestorProfitPayout) error {
	return r.db.Save(p).Error
}

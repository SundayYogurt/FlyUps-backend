package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type ProfitPoolRepository interface {
	Create(pool *domain.ProfitPool) error
	FindByID(id uint) (*domain.ProfitPool, error)
	ListAll() ([]domain.ProfitPool, error)
	ListByPioneerUserID(pioneerUserID uint) ([]domain.ProfitPool, error)
	ExistsByProjectAndQuarter(projectID uint, quarterNo int) (bool, error)
	Update(pool *domain.ProfitPool) error

	CreatePayout(p *domain.InvestorProfitPayout) error
	FindPayoutByID(id uint) (*domain.InvestorProfitPayout, error)
	ListPayoutsByPoolID(poolID uint) ([]domain.InvestorProfitPayout, error)
	ListPayoutsByBoosterUserID(boosterUserID uint) ([]domain.InvestorProfitPayout, error)
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

func (r *profitPoolRepository) ListByPioneerUserID(pioneerUserID uint) ([]domain.ProfitPool, error) {
	var list []domain.ProfitPool
	if err := r.db.Where("pioneer_user_id = ?", pioneerUserID).Order("created_at desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *profitPoolRepository) ExistsByProjectAndQuarter(projectID uint, quarterNo int) (bool, error) {
	var count int64
	err := r.db.Model(&domain.ProfitPool{}).
		Where("project_id = ? AND quarter_no = ?", projectID, quarterNo).
		Count(&count).Error
	return count > 0, err
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

func (r *profitPoolRepository) ListPayoutsByBoosterUserID(boosterUserID uint) ([]domain.InvestorProfitPayout, error) {
	var list []domain.InvestorProfitPayout
	if err := r.db.Where("booster_user_id = ?", boosterUserID).Order("created_at desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *profitPoolRepository) UpdatePayout(p *domain.InvestorProfitPayout) error {
	return r.db.Save(p).Error
}

package repository

import (
	"errors"
	"flyup/internal/domain"
	"gorm.io/gorm/clause"

	"gorm.io/gorm"
)

type ProfitPoolRepository interface {
	Create(pool *domain.ProfitPool) error
	CreateWithPayouts(pool *domain.ProfitPool, payouts []domain.InvestorProfitPayout, slip *domain.VerifiedSlip) error
	ConfirmWithSlip(p *domain.InvestorProfitPayout, slip *domain.VerifiedSlip) error
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

func insertVerifiedSlip(tx *gorm.DB, slip *domain.VerifiedSlip) error {
	if slip == nil {
		return errors.New("กรุณาตรวจสลิปก่อนบันทึกการโอน")
	}
	// Preserve references consumed before EasySlip was introduced, including
	// archived records. Legacy references have no sender-bank metadata, so an
	// exact reference collision is conservatively rejected.
	for _, model := range []any{&domain.ProfitPool{}, &domain.InvestorProfitPayout{}} {
		var count int64
		if err := tx.Unscoped().Model(model).Where("verified_slip_id IS NULL AND transfer_ref = ?", slip.TransRef).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("สลิปนี้ถูกใช้แล้ว")
		}
	}
	result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(slip)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("สลิปนี้ถูกใช้แล้ว")
	}
	return nil
}

func (r *profitPoolRepository) CreateWithPayouts(pool *domain.ProfitPool, payouts []domain.InvestorProfitPayout, slip *domain.VerifiedSlip) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := insertVerifiedSlip(tx, slip); err != nil {
			return err
		}
		pool.VerifiedSlipID = &slip.ID
		pool.SlipVerifiedAt = &slip.VerifiedAt
		if err := tx.Create(pool).Error; err != nil {
			return err
		}
		for i := range payouts {
			payouts[i].ProfitPoolID = pool.ID
		}
		if len(payouts) == 0 {
			return errors.New("no investor payouts")
		}
		return tx.Create(&payouts).Error
	})
}

func (r *profitPoolRepository) ConfirmWithSlip(p *domain.InvestorProfitPayout, slip *domain.VerifiedSlip) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := insertVerifiedSlip(tx, slip); err != nil {
			return err
		}
		p.VerifiedSlipID = &slip.ID
		p.SlipVerifiedAt = &slip.VerifiedAt
		// Conditional update prevents two simultaneous confirmations using different slips.
		result := tx.Model(&domain.InvestorProfitPayout{}).Where("id = ? AND status = ?", p.ID, domain.InvestorPayoutPending).Updates(map[string]any{
			"status": p.Status, "transfer_ref": p.TransferRef, "admin_note": p.AdminNote,
			"confirmed_at": p.ConfirmedAt, "confirmed_by": p.ConfirmedBy,
			"slip_image": p.SlipImage, "verified_slip_id": p.VerifiedSlipID, "slip_verified_at": p.SlipVerifiedAt,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("payout already confirmed")
		}
		return nil
	})
}

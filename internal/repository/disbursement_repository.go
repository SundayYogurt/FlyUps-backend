package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type DisbursementRepository interface {
	Create(d *domain.Disbursement) error
	FindByID(id uint) (*domain.Disbursement, error)
	FindByMilestoneID(milestoneID uint) (*domain.Disbursement, error)
	Update(d *domain.Disbursement) error
	ListAll() ([]domain.Disbursement, error)
	ListByStatus(status domain.DisbursementStatus) ([]domain.Disbursement, error)
	ListByPioneerID(pioneerID uint) ([]domain.Disbursement, error)
	ListByProjectID(projectID uint) ([]domain.Disbursement, error)
}

type disbursementRepository struct {
	db *gorm.DB
}

func NewDisbursementRepository(db *gorm.DB) DisbursementRepository {
	return &disbursementRepository{db}
}

// Create บันทึกรายการเบิกจ่ายเงิน (disbursement) ใหม่ลงฐานข้อมูล
func (r *disbursementRepository) Create(d *domain.Disbursement) error {
	return r.db.Create(d).Error
}

// FindByID ค้นหารายการเบิกจ่ายเงินจาก ID
func (r *disbursementRepository) FindByID(id uint) (*domain.Disbursement, error) {
	d := &domain.Disbursement{}
	if err := r.db.First(d, id).Error; err != nil {
		return nil, err
	}
	return d, nil
}

// FindByMilestoneID ค้นหารายการเบิกจ่ายเงินจาก milestone ID
func (r *disbursementRepository) FindByMilestoneID(milestoneID uint) (*domain.Disbursement, error) {
	d := &domain.Disbursement{}
	if err := r.db.Where("milestone_id = ?", milestoneID).First(d).Error; err != nil {
		return nil, err
	}
	return d, nil
}

// Update บันทึกการอัปเดตข้อมูลรายการเบิกจ่ายเงิน (save ทั้ง record)
func (r *disbursementRepository) Update(d *domain.Disbursement) error {
	return r.db.Save(d).Error
}

// ListAll ดึงรายการเบิกจ่ายเงินทั้งหมด เรียงจากล่าสุดไปเก่าสุด
func (r *disbursementRepository) ListAll() ([]domain.Disbursement, error) {
	var list []domain.Disbursement
	if err := r.db.Order("created_at desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListByStatus ดึงรายการเบิกจ่ายเงินตามสถานะที่ระบุ เรียงจากล่าสุดไปเก่าสุด
func (r *disbursementRepository) ListByStatus(status domain.DisbursementStatus) ([]domain.Disbursement, error) {
	var list []domain.Disbursement
	if err := r.db.Where("status = ?", status).Order("created_at desc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListByPioneerID ดึงรายการเบิกจ่ายเงินทั้งหมดของ pioneer (เจ้าของโปรเจกต์) เรียงตามโปรเจกต์และเฟส
func (r *disbursementRepository) ListByPioneerID(pioneerID uint) ([]domain.Disbursement, error) {
	var list []domain.Disbursement
	if err := r.db.Where("pioneer_user_id = ?", pioneerID).Order("project_id asc, phase_no asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListByProjectID ดึงรายการเบิกจ่ายเงินทั้งหมดของโปรเจกต์ เรียงตามเฟส (phase_no)
func (r *disbursementRepository) ListByProjectID(projectID uint) ([]domain.Disbursement, error) {
	var list []domain.Disbursement
	if err := r.db.Where("project_id = ?", projectID).Order("phase_no asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

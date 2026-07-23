package repository

import (
	"flyup/internal/domain"
	"flyup/internal/dto"

	"gorm.io/gorm"
)

type InvestmentRepository interface {
	Create(investment *domain.Investment) error
	FindByID(id uint) (*domain.Investment, error)
	FindByIDWithProject(id uint) (*domain.Investment, error)
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
	SumTotalFunding() (float64, error)
	CountUniqueBoosters() (int64, error)
}

type investmentRepository struct {
	db *gorm.DB
}

func NewInvestmentRepository(db *gorm.DB) InvestmentRepository {
	return &investmentRepository{db}
}

// Create บันทึกรายการลงทุนใหม่ลงฐานข้อมูล
func (r *investmentRepository) Create(investment *domain.Investment) error {
	return r.db.Create(investment).Error
}

// FindByID ค้นหารายการลงทุนจาก ID โดยตรง (ไม่ preload ความสัมพันธ์ใด ๆ)
func (r *investmentRepository) FindByID(id uint) (*domain.Investment, error) {
	inv := &domain.Investment{}
	err := r.db.First(inv, id).Error
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// FindByIDWithProject ค้นหารายการลงทุนจาก ID พร้อมโหลดข้อมูลโปรเจกต์ที่เกี่ยวข้อง
// (Project, Category, Media, Milestones, Stories) มาด้วย
func (r *investmentRepository) FindByIDWithProject(id uint) (*domain.Investment, error) {
	inv := &domain.Investment{}
	err := r.db.
		Preload("Project").
		Preload("Project.Category").
		Preload("Project.Media").
		Preload("Project.Milestones").
		Preload("Project.Stories").
		First(inv, id).Error
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// FindByReferenceNumber ค้นหารายการลงทุนจากเลขที่อ้างอิง (reference number) ของการชำระเงิน
func (r *investmentRepository) FindByReferenceNumber(ref string) (*domain.Investment, error) {
	inv := &domain.Investment{}

	err := r.db.Where("reference_number = ?", ref).First(inv).Error

	if err != nil {
		return nil, err
	}

	return inv, nil
}

// FindVerifiedByProjectID ดึงรายการลงทุนทั้งหมดของโปรเจกต์ที่มีสถานะ "verified" แล้ว
func (r *investmentRepository) FindVerifiedByProjectID(projectID uint) ([]domain.Investment, error) {
	var investments []domain.Investment
	err := r.db.
		Where("project_id = ? AND status = ?", projectID, string(domain.InvestmentVerified)).
		Find(&investments).Error
	return investments, err
}

// UpdateStatus อัปเดตสถานะ (status) ของรายการลงทุนตาม ID
func (r *investmentRepository) UpdateStatus(id uint, status domain.InvestmentStatus) error {
	return r.db.Model(&domain.Investment{}).Where("id = ?", id).Update("status", status).Error
}

// UpdatePaid บันทึกการอัปเดตรายการลงทุนหลังจากที่ผู้ใช้ชำระเงินแล้ว (save ทั้ง record)
func (r *investmentRepository) UpdatePaid(investment *domain.Investment) error {
	return r.db.Save(investment).Error
}

// UpdateRefunded บันทึกการอัปเดตรายการลงทุนหลังจากที่มีการคืนเงินแล้ว (save ทั้ง record)
func (r *investmentRepository) UpdateRefunded(investment *domain.Investment) error {
	return r.db.Save(investment).Error
}

// ListByBoosterUserID ดึงรายการลงทุนทั้งหมดของผู้ใช้ (booster) พร้อมข้อมูลโปรเจกต์ เรียงจากล่าสุดไปเก่าสุด
func (r *investmentRepository) ListByBoosterUserID(boosterUserID uint) ([]domain.Investment, error) {
	inv := []domain.Investment{}
	err := r.db.
		Preload("Project").
		Where("booster_user_id = ?", boosterUserID).
		Order("created_at desc").
		Find(&inv).Error
	return inv, err
}

// ListRefundPending ดึงรายการลงทุนที่รอการคืนเงิน (refund pending) เรียงตามเวลาที่ขอคืนเงินจากเก่าไปใหม่
func (r *investmentRepository) ListRefundPending() ([]domain.Investment, error) {
	var investments []domain.Investment
	err := r.db.Where("status = ?", domain.InvestmentRefundPending).Order("refunded_at ASC").Find(&investments).Error
	return investments, err
}

// IncrementProjectFunding เพิ่มยอดเงินระดมทุนสะสม (current_funding) ของโปรเจกต์ตามจำนวนเงินที่ระบุ
func (r *investmentRepository) IncrementProjectFunding(projectID uint, amount float64) error {
	return r.db.Model(&domain.Project{}).Where("id = ?", projectID).UpdateColumn("current_funding", gorm.Expr("current_funding + ?", amount)).Error
}

// SumActiveByProjectID รวมยอดเงินลงทุน (total_amount) ของโปรเจกต์ที่มีสถานะ verified แล้ว
func (r *investmentRepository) SumActiveByProjectID(projectID uint) (float64, error) {
	var total float64
	err := r.db.Model(&domain.Investment{}).
		Where("project_id = ? AND status = ?", projectID, string(domain.InvestmentVerified)).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&total).Error
	return total, err
}

// ListInvestorsByProjectID ดึงรายชื่อผู้ลงทุนของโปรเจกต์ พร้อมยอดเงินรวมและจำนวนครั้งที่ลงทุน (group by ผู้ใช้)
func (r *investmentRepository) ListInvestorsByProjectID(projectID uint) ([]dto.ProjectInvestorItem, error) {
	var result []dto.ProjectInvestorItem

	err := r.db.Model(&domain.Investment{}).
		Select(`users.id as user_id, users.first_name, users.last_name, users.email, users.picture,
            SUM(investments.principal_amount) as principal_amount,
            SUM(investments.total_amount) as total_amount,
            COUNT(investments.id) as investment_count,
            MIN(investments.paid_at) as first_invested_at`).
		Joins("JOIN users on users.id = investments.booster_user_id").
		Where("investments.project_id = ? AND investments.status = ?", projectID, string(domain.InvestmentVerified)).
		Group("users.id, users.first_name, users.last_name, users.email, users.picture").
		Order("first_invested_at ASC").
		Scan(&result).Error
	return result, err
}

// ListInvestedProjectsByUserID ดึงรายชื่อโปรเจกต์ที่ผู้ใช้เคยลงทุน พร้อมยอดเงินรวมและจำนวนครั้งที่ลงทุน (group by โปรเจกต์)
func (r *investmentRepository) ListInvestedProjectsByUserID(boosterUserID uint) ([]dto.InvestedProjectItem, error) {
	var result []dto.InvestedProjectItem
	err := r.db.Model(&domain.Investment{}).
		Select(`projects.id as project_id, projects.title, projects.state,
                projects.cover_image,
                projects.profit_share_pct,
                SUM(investments.total_amount) as total_amount,
                SUM(investments.principal_amount) as principal_amount,
                COUNT(investments.id) as investment_count,
                MIN(investments.paid_at) as first_invested_at`).
		Joins("JOIN projects on projects.id = investments.project_id").
		Where("investments.booster_user_id = ?", boosterUserID).
		Group("projects.id, projects.title, projects.state, projects.cover_image, projects.profit_share_pct").
		Order("first_invested_at DESC").
		Scan(&result).Error
	return result, err
}

// SumTotalFunding รวมยอดเงินต้น (principal_amount) ของรายการลงทุนที่ verified แล้วทั้งระบบ
func (r *investmentRepository) SumTotalFunding() (float64, error) {
	var total float64
	err := r.db.Model(&domain.Investment{}).
		Where("status = ?", string(domain.InvestmentVerified)).
		Select("COALESCE(SUM(principal_amount), 0)").
		Scan(&total).Error
	return total, err
}

// CountUniqueBoosters นับจำนวนผู้ลงทุน (booster) ที่ไม่ซ้ำกัน จากรายการลงทุนที่ verified แล้ว
func (r *investmentRepository) CountUniqueBoosters() (int64, error) {
	var count int64
	err := r.db.Model(&domain.Investment{}).
		Where("status = ?", string(domain.InvestmentVerified)).
		Distinct("booster_user_id").
		Count(&count).Error
	return count, err
}

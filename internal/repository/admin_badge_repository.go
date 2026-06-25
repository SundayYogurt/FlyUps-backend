package repository

import "gorm.io/gorm"

type AdminBadgeRepository interface {
	CountPendingProjects() (int64, error)
	CountSubmittedMilestones() (int64, error)
	CountPendingCancelReqs() (int64, error)
	CountOpenComplaints() (int64, error)
	CountPendingRefunds() (int64, error)
	CountPendingDisbursements() (int64, error)
	CountPendingProfitPools() (int64, error)
	CountPendingVerifications() (int64, error)
}

type adminBadgeRepository struct {
	db *gorm.DB
}

func NewAdminBadgeRepository(db *gorm.DB) AdminBadgeRepository {
	return &adminBadgeRepository{db}
}

func (r *adminBadgeRepository) CountPendingProjects() (int64, error) {
	var count int64
	err := r.db.Table("projects").Where("state = ? AND deleted_at IS NULL", "pending_review").Count(&count).Error
	return count, err
}

func (r *adminBadgeRepository) CountSubmittedMilestones() (int64, error) {
	var count int64
	err := r.db.Table("milestones").Where("status = ? AND deleted_at IS NULL", "submitted").Count(&count).Error
	return count, err
}

func (r *adminBadgeRepository) CountPendingCancelReqs() (int64, error) {
	var count int64
	err := r.db.Table("projects").Where("state = ? AND deleted_at IS NULL", "pending_cancel").Count(&count).Error
	return count, err
}

func (r *adminBadgeRepository) CountOpenComplaints() (int64, error) {
	var count int64
	err := r.db.Table("complaints").Where("status = ? AND deleted_at IS NULL", "open").Count(&count).Error
	return count, err
}

func (r *adminBadgeRepository) CountPendingRefunds() (int64, error) {
	var count int64
	err := r.db.Table("investments").Where("status = ? AND deleted_at IS NULL", "refund_pending").Count(&count).Error
	return count, err
}

func (r *adminBadgeRepository) CountPendingDisbursements() (int64, error) {
	var count int64
	err := r.db.Table("disbursements").Where("status = ? AND deleted_at IS NULL", "pending").Count(&count).Error
	return count, err
}

func (r *adminBadgeRepository) CountPendingProfitPools() (int64, error) {
	var count int64
	err := r.db.Table("profit_pools").Where("status = ? AND deleted_at IS NULL", "pending").Count(&count).Error
	return count, err
}

func (r *adminBadgeRepository) CountPendingVerifications() (int64, error) {
	var student, idCard int64
	if err := r.db.Table("student_card_verifications").Where("status = ? AND deleted_at IS NULL", "pending").Count(&student).Error; err != nil {
		return 0, err
	}
	if err := r.db.Table("id_card_verifications").Where("status = ? AND deleted_at IS NULL", "pending").Count(&idCard).Error; err != nil {
		return 0, err
	}
	return student + idCard, nil
}
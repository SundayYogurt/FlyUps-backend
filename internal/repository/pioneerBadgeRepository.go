package repository

import "gorm.io/gorm"

type PioneerBadgeRepository interface {
	CountActiveMilestones(pioneerID uint) (int64, error)
	CountUpcomingMeetings(pioneerID uint, today string) (int64, error)
	CountPendingPayouts(pioneerID uint) (int64, error)
}

type pioneerBadgeRepository struct {
	db *gorm.DB
}

func NewPioneerBadgeRepository(db *gorm.DB) PioneerBadgeRepository {
	return &pioneerBadgeRepository{db}
}

func (r *pioneerBadgeRepository) CountActiveMilestones(pioneerID uint) (int64, error) {
	var count int64
	err := r.db.Raw(`
		SELECT COUNT(m.id)
		FROM milestones m
		JOIN projects p ON p.id = m.project_id
		WHERE p.pioneer_user_id = ?
		  AND p.deleted_at IS NULL
		  AND m.status IN ('active', 'rejected')
		  AND m.deleted_at IS NULL
	`, pioneerID).Scan(&count).Error
	return count, err
}

func (r *pioneerBadgeRepository) CountUpcomingMeetings(pioneerID uint, today string) (int64, error) {
	var count int64
	err := r.db.Raw(`
		SELECT COUNT(mt.id)
		FROM meetings mt
		JOIN milestones m ON m.id = mt.milestone_id
		JOIN projects p ON p.id = m.project_id
		WHERE p.pioneer_user_id = ?
		  AND p.deleted_at IS NULL
		  AND mt.status = 'open'
		  AND mt.deleted_at IS NULL
		  AND mt.date >= ?
	`, pioneerID, today).Scan(&count).Error
	return count, err
}

func (r *pioneerBadgeRepository) CountPendingPayouts(pioneerID uint) (int64, error) {
	var count int64
	err := r.db.Table("disbursements").
		Where("pioneer_user_id = ? AND status = ? AND deleted_at IS NULL", pioneerID, "pending").
		Count(&count).Error
	return count, err
}

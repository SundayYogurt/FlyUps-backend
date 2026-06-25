package repository

import "gorm.io/gorm"

type BoosterBadgeRepository interface {
	CountPendingVotes(boosterID uint) (int64, error)
	CountUpcomingMeetings(boosterID uint, today string) (int64, error)
	CountPendingRefunds(boosterID uint) (int64, error)
	CountOpenComplaints(boosterID uint) (int64, error)
}

type boosterBadgeRepository struct {
	db *gorm.DB
}

func NewBoosterBadgeRepository(db *gorm.DB) BoosterBadgeRepository {
	return &boosterBadgeRepository{db}
}

func (r *boosterBadgeRepository) CountPendingVotes(boosterID uint) (int64, error) {
	var count int64
	err := r.db.Raw(`
		SELECT COUNT(DISTINCT m.id)
		FROM milestones m
		JOIN investments i ON i.project_id = m.project_id
		WHERE i.booster_user_id = ?
		  AND i.status IN ('verified', 'paid')
		  AND i.deleted_at IS NULL
		  AND m.voting_open = true
		  AND m.deleted_at IS NULL
		  AND NOT EXISTS (
		    SELECT 1 FROM milestone_votes mv
		    WHERE mv.milestone_id = m.id
		      AND mv.booster_user_id = ?
		  )
	`, boosterID, boosterID).Scan(&count).Error
	return count, err
}

func (r *boosterBadgeRepository) CountUpcomingMeetings(boosterID uint, today string) (int64, error) {
	var count int64
	err := r.db.Raw(`
		SELECT COUNT(DISTINCT mt.id)
		FROM meetings mt
		JOIN milestones m ON m.id = mt.milestone_id
		JOIN investments i ON i.project_id = m.project_id
		WHERE i.booster_user_id = ?
		  AND i.status IN ('verified', 'paid')
		  AND i.deleted_at IS NULL
		  AND mt.status = 'open'
		  AND mt.deleted_at IS NULL
		  AND mt.date >= ?
	`, boosterID, today).Scan(&count).Error
	return count, err
}

func (r *boosterBadgeRepository) CountPendingRefunds(boosterID uint) (int64, error) {
	var count int64
	err := r.db.Table("investments").
		Where("booster_user_id = ? AND status = ? AND deleted_at IS NULL", boosterID, "refund_pending").
		Count(&count).Error
	return count, err
}

func (r *boosterBadgeRepository) CountOpenComplaints(boosterID uint) (int64, error) {
	var count int64
	err := r.db.Table("complaints").
		Where("complainant_id = ? AND status = ? AND deleted_at IS NULL", boosterID, "open").
		Count(&count).Error
	return count, err
}

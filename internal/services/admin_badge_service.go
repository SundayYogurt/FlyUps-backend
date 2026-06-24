package services

import (
	"flyup/internal/dto"
	"time"

	"gorm.io/gorm"
)

type AdminBadgeService interface {
	GetCounts() (dto.AdminBadgeCounts, error)
}

type adminBadgeService struct {
	db *gorm.DB
}

func NewAdminBadgeService(db *gorm.DB) AdminBadgeService {
	return &adminBadgeService{db: db}
}

func (s *adminBadgeService) GetCounts() (dto.AdminBadgeCounts, error) {
	var counts dto.AdminBadgeCounts

	s.db.Table("projects").Where("state = ? AND deleted_at IS NULL", "pending_review").Count(&counts.PendingProjects)
	s.db.Table("milestones").Where("status = ? AND deleted_at IS NULL", "submitted").Count(&counts.SubmittedMilestones)
	s.db.Table("projects").Where("state = ? AND deleted_at IS NULL", "pending_cancel").Count(&counts.PendingCancelReqs)
	s.db.Table("complaints").Where("status = ? AND deleted_at IS NULL", "open").Count(&counts.OpenComplaints)
	s.db.Table("investments").Where("status = ? AND deleted_at IS NULL", "refund_pending").Count(&counts.PendingRefunds)
	s.db.Table("disbursements").Where("status = ? AND deleted_at IS NULL", "pending").Count(&counts.PendingDisbursements)
	s.db.Table("profit_pools").Where("status = ? AND deleted_at IS NULL", "pending").Count(&counts.PendingProfitPools)

	var studentCard, idCard int64
	s.db.Table("student_card_verifications").Where("status = ? AND deleted_at IS NULL", "pending").Count(&studentCard)
	s.db.Table("id_card_verifications").Where("status = ? AND deleted_at IS NULL", "pending").Count(&idCard)
	counts.PendingVerifications = studentCard + idCard

	return counts, nil
}

type BoosterBadgeService interface {
	GetCounts(userID uint) (dto.BoosterBadgeCounts, error)
}

type boosterBadgeService struct {
	db *gorm.DB
}

func NewBoosterBadgeService(db *gorm.DB) BoosterBadgeService {
	return &boosterBadgeService{db: db}
}

func (s *boosterBadgeService) GetCounts(userID uint) (dto.BoosterBadgeCounts, error) {
	var counts dto.BoosterBadgeCounts
	today := time.Now().UTC().Format("2006-01-02")

	s.db.Raw(`
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
	`, userID, userID).Scan(&counts.PendingVotes)

	s.db.Raw(`
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
	`, userID, today).Scan(&counts.UpcomingMeetings)

	s.db.Table("investments").
		Where("booster_user_id = ? AND status = ? AND deleted_at IS NULL", userID, "refund_pending").
		Count(&counts.PendingRefunds)

	s.db.Table("complaints").
		Where("complainant_id = ? AND status = ? AND deleted_at IS NULL", userID, "open").
		Count(&counts.OpenComplaints)

	return counts, nil
}

type PioneerBadgeService interface {
	GetCounts(userID uint) (dto.PioneerBadgeCounts, error)
}

type pioneerBadgeService struct {
	db *gorm.DB
}

func NewPioneerBadgeService(db *gorm.DB) PioneerBadgeService {
	return &pioneerBadgeService{db: db}
}

func (s *pioneerBadgeService) GetCounts(userID uint) (dto.PioneerBadgeCounts, error) {
	var counts dto.PioneerBadgeCounts
	today := time.Now().UTC().Format("2006-01-02")

	s.db.Raw(`
		SELECT COUNT(m.id)
		FROM milestones m
		JOIN projects p ON p.id = m.project_id
		WHERE p.pioneer_user_id = ?
		  AND p.deleted_at IS NULL
		  AND m.status IN ('active', 'rejected')
		  AND m.deleted_at IS NULL
	`, userID).Scan(&counts.ActiveMilestones)

	s.db.Raw(`
		SELECT COUNT(mt.id)
		FROM meetings mt
		JOIN milestones m ON m.id = mt.milestone_id
		JOIN projects p ON p.id = m.project_id
		WHERE p.pioneer_user_id = ?
		  AND p.deleted_at IS NULL
		  AND mt.status = 'open'
		  AND mt.deleted_at IS NULL
		  AND mt.date >= ?
	`, userID, today).Scan(&counts.UpcomingMeetings)

	s.db.Table("disbursements").
		Where("pioneer_user_id = ? AND status = ? AND deleted_at IS NULL", userID, "pending").
		Count(&counts.PendingPayouts)

	return counts, nil
}

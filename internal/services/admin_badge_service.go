package services

import (
	"flyup/internal/dto"
	"flyup/internal/repository"
	"time"
)

// --- AdminBadgeService ---

type AdminBadgeService interface {
	GetCounts() (dto.AdminBadgeCounts, error)
}

type adminBadgeService struct {
	repo repository.AdminBadgeRepository
}

func NewAdminBadgeService(repo repository.AdminBadgeRepository) AdminBadgeService {
	return &adminBadgeService{repo: repo}
}

func (s *adminBadgeService) GetCounts() (dto.AdminBadgeCounts, error) {
	var counts dto.AdminBadgeCounts
	var err error

	if counts.PendingProjects, err = s.repo.CountPendingProjects(); err != nil {
		return counts, err
	}
	if counts.SubmittedMilestones, err = s.repo.CountSubmittedMilestones(); err != nil {
		return counts, err
	}
	if counts.PendingCancelReqs, err = s.repo.CountPendingCancelReqs(); err != nil {
		return counts, err
	}
	if counts.OpenComplaints, err = s.repo.CountOpenComplaints(); err != nil {
		return counts, err
	}
	if counts.PendingRefunds, err = s.repo.CountPendingRefunds(); err != nil {
		return counts, err
	}
	if counts.PendingDisbursements, err = s.repo.CountPendingDisbursements(); err != nil {
		return counts, err
	}
	if counts.PendingProfitPools, err = s.repo.CountPendingProfitPools(); err != nil {
		return counts, err
	}
	if counts.PendingVerifications, err = s.repo.CountPendingVerifications(); err != nil {
		return counts, err
	}

	return counts, nil
}

// --- BoosterBadgeService ---

type BoosterBadgeService interface {
	GetCounts(userID uint) (dto.BoosterBadgeCounts, error)
}

type boosterBadgeService struct {
	repo repository.BoosterBadgeRepository
}

func NewBoosterBadgeService(repo repository.BoosterBadgeRepository) BoosterBadgeService {
	return &boosterBadgeService{repo: repo}
}

func (s *boosterBadgeService) GetCounts(userID uint) (dto.BoosterBadgeCounts, error) {
	var counts dto.BoosterBadgeCounts
	today := time.Now().UTC().Format("2006-01-02")

	var err error
	if counts.PendingVotes, err = s.repo.CountPendingVotes(userID); err != nil {
		return counts, err
	}
	if counts.UpcomingMeetings, err = s.repo.CountUpcomingMeetings(userID, today); err != nil {
		return counts, err
	}
	if counts.PendingRefunds, err = s.repo.CountPendingRefunds(userID); err != nil {
		return counts, err
	}
	if counts.OpenComplaints, err = s.repo.CountOpenComplaints(userID); err != nil {
		return counts, err
	}

	return counts, nil
}

// --- PioneerBadgeService ---

type PioneerBadgeService interface {
	GetCounts(userID uint) (dto.PioneerBadgeCounts, error)
}

type pioneerBadgeService struct {
	repo repository.PioneerBadgeRepository
}

func NewPioneerBadgeService(repo repository.PioneerBadgeRepository) PioneerBadgeService {
	return &pioneerBadgeService{repo: repo}
}

func (s *pioneerBadgeService) GetCounts(userID uint) (dto.PioneerBadgeCounts, error) {
	var counts dto.PioneerBadgeCounts
	today := time.Now().UTC().Format("2006-01-02")

	var err error
	if counts.ActiveMilestones, err = s.repo.CountActiveMilestones(userID); err != nil {
		return counts, err
	}
	if counts.UpcomingMeetings, err = s.repo.CountUpcomingMeetings(userID, today); err != nil {
		return counts, err
	}
	if counts.PendingPayouts, err = s.repo.CountPendingPayouts(userID); err != nil {
		return counts, err
	}

	return counts, nil
}
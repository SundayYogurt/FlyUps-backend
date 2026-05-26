package repository

import (
	"flyup/internal/domain"
	"flyup/internal/dto"
	"time"

	"gorm.io/gorm"
)

type AdminLogRepository interface {
	Create(log *domain.AdminLog) error
	ListAll(filter dto.AdminLogFilter) ([]domain.AdminLog, int64, error)
}

type adminLogRepository struct {
	db *gorm.DB
}

func NewAdminLogRepository(db *gorm.DB) AdminLogRepository {
	return &adminLogRepository{db}
}

func (r *adminLogRepository) Create(log *domain.AdminLog) error {
	return r.db.Create(log).Error
}

func (r *adminLogRepository) ListAll(filter dto.AdminLogFilter) ([]domain.AdminLog, int64, error) {
	var logs []domain.AdminLog
	var total int64

	q := r.db.Model(&domain.AdminLog{}).Preload("Admin")

	if filter.AdminID != nil {
		q = q.Where("admin_id = ?", *filter.AdminID)
	}
	if filter.Action != "" {
		q = q.Where("action = ?", filter.Action)
	}
	if filter.TargetType != "" {
		q = q.Where("target_type = ?", filter.TargetType)
	}
	if filter.TargetID != nil {
		q = q.Where("target_id = ?", *filter.TargetID)
	}
	if filter.From != "" {
		if t, err := time.Parse("2006-01-02", filter.From); err == nil {
			q = q.Where("admin_logs.created_at >= ?", t)
		}
	}
	if filter.To != "" {
		if t, err := time.Parse("2006-01-02", filter.To); err == nil {
			// เพิ่ม 1 วันเพื่อให้ครอบคลุมทั้งวัน to
			q = q.Where("admin_logs.created_at < ?", t.AddDate(0, 0, 1))
		}
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	pageSize := filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	if err := q.Order("admin_logs.created_at desc").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

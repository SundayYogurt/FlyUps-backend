package services

import (
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"log"
)

type AdminLogService interface {
	// LogAction บันทึก action ของ admin — error ไม่ block flow หลัก (log และ ignore)
	LogAction(adminID uint, action, targetType string, targetID *uint, note *string)
	// ListLogs ดึง log ทั้งหมดพร้อม filter + pagination
	ListLogs(filter dto.AdminLogFilter) ([]dto.AdminLogItem, int64, error)
}

type adminLogService struct {
	repo repository.AdminLogRepository
}

func NewAdminLogService(repo repository.AdminLogRepository) AdminLogService {
	return &adminLogService{repo}
}

func (s *adminLogService) LogAction(adminID uint, action, targetType string, targetID *uint, note *string) {
	entry := &domain.AdminLog{
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Note:       note,
	}
	if err := s.repo.Create(entry); err != nil {
		log.Printf("[AdminLog] failed to write log: adminID=%d action=%s err=%v", adminID, action, err)
	}
}

func (s *adminLogService) ListLogs(filter dto.AdminLogFilter) ([]dto.AdminLogItem, int64, error) {
	logs, total, err := s.repo.ListAll(filter)
	if err != nil {
		return nil, 0, err
	}
	items := make([]dto.AdminLogItem, 0, len(logs))
	for _, l := range logs {
		item := dto.AdminLogItem{
			ID:         l.ID,
			AdminID:    l.AdminID,
			Action:     l.Action,
			TargetType: l.TargetType,
			TargetID:   l.TargetID,
			Note:       l.Note,
			CreatedAt:  l.CreatedAt,
		}
		if l.Admin != nil {
			item.Admin = &dto.AdminLogAdmin{
				ID:        l.Admin.ID,
				FirstName: l.Admin.FirstName,
				LastName:  l.Admin.LastName,
				Email:     l.Admin.Email,
			}
		}
		items = append(items, item)
	}
	return items, total, nil
}

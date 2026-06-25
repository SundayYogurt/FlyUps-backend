package services

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"time"
)

const ComplaintSuspendThreshold = 3

type ComplaintService interface {
	Create(userID uint, req dto.CreateComplaintRequest) (*domain.Complaint, error)
	ListMine(userID uint) ([]dto.ComplaintItem, error)
	AdminList(status *domain.ComplaintStatus) ([]dto.ComplaintItem, error)
	AdminGet(id uint) (*dto.ComplaintItem, error)
	AdminResolve(id, adminID uint, note string) (*domain.Complaint, error)
	AdminReject(id, adminID uint, note string) (*domain.Complaint, error)
	GetProjectStats(projectID uint) (*dto.ProjectComplaintStats, error)
}

type complaintService struct {
	complaintRepo repository.ComplaintRepository
	projectRepo   repository.ProjectRepository
}

func NewComplaintService(
	complaintRepo repository.ComplaintRepository,
	projectRepo repository.ProjectRepository,
) ComplaintService {
	return &complaintService{complaintRepo, projectRepo}
}

func (s *complaintService) Create(userID uint, req dto.CreateComplaintRequest) (*domain.Complaint, error) {
	// project ต้องมีอยู่จริง
	project, err := s.projectRepo.FindProjectByID(req.ProjectID)
	if err != nil || project == nil {
		return nil, errors.New("project not found")
	}

	// 1 user ร้องเรียน 1 ครั้งต่อ project
	existing, err := s.complaintRepo.FindByUserAndProject(userID, req.ProjectID)
	if err != nil {
		return nil, errors.New("internal server error")
	}
	if existing != nil {
		return nil, errors.New("you have already filed a complaint for this project")
	}

	c := &domain.Complaint{
		ComplainantID: userID,
		ProjectID:     req.ProjectID,
		Subject:       req.Subject,
		Body:          req.Body,
		Status:        domain.ComplaintOpen,
	}
	if err := s.complaintRepo.Create(c); err != nil {
		return nil, errors.New("failed to create complaint")
	}
	return c, nil
}

func (s *complaintService) ListMine(userID uint) ([]dto.ComplaintItem, error) {
	list, err := s.complaintRepo.ListByUser(userID)
	if err != nil {
		return nil, errors.New("internal server error")
	}
	return s.toItems(list), nil
}

func (s *complaintService) AdminList(status *domain.ComplaintStatus) ([]dto.ComplaintItem, error) {
	list, err := s.complaintRepo.ListAll(status)
	if err != nil {
		return nil, errors.New("internal server error")
	}
	items := s.toItems(list)
	// inject per-project counts (cache by projectID to avoid N+1)
	cache := make(map[uint][2]int64)
	for i := range items {
		pid := items[i].ProjectID
		if _, ok := cache[pid]; !ok {
			total, _ := s.complaintRepo.CountByProjectID(pid)
			resolved, _ := s.complaintRepo.CountResolvedByProjectID(pid)
			cache[pid] = [2]int64{total, resolved}
		}
		items[i].TotalReports = cache[pid][0]
		items[i].ResolvedReports = cache[pid][1]
	}
	return items, nil
}

func (s *complaintService) AdminGet(id uint) (*dto.ComplaintItem, error) {
	c, err := s.complaintRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("complaint not found")
	}
	item := s.toItem(c)
	total, _ := s.complaintRepo.CountByProjectID(c.ProjectID)
	resolved, _ := s.complaintRepo.CountResolvedByProjectID(c.ProjectID)
	item.TotalReports = total
	item.ResolvedReports = resolved
	return &item, nil
}

func (s *complaintService) AdminResolve(id, adminID uint, note string) (*domain.Complaint, error) {
	return s.adminClose(id, adminID, note, domain.ComplaintResolved)
}

func (s *complaintService) AdminReject(id, adminID uint, note string) (*domain.Complaint, error) {
	return s.adminClose(id, adminID, note, domain.ComplaintRejected)
}

func (s *complaintService) adminClose(id, adminID uint, note string, target domain.ComplaintStatus) (*domain.Complaint, error) {
	c, err := s.complaintRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("complaint not found")
	}
	if c.Status != domain.ComplaintOpen {
		return nil, errors.New("complaint already closed")
	}
	now := time.Now().UTC()
	c.Status = target
	c.AdminNote = note
	c.ResolvedBy = &adminID
	c.ResolvedAt = &now
	if err := s.complaintRepo.Update(c); err != nil {
		return nil, errors.New("failed to update complaint")
	}

	// ถ้า resolve → เช็ค threshold แล้ว auto-suspend project
	if target == domain.ComplaintResolved {
		go s.checkAndSuspendProject(c.ProjectID)
	}

	return c, nil
}

func (s *complaintService) checkAndSuspendProject(projectID uint) {
	resolvedCount, err := s.complaintRepo.CountResolvedByProjectID(projectID)
	if err != nil || resolvedCount < ComplaintSuspendThreshold {
		return
	}
	project, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil || project == nil {
		return
	}
	// ระงับเฉพาะโปรเจกต์ที่ยังไม่ถูก suspend/cancel
	if project.State == domain.StateSuspended || project.State == domain.StateCancelled || project.State == domain.StateClosed {
		return
	}
	project.State = domain.StateSuspended
	project.Status = domain.StatusSuspended
	s.projectRepo.UpdateProject(project)
}

func (s *complaintService) GetProjectStats(projectID uint) (*dto.ProjectComplaintStats, error) {
	project, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil || project == nil {
		return nil, errors.New("project not found")
	}
	total, err := s.complaintRepo.CountByProjectID(projectID)
	if err != nil {
		return nil, errors.New("internal server error")
	}
	resolved, err := s.complaintRepo.CountResolvedByProjectID(projectID)
	if err != nil {
		return nil, errors.New("internal server error")
	}
	return &dto.ProjectComplaintStats{
		ProjectID:       projectID,
		ProjectTitle:    project.Title,
		TotalReports:    total,
		ResolvedReports: resolved,
		Threshold:       ComplaintSuspendThreshold,
	}, nil
}

func (s *complaintService) toItem(c *domain.Complaint) dto.ComplaintItem {
	item := dto.ComplaintItem{
		ID:            c.ID,
		ComplainantID: c.ComplainantID,
		ProjectID:     c.ProjectID,
		Subject:       c.Subject,
		Body:          c.Body,
		Status:        string(c.Status),
		AdminNote:     c.AdminNote,
		ResolvedAt:    c.ResolvedAt,
		CreatedAt:     c.CreatedAt,
	}
	if c.Complainant != nil {
		item.Complainant = &dto.ComplaintComplainant{
			ID:        c.Complainant.ID,
			FirstName: c.Complainant.FirstName,
			LastName:  c.Complainant.LastName,
			Email:     c.Complainant.Email,
		}
	}
	if c.Project != nil {
		item.Project = &dto.ComplaintProject{
			ID:    c.Project.ID,
			Title: c.Project.Title,
			State: string(c.Project.State),
		}
	}
	return item
}

func (s *complaintService) toItems(list []domain.Complaint) []dto.ComplaintItem {
	items := make([]dto.ComplaintItem, 0, len(list))
	for i := range list {
		items = append(items, s.toItem(&list[i]))
	}
	return items
}

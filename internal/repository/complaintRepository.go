package repository

import (
	"errors"
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type ComplaintRepository interface {
	Create(c *domain.Complaint) error
	FindByID(id uint) (*domain.Complaint, error)
	FindByUserAndProject(userID, projectID uint) (*domain.Complaint, error)
	ListAll(status *domain.ComplaintStatus) ([]domain.Complaint, error)
	ListByUser(userID uint) ([]domain.Complaint, error)
	Update(c *domain.Complaint) error
	CountByProjectID(projectID uint) (int64, error)
	CountResolvedByProjectID(projectID uint) (int64, error)
}

type complaintRepository struct {
	db *gorm.DB
}

func NewComplaintRepository(db *gorm.DB) ComplaintRepository {
	return &complaintRepository{db}
}

func (r *complaintRepository) Create(c *domain.Complaint) error {
	return r.db.Create(c).Error
}

func (r *complaintRepository) FindByID(id uint) (*domain.Complaint, error) {
	c := &domain.Complaint{}
	err := r.db.
		Preload("Complainant").
		Preload("Project").
		First(c, id).Error
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *complaintRepository) FindByUserAndProject(userID, projectID uint) (*domain.Complaint, error) {
	c := &domain.Complaint{}
	err := r.db.Where("complainant_id = ? AND project_id = ?", userID, projectID).First(c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return c, nil
}

func (r *complaintRepository) ListAll(status *domain.ComplaintStatus) ([]domain.Complaint, error) {
	var list []domain.Complaint
	q := r.db.
		Preload("Complainant").
		Preload("Project").
		Order("created_at desc")
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *complaintRepository) ListByUser(userID uint) ([]domain.Complaint, error) {
	var list []domain.Complaint
	err := r.db.
		Preload("Project").
		Where("complainant_id = ?", userID).
		Order("created_at desc").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (r *complaintRepository) Update(c *domain.Complaint) error {
	return r.db.Save(c).Error
}

func (r *complaintRepository) CountByProjectID(projectID uint) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Complaint{}).Where("project_id = ?", projectID).Count(&count).Error
	return count, err
}

func (r *complaintRepository) CountResolvedByProjectID(projectID uint) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Complaint{}).
		Where("project_id = ? AND status = ?", projectID, domain.ComplaintResolved).
		Count(&count).Error
	return count, err
}

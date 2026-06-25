package services

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// Mock: ComplaintRepository

type mockComplaintRepo struct{ mock.Mock }

func (m *mockComplaintRepo) Create(c *domain.Complaint) error {
	return m.Called(c).Error(0)
}

func (m *mockComplaintRepo) FindByID(id uint) (*domain.Complaint, error) {
	args := m.Called(id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Complaint), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockComplaintRepo) FindByUserAndProject(userID, projectID uint) (*domain.Complaint, error) {
	args := m.Called(userID, projectID)
	if v := args.Get(0); v != nil {
		return v.(*domain.Complaint), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockComplaintRepo) ListAll(status *domain.ComplaintStatus) ([]domain.Complaint, error) {
	args := m.Called(status)
	return args.Get(0).([]domain.Complaint), args.Error(1)
}

func (m *mockComplaintRepo) ListByUser(userID uint) ([]domain.Complaint, error) {
	args := m.Called(userID)
	return args.Get(0).([]domain.Complaint), args.Error(1)
}

func (m *mockComplaintRepo) Update(c *domain.Complaint) error {
	return m.Called(c).Error(0)
}

func (m *mockComplaintRepo) CountByProjectID(projectID uint) (int64, error) {
	args := m.Called(projectID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockComplaintRepo) CountResolvedByProjectID(projectID uint) (int64, error) {
	args := m.Called(projectID)
	return args.Get(0).(int64), args.Error(1)
}

// Create

func TestComplaintService_Create_Success(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	pr.On("FindProjectByID", uint(10)).Return(&domain.Project{ID: 10}, nil)
	cr.On("FindByUserAndProject", uint(1), uint(10)).Return(nil, nil)
	cr.On("Create", mock.Anything).Return(nil)

	result, err := svc.Create(1, dto.CreateComplaintRequest{ProjectID: 10, Subject: "ปัญหา", Body: "รายละเอียด"})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(1), result.ComplainantID)
	assert.Equal(t, domain.ComplaintOpen, result.Status)
}

func TestComplaintService_Create_ProjectNotFound(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	pr.On("FindProjectByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)

	result, err := svc.Create(1, dto.CreateComplaintRequest{ProjectID: 99, Subject: "ปัญหา", Body: "รายละเอียด"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "project not found")
}

func TestComplaintService_Create_AlreadyFiled(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	pr.On("FindProjectByID", uint(10)).Return(&domain.Project{ID: 10}, nil)
	cr.On("FindByUserAndProject", uint(1), uint(10)).Return(&domain.Complaint{ID: 5}, nil)

	result, err := svc.Create(1, dto.CreateComplaintRequest{ProjectID: 10, Subject: "ปัญหา", Body: "รายละเอียด"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already filed")
}

func TestComplaintService_Create_RepoError(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	pr.On("FindProjectByID", uint(10)).Return(&domain.Project{ID: 10}, nil)
	cr.On("FindByUserAndProject", uint(1), uint(10)).Return(nil, nil)
	cr.On("Create", mock.Anything).Return(errors.New("db error"))

	result, err := svc.Create(1, dto.CreateComplaintRequest{ProjectID: 10, Subject: "ปัญหา", Body: "รายละเอียด"})

	assert.Error(t, err)
	assert.Nil(t, result)
}

// ListMine

func TestComplaintService_ListMine_Success(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	cr.On("ListByUser", uint(1)).Return([]domain.Complaint{
		{ID: 1, ComplainantID: 1, ProjectID: 10, Subject: "ปัญหา", Status: domain.ComplaintOpen},
	}, nil)

	items, err := svc.ListMine(1)

	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "ปัญหา", items[0].Subject)
}

func TestComplaintService_ListMine_RepoError(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	cr.On("ListByUser", uint(1)).Return([]domain.Complaint(nil), errors.New("db error"))

	items, err := svc.ListMine(1)

	assert.Error(t, err)
	assert.Nil(t, items)
}

// AdminList

func TestComplaintService_AdminList_Success(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	complaints := []domain.Complaint{
		{ID: 1, ComplainantID: 2, ProjectID: 10, Subject: "ปัญหา", Status: domain.ComplaintOpen},
	}
	cr.On("ListAll", (*domain.ComplaintStatus)(nil)).Return(complaints, nil)
	cr.On("CountByProjectID", uint(10)).Return(int64(3), nil)
	cr.On("CountResolvedByProjectID", uint(10)).Return(int64(1), nil)

	items, err := svc.AdminList(nil)

	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, int64(3), items[0].TotalReports)
	assert.Equal(t, int64(1), items[0].ResolvedReports)
}

func TestComplaintService_AdminList_WithStatusFilter(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	status := domain.ComplaintOpen
	cr.On("ListAll", &status).Return([]domain.Complaint{}, nil)

	items, err := svc.AdminList(&status)

	assert.NoError(t, err)
	assert.Empty(t, items)
}

func TestComplaintService_AdminList_RepoError(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	cr.On("ListAll", (*domain.ComplaintStatus)(nil)).Return([]domain.Complaint(nil), errors.New("db error"))

	items, err := svc.AdminList(nil)

	assert.Error(t, err)
	assert.Nil(t, items)
}

// AdminGet

func TestComplaintService_AdminGet_Success(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	cr.On("FindByID", uint(3)).Return(&domain.Complaint{
		ID: 3, ComplainantID: 1, ProjectID: 10, Subject: "ปัญหา", Status: domain.ComplaintOpen,
	}, nil)
	cr.On("CountByProjectID", uint(10)).Return(int64(2), nil)
	cr.On("CountResolvedByProjectID", uint(10)).Return(int64(0), nil)

	item, err := svc.AdminGet(3)

	assert.NoError(t, err)
	assert.NotNil(t, item)
	assert.Equal(t, uint(3), item.ID)
	assert.Equal(t, int64(2), item.TotalReports)
}

func TestComplaintService_AdminGet_NotFound(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	cr.On("FindByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)

	item, err := svc.AdminGet(99)

	assert.Error(t, err)
	assert.Nil(t, item)
	assert.Contains(t, err.Error(), "not found")
}

// AdminResolve

func TestComplaintService_AdminResolve_Success(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	complaint := &domain.Complaint{ID: 3, ProjectID: 10, Status: domain.ComplaintOpen}
	cr.On("FindByID", uint(3)).Return(complaint, nil)
	cr.On("Update", mock.Anything).Return(nil)
	// goroutine checkAndSuspendProject → mock ไว้รองรับ (may or may not be called)
	cr.On("CountResolvedByProjectID", uint(10)).Return(int64(1), nil).Maybe()
	pr.On("FindProjectByID", uint(10)).Return(&domain.Project{ID: 10, State: domain.StateFunding}, nil).Maybe()

	result, err := svc.AdminResolve(3, 99, "ตรวจสอบแล้ว")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, domain.ComplaintResolved, result.Status)
	assert.Equal(t, "ตรวจสอบแล้ว", result.AdminNote)
}

func TestComplaintService_AdminResolve_NotFound(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	cr.On("FindByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)

	result, err := svc.AdminResolve(99, 1, "note")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

func TestComplaintService_AdminResolve_AlreadyClosed(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	cr.On("FindByID", uint(3)).Return(&domain.Complaint{ID: 3, Status: domain.ComplaintResolved}, nil)

	result, err := svc.AdminResolve(3, 1, "note")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already closed")
}

// AdminReject

func TestComplaintService_AdminReject_Success(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	complaint := &domain.Complaint{ID: 3, ProjectID: 10, Status: domain.ComplaintOpen}
	cr.On("FindByID", uint(3)).Return(complaint, nil)
	cr.On("Update", mock.Anything).Return(nil)

	result, err := svc.AdminReject(3, 99, "ไม่พบปัญหาจริง")

	assert.NoError(t, err)
	assert.Equal(t, domain.ComplaintRejected, result.Status)
	assert.Equal(t, "ไม่พบปัญหาจริง", result.AdminNote)
}

func TestComplaintService_AdminReject_AlreadyClosed(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	cr.On("FindByID", uint(3)).Return(&domain.Complaint{ID: 3, Status: domain.ComplaintRejected}, nil)

	result, err := svc.AdminReject(3, 1, "note")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already closed")
}

// GetProjectStats

func TestComplaintService_GetProjectStats_Success(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	pr.On("FindProjectByID", uint(10)).Return(&domain.Project{ID: 10, Title: "Project A"}, nil)
	cr.On("CountByProjectID", uint(10)).Return(int64(5), nil)
	cr.On("CountResolvedByProjectID", uint(10)).Return(int64(2), nil)

	stats, err := svc.GetProjectStats(10)

	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, "Project A", stats.ProjectTitle)
	assert.Equal(t, int64(5), stats.TotalReports)
	assert.Equal(t, int64(2), stats.ResolvedReports)
	assert.Equal(t, ComplaintSuspendThreshold, stats.Threshold)
}

func TestComplaintService_GetProjectStats_ProjectNotFound(t *testing.T) {
	cr := new(mockComplaintRepo)
	pr := new(ProjectRepository)
	svc := NewComplaintService(cr, pr)

	pr.On("FindProjectByID", uint(99)).Return(nil, gorm.ErrRecordNotFound)

	stats, err := svc.GetProjectStats(99)

	assert.Error(t, err)
	assert.Nil(t, stats)
	assert.Contains(t, err.Error(), "project not found")
}

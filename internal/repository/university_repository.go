package repository

import (
	"context"
	"flyup/internal/domain"
	"strings"

	"gorm.io/gorm"
)

type UniversityRepository interface {
	Create(u *domain.University) error
	FindAll() ([]domain.University, error)
	FindByID(id uint) (*domain.University, error)
	Update(u *domain.University) error
	Delete(id uint) error
	FindByName(nameTH *string, nameEN *string) (*domain.University, error)
	// domain
	CreateDomain(d *domain.UniversityDomain) error
	DeleteDomain(id uint) error
	GetUniversityByDomain(ctx context.Context, domainStr string) (*domain.UniversityDomain, error)
	FindDomainByID(ID uint) (*domain.UniversityDomain, error)
	UpdateDomain(d *domain.UniversityDomain) error
}

type universityRepository struct {
	db *gorm.DB
}

func (r *universityRepository) FindByName(nameTH *string, nameEN *string) (*domain.University, error) {
	var u domain.University

	query := r.db.Model(&domain.University{})

	if nameTH != nil {
		query = query.Or("LOWER(name_th) = ?", strings.ToLower(strings.TrimSpace(*nameTH)))
	}

	if nameEN != nil {
		query = query.Or("LOWER(name_en) = ?", strings.ToLower(strings.TrimSpace(*nameEN)))
	}

	err := query.First(&u).Error
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (r *universityRepository) UpdateDomain(d *domain.UniversityDomain) error {
	return r.db.Save(d).Error
}

func (r *universityRepository) DeleteDomain(id uint) error {
	return r.db.Delete(&domain.UniversityDomain{}, id).Error
}

func (r *universityRepository) CreateDomain(d *domain.UniversityDomain) error {
	return r.db.Create(d).Error
}

func (r *universityRepository) Create(u *domain.University) error {
	return r.db.Create(u).Error
}

func (r *universityRepository) FindAll() ([]domain.University, error) {
	var list []domain.University
	err := r.db.Preload("Domains").Find(&list).Error
	return list, err
}

func (r *universityRepository) FindByID(id uint) (*domain.University, error) {
	var u domain.University
	err := r.db.Preload("Domains").First(&u, id).Error
	return &u, err
}

func (r *universityRepository) Update(u *domain.University) error {
	return r.db.Save(u).Error
}

func (r *universityRepository) Delete(id uint) error {
	return r.db.Select("Domains").Delete(&domain.University{ID: id}).Error
}

func (r *universityRepository) FindDomainByID(ID uint) (*domain.UniversityDomain, error) {
	var university domain.UniversityDomain

	err := r.db.Where("id = ?", ID).First(&university).Error
	if err != nil {
		return nil, err
	}
	return &university, nil
}

func (r *universityRepository) GetUniversityByDomain(ctx context.Context, domainStr string) (*domain.UniversityDomain, error) {
	var uniDomain domain.UniversityDomain

	err := r.db.WithContext(ctx).Preload("University").
		Where("domain = ?", domainStr).
		First(&uniDomain).Error

	if err != nil {
		return nil, err
	}

	// ส่งคืน Pointer ของ University ที่โหลดมาพร้อมกับ Domain
	return &uniDomain, nil
}

func NewUniversityRepository(db *gorm.DB) UniversityRepository {
	return &universityRepository{
		db: db,
	}
}

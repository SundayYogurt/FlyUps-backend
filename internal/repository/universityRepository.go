package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type UniversityRepository interface {
	GetUniversityByDomain(domain string) (*domain.UniversityDomain, error)
	FindUniversityByID(ID uint) (*domain.UniversityDomain, error)
}

type universityRepository struct {
	db *gorm.DB
}

func (u *universityRepository) FindUniversityByID(ID uint) (*domain.UniversityDomain, error) {
	var university domain.UniversityDomain

	err := u.db.Where("id = ?", ID).First(&university).Error
	if err != nil {
		return nil, err
	}
	return &university, nil
}

func (u *universityRepository) GetUniversityByDomain(domainStr string) (*domain.UniversityDomain, error) {
	var uniDomain domain.UniversityDomain

	err := u.db.Preload("University").
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

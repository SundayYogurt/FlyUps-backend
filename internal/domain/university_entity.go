package domain

import (
	"gorm.io/gorm"
)

// University มหาลัย
type University struct {
	ID       uint               `json:"id"`
	NameTH   *string            `json:"name_th,omitempty"`
	NameEN   *string            `json:"name_en,omitempty"`
	Province *string            `json:"province,omitempty"`
	Domains  []UniversityDomain `json:"domains" gorm:"foreignKey:UniversityID"`
	gorm.Model
}

// UniversityDomain โดเมนเมลของมหาลัย
type UniversityDomain struct {
	ID           uint       `json:"id"`
	UniversityID uint       `json:"university_id"`
	University   University `gorm:"foreignKey:UniversityID"`
	Domain       string     `json:"domain"` // e.g. kmutnb.ac.th
	IsActive     bool       `json:"is_active"`
	gorm.Model
}

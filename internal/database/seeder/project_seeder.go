package seeder

import (
	"errors"
	"flyup/internal/domain"

	"gorm.io/gorm"
)

func SeedProjectCategories(db *gorm.DB) error {

	categories := []domain.ProjectCategory{
		{Name: "Technology"},
		{Name: "AI"},
		{Name: "FinTech"},
		{Name: "EdTech"},
		{Name: "HealthTech"},
		{Name: "Gaming"},
		{Name: "Environment"},
		{Name: "Social Impact"},
		{Name: "Education"},
		{Name: "Others"},
	}

	for _, category := range categories {

		var existing domain.ProjectCategory

		err := db.Where("name = ?", category.Name).
			First(&existing).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&category).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

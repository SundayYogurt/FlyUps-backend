package database

import (
	"flyup/internal/domain"
	"flyup/internal/helper"
	"log"

	"gorm.io/gorm"
)

// BackfillProjectSlugs ensures the projects.slug column exists and fills in
// slugs for legacy rows that predate the slug feature, avoiding duplicate
// empty-slug collisions when the unique index migration runs.
func BackfillProjectSlugs(db *gorm.DB) error {
	if err := db.Exec("ALTER TABLE projects ADD COLUMN IF NOT EXISTS slug text DEFAULT ''").Error; err != nil {
		return err
	}

	var legacyProjects []domain.Project
	if err := db.Where("slug = '' OR slug IS NULL").Find(&legacyProjects).Error; err != nil {
		return err
	}

	if len(legacyProjects) == 0 {
		return nil
	}

	log.Printf("found %d projects with empty slug, generating slugs...", len(legacyProjects))
	for _, p := range legacyProjects {
		newSlug := helper.GenerateProjectSlug(p.Title, p.ID)
		if err := db.Model(&domain.Project{}).Where("id = ?", p.ID).Update("slug", newSlug).Error; err != nil {
			return err
		}
	}

	return nil
}
package repository

import (
	"testing"

	"flyup/internal/domain"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUpsertStudentProfileByUserIDPersistsPioneerFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.StudentProfile{}))

	original := domain.StudentProfile{UserID: 42, UniversityID: 1}
	require.NoError(t, db.Create(&original).Error)

	bio := "updated bio"
	portfolio := "https://example.com/portfolio"
	skills := "Go, React"
	faculty := "Engineering"
	major := "Computer Engineering"
	repo := NewUserRepository(db)
	require.NoError(t, repo.UpsertStudentProfileByUserID(&domain.StudentProfile{
		UserID:       42,
		UniversityID: 1,
		Bio:          &bio,
		Portfolio:    &portfolio,
		Skills:       &skills,
		Faculty:      &faculty,
		Major:        &major,
	}))

	var reloaded domain.StudentProfile
	require.NoError(t, db.Where("user_id = ?", 42).First(&reloaded).Error)
	require.Equal(t, bio, requireValue(t, reloaded.Bio))
	require.Equal(t, portfolio, requireValue(t, reloaded.Portfolio))
	require.Equal(t, skills, requireValue(t, reloaded.Skills))
	require.Equal(t, faculty, requireValue(t, reloaded.Faculty))
	require.Equal(t, major, requireValue(t, reloaded.Major))
}

func requireValue(t *testing.T, value *string) string {
	t.Helper()
	require.NotNil(t, value)
	return *value
}

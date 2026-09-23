package repository

import (
	"testing"

	"flyup/internal/domain"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestIdVerificationOCRFieldsPersist(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:id-verification-ocr?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.IdCardVerification{}); err != nil {
		t.Fatal(err)
	}
	repo := NewUserRepository(db)
	payload := `{"total":{"isSamePerson":"true","confidence":91.5}}`
	score := 91.5
	verify := &domain.IdCardVerification{UserID: 7, Status: domain.VerifyStatusApproved, OcrPayload: &payload, FaceScore: &score}
	if err := repo.CreateIdVerification(verify); err != nil {
		t.Fatal(err)
	}
	got, err := repo.FindLatestIdVerification(7)
	if err != nil {
		t.Fatal(err)
	}
	if got.OcrPayload == nil || *got.OcrPayload != payload || got.FaceScore == nil || *got.FaceScore != score {
		t.Fatalf("created OCR fields lost: payload=%v score=%v", got.OcrPayload, got.FaceScore)
	}

	updatedPayload := `{"total":{"isSamePerson":"false","confidence":42}}`
	updatedScore := 42.0
	verify.OcrPayload = &updatedPayload
	verify.FaceScore = &updatedScore
	if err := repo.UpdateIdVerification(verify); err != nil {
		t.Fatal(err)
	}
	got, err = repo.FindLatestIdVerification(7)
	if err != nil {
		t.Fatal(err)
	}
	if got.OcrPayload == nil || *got.OcrPayload != updatedPayload || got.FaceScore == nil || *got.FaceScore != updatedScore {
		t.Fatalf("updated OCR fields lost: payload=%v score=%v", got.OcrPayload, got.FaceScore)
	}
}

package repository

import (
	"errors"
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

func TestIdCardFingerprintIsUniqueAcrossUsers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:id-card-fingerprint?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.IdCardVerification{}); err != nil {
		t.Fatal(err)
	}
	repo := NewUserRepository(db)
	fingerprint := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	first := &domain.IdCardVerification{UserID: 7, CardFingerprint: &fingerprint, Status: domain.VerifyStatusApproved}
	if err := repo.CreateIdVerification(first); err != nil {
		t.Fatal(err)
	}
	owner, err := repo.FindIdVerificationByCardFingerprint(fingerprint)
	if err != nil || owner == nil || owner.UserID != 7 {
		t.Fatalf("card owner was not persisted: owner=%+v err=%v", owner, err)
	}
	second := &domain.IdCardVerification{UserID: 8, CardFingerprint: &fingerprint, Status: domain.VerifyStatusPending}
	if err := repo.CreateIdVerification(second); !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate card was inserted: %v", err)
	}
	otherFingerprint := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	second.CardFingerprint = &otherFingerprint
	if err := repo.CreateIdVerification(second); err != nil {
		t.Fatal(err)
	}
	second.CardFingerprint = &fingerprint
	if err := repo.UpdateIdVerification(second); !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate card replaced another user's card: %v", err)
	}
	if err := db.Delete(first).Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateIdVerification(&domain.IdCardVerification{UserID: 9, CardFingerprint: &fingerprint}); !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("soft-deleted user's card was reused: %v", err)
	}
}

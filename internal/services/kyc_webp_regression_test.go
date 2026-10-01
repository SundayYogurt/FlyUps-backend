package services

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"flyup/config"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"

	"github.com/jarcoal/httpmock"
)

type kycRepoStub struct {
	repository.UserRepository
	saved    *domain.IdCardVerification
	existing *domain.IdCardVerification
	creates  int
	updates  int
}

func (r *kycRepoStub) FindLatestIdVerification(uint) (*domain.IdCardVerification, error) {
	return r.existing, nil
}
func (r *kycRepoStub) FindIdVerificationByCardFingerprint(fingerprint string) (*domain.IdCardVerification, error) {
	if r.saved != nil && r.saved.CardFingerprint != nil && *r.saved.CardFingerprint == fingerprint {
		return r.saved, nil
	}
	return nil, nil
}
func (r *kycRepoStub) CreateIdVerification(v *domain.IdCardVerification) error {
	r.saved = v
	r.creates++
	return nil
}
func (r *kycRepoStub) UpdateIdVerification(v *domain.IdCardVerification) error {
	r.saved = v
	r.updates++
	return nil
}
func (*kycRepoStub) CreateConsents([]*domain.UserConsent) error { return nil }

func registerValidIDCardOCR() {
	httpmock.RegisterResponder(http.MethodPost, "https://api.iapp.co.th/v3/store/ekyc/thai-national-id-card/front", httpmock.NewStringResponder(http.StatusOK, `{"id_number":"1234567890121"}`))
}

func TestVerifyIDRejectsCardAlreadyUsedByAnotherUser(t *testing.T) {
	webp, err := os.ReadFile("../helper/testdata/gopher-doc.1bpp.lossless.webp")
	if err != nil {
		t.Fatal(err)
	}
	const cardURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/duplicate-card.webp"
	const selfieURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/duplicate-selfie.webp"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder(http.MethodGet, cardURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodGet, selfieURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodPost, "https://api.iapp.co.th/v3/store/ekyc/face-and-id-card-verification", httpmock.NewStringResponder(http.StatusOK, `{"total":{"isSamePerson":"true","confidence":90}}`))
	registerValidIDCardOCR()
	repo := &kycRepoStub{}
	svc := NewUserService(repo, nil, nil, config.AppConfig{IAppAPIKey: "test-key", AppSecret: "test-secret"}, nil, nil)
	consent := true
	input := dto.VerifyIDInput{IDCardURL: cardURL, SelfieURL: selfieURL, DeclareTruth: &consent}
	if err := svc.VerifyID(7, input); err != nil {
		t.Fatal(err)
	}
	if err := svc.VerifyID(8, input); err == nil || !strings.Contains(err.Error(), "already used") {
		t.Fatalf("second user reused the same ID card: %v", err)
	}
}

func TestVerifyIDDoesNotSaveCardWithoutValidOCRNumber(t *testing.T) {
	webp, err := os.ReadFile("../helper/testdata/gopher-doc.1bpp.lossless.webp")
	if err != nil {
		t.Fatal(err)
	}
	const cardURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/invalid-number-card.webp"
	const selfieURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/invalid-number-selfie.webp"
	for _, payload := range []string{`{}`, `{"id_number":"1234567890123"}`} {
		httpmock.Activate()
		httpmock.RegisterResponder(http.MethodGet, cardURL, httpmock.NewBytesResponder(http.StatusOK, webp))
		httpmock.RegisterResponder(http.MethodPost, "https://api.iapp.co.th/v3/store/ekyc/thai-national-id-card/front", httpmock.NewStringResponder(http.StatusOK, payload))
		repo := &kycRepoStub{}
		svc := NewUserService(repo, nil, nil, config.AppConfig{IAppAPIKey: "test-key", AppSecret: "test-secret"}, nil, nil)
		consent := true
		err := svc.VerifyID(7, dto.VerifyIDInput{IDCardURL: cardURL, SelfieURL: selfieURL, DeclareTruth: &consent})
		if err == nil || repo.saved != nil {
			t.Fatalf("invalid OCR number was accepted: payload=%s err=%v saved=%+v", payload, err, repo.saved)
		}
		httpmock.DeactivateAndReset()
	}
}

func TestVerifyIDPreservesRejectedCardWhenUserSubmitsDifferentCard(t *testing.T) {
	webp, err := os.ReadFile("../helper/testdata/gopher-doc.1bpp.lossless.webp")
	if err != nil {
		t.Fatal(err)
	}
	const cardURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/replacement-card.webp"
	const selfieURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/replacement-selfie.webp"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerValidIDCardOCR()
	httpmock.RegisterResponder(http.MethodGet, cardURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodGet, selfieURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodPost, "https://api.iapp.co.th/v3/store/ekyc/face-and-id-card-verification", httpmock.NewStringResponder(http.StatusOK, `{"total":{"isSamePerson":"true","confidence":90}}`))
	oldFingerprint := "old-fingerprint"
	repo := &kycRepoStub{existing: &domain.IdCardVerification{ID: 11, UserID: 7, Status: domain.VerifyStatusRejected, CardFingerprint: &oldFingerprint}}
	svc := NewUserService(repo, nil, nil, config.AppConfig{IAppAPIKey: "test-key", AppSecret: "test-secret"}, nil, nil)
	consent := true
	if err := svc.VerifyID(7, dto.VerifyIDInput{IDCardURL: cardURL, SelfieURL: selfieURL, DeclareTruth: &consent}); err != nil {
		t.Fatal(err)
	}
	if repo.creates != 1 || repo.updates != 0 || repo.existing.CardFingerprint == nil || *repo.existing.CardFingerprint != oldFingerprint {
		t.Fatalf("previous card claim was overwritten: creates=%d updates=%d existing=%+v", repo.creates, repo.updates, repo.existing)
	}
}

func TestVerifyIDApprovesAtThirtyPercent(t *testing.T) {
	webp, err := os.ReadFile("../helper/testdata/gopher-doc.1bpp.lossless.webp")
	if err != nil {
		t.Fatal(err)
	}
	const cardURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/threshold-card.webp"
	const selfieURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/threshold-selfie.webp"
	for _, tc := range []struct {
		name    string
		payload string
		want    domain.VerifyStatus
	}{
		{"at threshold", `{"total":{"isSamePerson":"true","confidence":30}}`, domain.VerifyStatusApproved},
		{"reported 57.094 result", `{"total":{"isSamePerson":"true","confidence":57.094}}`, domain.VerifyStatusApproved},
		{"below threshold", `{"total":{"isSamePerson":"true","confidence":29.999}}`, domain.VerifyStatusPending},
		{"different person", `{"total":{"isSamePerson":"false","confidence":57.094}}`, domain.VerifyStatusPending},
		{"missing same-person result", `{"total":{"confidence":30}}`, domain.VerifyStatusPending},
	} {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			registerValidIDCardOCR()
			httpmock.RegisterResponder(http.MethodGet, cardURL, httpmock.NewBytesResponder(http.StatusOK, webp))
			httpmock.RegisterResponder(http.MethodGet, selfieURL, httpmock.NewBytesResponder(http.StatusOK, webp))
			httpmock.RegisterResponder(http.MethodPost, "https://api.iapp.co.th/v3/store/ekyc/face-and-id-card-verification", httpmock.NewStringResponder(http.StatusOK, tc.payload))
			repo := &kycRepoStub{}
			svc := NewUserService(repo, nil, nil, config.AppConfig{IAppAPIKey: "test-key", AppSecret: "test-secret"}, nil, nil)
			consent := true
			if err := svc.VerifyID(7, dto.VerifyIDInput{IDCardURL: cardURL, SelfieURL: selfieURL, DeclareTruth: &consent}); err != nil {
				t.Fatal(err)
			}
			if repo.saved == nil || repo.saved.Status != tc.want || (repo.saved.VerifiedAt != nil) != (tc.want == domain.VerifyStatusApproved) {
				t.Fatalf("unexpected verification: %+v", repo.saved)
			}
		})
	}
}

func TestVerifyIDUsesUploadedWebPWithoutCloudinaryURLRewrite(t *testing.T) {
	webp, err := os.ReadFile("../helper/testdata/gopher-doc.1bpp.lossless.webp")
	if err != nil {
		t.Fatal(err)
	}
	const cardURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/card.webp"
	const selfieURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/selfie.webp"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerValidIDCardOCR()
	httpmock.RegisterResponder(http.MethodGet, cardURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodGet, selfieURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodPost, "https://api.iapp.co.th/v3/store/ekyc/face-and-id-card-verification", httpmock.NewStringResponder(http.StatusOK, `{"total":{"isSamePerson":"true","confidence":90}}`))
	repo := &kycRepoStub{}
	svc := NewUserService(repo, nil, nil, config.AppConfig{IAppAPIKey: "test-key", AppSecret: "test-secret"}, nil, nil)
	consent := true
	err = svc.VerifyID(7, dto.VerifyIDInput{IDCardURL: cardURL, SelfieURL: selfieURL, DeclareTruth: &consent})
	if err != nil {
		t.Fatal(err)
	}
	if repo.saved == nil || repo.saved.Status != domain.VerifyStatusApproved || repo.saved.Document != cardURL || repo.saved.SelfieURL != selfieURL {
		t.Fatalf("WebP verification was not saved correctly: %+v", repo.saved)
	}
	if repo.saved.OcrPayload == nil || *repo.saved.OcrPayload == "" || repo.saved.FaceScore == nil || *repo.saved.FaceScore != 90 {
		t.Fatalf("iApp response was not saved: payload=%v score=%v", repo.saved.OcrPayload, repo.saved.FaceScore)
	}
}

func TestVerifyIDRetriesPendingRecordWithMissingOCRResult(t *testing.T) {
	webp, err := os.ReadFile("../helper/testdata/gopher-doc.1bpp.lossless.webp")
	if err != nil {
		t.Fatal(err)
	}
	const cardURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/retry-card.webp"
	const selfieURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/retry-selfie.webp"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerValidIDCardOCR()
	httpmock.RegisterResponder(http.MethodGet, cardURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodGet, selfieURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodPost, "https://api.iapp.co.th/v3/store/ekyc/face-and-id-card-verification", httpmock.NewStringResponder(http.StatusOK, `{"total":{"isSamePerson":"true","confidence":90}}`))
	repo := &kycRepoStub{existing: &domain.IdCardVerification{ID: 11, UserID: 7, Status: domain.VerifyStatusPending}}
	svc := NewUserService(repo, nil, nil, config.AppConfig{IAppAPIKey: "test-key", AppSecret: "test-secret"}, nil, nil)
	consent := true
	if err := svc.VerifyID(7, dto.VerifyIDInput{IDCardURL: cardURL, SelfieURL: selfieURL, DeclareTruth: &consent}); err != nil {
		t.Fatal(err)
	}
	if repo.saved == nil || repo.saved.ID != 11 || repo.saved.Status != domain.VerifyStatusApproved || repo.saved.FaceScore == nil || *repo.saved.FaceScore != 90 || repo.saved.OcrPayload == nil || *repo.saved.OcrPayload == "" {
		t.Fatalf("pending verification was not refreshed with OCR result: %+v", repo.saved)
	}
}

func TestVerifyIDDoesNotRetryPendingRecordWithOCRResult(t *testing.T) {
	payload := `{"total":{"isSamePerson":"false","confidence":40}}`
	score := 40.0
	repo := &kycRepoStub{existing: &domain.IdCardVerification{
		ID: 11, UserID: 7, Status: domain.VerifyStatusPending,
		Document:   "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/card.webp",
		SelfieURL:  "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/selfie.webp",
		OcrPayload: &payload, FaceScore: &score,
	}}
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)
	consent := true
	err := svc.VerifyID(7, dto.VerifyIDInput{
		IDCardURL:    "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/card.webp",
		SelfieURL:    "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/selfie.webp",
		DeclareTruth: &consent,
	})
	if err == nil || err.Error() != "verification is already pending" || repo.saved != nil {
		t.Fatalf("pending verification with OCR result was retried: err=%v saved=%+v", err, repo.saved)
	}
}

func TestVerifyIDRejectsUndetectableImagesWithoutCreatingPendingRecord(t *testing.T) {
	webp, err := os.ReadFile("../helper/testdata/gopher-doc.1bpp.lossless.webp")
	if err != nil {
		t.Fatal(err)
	}
	const cardURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/bad-card.webp"
	const selfieURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/bad-selfie.webp"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerValidIDCardOCR()
	httpmock.RegisterResponder(http.MethodGet, cardURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodGet, selfieURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodPost, "https://api.iapp.co.th/v3/store/ekyc/face-and-id-card-verification", httpmock.NewStringResponder(421, `{"message":"face on id card or id card not found in image [file1]"}`))
	repo := &kycRepoStub{}
	svc := NewUserService(repo, nil, nil, config.AppConfig{IAppAPIKey: "test-key", AppSecret: "test-secret"}, nil, nil)
	consent := true
	err = svc.VerifyID(7, dto.VerifyIDInput{IDCardURL: cardURL, SelfieURL: selfieURL, DeclareTruth: &consent})
	if err == nil || !strings.Contains(err.Error(), "retake") || repo.saved != nil {
		t.Fatalf("undetectable images were accepted: err=%v saved=%+v", err, repo.saved)
	}
}

func TestVerifyIDKeepsProviderOutagePendingForReview(t *testing.T) {
	webp, err := os.ReadFile("../helper/testdata/gopher-doc.1bpp.lossless.webp")
	if err != nil {
		t.Fatal(err)
	}
	const cardURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/outage-card.webp"
	const selfieURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/outage-selfie.webp"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerValidIDCardOCR()
	httpmock.RegisterResponder(http.MethodGet, cardURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodGet, selfieURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodPost, "https://api.iapp.co.th/v3/store/ekyc/face-and-id-card-verification", httpmock.NewStringResponder(http.StatusPaymentRequired, `{"message":"insufficient credits"}`))
	repo := &kycRepoStub{}
	svc := NewUserService(repo, nil, nil, config.AppConfig{IAppAPIKey: "test-key", AppSecret: "test-secret"}, nil, nil)
	consent := true
	if err := svc.VerifyID(7, dto.VerifyIDInput{IDCardURL: cardURL, SelfieURL: selfieURL, DeclareTruth: &consent}); err != nil {
		t.Fatal(err)
	}
	if repo.saved == nil || repo.saved.Status != domain.VerifyStatusPending || repo.saved.FaceScore != nil {
		t.Fatalf("provider outage did not fall back to review: %+v", repo.saved)
	}
}

func TestVerifyIDExplainsSelfieCardDetectionFailure(t *testing.T) {
	webp, err := os.ReadFile("../helper/testdata/gopher-doc.1bpp.lossless.webp")
	if err != nil {
		t.Fatal(err)
	}
	const cardURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/selfie-error-card.webp"
	const selfieURL = "https://res.cloudinary.com/dsvexmpb6/image/upload/v1/selfie-error-selfie.webp"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	registerValidIDCardOCR()
	httpmock.RegisterResponder(http.MethodGet, cardURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodGet, selfieURL, httpmock.NewBytesResponder(http.StatusOK, webp))
	httpmock.RegisterResponder(http.MethodPost, "https://api.iapp.co.th/v3/store/ekyc/face-and-id-card-verification", httpmock.NewStringResponder(422, `{"message":"face on selfie not found in image [file0]"}`))
	repo := &kycRepoStub{}
	svc := NewUserService(repo, nil, nil, config.AppConfig{IAppAPIKey: "test-key", AppSecret: "test-secret"}, nil, nil)
	consent := true
	err = svc.VerifyID(7, dto.VerifyIDInput{IDCardURL: cardURL, SelfieURL: selfieURL, DeclareTruth: &consent})
	if err == nil || !strings.Contains(err.Error(), "entire card in frame") || repo.saved != nil {
		t.Fatalf("selfie card detection error was not explained: err=%v saved=%+v", err, repo.saved)
	}
}

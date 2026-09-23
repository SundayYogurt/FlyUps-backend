package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type stubPaymentRepository struct {
	complete func(string, float64, float64, float64) (repository.PaymentResult, error)
	fail     func(string) (repository.PaymentResult, error)
}

func (s stubPaymentRepository) Complete(id string, fee, vat, net float64) (repository.PaymentResult, error) {
	return s.complete(id, fee, vat, net)
}
func (s stubPaymentRepository) Fail(id string) (repository.PaymentResult, error) {
	return s.fail(id)
}
func (s stubPaymentRepository) Expire(uint, time.Time) (bool, error) { return false, nil }

func signedPaymentEvent(secret, eventType, intentID string) ([]byte, string) {
	payload := []byte(fmt.Sprintf(`{"id":"evt_test","object":"event","type":%q,"data":{"object":{"id":%q}}}`, eventType, intentID))
	timestamp := fmt.Sprint(time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "." + string(payload)))
	return payload, "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestWebhookReturnsDatabaseFailureForRetry(t *testing.T) {
	want := errors.New("database unavailable")
	for _, eventType := range []string{"payment_intent.succeeded", "payment_intent.payment_failed"} {
		t.Run(eventType, func(t *testing.T) {
			svc := &investmentService{webhookSecret: "whsec_test", feeLookup: func(string) (float64, float64, float64) { return 0, 0, 0 }}
			svc.paymentRepo = stubPaymentRepository{
				complete: func(id string, _, _, _ float64) (repository.PaymentResult, error) {
					return repository.PaymentResult{}, want
				},
				fail: func(id string) (repository.PaymentResult, error) { return repository.PaymentResult{}, want },
			}
			payload, signature := signedPaymentEvent(svc.webhookSecret, eventType, "pi_1")
			if err := svc.HandleStripeWebhook(payload, signature); !errors.Is(err, want) {
				t.Fatalf("expected retriable DB error, got %v", err)
			}
		})
	}
}

func TestWebhookRejectsMalformedIntentID(t *testing.T) {
	svc := &investmentService{webhookSecret: "whsec_test"}
	payload, signature := signedPaymentEvent(svc.webhookSecret, "payment_intent.succeeded", "")
	err := svc.HandleStripeWebhook(payload, signature)
	if !errors.Is(err, ErrInvalidStripeWebhook) || !strings.Contains(err.Error(), "missing payment intent id") {
		t.Fatalf("expected bad payload, got %v", err)
	}
}

func TestSecondQRUsesAvailableFundingAfterReservations(t *testing.T) {
	project := &domain.Project{FundingGoal: 1000, CurrentFunding: 0, MinInvestAmount: 20}
	if err := validateAmount(project, 1000); err != nil {
		t.Fatalf("first QR should fit: %v", err)
	}
	if err := validateAmountWithReserved(project, 1000, 1000); err == nil {
		t.Fatal("second QR exceeded the remaining capacity")
	}
}

type verifiedQRUserRepo struct{ repository.UserRepository }

func (verifiedQRUserRepo) FindUserById(id uint) (*domain.User, error) {
	return &domain.User{ID: id, Role: "booster", IdCardVerification: &domain.IdCardVerification{Status: domain.VerifyStatusApproved}}, nil
}

func TestCreateInvestmentRejectsSecondActiveQR(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:second-qr?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, schema := range []string{
		`CREATE TABLE projects (id integer primary key, owner_user_id integer, title text, state text, status text, funding_goal real, current_funding real, min_invest_amount real, max_invest_amount real, softcap real, platform_fee real, deleted_at datetime)`,
		`CREATE TABLE investments (id integer primary key, project_id integer, status text, total_amount real, created_at datetime, deleted_at datetime)`,
		`CREATE TABLE transactions (id integer primary key, investment_id integer, status text, expires_at datetime, deleted_at datetime)`,
	} {
		if err := db.Exec(schema).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { sqlDB, _ := db.DB(); _ = sqlDB.Close() })
	if err := db.Exec(`INSERT INTO projects (id, owner_user_id, title, state, status, funding_goal, current_funding, min_invest_amount) VALUES (1, 10, 'P', 'funding', 'active', 1000, 0, 20)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO investments (id, project_id, status, total_amount, created_at) VALUES (1, 1, 'pending_payment', 1000, ?)`, time.Now()).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO transactions (id, investment_id, status, expires_at) VALUES (1, 1, 'pending', ?)`, time.Now().Add(5*time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewInvestmentService(repository.NewProjectRepository(db), nil, nil, verifiedQRUserRepo{}, nil, "", "", nil, nil, db)
	if _, err := svc.CreateInvestment(11, "booster@example.com", dto.CreateInvestmentRequest{ProjectID: 1, Amount: 1000}); err == nil {
		t.Fatal("second active QR was accepted for the last 1000 baht")
	}
}

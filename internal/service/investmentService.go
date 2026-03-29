package service

import (
	"crypto/rand"
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"fmt"
	"log"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/paymentintent"
	"github.com/stripe/stripe-go/v85/paymentmethod"
	"github.com/stripe/stripe-go/v85/webhook"
	"gorm.io/gorm"
)

type InvestmentService interface {
	GetInvestment(boosterUserID uint, investmentID uint) (*domain.Investment, *domain.Transaction, error)
	CreateInvestment(boosterUserID uint, boosterEmail string, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error)
	ListUserInvestments(boosterUserID uint) ([]domain.Investment, error)
	HandleStripeWebhook(payload []byte, sigHeader string) error
}

type investmentService struct {
	projectRepo     repository.ProjectRepository
	investmentRepo  repository.InvestmentRepository
	transactionRepo repository.TransactionRepository
	stripeSecretKey string
	webhookSecret   string
}

func NewInvestmentService(projectRepo repository.ProjectRepository, investmentRepo repository.InvestmentRepository, transactionRepo repository.TransactionRepository, stripeSecretKey string, webhookSecret string) InvestmentService {
	return &investmentService{projectRepo, investmentRepo, transactionRepo, stripeSecretKey, webhookSecret}
}

func (s *investmentService) GetInvestment(boosterUserID uint, investmentID uint) (*domain.Investment, *domain.Transaction, error) {
	investment, err := s.investmentRepo.FindByID(investmentID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("investment not found")
		}
		return nil, nil, errors.New("internal server error")
	}

	if investment.BoosterUserID != boosterUserID {
		return nil, nil, errors.New("investment not found")
	}

	txn, err := s.transactionRepo.FindByInvestmentID(investmentID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, errors.New("internal server error")
	}

	return investment, txn, nil
}

func (s *investmentService) CreateInvestment(boosterUserID uint, boosterEmail string, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error) {
	project, err := s.projectRepo.FindProjectByID(req.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("project not found")
		}
		return nil, errors.New("internal server error")
	}

	if project.State != domain.StateFunding {
		return nil, errors.New("project is not open for investment")
	}

	if err := validateAmount(project, req.Amount); err != nil {
		return nil, err
	}

	if project.FundingGoal > 0 {
		currentTotal, err := s.investmentRepo.SumActiveByProjectID(req.ProjectID)
		if err != nil {
			return nil, errors.New("internal server error")
		}
		if currentTotal+req.Amount > project.FundingGoal {
			remaining := project.FundingGoal - currentTotal
			return nil, fmt.Errorf("investment exceeds funding goal, remaining ฿%.0f", remaining)
		}
	}

	fee, vat, principal := calculateFees(req.Amount, project.PlatformFee)

	refNum, err := generateReferenceNumber()
	if err != nil {
		return nil, errors.New("failed to generate reference number")
	}

	investment := &domain.Investment{
		ReferenceNumber: refNum,
		ProjectID:       req.ProjectID,
		BoosterUserID:   boosterUserID,
		TotalAmount:     req.Amount,
		PlatformFee:     fee,
		VATAmount:       vat,
		PrincipalAmount: principal,
		ProfitSharePct:  project.ProfitSharePct,
		Status:          domain.InvestmentPending,
	}

	if err := s.investmentRepo.Create(investment); err != nil {
		log.Printf("[CreateInvestment] db error: %v", err)
		return nil, errors.New("failed to create investment")
	}

	// สร้าง Stripe QR Code
	qrURL, intentID, clientSecret, expiresAt, err := s.createStripePromptPay(req.Amount, refNum, project.Title, boosterEmail)
	if err != nil {
		// Stripe ล้มเหลว → mark investment เป็น rejected
		_ = s.investmentRepo.UpdateStatus(investment.ID, domain.InvestmentRejected)
		log.Printf("[CreateInvestment] stripe error: %v", err)
		return nil, errors.New("failed to create payment QR code")
	}

	txn := &domain.Transaction{
		InvestmentID:          investment.ID,
		StripePaymentIntentID: intentID,
		StripeClientSecret:    clientSecret,
		QRCodeImageURL:        qrURL,
		ExpiresAt:             expiresAt,
		Status:                domain.TransactionPending,
	}

	if err := s.transactionRepo.Create(txn); err != nil {
		log.Printf("[CreateInvestment] transaction db error: %v", err)
		return nil, errors.New("failed to save transaction")
	}

	return &dto.InvestmentResponse{
		InvestmentID:    investment.ID,
		ReferenceNumber: refNum,
		QRCodeImageURL:  qrURL,
		ExpiresAt:       expiresAt.Format(time.RFC3339),
		TotalAmount:     req.Amount,
		Title:           project.Title, // ตรงกับ Project.Title ของ friend
	}, nil
}

// all my investments
func (s *investmentService) ListUserInvestments(boosterUserID uint) ([]domain.Investment, error) {
	return s.investmentRepo.ListByBoosterUserID(boosterUserID)
}

func (s *investmentService) HandleStripeWebhook(payload []byte, sigHeader string) error {
	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, s.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return fmt.Errorf("webhook signture verification failed: %v", err)
	}

	switch event.Type {
	case "payment_intent.succeeded":
		if id, ok := event.Data.Object["id"].(string); ok {
			s.handlePaymentSucceeded(id)
		}
	case "payment_intent.payment_failed":
		if id, ok := event.Data.Object["id"].(string); ok {
			s.handlePaymentFailed(id)
		}
	}

	return nil
}

// // private methods

// // เรียก Stripe API เพื่อสร้าง QR Code PromptPay
func (s *investmentService) createStripePromptPay(amount float64, refNum string, projectTitle string, email string) (qrURL, intentID, clientSecret string, expiresAt time.Time, err error) {
	stripe.Key = s.stripeSecretKey

	pm, err := paymentmethod.New(&stripe.PaymentMethodParams{
		Type: stripe.String("promptpay"),
		BillingDetails: &stripe.PaymentMethodBillingDetailsParams{
			Email: stripe.String(email),
		},
	})
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("create payment method error: %v", err)
	}

	amountInSatang := int64(math.Round(amount * 100))

	pi, err := paymentintent.New(&stripe.PaymentIntentParams{
		Amount:             stripe.Int64(amountInSatang),
		Currency:           stripe.String("thb"),
		PaymentMethodTypes: []*string{stripe.String("promptpay")},
		PaymentMethod:      stripe.String(pm.ID),
		Confirm:            stripe.Bool(true),
		Metadata: map[string]string{
			"reference_number": refNum,
			"project_title":    projectTitle, // เปลี่ยนจาก project_name → project_title
		},
	})
	if err != nil {
		return "", "", "", time.Time{}, fmt.Errorf("create payment intent error: %v", err)
	}

	if pi.NextAction == nil || pi.NextAction.PromptPayDisplayQRCode == nil {
		return "", "", "", time.Time{}, errors.New("promptpay QR code not returned by Stripe")
	}

	return pi.NextAction.PromptPayDisplayQRCode.ImageURLPNG,
		pi.ID,
		pi.ClientSecret,
		time.Now().Add(5 * time.Minute),
		nil
}

func (s *investmentService) handlePaymentSucceeded(intentID string) {
	txn, err := s.transactionRepo.FindByPaymentIntentID(intentID)
	if err != nil {
		log.Printf("[Webhook] transaction not found for intent %s: %v", intentID, err)
		return
	}

	if err := s.transactionRepo.UpdateStatus(txn.ID, domain.TransactionSucceeded); err != nil {
		log.Printf("[Webhook] update transaction status error: %v", err)
		return
	}

	investment, err := s.investmentRepo.FindByID(txn.InvestmentID)
	if err != nil {
		log.Printf("[Webhook] investment not found: %v", err)
		return
	}

	now := time.Now()
	investment.Status = domain.InvestmentVerified // "verified" ตาม friend's constant
	investment.PaidAt = &now

	if err := s.investmentRepo.UpdatePaid(investment); err != nil {
		log.Printf("[Webhook] update investment error: %v", err)
	}
}

func (s *investmentService) handlePaymentFailed(intentID string) {
	txn, err := s.transactionRepo.FindByPaymentIntentID(intentID)
	if err != nil {
		log.Printf("[Webhook] transaction not found for intent %s: %v", intentID, err)
		return
	}

	_ = s.transactionRepo.UpdateStatus(txn.ID, domain.TransactionFailed)
	_ = s.investmentRepo.UpdateStatus(txn.InvestmentID, domain.InvestmentRejected)
}

// // helper functions

func validateAmount(project *domain.Project, amount float64) error {
	effectiveMin := math.Max(project.MinInvestAmount, project.FundingGoal*0.01)
	if amount < effectiveMin {
		return fmt.Errorf("minimum investment is ฿%.0f (1%% of funding goal)", effectiveMin)
	}

	if project.MaxInvestAmount > 0 && amount > project.MaxInvestAmount {
		return fmt.Errorf("maximum investment is ฿%.0f", project.MaxInvestAmount)
	}

	return nil
}

// // คำนวณค่าธรรมเนียมและยอดสุทธิ
func calculateFees(amount float64, platformFeePct float64) (fee, vat, principal float64) {
	fee = math.Round(amount*(platformFeePct/100)*100) / 100

	vat = math.Round(fee*0.07*100) / 100

	principal = math.Round((amount-fee-vat)*100) / 100

	return fee, vat, principal
}

// // สร้างเลขอ้างอิงแบบสุ่ม เช่น "INV-ABCDEFGH"
func generateReferenceNumber() (string, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	result := make([]byte, 8)

	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		result[i] = chars[n.Int64()]
	}

	return "INV-" + strings.ToUpper(string(result)), nil
}

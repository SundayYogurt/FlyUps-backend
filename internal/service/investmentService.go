package service

import (
	"crypto/rand"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"math"
	"math/big"
	"strings"
)

type InvestmentService interface {
	CreateInvestment(userID uint, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error)
}

type investmentService struct {
	investmentRepo  repository.InvestmentRepository
	stripeSecretKey string
	webhookSecret   string
}

// func NewInvestmentService(investmentRepo repository.InvestmentRepository, stripeSecretKey string, webhookSecret string) InvestmentService {
// 	return &investmentService{investmentRepo, stripeSecretKey, webhookSecret}
// }

// func (s *investmentService) CreateInvestment(userID uint, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error) {
// 	project, err := s.projectRepo.FindByID(req.ProjectID)

// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			return nil, errors.New("project not found")
// 		}
// 		return nil, errors.New("internal server error")
// 	}

// 	if project.Status != domain.ProjectStatusActive {
// 		return nil, errors.New("project is not open for investment")
// 	}

// 	if err := validateAmount(project, req.Amount); err != nil {
// 		return nil, err
// 	}

// 	fee, vat, net := calculateFees(req.Amount, project.PlatformFeePct)

// 	refNum, err := generateReferenceNumber()
// 	if err != nil {
// 		return nil, errors.New("failed to generate reference number")
// 	}

// 	investment := &domain.Investment{
// 		ReferenceNumber: refNum,
// 		UserID:          userID,
// 		ProjectID:       req.ProjectID,
// 		Amount:          req.Amount,
// 		PlatformFee:     fee,
// 		VATAmount:       vat,
// 		NetAmount:       net,
// 		ProfitSharePct:  project.ProfitSharePct,
// 		Status:          domain.InvestmentStatusPending,
// 	}

// 	if err := s.investmentRepo.Create(investment); err != nil {
// 		log.Printf("[CreateInvestment] db error: %v", err)
// 		return nil, errors.New("failed to create investment")
// 	}

// 	// qrURL, intentID, clientSecret, expiresAt, err := s.

// }

// private methods

// เรียก Stripe API เพื่อสร้าง QR Code PromptPay
// func (s *investmentService) createStripePromptPay(amount float64, refNum string, projectName string) (string, string, string, time.Time, error) {
// 	// set Stripe API key ก่อนเรียก API ทุกครั้ง
// 	stripe.Key = s.stripeSecretKey

// 	pm, err := paymentmethod.New(&stripe.PaymentMethodParams{
// 		Type: stripe.String("propmtpay"),
// 	})

// 	if err != nil {

// 	}
// }

// // helper functions

// func validateAmount(project *domain.Project, amount float64) error {
// 	if amount < project.MinInvestAmount {
// 		return fmt.Errorf("minimum investment is ฿%.0f", project.MinInvestAmount)
// 	}

// 	if project.MaxInvestAmount > 0 && amount > project.MaxInvestAmount {
// 		return fmt.Errorf("maximum investment is ฿%.0f", project.MaxInvestAmount)
// 	}

// 	return nil
// }

// คำนวณค่าธรรมเนียมและยอดสุทธิ
func calculateFees(amount float64, platformFeePct float64) (fee, vat, net float64) {
	fee = math.Round(amount*(platformFeePct/100)*100) / 100

	vat = math.Round(fee*0.07*100) / 100

	net = math.Round((amount-fee-vat)*100) / 100

	return fee, vat, net
}

// สร้างเลขอ้างอิงแบบสุ่ม เช่น "INV-ABCDEFGH"
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

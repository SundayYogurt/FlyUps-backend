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
	"github.com/stripe/stripe-go/v85/charge"
	"github.com/stripe/stripe-go/v85/paymentintent"
	"github.com/stripe/stripe-go/v85/paymentmethod"
	"github.com/stripe/stripe-go/v85/webhook"
	"gorm.io/gorm"
)

// MaxInvestmentPerTransaction คือเพดานต่อรายการ (THB)
// — ต่ำกว่า limit ของ Stripe API (~999,999.99 THB) เพื่อเผื่อ fees/rounding
// — เลขกลมตามมาตรฐาน fintech ไทย; ผู้ใช้ที่ลงทุนสูงกว่านี้ต้องแบ่งหลายรายการ
const MaxInvestmentPerTransaction = 500_000.0

type InvestmentService interface {
	GetInvestment(boosterUserID uint, investmentID uint) (*domain.Investment, *domain.Transaction, error)
	CreateInvestment(boosterUserID uint, boosterEmail string, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error)
	ListUserInvestments(boosterUserID uint) ([]domain.Investment, error)
	HandleStripeWebhook(payload []byte, sigHeader string) error
	RefundInvestment(boosterUserID uint, investmentID uint, note string) (*dto.RefundResponse, error)
	ApproveRefund(investmentID uint) error
	ListRefundRequests() ([]dto.RefundRequestItem, error)
	GetProjectInvestors(projectID uint) ([]dto.ProjectInvestorItem, error)
	ListInvestedProjects(boosterUserID uint) ([]dto.InvestedProjectItem, error)
	VoteMilestone(boosterUserID uint, milestoneID uint, choice domain.MilestoneVoteChoice) (*domain.MilestoneVote, error)
}

type investmentService struct {
	projectRepo      repository.ProjectRepository
	investmentRepo   repository.InvestmentRepository
	transactionRepo  repository.TransactionRepository
	userRepo         repository.UserRepository
	disbursementRepo repository.DisbursementRepository
	stripeSecretKey  string
	webhookSecret    string
	notifSvc         NotificationService
}

func NewInvestmentService(projectRepo repository.ProjectRepository, investmentRepo repository.InvestmentRepository, transactionRepo repository.TransactionRepository, userRepo repository.UserRepository, disbursementRepo repository.DisbursementRepository, stripeSecretKey string, webhookSecret string, notifSvc NotificationService) InvestmentService {
	return &investmentService{projectRepo, investmentRepo, transactionRepo, userRepo, disbursementRepo, stripeSecretKey, webhookSecret, notifSvc}
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
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return investment, nil, nil
		}
		return nil, nil, errors.New("internal server error")
	}

	return investment, txn, nil
}

func (s *investmentService) CreateInvestment(boosterUserID uint, boosterEmail string, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error) {
	user, err := s.userRepo.FindUserById(boosterUserID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if user.Role == "admin" {
		return nil, errors.New("admin cannot invest")
	}

	project, err := s.projectRepo.FindProjectByID(req.ProjectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("project not found")
		}
		return nil, errors.New("internal server error")
	}

	if project.OwnerUserID == boosterUserID {
		return nil, errors.New("cannot invest in your own project")
	}

	if project.State != domain.StateFunding {
		return nil, errors.New("project is not open for investment")
	}

	if err := validateAmount(project, req.Amount); err != nil {
		return nil, err
	}

	if req.Amount > MaxInvestmentPerTransaction {
		return nil, fmt.Errorf("ยอดลงทุนต่อรายการต้องไม่เกิน ฿%.0f (กรุณาแบ่งเป็นหลายรายการหากต้องการลงทุนสูงกว่านี้)", MaxInvestmentPerTransaction)
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

func (s *investmentService) RefundInvestment(boosterUserID uint, investmentID uint, note string) (*dto.RefundResponse, error) {
	investment, err := s.investmentRepo.FindByID(investmentID)
	if err != nil {
		return nil, errors.New("investment not found")
	}

	if investment.BoosterUserID != boosterUserID {
		return nil, errors.New("investment not found")
	}

	if investment.Status != domain.InvestmentVerified {
		return nil, errors.New("only verified investments can be refunded")
	}

	project, err := s.projectRepo.FindProjectByID(investment.ProjectID)
	if err != nil {
		return nil, errors.New("project not found")
	}

	if project.State != domain.StateFunding {
		return nil, errors.New("refund is only allowed while project project is in funding state")
	}

	refundAmount := investment.PrincipalAmount

	now := time.Now()
	investment.Status = domain.InvestmentRefundPending
	investment.RefundAmount = refundAmount
	investment.RefundNote = note
	investment.RefundedAt = &now

	if err := s.investmentRepo.UpdateRefunded(investment); err != nil {
		log.Printf("[RefundInvestment] db update error: %v", err)
		return nil, errors.New("refund processed but failed to update record")
	}

	if err := s.investmentRepo.IncrementProjectFunding(investment.ProjectID, -investment.TotalAmount); err != nil {
		log.Printf("[RefundInvestment] deccrement current_funding error: %v", err)
	}

	feesDeducted := investment.TotalAmount - refundAmount
	return &dto.RefundResponse{
		InvestmentID:    investment.ID,
		ReferenceNumber: investment.ReferenceNumber,
		RefundAmount:    refundAmount,
		TotalPaid:       investment.TotalAmount,
		FeesDeducted:    math.Round(feesDeducted*100) / 100,
	}, nil
}

func (s *investmentService) ListRefundRequests() ([]dto.RefundRequestItem, error) {
	investments, err := s.investmentRepo.ListRefundPending()
	if err != nil {
		return nil, errors.New("internal server error")
	}

	var result []dto.RefundRequestItem
	for _, inv := range investments {
		item := dto.RefundRequestItem{
			InvestmentID:    inv.ID,
			ReferenceNumber: inv.ReferenceNumber,
			BoosterUserID:   inv.BoosterUserID,
			RefundAmount:    inv.RefundAmount,
			TotalPaid:       inv.TotalAmount,
		}

		if inv.RefundedAt != nil {
			item.RequestedAt = inv.RefundedAt.Format(time.RFC3339)
		}

		user, err := s.userRepo.FindUserById(inv.BoosterUserID)
		if err == nil {
			item.BoosterName = user.FirstName + " " + user.LastName
			if user.BankAccount != nil {
				item.BankAccount = &dto.RefundBankAccount{
					BankName:      user.BankAccount.BankName,
					AccountName:   user.BankAccount.AccountName,
					AccountNumber: user.BankAccount.AccountNumber,
				}
			}
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *investmentService) ApproveRefund(investmentID uint) error {
	investment, err := s.investmentRepo.FindByID(investmentID)
	if err != nil {
		return errors.New("investment not found")
	}

	if investment.Status != domain.InvestmentRefundPending {
		return errors.New("investment is not pending refund")
	}

	now := time.Now()
	investment.Status = domain.InvestmentRefunded
	investment.RefundedAt = &now

	if err := s.investmentRepo.UpdateRefunded(investment); err != nil {
		return errors.New("failed to approve refund")
	}

	if err := s.investmentRepo.IncrementProjectFunding(investment.ProjectID, -investment.TotalAmount); err != nil {
		log.Printf("[ApproveRefund] decrement current_funding error: %v", err)
	}

	return nil
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

func (s *investmentService) GetProjectInvestors(projectID uint) ([]dto.ProjectInvestorItem, error) {
	_, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return nil, errors.New("project not found")
	}
	return s.investmentRepo.ListInvestorsByProjectID(projectID)
}

func (s *investmentService) ListInvestedProjects(boosterUserID uint) ([]dto.InvestedProjectItem, error) {
	return s.investmentRepo.ListInvestedProjectsByUserID(boosterUserID)
}

func (s *investmentService) VoteMilestone(boosterUserID uint, milestoneID uint, choice domain.MilestoneVoteChoice) (*domain.MilestoneVote, error) {
	if boosterUserID == 0 {
		return nil, errors.New("unauthorized")
	}
	if choice != domain.MilestoneVoteApprove && choice != domain.MilestoneVoteReject {
		return nil, errors.New("invalid vote choice")
	}

	m, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil {
		return nil, errors.New("milestone not found")
	}

	// allow voting only when pioneer opened voting after admin approval
	if m.Status != domain.MilestoneApproved || !m.VotingOpen {
		return nil, errors.New("voting is not open")
	}

	ok, err := s.projectRepo.HasVerifiedInvestment(m.ProjectID, boosterUserID)
	if err != nil {
		return nil, errors.New("internal server error")
	}
	if !ok {
		return nil, errors.New("only verified investors can vote")
	}

	vote := &domain.MilestoneVote{
		MilestoneID:   milestoneID,
		ProjectID:     m.ProjectID,
		BoosterUserID: boosterUserID,
		Choice:        choice,
	}
	if err := s.projectRepo.UpsertMilestoneVote(vote); err != nil {
		return nil, errors.New("failed to save vote")
	}

	// auto-finalize: if approval reaches strict majority of eligible verified investors -> paid
	eligible, err := s.projectRepo.CountVerifiedBoostersByProjectID(m.ProjectID)
	if err == nil && eligible > 0 {
		approveCount, err2 := s.projectRepo.CountMilestoneVotes(milestoneID, domain.MilestoneVoteApprove)
		if err2 == nil && approveCount*2 > eligible {
			now := time.Now().UTC()
			m.Status = domain.MilestonePaid
			m.VotingOpen = false
			m.VotingClosedAt = &now
			_ = s.projectRepo.UpdateMilestone(m)
			s.createDisbursementForMilestone(m)
		}

		rejectCount, err3 := s.projectRepo.CountMilestoneVotes(milestoneID, domain.MilestoneVoteReject)
		if err3 == nil && rejectCount*2 > eligible {
			now := time.Now().UTC()
			m.Status = domain.MilestoneRejected
			m.VotingOpen = false
			m.VotingClosedAt = &now
			_ = s.projectRepo.UpdateMilestone(m)
		}
	}

	return vote, nil
}

// // private methods

// createDisbursementForMilestone creates a pending disbursement record when
// a milestone passes booster voting. Idempotent — safe to call if one already exists.
func (s *investmentService) createDisbursementForMilestone(m *domain.Milestone) {
	if existing, err := s.disbursementRepo.FindByMilestoneID(m.ID); err == nil && existing != nil {
		return
	}

	project, err := s.projectRepo.FindProjectByID(m.ProjectID)
	if err != nil {
		log.Printf("[createDisbursement] project lookup failed: %v", err)
		return
	}

	amount := project.CurrentFunding * float64(m.PercentRelease) / 100.0

	d := &domain.Disbursement{
		MilestoneID:    m.ID,
		ProjectID:      project.ID,
		PioneerUserID:  project.OwnerUserID,
		Amount:         amount,
		PhaseNo:        m.PhaseNo,
		PercentRelease: m.PercentRelease,
		Status:         domain.DisbursementPending,
	}

	if err := s.disbursementRepo.Create(d); err != nil {
		log.Printf("[createDisbursement] create error: %v", err)
	}
}

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

	stripe.Key = s.stripeSecretKey
	stripeFee, stripeFeeVAT, netAmount := s.fetchStripeFeesFromIntent(intentID)
	if err := s.transactionRepo.UpdateStripeFeesAndNet(txn.ID, stripeFee, stripeFeeVAT, netAmount); err != nil {
		log.Printf("[Webhook] update stripe fees error: %v", err)
	}

	investment, err := s.investmentRepo.FindByID(txn.InvestmentID)
	if err != nil {
		log.Printf("[Webhook] investment not found: %v", err)
		return
	}

	now := time.Now()
	investment.Status = domain.InvestmentVerified
	investment.PaidAt = &now
	// principal ที่แท้จริง = net จาก Stripe - platform_fee - vat
	if netAmount > 0 {
		investment.PrincipalAmount = math.Round((netAmount-investment.PlatformFee-investment.VATAmount)*100) / 100
	}

	if err := s.investmentRepo.UpdatePaid(investment); err != nil {
		log.Printf("[Webhook] update investment error: %v", err)
	}

	if err := s.investmentRepo.IncrementProjectFunding(investment.ProjectID, investment.TotalAmount); err != nil {
		log.Printf("[Webhook] update project current_funding error: %v", err)
	}

	// notify pioneer ที่เป็นเจ้าของโปรเจกต์
	if s.notifSvc != nil {
		project, err := s.projectRepo.FindProjectByID(investment.ProjectID)
		if err == nil {
			relatedID := investment.ProjectID
			relatedType := "project"
			body := fmt.Sprintf("มีการลงทุนใหม่ในโปรเจกต์ %s จำนวน %.2f บาท", project.Title, investment.TotalAmount)
			if err := s.notifSvc.CreateAndPush(project.OwnerUserID, domain.NotifNewInvestment, "มีการลงทุนใหม่", body, &relatedID, &relatedType); err != nil {
				log.Printf("[Webhook] send notification error: %v", err)
			}
		}
	}
}

func (s *investmentService) fetchStripeFeesFromIntent(intentID string) (stripeFee, stripeFeeVAT, netAmount float64) {
	pi, err := paymentintent.Get(intentID, &stripe.PaymentIntentParams{
		Params: stripe.Params{
			Expand: []*string{stripe.String("latest_charge.balance_transaction")},
		},
	})

	if err != nil {
		log.Printf("[Webhook] fetch payment intent error: %v", err)
		return
	}

	if pi.LatestCharge == nil {
		return
	}

	ch, err := charge.Get(pi.LatestCharge.ID, &stripe.ChargeParams{
		Params: stripe.Params{
			Expand: []*string{stripe.String("balance_transaction")},
		},
	})
	if err != nil || ch.BalanceTransaction == nil {
		log.Printf("[Webhook] fetch charge/balance_transaction error: %v", err)
		return
	}

	bt := ch.BalanceTransaction
	netAmount = float64(bt.Net) / 100
	for _, detail := range bt.FeeDetails {
		if detail.Type == "tax" {
			stripeFeeVAT += float64(detail.Amount) / 100
		} else {
			stripeFee += float64(detail.Amount) / 100
		}
	}
	return
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

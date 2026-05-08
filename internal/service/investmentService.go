package service

import (
	"bytes"
	"crypto/rand"
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"fmt"
	"html/template"
	"log"
	"math"
	"math/big"
	"strings"
	"time"

	"flyup/pkg/notification"

	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/charge"
	"github.com/stripe/stripe-go/v85/paymentintent"
	"github.com/stripe/stripe-go/v85/paymentmethod"
	"github.com/stripe/stripe-go/v85/refund"
	"github.com/stripe/stripe-go/v85/webhook"

	"gorm.io/gorm"
)

// MaxInvestmentPerTransaction คือเพดานต่อรายการ (THB)
// — ต่ำกว่า limit ของ Stripe API (~999,999.99 THB) เพื่อเผื่อ fees/rounding
// — เลขกลมตามมาตรฐาน fintech ไทย; ผู้ใช้ที่ลงทุนสูงกว่านี้ต้องแบ่งหลายรายการ
const MaxInvestmentPerTransaction = 500_000.0

type InvestmentService interface {
	GetInvestment(boosterUserID uint, investmentID uint) (*domain.Investment, *domain.Transaction, error)
	GenerateContractHTML(boosterUserID uint, investmentID uint) ([]byte, error)
	CreateInvestment(boosterUserID uint, boosterEmail string, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error)
	ListUserInvestments(boosterUserID uint) ([]domain.Investment, error)
	HandleStripeWebhook(payload []byte, sigHeader string) error
	RefundInvestment(boosterUserID uint, investmentID uint, note string) (*dto.RefundResponse, error)
	ApproveRefund(investmentID uint) error
	ListRefundRequests() ([]dto.RefundRequestItem, error)
	GetProjectInvestors(projectID uint) ([]dto.ProjectInvestorItem, error)
	ListInvestedProjects(boosterUserID uint) ([]dto.InvestedProjectItem, error)
	VoteMilestone(boosterUserID uint, milestoneID uint, choice domain.MilestoneVoteChoice) (*domain.MilestoneVote, error)
	GetMyVote(boosterUserID uint, milestoneID uint) (*domain.MilestoneVote, error)
	GetMilestoneVoters(pioneerUserID uint, milestoneID uint) ([]dto.MilestoneVoterItem, error)
	// RefundProjectInvestments คืนเงินนักลงทุนทุกคนเมื่อโปรเจกต์ถูก cancel
	RefundProjectInvestments(project domain.Project)
	GetCancelPreview(projectID uint) (*dto.CancelPreviewResponse, error)
	FinalizeVotingIfExpired(milestoneID uint) error
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
	emailClient      notification.NotificationClient
}

func NewInvestmentService(projectRepo repository.ProjectRepository, investmentRepo repository.InvestmentRepository, transactionRepo repository.TransactionRepository, userRepo repository.UserRepository, disbursementRepo repository.DisbursementRepository, stripeSecretKey string, webhookSecret string, notifSvc NotificationService, emailClient notification.NotificationClient) InvestmentService {
	return &investmentService{projectRepo, investmentRepo, transactionRepo, userRepo, disbursementRepo, stripeSecretKey, webhookSecret, notifSvc, emailClient}
}

func (s *investmentService) GetInvestment(boosterUserID uint, investmentID uint) (*domain.Investment, *domain.Transaction, error) {
	investment, err := s.investmentRepo.FindByIDWithProject(investmentID)

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

func (s *investmentService) GenerateContractHTML(boosterUserID uint, investmentID uint) ([]byte, error) {
	investment, err := s.investmentRepo.FindByIDWithProject(investmentID)
	if err != nil || investment.BoosterUserID != boosterUserID {
		return nil, errors.New("investment not found")
	}

	user, err := s.userRepo.FindUserById(boosterUserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	projectTitle := "—"
	if investment.Project != nil {
		projectTitle = investment.Project.Title
	}

	paidAt := "ยังไม่ชำระ"
	if investment.PaidAt != nil {
		paidAt = investment.PaidAt.Format("02 January 2006")
	}

	type contractData struct {
		RefNumber    string
		BoosterName  string
		BoosterEmail string
		ProjectTitle string
		Amount       float64
		PlatformFee  float64
		VAT          float64
		NetAmount    float64
		ProfitShare  float64
		PaidAt       string
		GeneratedAt  string
	}

	data := contractData{
		RefNumber:    investment.ReferenceNumber,
		BoosterName:  user.FirstName + " " + user.LastName,
		BoosterEmail: user.Email,
		ProjectTitle: projectTitle,
		Amount:       investment.TotalAmount,
		PlatformFee:  investment.PlatformFee,
		VAT:          investment.VATAmount,
		NetAmount:    investment.PrincipalAmount,
		ProfitShare:  investment.ProfitSharePct,
		PaidAt:       paidAt,
		GeneratedAt:  time.Now().Format("02 January 2006 15:04"),
	}

	const tmpl = `<!DOCTYPE html>
<html lang="th">
<head>
<meta charset="UTF-8" />
<title>สัญญาการลงทุน - {{.RefNumber}}</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: 'Sarabun', 'Helvetica Neue', Arial, sans-serif; font-size: 14px; color: #1a1a1a; background: #fff; padding: 40px; max-width: 800px; margin: auto; }
  .header { text-align: center; border-bottom: 3px solid #7c3aed; padding-bottom: 20px; margin-bottom: 32px; }
  .logo { font-size: 28px; font-weight: 900; color: #7c3aed; letter-spacing: -1px; margin-bottom: 4px; }
  .title { font-size: 20px; font-weight: 700; color: #111; margin-bottom: 4px; }
  .ref { font-size: 13px; color: #6b7280; }
  .section { margin-bottom: 28px; }
  .section-title { font-size: 13px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; color: #7c3aed; border-bottom: 1px solid #e5e7eb; padding-bottom: 6px; margin-bottom: 14px; }
  .row { display: flex; justify-content: space-between; padding: 7px 0; border-bottom: 1px dashed #f3f4f6; }
  .row:last-child { border-bottom: none; }
  .label { color: #6b7280; }
  .value { font-weight: 600; }
  .total-row { display: flex; justify-content: space-between; padding: 12px 16px; background: #f5f3ff; border-radius: 8px; margin-top: 10px; }
  .total-label { font-weight: 700; color: #7c3aed; }
  .total-value { font-weight: 800; font-size: 16px; color: #7c3aed; }
  .terms { font-size: 12px; color: #9ca3af; line-height: 1.7; background: #f9fafb; border-radius: 8px; padding: 16px; }
  .footer { text-align: center; margin-top: 40px; padding-top: 20px; border-top: 1px solid #e5e7eb; font-size: 12px; color: #9ca3af; }
  @media print { body { padding: 20px; } }
</style>
</head>
<body>
  <div class="header">
    <div class="logo">FlyUp</div>
    <div class="title">สัญญาการลงทุนโปรเจกต์นักศึกษา</div>
    <div class="ref">เลขอ้างอิง: {{.RefNumber}}</div>
  </div>

  <div class="section">
    <div class="section-title">ข้อมูลผู้ลงทุน (Booster)</div>
    <div class="row"><span class="label">ชื่อ-นามสกุล</span><span class="value">{{.BoosterName}}</span></div>
    <div class="row"><span class="label">อีเมล</span><span class="value">{{.BoosterEmail}}</span></div>
  </div>

  <div class="section">
    <div class="section-title">ข้อมูลโปรเจกต์</div>
    <div class="row"><span class="label">ชื่อโปรเจกต์</span><span class="value">{{.ProjectTitle}}</span></div>
    <div class="row"><span class="label">ส่วนแบ่งกำไร</span><span class="value">{{.ProfitShare}}%</span></div>
    <div class="row"><span class="label">วันที่ชำระเงิน</span><span class="value">{{.PaidAt}}</span></div>
  </div>

  <div class="section">
    <div class="section-title">รายละเอียดการชำระเงิน</div>
    <div class="row"><span class="label">ยอดลงทุน</span><span class="value">฿{{printf "%.2f" .Amount}}</span></div>
    <div class="row"><span class="label">ค่าธรรมเนียมแพลตฟอร์ม</span><span class="value">฿{{printf "%.2f" .PlatformFee}}</span></div>
    <div class="row"><span class="label">VAT (7%)</span><span class="value">฿{{printf "%.2f" .VAT}}</span></div>
    <div class="total-row"><span class="total-label">ยอดชำระสุทธิ</span><span class="total-value">฿{{printf "%.2f" .NetAmount}}</span></div>
  </div>

  <div class="section">
    <div class="section-title">เงื่อนไขและข้อตกลง</div>
    <div class="terms">
      1. ผู้ลงทุนรับทราบว่าการลงทุนนี้มีความเสี่ยง และไม่ได้รับประกันผลตอบแทน<br/>
      2. ส่วนแบ่งกำไรจะคำนวณตามผลประกอบการจริงของโปรเจกต์<br/>
      3. FlyUp ทำหน้าที่เป็นตัวกลางในการระดมทุนเท่านั้น ไม่ได้ค้ำประกันความสำเร็จของโปรเจกต์<br/>
      4. ในกรณีที่โปรเจกต์ถูกยกเลิก ผู้ลงทุนจะได้รับเงินคืนตามนโยบายของแพลตฟอร์ม<br/>
      5. เอกสารนี้ออกโดยระบบอัตโนมัติ ณ วันที่ {{.GeneratedAt}}
    </div>
  </div>

  <div class="footer">FlyUp Platform · fly-up.app · เอกสารนี้สร้างโดยระบบอัตโนมัติ</div>
</body>
</html>`

	t, err := template.New("contract").Parse(tmpl)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *investmentService) CreateInvestment(boosterUserID uint, boosterEmail string, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error) {
	user, err := s.userRepo.FindUserById(boosterUserID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if user.Role == "admin" {
		return nil, errors.New("admin cannot invest")
	}

	if user.IdCardVerification == nil || user.IdCardVerification.Status != domain.VerifyStatusApproved {
		return nil, errors.New("identity verification required before investing")
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
			Status:          string(inv.Status),
		}

		if inv.RefundedAt != nil {
			item.RequestedAt = inv.RefundedAt.Format(time.RFC3339)
		}

		if p, err := s.projectRepo.FindProjectByID(inv.ProjectID); err == nil {
			item.ProjectTitle = p.Title
		}

		user, err := s.userRepo.FindUserById(inv.BoosterUserID)
		if err == nil {
			item.BoosterName = user.FirstName + " " + user.LastName
			item.BoosterEmail = user.Email
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

// RefundProjectInvestments คืนเงินนักลงทุนทุกคนเมื่อโปรเจกต์ถูก cancel
func (s *investmentService) RefundProjectInvestments(project domain.Project) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[RefundProjectInvestments] panic: %v", r)
		}
	}()

	investments, err := s.investmentRepo.FindVerifiedByProjectID(project.ID)
	if err != nil {
		log.Printf("[RefundProjectInvestments] find investments error: %v", err)
		return
	}

	if len(investments) == 0 {
		return
	}

	// รวมยอดที่ต้องคืน
	var totalFunding float64
	for _, inv := range investments {
		totalFunding += inv.TotalAmount
	}

	remaining := project.CurrentFunding
	if totalFunding == 0 || remaining == 0 {
		return
	}

	var totalRefunded float64
	stripe.Key = s.stripeSecretKey

	for i, inv := range investments {
		// กัน refund ซ้ำ
		if inv.Status == domain.InvestmentRefunded {
			continue
		}

		var refundAmount float64
		// กัน rounding error ของรายการสุดท้าย
		if i == len(investments)-1 {
			refundAmount = remaining - totalRefunded
		} else {
			ratio := inv.TotalAmount / totalFunding
			refundAmount = math.Round(ratio*remaining*100) / 100
			totalRefunded += refundAmount
		}

		// กัน refund เกินยอดที่จ่ายจริง
		if refundAmount > inv.TotalAmount {
			refundAmount = inv.TotalAmount
		}
		if refundAmount <= 0 {
			continue
		}

		// หา transaction เพื่อเอา Stripe PaymentIntent ID
		txn, err := s.transactionRepo.FindByInvestmentID(inv.ID)
		if err != nil {
			log.Printf("[RefundProjectInvestments] txn not found inv %d: %v", inv.ID, err)
			continue
		}

		// ยิง Stripe Refund
		refundAmountSatang := int64(math.Round(refundAmount * 100))
		_, err = refund.New(&stripe.RefundParams{
			PaymentIntent: stripe.String(txn.StripePaymentIntentID),
			Amount:        stripe.Int64(refundAmountSatang),
		})
		if err != nil {
			log.Printf("[RefundProjectInvestments] stripe refund fail inv %d: %v", inv.ID, err)
			continue
		}

		// อัพเดต DB
		now := time.Now()
		inv.Status = domain.InvestmentRefunded
		inv.RefundAmount = refundAmount
		inv.RefundedAt = &now
		if err := s.investmentRepo.UpdateRefunded(&inv); err != nil {
			log.Printf("[RefundProjectInvestments] update refund fail inv %d: %v", inv.ID, err)
		}

		// notify นักลงทุน
		if s.notifSvc != nil {
			relatedID := project.ID
			relatedType := "project"
			body := fmt.Sprintf(
				"โปรเจกต์ \"%s\" ถูกยกเลิก คุณได้รับเงินคืน %.2f บาท",
				project.Title,
				refundAmount,
			)
			_ = s.notifSvc.CreateAndPush(
				inv.BoosterUserID,
				domain.NotifProjectStatus,
				"คืนเงินจากโปรเจกต์",
				body,
				&relatedID,
				&relatedType,
			)
		}

		log.Printf("[RefundProjectInvestments] refunded user %d amount %.2f", inv.BoosterUserID, refundAmount)
	}
}

func (s *investmentService) GetCancelPreview(projectID uint) (*dto.CancelPreviewResponse, error) {
	project, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return nil, err
	}

	milestones, err := s.projectRepo.FindMilestonesByProjectID(projectID)
	if err != nil {
		return nil, err
	}

	disbursements, err := s.disbursementRepo.ListByProjectID(projectID)
	if err != nil {
		return nil, err
	}

	// index disbursements by milestone_id for quick lookup
	disbByMilestone := make(map[uint]domain.Disbursement, len(disbursements))
	var totalDisbursed float64
	for _, d := range disbursements {
		disbByMilestone[d.MilestoneID] = d
		if d.Status == domain.DisbursementConfirmed {
			totalDisbursed += d.Amount
		}
	}

	previewMilestones := make([]dto.CancelPreviewMilestone, 0, len(milestones))
	for _, m := range milestones {
		d, hasDisbursement := disbByMilestone[m.ID]
		var disbursedAmount float64
		isConfirmed := false
		if hasDisbursement && d.Status == domain.DisbursementConfirmed {
			disbursedAmount = d.Amount
			isConfirmed = true
		}
		previewMilestones = append(previewMilestones, dto.CancelPreviewMilestone{
			PhaseNo:         m.PhaseNo,
			Title:           m.Title,
			PercentRelease:  m.PercentRelease,
			DisbursedAmount: disbursedAmount,
			IsConfirmed:     isConfirmed,
		})
	}

	// use same logic as RefundProjectInvestments
	investments, err := s.investmentRepo.FindVerifiedByProjectID(projectID)
	if err != nil {
		return nil, err
	}

	remaining := project.CurrentFunding
	var totalInvested float64
	for _, inv := range investments {
		totalInvested += inv.TotalAmount
	}

	previewInvestors := make([]dto.CancelPreviewInvestor, 0, len(investments))
	// aggregate by booster_user_id
	type aggEntry struct {
		dto.CancelPreviewInvestor
		totalAmount float64
	}
	aggMap := make(map[uint]*aggEntry)

	for _, inv := range investments {
		e, ok := aggMap[inv.BoosterUserID]
		if !ok {
			aggMap[inv.BoosterUserID] = &aggEntry{
				CancelPreviewInvestor: dto.CancelPreviewInvestor{
					UserID:      inv.BoosterUserID,
					TotalAmount: inv.TotalAmount,
				},
				totalAmount: inv.TotalAmount,
			}
		} else {
			e.TotalAmount += inv.TotalAmount
			e.totalAmount += inv.TotalAmount
		}
	}

	// fetch user info and calculate refund
	for userID, entry := range aggMap {
		u, _ := s.userRepo.FindUserById(userID)
		if u != nil {
			entry.FirstName = u.FirstName
			entry.LastName = u.LastName
			entry.Email = u.Email
		}
		var refundAmount float64
		if totalInvested > 0 {
			ratio := entry.totalAmount / totalInvested
			refundAmount = math.Round(ratio*remaining*100) / 100
		}
		entry.RefundAmount = refundAmount
		previewInvestors = append(previewInvestors, entry.CancelPreviewInvestor)
	}

	return &dto.CancelPreviewResponse{
		ProjectID:        projectID,
		Title:            project.Title,
		TotalFunding:     project.CurrentFunding,
		TotalDisbursed:   totalDisbursed,
		RefundableAmount: remaining,
		Milestones:       previewMilestones,
		Investors:        previewInvestors,
	}, nil
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

	existing, _ := s.projectRepo.FindVote(milestoneID, boosterUserID)
	if existing != nil {
		return nil, errors.New("you have already voted")
	}

	vote := &domain.MilestoneVote{
		MilestoneID:   milestoneID,
		ProjectID:     m.ProjectID,
		BoosterUserID: boosterUserID,
		Choice:        choice,
	}
	if err := s.projectRepo.UpsertMilestoneVote(vote); err != nil {
		if helper.IsUniqueConstraintError(err) {
			return nil, errors.New("you have already voted")
		}
		return nil, errors.New("failed to save vote")
	}

	m, _ = s.projectRepo.FindMilestoneByID(milestoneID)
	if !m.VotingOpen {
		return vote, nil // voting already closed
	}

	// auto-finalize: if approval reaches strict majority of eligible verified investment -> paid
	eligibleAmount, err := s.projectRepo.SumVerifiedInvestmentByProjectID(m.ProjectID)
	if err == nil && eligibleAmount > 0 {
		approveAmount, err2 := s.projectRepo.SumMilestoneVotes(milestoneID, domain.MilestoneVoteApprove)
		if err2 == nil && approveAmount*2 > eligibleAmount {
			now := time.Now().UTC()
			m.Status = domain.MilestonePaid
			m.VotingOpen = false
			m.VotingClosedAt = &now
			
			// Calculate delay to shift upcoming milestones
			var delay time.Duration
			if m.OriginalDueDate != nil && now.After(*m.OriginalDueDate) {
				delay = now.Sub(*m.OriginalDueDate)
			}

			_ = s.projectRepo.UpdateMilestone(m)
			_ = s.projectRepo.CloseMeetingsByMilestoneID(m.ID)
			s.createDisbursementForMilestone(m)

			// activate next phase and shift deadlines
			if allMs, err := s.projectRepo.FindMilestonesByProjectID(m.ProjectID); err == nil {
				for i := range allMs {
					needsUpdate := false
					if allMs[i].PhaseNo > m.PhaseNo {
						if delay > 0 && allMs[i].DueDate != nil {
							newDue := allMs[i].DueDate.Add(delay)
							allMs[i].DueDate = &newDue
							needsUpdate = true
						}
					}
					if allMs[i].PhaseNo == m.PhaseNo+1 && (allMs[i].Status == domain.MilestoneWaiting || allMs[i].Status == domain.MilestoneDraft) {
						allMs[i].Status = domain.MilestoneActive
						needsUpdate = true
					}
					if needsUpdate {
						_ = s.projectRepo.UpdateMilestone(&allMs[i])
					}
				}
			}

			if p, pErr := s.projectRepo.FindProjectByID(m.ProjectID); pErr == nil {
				if s.notifSvc != nil {
					relatedID := m.ID
					relatedType := "milestone"
					body := fmt.Sprintf("Milestone Phase %d: %s ผ่านการโหวตแล้ว กำลังดำเนินการปล่อยทุน", m.PhaseNo, m.Title)
					_ = s.notifSvc.CreateAndPush(p.OwnerUserID, domain.NotifMilestone, "Milestone ผ่านการโหวต", body, &relatedID, &relatedType)
					// notify all admins to process disbursement
					adminBody := fmt.Sprintf("โปรเจกต์ %s – Phase %d: %s ผ่านการโหวตแล้ว กรุณาโอนเงินให้ Pioneer", p.Title, m.PhaseNo, m.Title)
					if adminIDs, err := s.userRepo.FindAdminUserIDs(); err == nil {
						for _, aid := range adminIDs {
							_ = s.notifSvc.CreateAndPush(aid, domain.NotifMilestone, "Phase ผ่านการโหวต – รอโอนเงิน", adminBody, &relatedID, &relatedType)
						}
					}
				}
				if s.emailClient != nil {
					projectTitle := p.Title
					phaseNo := m.PhaseNo
					phaseTitle := m.Title
					projectID := m.ProjectID
					go func() {
						defer func() {
							if r := recover(); r != nil {
								log.Printf("milestone vote email panic: %v", r)
							}
						}()
						investors, _ := s.investmentRepo.ListInvestorsByProjectID(projectID)
						for _, inv := range investors {
							if err := s.emailClient.SendMilestoneVoteResultEmail(inv.Email, projectTitle, phaseNo, phaseTitle, true); err != nil {
								log.Printf("send milestone vote result email error: %v", err)
							}
						}
					}()
				}
			}
		} else {
			rejectAmount, err3 := s.projectRepo.SumMilestoneVotes(milestoneID, domain.MilestoneVoteReject)
			if err3 == nil && rejectAmount*2 > eligibleAmount {
				now := time.Now().UTC()
				m.VotingOpen = false
				m.VotingClosedAt = &now
				_ = s.projectRepo.CloseMeetingsByMilestoneID(m.ID)
				
				m.RetryCount += 1
				if m.RetryCount > 1 {
					m.Status = domain.MilestoneFailed
					_ = s.projectRepo.UpdateMilestone(m)
					
					// Suspend project because retry failed
					if p, pErr := s.projectRepo.FindProjectByID(m.ProjectID); pErr == nil {
						p.State = domain.StateSuspended
						p.Status = domain.StatusFailed
						_, _ = s.projectRepo.UpdateProject(p)
						
						// Notify project suspension
						if s.notifSvc != nil {
							relatedID := p.ID
							relatedType := "project"
							body := fmt.Sprintf("โปรเจกต์ %s ถูกระงับเนื่องจาก Milestone Phase %d ไม่ผ่านการโหวตในรอบแก้ไข", p.Title, m.PhaseNo)
							_ = s.notifSvc.CreateAndPush(p.OwnerUserID, domain.NotifProjectStatus, "โปรเจกต์ถูกระงับ", body, &relatedID, &relatedType)
						}
					}
				} else {
					m.Status = domain.MilestoneRejected
					retryDeadline := now.Add(7 * 24 * time.Hour)
					m.RetryDeadline = &retryDeadline
					_ = s.projectRepo.UpdateMilestone(m)
				}

				if p, pErr := s.projectRepo.FindProjectByID(m.ProjectID); pErr == nil {
					if s.notifSvc != nil && m.Status == domain.MilestoneRejected {
						relatedID := m.ID
						relatedType := "milestone"
						body := fmt.Sprintf("Milestone Phase %d: %s ไม่ผ่านการโหวต คุณมีเวลาแก้ไข 7 วัน (ถึง %s)", m.PhaseNo, m.Title, m.RetryDeadline.Format("02 Jan 2006"))
						_ = s.notifSvc.CreateAndPush(p.OwnerUserID, domain.NotifMilestone, "Milestone ไม่ผ่านการโหวต", body, &relatedID, &relatedType)
					}
					if s.emailClient != nil {
						projectTitle := p.Title
						phaseNo := m.PhaseNo
						phaseTitle := m.Title
						projectID := m.ProjectID
						go func() {
							defer func() {
								if r := recover(); r != nil {
									log.Printf("milestone vote email panic: %v", r)
								}
							}()
							investors, _ := s.investmentRepo.ListInvestorsByProjectID(projectID)
							for _, inv := range investors {
								if err := s.emailClient.SendMilestoneVoteResultEmail(inv.Email, projectTitle, phaseNo, phaseTitle, false); err != nil {
									log.Printf("send milestone vote result email error: %v", err)
								}
							}
						}()
					}
				}
			}
		}
	}

	return vote, nil
}

func (s *investmentService) FinalizeVotingIfExpired(milestoneID uint) error {
	m, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil {
		return err
	}

	if m.Status != domain.MilestoneApproved || !m.VotingOpen {
		return nil
	}

	now := time.Now().UTC()
	// Use milestone DueDate as voting deadline; fall back to 7 days from open if unset
	var deadline time.Time
	if m.DueDate != nil {
		deadline = m.DueDate.UTC()
	} else if m.VotingOpenedAt != nil {
		deadline = m.VotingOpenedAt.Add(7 * 24 * time.Hour)
	} else {
		return nil
	}
	if !now.After(deadline) {
		return nil // Voting window still open
	}

	// Expired, tally votes
	eligibleAmount, err := s.projectRepo.SumVerifiedInvestmentByProjectID(m.ProjectID)
	if err != nil || eligibleAmount <= 0 {
		return nil // No investors? Should not happen if it passed
	}

	approveAmount, _ := s.projectRepo.SumMilestoneVotes(milestoneID, domain.MilestoneVoteApprove)
	rejectAmount, _ := s.projectRepo.SumMilestoneVotes(milestoneID, domain.MilestoneVoteReject)

	// In case of a tie or no votes, default to reject (strict rule) or default to approve?
	// Given strict rules: if approve > reject, it passes.
	if approveAmount > rejectAmount {
		m.Status = domain.MilestonePaid
		m.VotingOpen = false
		m.VotingClosedAt = &now

		var delay time.Duration
		if m.OriginalDueDate != nil && now.After(*m.OriginalDueDate) {
			delay = now.Sub(*m.OriginalDueDate)
		}

		_ = s.projectRepo.UpdateMilestone(m)
		_ = s.projectRepo.CloseMeetingsByMilestoneID(m.ID)
		s.createDisbursementForMilestone(m)

		if allMs, err := s.projectRepo.FindMilestonesByProjectID(m.ProjectID); err == nil {
			for i := range allMs {
				needsUpdate := false
				if allMs[i].PhaseNo > m.PhaseNo {
					if delay > 0 && allMs[i].DueDate != nil {
						newDue := allMs[i].DueDate.Add(delay)
						allMs[i].DueDate = &newDue
						needsUpdate = true
					}
				}
				if allMs[i].PhaseNo == m.PhaseNo+1 && (allMs[i].Status == domain.MilestoneWaiting || allMs[i].Status == domain.MilestoneDraft) {
					allMs[i].Status = domain.MilestoneActive
					needsUpdate = true
				}
				if needsUpdate {
					_ = s.projectRepo.UpdateMilestone(&allMs[i])
				}
			}
		}

		if p, pErr := s.projectRepo.FindProjectByID(m.ProjectID); pErr == nil {
			if s.notifSvc != nil {
				relatedID := m.ID
				relatedType := "milestone"
				body := fmt.Sprintf("Milestone Phase %d: %s ปิดโหวตและผ่านแล้ว", m.PhaseNo, m.Title)
				_ = s.notifSvc.CreateAndPush(p.OwnerUserID, domain.NotifMilestone, "Milestone ผ่านการโหวตอัตโนมัติ", body, &relatedID, &relatedType)
				// notify admins
				adminBody := fmt.Sprintf("โปรเจกต์ %s – Phase %d: %s ผ่านการโหวต (หมดเวลา) กรุณาโอนเงินให้ Pioneer", p.Title, m.PhaseNo, m.Title)
				if adminIDs, err := s.userRepo.FindAdminUserIDs(); err == nil {
					for _, aid := range adminIDs {
						_ = s.notifSvc.CreateAndPush(aid, domain.NotifMilestone, "Phase ผ่านการโหวต – รอโอนเงิน", adminBody, &relatedID, &relatedType)
					}
				}
			}
		}
	} else {
		m.VotingOpen = false
		m.VotingClosedAt = &now
		_ = s.projectRepo.CloseMeetingsByMilestoneID(m.ID)

		m.RetryCount += 1
		if m.RetryCount > 1 {
			m.Status = domain.MilestoneFailed
			_ = s.projectRepo.UpdateMilestone(m)

			if p, pErr := s.projectRepo.FindProjectByID(m.ProjectID); pErr == nil {
				p.State = domain.StateSuspended
				p.Status = domain.StatusFailed
				_, _ = s.projectRepo.UpdateProject(p)
			}
		} else {
			m.Status = domain.MilestoneRejected
			retryDeadline := now.Add(7 * 24 * time.Hour)
			m.RetryDeadline = &retryDeadline
			_ = s.projectRepo.UpdateMilestone(m)
		}

		if p, pErr := s.projectRepo.FindProjectByID(m.ProjectID); pErr == nil {
			if s.notifSvc != nil && m.Status == domain.MilestoneRejected {
				relatedID := m.ID
				relatedType := "milestone"
				body := fmt.Sprintf("Milestone Phase %d: %s หมดเวลาโหวตและได้คะแนนไม่ผ่าน คุณมีเวลาแก้ไข 7 วัน (ถึง %s)", m.PhaseNo, m.Title, m.RetryDeadline.Format("02 Jan 2006"))
				_ = s.notifSvc.CreateAndPush(p.OwnerUserID, domain.NotifMilestone, "Milestone ไม่ผ่านการโหวต", body, &relatedID, &relatedType)
			}
		}
	}
	return nil
}

func (s *investmentService) GetMyVote(boosterUserID uint, milestoneID uint) (*domain.MilestoneVote, error) {
	if boosterUserID == 0 {
		return nil, errors.New("unauthorized")
	}
	return s.projectRepo.FindVote(milestoneID, boosterUserID)
}

func (s *investmentService) GetMilestoneVoters(pioneerUserID uint, milestoneID uint) ([]dto.MilestoneVoterItem, error) {
	milestone, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil || milestone == nil {
		return nil, errors.New("milestone not found")
	}

	project, err := s.projectRepo.FindProjectByID(milestone.ProjectID)
	if err != nil || project == nil {
		return nil, errors.New("project not found")
	}

	if project.OwnerUserID != pioneerUserID {
		return nil, errors.New("forbidden")
	}

	investors, err := s.investmentRepo.ListInvestorsByProjectID(project.ID)
	if err != nil {
		return nil, err
	}

	votes, err := s.projectRepo.ListVotesByMilestoneID(milestoneID)
	if err != nil {
		return nil, err
	}

	voteMap := make(map[uint]domain.MilestoneVoteChoice, len(votes))
	for _, v := range votes {
		voteMap[v.BoosterUserID] = v.Choice
	}

	result := make([]dto.MilestoneVoterItem, 0, len(investors))
	for _, inv := range investors {
		item := dto.MilestoneVoterItem{
			UserID:    inv.UserID,
			FirstName: inv.FirstName,
			LastName:  inv.LastName,
			Picture:   inv.Picture,
		}
		if choice, voted := voteMap[inv.UserID]; voted {
			item.Voted = true
			item.Choice = string(choice)
		}
		result = append(result, item)
	}
	return result, nil
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

	// ตรวจสอบว่าถึงเป้าหมายหรือยัง → เปลี่ยน state เป็น executing อัตโนมัติ
	project, projErr := s.projectRepo.FindProjectByID(investment.ProjectID)
	if projErr == nil && project.State == domain.StateFunding && project.CurrentFunding >= project.FundingGoal {
		project.State = domain.StateExecuting
		project.Status = domain.StatusActive
		if _, err := s.projectRepo.UpdateProject(project); err != nil {
			log.Printf("[Webhook] auto-transition to executing error: %v", err)
		} else {
			log.Printf("[Webhook] project %d reached funding goal → state=executing", project.ID)
		}
	}

	// notify pioneer ที่เป็นเจ้าของโปรเจกต์
	if s.notifSvc != nil && projErr == nil {
		relatedID := investment.ProjectID
		relatedType := "project"
		body := fmt.Sprintf("มีการลงทุนใหม่ในโปรเจกต์ %s จำนวน %.2f บาท", project.Title, investment.TotalAmount)
		if err := s.notifSvc.CreateAndPush(project.OwnerUserID, domain.NotifNewInvestment, "มีการลงทุนใหม่", body, &relatedID, &relatedType); err != nil {
			log.Printf("[Webhook] send notification error: %v", err)
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

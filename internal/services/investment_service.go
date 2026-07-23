package services

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
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

	"github.com/skip2/go-qrcode"
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

// MinInvestmentPerTransaction คือขั้นต่ำต่อรายการ (THB) ตามขั้นต่ำของช่องทางชำระเงิน Stripe (PromptPay)
const MinInvestmentPerTransaction = 20.0

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
	GetTotalFunding() (float64, error)
	GetUniqueBoostersCount() (int64, error)
	// SyncProjectPrincipalAmounts ดึงค่าธรรมเนียม Stripe จริงมาคำนวณ PrincipalAmount ใหม่
	// ให้กับ investment ที่ webhook ยังดึงค่าธรรมเนียมไม่สำเร็จ (NetAmount = 0)
	SyncProjectPrincipalAmounts(projectID uint) error
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

// GetInvestment ดึงข้อมูลการลงทุนพร้อมธุรกรรมที่เกี่ยวข้อง โดยตรวจสอบว่าเป็นของ boosterUserID นี้เท่านั้น
func (s *investmentService) GetInvestment(boosterUserID uint, investmentID uint) (*domain.Investment, *domain.Transaction, error) {
	investment, err := s.investmentRepo.FindByIDWithProject(investmentID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("investment not found")
		}
		return nil, nil, errors.New("internal server error")
	}

	// เช็คเจ้าของ ไม่งั้นใครก็เดา id แล้วดูของคนอื่นได้
	if investment.BoosterUserID != boosterUserID {
		return nil, nil, errors.New("investment not found")
	}

	// บาง investment อาจยังไม่มี transaction (เช่นเพิ่งสร้าง ยังไม่ทันมี record) ถือว่าไม่ error
	txn, err := s.transactionRepo.FindByInvestmentID(investmentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return investment, nil, nil
		}
		return nil, nil, errors.New("internal server error")
	}

	return investment, txn, nil
}

// GenerateContractHTML สร้างเอกสารสัญญาการลงทุนเป็น HTML สำหรับดาวน์โหลด/พิมพ์
func (s *investmentService) GenerateContractHTML(boosterUserID uint, investmentID uint) ([]byte, error) {
	investment, err := s.investmentRepo.FindByIDWithProject(investmentID)
	if err != nil || investment.BoosterUserID != boosterUserID {
		return nil, errors.New("investment not found")
	}

	user, err := s.userRepo.FindUserById(boosterUserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// เผื่อโปรเจกต์โดนลบไปแล้วแต่ investment ยังอยู่ ไม่ให้หน้าเอกสารพัง
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
		GeneratedAt: func() string {
			thaiMonths := [13]string{"", "มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน", "กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม"}
			// เวลาไทยล้วน ๆ ปีเป็น พ.ศ. เผื่อ server timezone ไม่ตรงเลยลง fallback ไว้กันพัง
			loc, err := time.LoadLocation("Asia/Bangkok")
			if err != nil {
				loc = time.FixedZone("ICT", 7*60*60)
			}
			now := time.Now().In(loc)
			return fmt.Sprintf("%d %s %d เวลา %02d:%02d น.", now.Day(), thaiMonths[now.Month()], now.Year()+543, now.Hour(), now.Minute())
		}(),
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

	// ยัดข้อมูลลง template แล้วได้ไฟล์ HTML กลับมาเป็น []byte ให้ handler ส่งดาวน์โหลดต่อ
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

// CreateInvestment สร้างรายการลงทุนใหม่ ตรวจสอบสิทธิ์ผู้ใช้ เงื่อนไขโปรเจกต์ และยอดเงิน
// แล้วสร้าง QR Code สำหรับชำระเงินผ่าน Stripe PromptPay
func (s *investmentService) CreateInvestment(boosterUserID uint, boosterEmail string, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error) {
	user, err := s.userRepo.FindUserById(boosterUserID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	// admin กับ pioneer ไม่ใช่ role ที่มาลงทุน กันไว้ตั้งแต่ต้นทาง
	if user.Role == "admin" {
		return nil, errors.New("admin cannot invest")
	}

	if user.Role == "pioneer" {
		return nil, errors.New("pioneer cannot invest")
	}

	// ต้องยืนยันตัวตนผ่านก่อนถึงจะลงทุนได้ (KYC)
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

	// เจ้าของโปรเจกต์ลงทุนในโปรเจกต์ตัวเองไม่ได้
	if project.OwnerUserID == user.ID {
		return nil, errors.New("cannot invest in your own project")
	}

	if project.State != domain.StateFunding {
		return nil, errors.New("project is not open for investment")
	}

	// reload โปรเจกต์ก่อนคำนวณยอดคงเหลือ เพื่อให้ current_funding เป็นค่าล่าสุดจาก DB
	// (project ที่โหลดไว้ด้านบนอาจค้างจากการลงทุนอื่นที่ verified แทรกเข้ามาระหว่างนี้)
	project, err = s.projectRepo.FindProjectByID(req.ProjectID)
	if err != nil {
		return nil, errors.New("internal server error")
	}

	if err := validateAmount(project, req.Amount); err != nil {
		return nil, err
	}

	if req.Amount > MaxInvestmentPerTransaction {
		return nil, fmt.Errorf("ยอดลงทุนต่อรายการต้องไม่เกิน ฿%.0f (กรุณาแบ่งเป็นหลายรายการหากต้องการลงทุนสูงกว่านี้)", MaxInvestmentPerTransaction)
	}

	// คำนวณค่าธรรมเนียม/VAT/ยอดสุทธิไว้ก่อน (ยังไม่รู้ค่าธรรมเนียม Stripe จริง จะมาปรับอีกทีตอน webhook)
	fee, vat, principal := calculateFees(req.Amount, project.PlatformFee)

	refNum, err := generateReferenceNumber()
	if err != nil {
		return nil, errors.New("failed to generate reference number")
	}

	// สร้าง record ไว้ก่อนเป็นสถานะ pending รอผู้ใช้สแกน QR จ่ายเงิน
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
	qrURL, qrData, intentID, clientSecret, expiresAt, err := s.createStripePromptPay(req.Amount, refNum, project.Title, boosterEmail)
	if err != nil {
		// Stripe ล้มเหลว → mark investment เป็น rejected
		_ = s.investmentRepo.UpdateStatus(investment.ID, domain.InvestmentRejected)
		log.Printf("[CreateInvestment] stripe error: %v", err)
		return nil, errors.New("failed to create payment QR code")
	}

	qrBase64, err := generateQRBase64(qrData)
	if err != nil {
		// แปลงรูปไม่ผ่านก็ไม่ถึงกับ fail ทั้ง flow เพราะยังมี qrURL จาก Stripe ให้ใช้อยู่
		log.Printf("[CreateInvestment] generate qr base64 error: %v", err)
	}

	txn := &domain.Transaction{
		InvestmentID:          investment.ID,
		StripePaymentIntentID: intentID,
		StripeClientSecret:    clientSecret,
		QRCodeImageURL:        qrURL,
		QRCodeBase64:          qrBase64,
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
		QRCodeBase64:    qrBase64,
		ExpiresAt:       expiresAt.Format(time.RFC3339),
		TotalAmount:     req.Amount,
		Title:           project.Title,
	}, nil
}

// ListUserInvestments ดึงรายการลงทุนทั้งหมดของผู้ใช้
func (s *investmentService) ListUserInvestments(boosterUserID uint) ([]domain.Investment, error) {
	return s.investmentRepo.ListByBoosterUserID(boosterUserID)
}

// RefundInvestment ให้ผู้ลงทุนขอคืนเงินสำหรับการลงทุนที่ verified แล้ว
// (เปลี่ยนสถานะเป็น refund pending รอแอดมินอนุมัติ)
func (s *investmentService) RefundInvestment(boosterUserID uint, investmentID uint, note string) (*dto.RefundResponse, error) {
	investment, err := s.investmentRepo.FindByID(investmentID)
	if err != nil {
		return nil, errors.New("investment not found")
	}

	// ต้องเป็นเจ้าของรายการลงทุนนี้เท่านั้น
	if investment.BoosterUserID != boosterUserID {
		return nil, errors.New("investment not found")
	}

	// ขอคืนได้เฉพาะรายการที่จ่ายเงินสำเร็จแล้วเท่านั้น (pending/rejected ไม่มีเงินให้คืน)
	if investment.Status != domain.InvestmentVerified {
		return nil, errors.New("only verified investments can be refunded")
	}

	project, err := s.projectRepo.FindProjectByID(investment.ProjectID)
	if err != nil {
		return nil, errors.New("project not found")
	}

	// พ้นช่วงระดมทุนไปแล้ว (เช่นเริ่ม executing) จะคืนเงินไม่ได้ เพราะเงินอาจถูกเบิกไปให้ pioneer แล้ว
	if project.State != domain.StateFunding {
		return nil, errors.New("refund is only allowed while project project is in funding state")
	}

	// ถ้าตอน webhook ยังดึงค่าธรรมเนียม Stripe ไม่สำเร็จ (balance_transaction ยังไม่พร้อม)
	// ให้ลองดึงใหม่ตอนขอ refund เพื่อให้ PrincipalAmount หักค่าธรรมเนียม Stripe จริงเสมอ
	txn, err := s.transactionRepo.FindByInvestmentID(investment.ID)
	if err != nil {
		return nil, errors.New("transaction not found")
	}

	s.refreshPrincipalFromStripe(investment, txn)

	// คืนแค่ PrincipalAmount เท่านั้น ค่าธรรมเนียม platform/VAT ไม่คืนให้
	refundAmount := investment.PrincipalAmount

	// ยังไม่ยิง Stripe refund ตรงนี้ แค่ mark เป็น pending รอแอดมินกดอนุมัติอีกที
	now := time.Now()
	investment.Status = domain.InvestmentRefundPending
	investment.RefundAmount = refundAmount
	investment.RefundNote = note
	investment.RefundedAt = &now

	if err := s.investmentRepo.UpdateRefunded(investment); err != nil {
		log.Printf("[RefundInvestment] db update error: %v", err)
		return nil, errors.New("refund processed but failed to update record")
	}

	// หักยอดที่นับรวมไว้ในโปรเจกต์ออก เพราะเงินก้อนนี้กำลังจะถูกคืนไม่ใช่ของโปรเจกต์แล้ว
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

// refreshPrincipalFromStripe ดึงค่าธรรมเนียม Stripe จริงมาคำนวณ PrincipalAmount ใหม่
// เฉพาะกรณีที่ webhook ตอนจ่ายเงินยังไม่สามารถดึง balance_transaction ได้ (NetAmount = 0)
// คืนค่า true ถ้ามีการอัปเดต PrincipalAmount
func (s *investmentService) refreshPrincipalFromStripe(investment *domain.Investment, txn *domain.Transaction) bool {
	if txn.NetAmount > 0 {
		return false
	}

	stripe.Key = s.stripeSecretKey
	stripeFee, stripeFeeVAT, netAmount := s.fetchStripeFeesFromIntent(txn.StripePaymentIntentID)
	if netAmount <= 0 {
		return false
	}

	if err := s.transactionRepo.UpdateStripeFeesAndNet(txn.ID, stripeFee, stripeFeeVAT, netAmount); err != nil {
		log.Printf("[refreshPrincipalFromStripe] update stripe fees error: %v", err)
	}
	investment.PrincipalAmount = math.Round((netAmount-investment.PlatformFee-investment.VATAmount)*100) / 100
	return true
}

// SyncProjectPrincipalAmounts ไล่ดู investment ที่ verified ทั้งหมดของโปรเจกต์
// แล้วอัปเดต PrincipalAmount ให้หักค่าธรรมเนียม Stripe จริง สำหรับรายการที่ webhook
// ดึงค่าธรรมเนียมไม่สำเร็จตอนจ่ายเงิน — ใช้ก่อนคำนวณสัดส่วนปันผล เพื่อให้นักลงทุนทุกคน
// ใช้ฐาน PrincipalAmount ที่คำนวณด้วยสูตรเดียวกัน
func (s *investmentService) SyncProjectPrincipalAmounts(projectID uint) error {
	investments, err := s.investmentRepo.FindVerifiedByProjectID(projectID)
	if err != nil {
		return err
	}

	for i := range investments {
		inv := &investments[i]
		txn, err := s.transactionRepo.FindByInvestmentID(inv.ID)
		if err != nil {
			// ไม่มี transaction ก็ข้ามไปตัวถัดไป ไม่ทำให้ทั้ง batch fail
			log.Printf("[SyncProjectPrincipalAmounts] transaction not found for investment %d: %v", inv.ID, err)
			continue
		}

		// ถ้ามีการอัปเดต PrincipalAmount จริง (แปลว่าก่อนหน้านี้ยังไม่เคยดึงค่าธรรมเนียมสำเร็จ) ค่อย save
		if s.refreshPrincipalFromStripe(inv, txn) {
			if err := s.investmentRepo.UpdatePaid(inv); err != nil {
				log.Printf("[SyncProjectPrincipalAmounts] update investment %d error: %v", inv.ID, err)
			}
		}
	}

	return nil
}

// ListRefundRequests ดึงรายการคำขอคืนเงินที่รอดำเนินการทั้งหมด พร้อมข้อมูลผู้ลงทุนและบัญชีธนาคารสำหรับโอนคืน
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

		// พวก lookup พวกนี้ error ก็ปล่อยผ่าน แค่ field เป็นค่าว่าง ไม่ต้อง fail ทั้ง list
		if p, err := s.projectRepo.FindProjectByID(inv.ProjectID); err == nil {
			item.ProjectTitle = p.Title
		}

		user, err := s.userRepo.FindUserById(inv.BoosterUserID)
		if err == nil {
			item.BoosterName = user.FirstName + " " + user.LastName
			item.BoosterEmail = user.Email

			// หาบัญชีที่ตั้งเป็น default ไว้ก่อน ถ้าไม่มีเลยก็เอาบัญชีแรกที่เจอ
			var defaultBank *domain.BankAccount
			for _, b := range user.BankAccounts {
				if b.IsDefault {
					defaultBank = &b
					break
				}
			}
			if defaultBank == nil && len(user.BankAccounts) > 0 {
				defaultBank = &user.BankAccounts[0]
			}

			if defaultBank != nil {
				item.BankAccount = &dto.RefundBankAccount{
					BankName:      defaultBank.BankName,
					AccountName:   defaultBank.AccountName,
					AccountNumber: defaultBank.AccountNumber,
				}
			}
		}
		result = append(result, item)
	}
	return result, nil
}

// ApproveRefund อนุมัติคำขอคืนเงิน ยิง Stripe Refund จริง แล้วอัปเดตสถานะและแจ้งเตือนผู้ลงทุน
func (s *investmentService) ApproveRefund(investmentID uint) error {
	investment, err := s.investmentRepo.FindByID(investmentID)
	if err != nil {
		return errors.New("investment not found")
	}

	// กันอนุมัติซ้ำ หรืออนุมัติรายการที่ยังไม่ได้ขอ refund
	if investment.Status != domain.InvestmentRefundPending {
		return errors.New("investment is not pending refund")
	}

	txn, err := s.transactionRepo.FindByInvestmentID(investment.ID)
	if err != nil {
		return errors.New("transaction not found")
	}

	// ยิง Stripe Refund จริง — ก่อนหน้านี้ ApproveRefund แค่เปลี่ยน status ในฐานข้อมูล
	// โดยไม่ได้คืนเงินผ่าน Stripe เลย ทำให้ booster ไม่ได้รับเงินคืนจริง
	stripe.Key = s.stripeSecretKey
	refundAmountSatang := int64(math.Round(investment.RefundAmount * 100))
	if _, err := refund.New(&stripe.RefundParams{
		PaymentIntent: stripe.String(txn.StripePaymentIntentID),
		Amount:        stripe.Int64(refundAmountSatang),
	}); err != nil {
		log.Printf("[ApproveRefund] stripe refund failed for investment %d: %v", investment.ID, err)
		return fmt.Errorf("stripe refund failed: %v", err)
	}

	now := time.Now()
	investment.Status = domain.InvestmentRefunded
	investment.RefundedAt = &now

	if err := s.investmentRepo.UpdateRefunded(investment); err != nil {
		return errors.New("failed to approve refund")
	}

	// notify booster ว่าคำขอคืนเงินได้รับการอนุมัติแล้ว
	if s.notifSvc != nil {
		relatedID := investment.ProjectID
		relatedType := "project"
		body := fmt.Sprintf("คำขอคืนเงินจำนวน %.2f บาท ของคุณได้รับการอนุมัติแล้ว เงินจะถูกโอนเข้าช่องทางการชำระเงินเดิมภายในไม่กี่วัน หากชำระผ่าน PromptPay กรุณาตรวจสอบอีเมลจาก Stripe เพื่อกรอกข้อมูลบัญชีธนาคารสำหรับรับเงินคืน", investment.RefundAmount)
		if err := s.notifSvc.CreateAndPush(investment.BoosterUserID, domain.NotifProjectStatus, "คำขอคืนเงินได้รับการอนุมัติ", body, &relatedID, &relatedType); err != nil {
			log.Printf("[ApproveRefund] send notification error: %v", err)
		}
	}

	// ส่งอีเมลแจ้ง booster ว่าคำขอคืนเงินได้รับการอนุมัติแล้ว
	if s.emailClient != nil {
		if user, err := s.userRepo.FindUserById(investment.BoosterUserID); err == nil {
			projectTitle := ""
			if p, err := s.projectRepo.FindProjectByID(investment.ProjectID); err == nil {
				projectTitle = p.Title
			}
			if err := s.emailClient.SendRefundApprovedEmail(user.Email, projectTitle, investment.RefundAmount); err != nil {
				log.Printf("[ApproveRefund] send email error: %v", err)
			}
		}
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

	if err := s.SyncProjectPrincipalAmounts(project.ID); err != nil {
		log.Printf("[RefundProjectInvestments] sync principal amounts error: %v", err)
	}

	investments, err := s.investmentRepo.FindVerifiedByProjectID(project.ID)
	if err != nil {
		log.Printf("[RefundProjectInvestments] find investments error: %v", err)
		return
	}

	if len(investments) == 0 {
		return
	}

	// รวมยอด PrincipalAmount (สุทธิหลังหักค่าธรรมเนียม Stripe + ค่าธรรมเนียมแพลตฟอร์ม + VAT)
	// เพื่อใช้เป็นฐานคำนวณสัดส่วนคืนเงิน ให้สอดคล้องกับฐานที่ใช้คำนวณ disbursement ให้ pioneer
	var totalPrincipal float64
	for _, inv := range investments {
		totalPrincipal += inv.PrincipalAmount
	}

	// เงินที่เบิกจ่ายให้ pioneer ไปแล้ว (confirmed) คำนวณจากฐาน PrincipalAmount เช่นกัน ไม่อยู่ใน escrow แล้ว ไม่สามารถคืนผ่าน Stripe ได้
	var totalDisbursed float64
	if disbursements, err := s.disbursementRepo.ListByProjectID(project.ID); err == nil {
		for _, d := range disbursements {
			if d.Status == domain.DisbursementConfirmed {
				totalDisbursed += d.Amount
			}
		}
	}

	remaining := totalPrincipal - totalDisbursed
	if remaining < 0 {
		remaining = 0
	}
	if totalPrincipal == 0 || remaining == 0 {
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
			ratio := inv.PrincipalAmount / totalPrincipal
			refundAmount = math.Round(ratio*remaining*100) / 100
			totalRefunded += refundAmount
		}

		// กัน refund เกินยอดสุทธิที่นักลงทุนพึงได้รับ
		if refundAmount > inv.PrincipalAmount {
			refundAmount = inv.PrincipalAmount
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

// GetCancelPreview คำนวณและแสดงตัวอย่างผลกระทบก่อนยกเลิกโปรเจกต์
// เช่น ยอดเงินที่เบิกจ่ายให้ pioneer ไปแล้ว และยอดที่จะคืนให้นักลงทุนแต่ละคน
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

	// เตรียมรายการ milestone ให้ frontend เห็นว่า phase ไหนเบิกไปแล้วบ้าง
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

	if err := s.SyncProjectPrincipalAmounts(projectID); err != nil {
		log.Printf("[GetCancelPreview] sync principal amounts error: %v", err)
	}

	// use same logic as RefundProjectInvestments — ใช้ฐาน PrincipalAmount (สุทธิหลังหักค่าธรรมเนียม)
	investments, err := s.investmentRepo.FindVerifiedByProjectID(projectID)
	if err != nil {
		return nil, err
	}

	var totalPrincipal float64
	for _, inv := range investments {
		totalPrincipal += inv.PrincipalAmount
	}

	remaining := totalPrincipal - totalDisbursed
	if remaining < 0 {
		remaining = 0
	}

	previewInvestors := make([]dto.CancelPreviewInvestor, 0, len(investments))
	// aggregate by booster_user_id
	type aggEntry struct {
		dto.CancelPreviewInvestor
		principalAmount float64
	}
	aggMap := make(map[uint]*aggEntry)

	// คนเดียวอาจลงทุนหลายรอบในโปรเจกต์เดียวกัน เลยต้องรวมยอดตาม user ก่อนคำนวณคืนเงิน
	for _, inv := range investments {
		e, ok := aggMap[inv.BoosterUserID]
		if !ok {
			aggMap[inv.BoosterUserID] = &aggEntry{
				CancelPreviewInvestor: dto.CancelPreviewInvestor{
					UserID:      inv.BoosterUserID,
					TotalAmount: inv.TotalAmount,
				},
				principalAmount: inv.PrincipalAmount,
			}
		} else {
			e.TotalAmount += inv.TotalAmount
			e.principalAmount += inv.PrincipalAmount
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
		if totalPrincipal > 0 {
			ratio := entry.principalAmount / totalPrincipal
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

// HandleStripeWebhook รับและประมวลผล webhook event จาก Stripe (ชำระเงินสำเร็จ/ไม่สำเร็จ) หลังตรวจสอบลายเซ็นแล้ว
func (s *investmentService) HandleStripeWebhook(payload []byte, sigHeader string) error {
	event, err := webhook.ConstructEventWithOptions(payload, sigHeader, s.webhookSecret, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return fmt.Errorf("webhook signture verification failed: %v", err)
	}

	// event อื่นนอกจากสองแบบนี้ไม่สนใจ ปล่อยผ่านเงียบ ๆ
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

// GetProjectInvestors ดึงรายชื่อผู้ลงทุนทั้งหมดของโปรเจกต์ (ตรวจสอบก่อนว่าโปรเจกต์มีอยู่จริง)
func (s *investmentService) GetProjectInvestors(projectID uint) ([]dto.ProjectInvestorItem, error) {
	_, err := s.projectRepo.FindProjectByID(projectID)
	if err != nil {
		return nil, errors.New("project not found")
	}
	return s.investmentRepo.ListInvestorsByProjectID(projectID)
}

// ListInvestedProjects ดึงรายชื่อโปรเจกต์ทั้งหมดที่ผู้ใช้เคยลงทุน
func (s *investmentService) ListInvestedProjects(boosterUserID uint) ([]dto.InvestedProjectItem, error) {
	return s.investmentRepo.ListInvestedProjectsByUserID(boosterUserID)
}

// VoteMilestone บันทึกคะแนนโหวตของผู้ลงทุนต่อ milestone และตรวจสอบผลโหวตอัตโนมัติ
// (ผ่าน/ไม่ผ่าน) ตามสัดส่วนยอดเงินลงทุนที่ verified แล้วในโปรเจกต์
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

	// ต้องมีเงินลงทุนจริง (verified) ในโปรเจกต์นี้ถึงจะมีสิทธิ์โหวต
	ok, err := s.projectRepo.HasVerifiedInvestment(m.ProjectID, boosterUserID)
	if err != nil {
		return nil, errors.New("internal server error")
	}
	if !ok {
		return nil, errors.New("only verified investors can vote")
	}

	// โหวตได้คนละครั้งต่อ milestone
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

	// เผื่อระหว่างที่บันทึกโหวตอยู่ มีคนอื่นปิดโหวตไปพอดี ก็แค่บันทึกไว้เฉย ๆ ไม่ต้องไปนับผลต่อ
	m, _ = s.projectRepo.FindMilestoneByID(milestoneID)
	if !m.VotingOpen {
		return vote, nil // voting already closed
	}

	// เช็คทุกครั้งที่มีคนโหวตว่าเกินครึ่งหนึ่งของเงินลงทุนทั้งหมดหรือยัง ถ้าเกินก็ปิดโหวตและผ่านทันทีไม่ต้องรอครบเวลา
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
			// เช็คฝั่งตรงข้ามด้วย ถ้าโหวต reject เกินครึ่งแล้วก็ปิดโหวตเลย ไม่ต้องรอให้ครบเวลา
			rejectAmount, err3 := s.projectRepo.SumMilestoneVotes(milestoneID, domain.MilestoneVoteReject)
			if err3 == nil && rejectAmount*2 > eligibleAmount {
				now := time.Now().UTC()
				m.VotingOpen = false
				m.VotingClosedAt = &now
				_ = s.projectRepo.CloseMeetingsByMilestoneID(m.ID)

				// รีทรายได้แค่ 1 ครั้ง ถ้าตกรอบสองก็จบเลย โปรเจกต์โดนระงับ
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

// FinalizeVotingIfExpired ปิดรอบโหวต milestone อัตโนมัติเมื่อเลยกำหนดเวลาโหวตแล้ว แล้วสรุปผลตามคะแนนที่มีอยู่
func (s *investmentService) FinalizeVotingIfExpired(milestoneID uint) error {
	m, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil {
		return err
	}

	// ไม่ได้อยู่ในช่วงเปิดโหวตก็ไม่มีอะไรให้ทำ
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

// GetMyVote ดึงคะแนนโหวตของผู้ใช้ที่มีต่อ milestone ที่ระบุ
func (s *investmentService) GetMyVote(boosterUserID uint, milestoneID uint) (*domain.MilestoneVote, error) {
	if boosterUserID == 0 {
		return nil, errors.New("unauthorized")
	}
	return s.projectRepo.FindVote(milestoneID, boosterUserID)
}

// GetMilestoneVoters ดึงรายชื่อผู้ลงทุนทั้งหมดของโปรเจกต์พร้อมสถานะการโหวตต่อ milestone
// (เฉพาะเจ้าของโปรเจกต์ (pioneer) เท่านั้นที่เรียกดูได้)
func (s *investmentService) GetMilestoneVoters(pioneerUserID uint, milestoneID uint) ([]dto.MilestoneVoterItem, error) {
	milestone, err := s.projectRepo.FindMilestoneByID(milestoneID)
	if err != nil || milestone == nil {
		return nil, errors.New("milestone not found")
	}

	project, err := s.projectRepo.FindProjectByID(milestone.ProjectID)
	if err != nil || project == nil {
		return nil, errors.New("project not found")
	}

	// เฉพาะ pioneer เจ้าของโปรเจกต์เท่านั้นที่ดูรายชื่อคนโหวตได้
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

	// ทำเป็น map ไว้ก่อนเพื่อเช็คได้เร็วว่าใครโหวตอะไรไปแล้วบ้าง
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

	if err := s.SyncProjectPrincipalAmounts(m.ProjectID); err != nil {
		log.Printf("[createDisbursement] sync principal amounts error: %v", err)
	}

	investments, err := s.investmentRepo.FindVerifiedByProjectID(m.ProjectID)
	if err != nil {
		log.Printf("[createDisbursement] list investments failed: %v", err)
		return
	}

	var totalPrincipal float64
	for _, inv := range investments {
		totalPrincipal += inv.PrincipalAmount
	}

	// ใช้ยอด PrincipalAmount (หักค่าธรรมเนียม Stripe + ค่าธรรมเนียมแพลตฟอร์ม + VAT แล้ว) เป็นฐาน
	// เพื่อไม่ให้ฝั่งแพลตฟอร์มต้องแบกรับค่าธรรมเนียมเหล่านี้เองตอนโอนเงินให้ pioneer
	amount := totalPrincipal * float64(m.PercentRelease) / 100.0

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
func (s *investmentService) createStripePromptPay(amount float64, refNum string, projectTitle string, email string) (qrURL, qrData, intentID, clientSecret string, expiresAt time.Time, err error) {
	stripe.Key = s.stripeSecretKey

	// สร้าง payment method แบบ promptpay ผูกกับอีเมลผู้ลงทุนก่อน
	pm, err := paymentmethod.New(&stripe.PaymentMethodParams{
		Type: stripe.String("promptpay"),
		BillingDetails: &stripe.PaymentMethodBillingDetailsParams{
			Email: stripe.String(email),
		},
	})
	if err != nil {
		return "", "", "", "", time.Time{}, fmt.Errorf("create payment method error: %v", err)
	}

	// Stripe คิดหน่วยเป็นสตางค์ (satang) ไม่ใช่บาท เลยต้อง x100
	amountInSatang := int64(math.Round(amount * 100))

	// Confirm: true เพื่อให้ Stripe สร้าง QR กลับมาให้เลยในขั้นตอนเดียว ไม่ต้อง confirm แยกอีกรอบ
	pi, err := paymentintent.New(&stripe.PaymentIntentParams{
		Amount:             stripe.Int64(amountInSatang),
		Currency:           stripe.String("thb"),
		PaymentMethodTypes: []*string{stripe.String("promptpay")},
		PaymentMethod:      stripe.String(pm.ID),
		Confirm:            stripe.Bool(true),
		Metadata: map[string]string{
			"reference_number": refNum,
			"project_title":    projectTitle,
		},
	})
	if err != nil {
		return "", "", "", "", time.Time{}, fmt.Errorf("create payment intent error: %v", err)
	}

	// ปกติ Stripe จะแนบ QR code มาให้ใน next_action ถ้าไม่มีแปลว่ามีอะไรผิดปกติ
	if pi.NextAction == nil || pi.NextAction.PromptPayDisplayQRCode == nil {
		return "", "", "", "", time.Time{}, errors.New("promptpay QR code not returned by Stripe")
	}

	return pi.NextAction.PromptPayDisplayQRCode.ImageURLPNG,
		pi.NextAction.PromptPayDisplayQRCode.Data,
		pi.ID,
		pi.ClientSecret,
		time.Now().Add(5 * time.Minute), // ให้เวลาสแกนจ่าย 5 นาทีตามเงื่อนไข PromptPay
		nil
}

// generateQRBase64 แปลงข้อมูล QR code เป็นรูปภาพ PNG ในรูปแบบ base64 data URI
func generateQRBase64(data string) (string, error) {
	png, err := qrcode.Encode(data, qrcode.Medium, 512)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}

// handlePaymentSucceeded ประมวลผลเมื่อชำระเงินสำเร็จ: อัปเดตสถานะธุรกรรม/การลงทุน คำนวณ PrincipalAmount
// จากค่าธรรมเนียม Stripe จริง เพิ่มยอดระดมทุนของโปรเจกต์ และเปลี่ยนสถานะโปรเจกต์เป็น executing หากถึงเป้าหมายแล้ว
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

	// ไปดึงค่าธรรมเนียม Stripe ตัวจริงมา (ตอนสร้าง investment เรารู้แค่ค่าธรรมเนียม platform เอง)
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

	// ตอนนี้ถึงค่อยนับเงินก้อนนี้เข้ายอดระดมทุนของโปรเจกต์ (ตอน pending ยังไม่นับ)
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
			// เริ่ม milestone phase 1 ทันทีที่ครบเป้า
			milestones, err := s.projectRepo.FindMilestonesByProjectID(project.ID)
			if err == nil {
				for i := range milestones {
					if milestones[i].Status == domain.MilestoneWaiting && milestones[i].PhaseNo == 1 {
						milestones[i].Status = domain.MilestoneActive
						if err := s.projectRepo.UpdateMilestone(&milestones[i]); err != nil {
							log.Printf("[Webhook] activate milestone phase 1 error: %v", err)
						}
					}
				}
			}
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

// fetchStripeFeesFromIntent ดึงค่าธรรมเนียม Stripe จริงและยอดสุทธิจาก payment intent
// (ผ่าน latest charge และ balance transaction ของมัน)
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

	// ยังไม่มี charge แปลว่ายังไม่ได้เงินจริง ๆ ก็ยังไม่มีค่าธรรมเนียมให้ดึง
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

	// balance_transaction คือของจริงจาก Stripe (net = ยอดที่เข้าบัญชีจริงหลังหักค่าธรรมเนียม)
	bt := ch.BalanceTransaction
	netAmount = float64(bt.Net) / 100
	// แยก fee ปกติกับ VAT ของ fee ออกจากกัน เพราะฝั่ง fee detail ของ Stripe มัดรวมมาเป็นก้อนเดียว
	for _, detail := range bt.FeeDetails {
		if detail.Type == "tax" {
			stripeFeeVAT += float64(detail.Amount) / 100
		} else {
			stripeFee += float64(detail.Amount) / 100
		}
	}
	return
}

// handlePaymentFailed ประมวลผลเมื่อชำระเงินไม่สำเร็จ: อัปเดตสถานะธุรกรรม/การลงทุนเป็นล้มเหลว และแจ้งเตือนผู้ลงทุน
func (s *investmentService) handlePaymentFailed(intentID string) {
	txn, err := s.transactionRepo.FindByPaymentIntentID(intentID)
	if err != nil {
		log.Printf("[Webhook] transaction not found for intent %s: %v", intentID, err)
		return
	}

	_ = s.transactionRepo.UpdateStatus(txn.ID, domain.TransactionFailed)
	_ = s.investmentRepo.UpdateStatus(txn.InvestmentID, domain.InvestmentRejected)

	// notify booster ว่าการชำระเงินไม่สำเร็จ/QR หมดอายุ เพื่อให้รู้ว่าต้องทำรายการใหม่
	if s.notifSvc != nil {
		investment, err := s.investmentRepo.FindByID(txn.InvestmentID)
		if err != nil {
			log.Printf("[Webhook] investment not found: %v", err)
			return
		}
		relatedID := investment.ProjectID
		relatedType := "project"
		if err := s.notifSvc.CreateAndPush(investment.BoosterUserID, domain.NotifPaymentFailed, "การชำระเงินไม่สำเร็จ", "การชำระเงินสำหรับการลงทุนของคุณไม่สำเร็จ หรือ QR Code หมดอายุ กรุณาทำรายการใหม่", &relatedID, &relatedType); err != nil {
			log.Printf("[Webhook] send notification error: %v", err)
		}
	}
}

// // helper functions

// amountEpsilon คือค่าความคลาดเคลื่อนที่ยอมรับได้เมื่อเปรียบเทียบจำนวนเงิน (float64)
// ห้ามเทียบ float ด้วย == ตรง ๆ เพราะการคำนวณสะสม (เช่น current_funding ที่บวกทีละรายการ)
// อาจมีเศษความคลาดเคลื่อนระดับ 1e-9 ที่ทำให้เงื่อนไข == พลาดได้
const amountEpsilon = 0.01

// amountLess เทียบว่า a น้อยกว่า b หรือไม่ โดยเผื่อ epsilon กันความคลาดเคลื่อนของ float
func amountLess(a, b float64) bool { return a < b-amountEpsilon }

// amountGreater เทียบว่า a มากกว่า b หรือไม่ โดยเผื่อ epsilon กันความคลาดเคลื่อนของ float
func amountGreater(a, b float64) bool { return a > b+amountEpsilon }

// amountEqual เทียบว่า a เท่ากับ b หรือไม่ โดยเผื่อ epsilon กันความคลาดเคลื่อนของ float
func amountEqual(a, b float64) bool { return math.Abs(a-b) < amountEpsilon }

// validateAmount ตรวจสอบยอดลงทุนตาม invariant: หลังทุกการลงทุน ยอดคงเหลือของโปรเจกต์
// ต้องเป็น 0 (ปิดยอดพอดี) หรือ ≥ ฿20 (ขั้นต่ำของช่องทางชำระเงิน Stripe PromptPay) เสมอ
// ป้องกันกรณีที่ยอดคงเหลือตกค้างต่ำกว่าขั้นต่ำจนไม่มีใครสามารถลงทุนปิดยอดที่เหลือได้ (Jira F2-116)
//
// ลำดับการเช็คมีผลต่อ error message ที่ผู้ใช้เห็น จึงต้องเรียงตามนี้เท่านั้น:
// 1) ขั้นต่ำ ฿20 ต่อครั้ง (ไม่มีข้อยกเว้น) 2) ห้ามเกินยอดคงเหลือ 3) ขั้นต่ำ 1% ของเป้าหมาย
// (ยกเว้นเมื่อปิดยอดพอดี หรือถึง softcap แล้ว) 4) เพดานสูงสุดของโปรเจกต์ 5) ห้ามทิ้งเศษ (ไม่มีข้อยกเว้น)
func validateAmount(project *domain.Project, amount float64) error {
	remaining := project.FundingGoal - project.CurrentFunding
	left := remaining - amount

	// 1. ขั้นต่ำของช่องทางชำระเงิน Stripe (PromptPay) — บังคับใช้เสมอ ไม่มีข้อยกเว้นใด ๆ แม้ถึง softcap แล้ว
	if amountLess(amount, MinInvestmentPerTransaction) {
		return fmt.Errorf("จำนวนเงินขั้นต่ำต่อครั้งคือ ฿%.2f", MinInvestmentPerTransaction)
	}

	// 2. ห้ามลงทุนเกินยอดคงเหลือของโปรเจกต์
	if amountGreater(amount, remaining) {
		return fmt.Errorf("ยอดคงเหลือให้ลงทุนคือ ฿%.2f", remaining)
	}

	// 3. ขั้นต่ำ 1% ของเป้าหมาย — ยกเว้นเมื่อลงทุนปิดยอดพอดี หรือระดมทุนถึง softcap แล้ว
	// เพื่อให้นักลงทุนสามารถลงทุนยอด remaining ที่เหลือได้แม้จะน้อยกว่าขั้นต่ำปกติ
	isClosing := amountEqual(remaining, amount)
	softcapReached := project.Softcap > 0 && !amountLess(project.CurrentFunding, project.Softcap)
	if !isClosing && !softcapReached {
		effectiveMin := math.Max(project.MinInvestAmount, project.FundingGoal*0.01)
		if amountLess(amount, effectiveMin) {
			return fmt.Errorf("จำนวนเงินขั้นต่ำคือ ฿%.2f (1%% ของยอดเป้าหมาย)", effectiveMin)
		}
	}

	// 4. เพดานสูงสุดต่อรายการของโปรเจกต์ (0 = ไม่มีเพดาน)
	if project.MaxInvestAmount > 0 && amountGreater(amount, project.MaxInvestAmount) {
		return fmt.Errorf("จำนวนเงินสูงสุดคือ ฿%.2f", project.MaxInvestAmount)
	}

	// 5. ห้ามทิ้งเศษ: เช็คนี้อยู่นอกข้อยกเว้นของข้อ 3 (softcap/ปิดยอดพอดี) — ต้องเช็คเสมอไม่มีข้อยกเว้น
	// เพราะเป็น invariant หลักที่ป้องกันไม่ให้โปรเจกต์เหลือยอดค้างที่ไม่มีใครลงทุนปิดได้อีก
	if left >= amountEpsilon && amountLess(left, MinInvestmentPerTransaction) {
		// ไม่มีเพดาน (MaxInvestAmount <= 0) หรือยอดคงเหลือยังอยู่ในเพดาน → มีคนปิดยอดคนเดียวได้
		canCloseAlone := project.MaxInvestAmount <= 0 || !amountGreater(remaining, project.MaxInvestAmount)
		if canCloseAlone {
			return fmt.Errorf("การลงทุนนี้จะเหลือยอดค้าง ฿%.2f ซึ่งระดมต่อไม่ได้ กรุณาปิดยอดที่เหลือ ฿%.2f", left, remaining)
		}
		return fmt.Errorf("การลงทุนนี้จะเหลือยอดค้าง ฿%.2f ซึ่งระดมต่อไม่ได้ กรุณาลดยอดเหลือไม่เกิน ฿%.2f", left, remaining-MinInvestmentPerTransaction)
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

	// สุ่มทีละตัวอักษรจาก crypto/rand ให้แน่ใจว่าเดาไม่ได้ (เลขอ้างอิงนี้เอาไปผูกกับ Stripe metadata ด้วย)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		result[i] = chars[n.Int64()]
	}

	return "INV-" + strings.ToUpper(string(result)), nil
}

// GetTotalFunding รวมยอดเงินระดมทุนทั้งหมดในระบบ
func (s *investmentService) GetTotalFunding() (float64, error) {
	return s.investmentRepo.SumTotalFunding()
}

// GetUniqueBoostersCount นับจำนวนผู้ลงทุนที่ไม่ซ้ำกันในระบบ
func (s *investmentService) GetUniqueBoostersCount() (int64, error) {
	return s.investmentRepo.CountUniqueBoosters()
}

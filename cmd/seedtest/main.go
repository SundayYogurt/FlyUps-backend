// cmd/seedtest/main.go
// Seed ข้อมูลสำหรับรัน System Test UC1-UC8 (FlyUp_TestPlan_UC1-UC8.docx)
//
// ใช้กับ STAGING เท่านั้น ห้ามชี้ DSN ไป production เด็ดขาด
//
// วิธีรัน:
//
//	DSN="host=localhost user=... dbname=flyup_staging ..." go run ./cmd/seedtest
//
// สคริปต์นี้ idempotent: รันซ้ำได้ จะลบเฉพาะข้อมูล seed เดิม
// (email ลงท้าย @seed.flyup.test และ slug ขึ้นต้น seed-) แล้วสร้างใหม่ทุกครั้ง
package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"flyup/internal/domain"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	seedEmailSuffix = "@seed.flyup.test"
	seedSlugPrefix  = "seed-"
	defaultPassword = "Test1234" // ทุกบัญชี seed ใช้รหัสเดียวกัน
)

func main() {
	_ = godotenv.Load()

	dsn := strings.TrimSpace(os.Getenv("DSN"))
	if dsn == "" {
		log.Fatal("DSN env var is required")
	}
	// กันพลาดยิงใส่ prod (ปรับ keyword ให้ตรงชื่อ db จริงของทีม)
	if strings.Contains(strings.ToLower(dsn), "prod") {
		log.Fatal("refusing to seed: DSN looks like production")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("database connection error: %v", err)
	}

	start := time.Now()
	if err := db.Transaction(seedAll); err != nil {
		log.Fatalf("seed failed: %v", err)
	}
	log.Printf("seed test data ok in %s", time.Since(start))
	printAccounts()
}

func seedAll(tx *gorm.DB) error {
	if err := cleanup(tx); err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}

	// ---------- Users ----------
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now()

	mkUser := func(email, first, last, phone, role string) *domain.User {
		return &domain.User{
			Email: email, PasswordHash: string(hash),
			FirstName: first, LastName: last, Phone: phone,
			Status: domain.ACTIVE, Role: role, EmailVerifiedAt: &now,
		}
	}

	admin := mkUser("admin"+seedEmailSuffix, "แอดมิน", "ระบบ", "0800000001", "admin")
	booster1 := mkUser("booster1"+seedEmailSuffix, "บูสเตอร์", "หนึ่ง", "0800000011", "booster")     // ยืนยันแล้ว ลงทุน 60%
	booster2 := mkUser("booster2"+seedEmailSuffix, "บูสเตอร์", "สอง", "0800000012", "booster")       // ยืนยันแล้ว ลงทุน 25% + เคยร้องเรียน
	booster3 := mkUser("booster3"+seedEmailSuffix, "บูสเตอร์", "สาม", "0800000013", "booster")       // ยืนยันแล้ว ลงทุน 15%
	boosterNew := mkUser("booster_new"+seedEmailSuffix, "บูสเตอร์", "ใหม่", "0800000014", "booster") // ยังไม่ยืนยัน (TC0406)
	pioneer1 := mkUser("pioneer1"+seedEmailSuffix, "ไพโอเนียร์", "เจ้าของ", "0800000021", "pioneer") // เจ้าของโปรเจกต์ (TC0408)
	pioneer2 := mkUser("pioneer2"+seedEmailSuffix, "ไพโอเนียร์", "คนอื่น", "0800000022", "pioneer")  // ไม่ใช่เจ้าของ (TC0409)

	users := []*domain.User{admin, booster1, booster2, booster3, boosterNew, pioneer1, pioneer2}
	for _, u := range users {
		if err := tx.Create(u).Error; err != nil {
			return fmt.Errorf("create user %s: %w", u.Email, err)
		}
	}

	// ยืนยันตัวตน (IdCard approved) + ผูกบัญชีธนาคาร ให้ booster1-3 (precondition UC4)
	for i, b := range []*domain.User{booster1, booster2, booster3} {
		v := domain.IdCardVerification{
			UserID: b.ID, Document: "seed-idcard.jpg",
			Status: domain.VerifyStatusApproved, VerifiedAt: &now, ReviewedBy: &admin.ID,
		}
		if err := tx.Create(&v).Error; err != nil {
			return err
		}
		ba := domain.BankAccount{
			UserID: b.ID, BankName: "KBANK",
			AccountName:   b.FirstName + " " + b.LastName,
			AccountNumber: fmt.Sprintf("123456789%d", i), IsDefault: true,
		}
		if err := tx.Create(&ba).Error; err != nil {
			return err
		}
	}

	// StudentProfile ให้ pioneer1 (ข้อมูลผู้สร้างใน TC0102)
	var uni domain.University
	if err := tx.First(&uni).Error; err != nil {
		return fmt.Errorf("no university found - run `go run ./cmd/seed` first: %w", err)
	}
	bio := "นักศึกษาวิศวกรรมซอฟต์แวร์ ชอบทำ IoT"
	sp := domain.StudentProfile{UserID: pioneer1.ID, UniversityID: uni.ID, Bio: &bio, VerifiedAt: &now}
	if err := tx.Create(&sp).Error; err != nil {
		return err
	}

	// ---------- Categories ----------
	catIOT := domain.ProjectCategory{Name: "IOT"}
	catWeb := domain.ProjectCategory{Name: "Web Application"} // เว้นว่างไม่ใส่โปรเจกต์ (TC0303)
	if err := tx.Where("name = ?", catIOT.Name).FirstOrCreate(&catIOT).Error; err != nil {
		return err
	}
	if err := tx.Where("name = ?", catWeb.Name).FirstOrCreate(&catWeb).Error; err != nil {
		return err
	}

	// ---------- Projects ----------
	desc := "ระบบรดน้ำอัจฉริยะควบคุมผ่านมือถือ"
	endFuture := now.AddDate(0, 1, 0)
	endPast := now.AddDate(0, -1, 0)

	// P1: กำลังระดมทุน — ใช้กับ UC1, UC2, UC4, UC6, TC0703
	p1 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID,
		Title: "Smart Farm IoT ระบบรดน้ำอัจฉริยะ", Description: &desc,
		State: domain.StateFunding, Status: domain.StatusActive, Visibility: domain.VisibilityPublic,
		FundingGoal: 100000, Softcap: 50000, CurrentFunding: 10000,
		DurationDays: 60, DurationMonths: 6, EndDate: endFuture, FundingAt: now.AddDate(0, 0, -14),
		ProfitSharePct: 10, MinInvestAmount: 1000, MaxInvestAmount: 10000, PlatformFee: 5, // ขั้นต่ำ = 1% ของเป้า 100,000
		Slug: seedSlugPrefix + "smart-farm",
	}
	// P2: กำลังดำเนินการ (executing/funded) — ใช้กับ UC5, UC7, UC8
	p2 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID,
		Title: "AI Chatbot ช่วยติวสอบ", Description: &desc,
		State: domain.StateExecuting, Status: domain.StatusFunded, Visibility: domain.VisibilityPublic,
		FundingGoal: 10000, Softcap: 5000, CurrentFunding: 10000,
		DurationDays: 30, DurationMonths: 6, EndDate: endPast, FundingAt: now.AddDate(0, -2, 0),
		ProfitSharePct: 10, MinInvestAmount: 100, MaxInvestAmount: 10000, PlatformFee: 5,
		Slug: seedSlugPrefix + "ai-chatbot",
	}
	// P3: ล้มเหลว — ใช้กับ TC0505
	p3 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID,
		Title: "Drone ส่งของในมหาวิทยาลัย", Description: &desc,
		State: domain.StateClosed, Status: domain.StatusFailed, Visibility: domain.VisibilityPublic,
		FundingGoal: 200000, Softcap: 100000, CurrentFunding: 3000,
		DurationDays: 30, DurationMonths: 6, EndDate: endPast, FundingAt: now.AddDate(0, -3, 0),
		ProfitSharePct: 12, MinInvestAmount: 2000, MaxInvestAmount: 20000, PlatformFee: 5, // ขั้นต่ำ = 1% ของเป้า 200,000
		Slug: seedSlugPrefix + "drone-delivery",
	}
	for _, p := range []*domain.Project{&p1, &p2, &p3} {
		if err := tx.Create(p).Error; err != nil {
			return fmt.Errorf("create project %s: %w", p.Slug, err)
		}
	}

	// ---------- Investments ----------
	// helper คำนวณตาม calculateFees ใน investment_service.go:
	// จ่าย amount -> fee 5% ของ amount -> VAT 7% ของ fee -> โปรเจกต์ได้ principal = amount - fee - vat
	round2 := func(v float64) float64 { return math.Round(v*100) / 100 }
	mkInvest := func(ref string, projectID, boosterID uint, amount, sharePct float64, status domain.InvestmentStatus, paid bool) domain.Investment {
		fee := round2(amount * 0.05)
		vat := round2(fee * 0.07)
		inv := domain.Investment{
			ReferenceNumber: ref, ProjectID: projectID, BoosterUserID: boosterID,
			PrincipalAmount: round2(amount - fee - vat), PlatformFee: fee, VATAmount: vat,
			TotalAmount: amount, ProfitSharePct: sharePct, Status: status,
		}
		if paid {
			inv.PaidAt = &now
		}
		return inv
	}

	investments := []domain.Investment{
		// P2: ลงทุน 60/25/15 ของยอด 10,000 → ใช้จำลองโหวตถ่วงน้ำหนัก >51% (TC0804)
		mkInvest(seedRef("P2B1"), p2.ID, booster1.ID, 6000, 6.0, domain.InvestmentVerified, true),
		mkInvest(seedRef("P2B2"), p2.ID, booster2.ID, 2500, 2.5, domain.InvestmentVerified, true),
		mkInvest(seedRef("P2B3"), p2.ID, booster3.ID, 1500, 1.5, domain.InvestmentVerified, true),
		// P1: booster1 ลงทุนในโปรเจกต์กำลังระดมทุน → ใช้ทดสอบขอคืนเงิน (TC0703)
		mkInvest(seedRef("P1B1"), p1.ID, booster1.ID, 1000, 0.1, domain.InvestmentVerified, true),
		// P3 (ล้มเหลว): booster1 ลงทุนไว้ → ปุ่มต้องเปลี่ยนเป็น "ขอเงินคืน" (TC0505)
		mkInvest(seedRef("P3B1"), p3.ID, booster1.ID, 2000, 1.0, domain.InvestmentVerified, true),
	}
	for _, inv := range investments {
		if err := tx.Create(&inv).Error; err != nil {
			return fmt.Errorf("create investment %s: %w", inv.ReferenceNumber, err)
		}
	}

	// ---------- Milestones (P2) ----------
	crit := "ส่งมอบตามเกณฑ์ที่ตกลง"
	summary := "พัฒนาเสร็จตามแผน แนบหลักฐาน GitHub"
	closedAt := now.AddDate(0, 0, -20)

	// Phase 1: จ่ายแล้ว + ปิดโหวตแล้ว (TC0805 ดูผลโหวตรวมได้ แต่โหวตเพิ่มไม่ได้)
	m1 := domain.Milestone{
		ProjectID: p2.ID, PhaseNo: 1, Title: "Phase 1: ออกแบบระบบ",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary,
		SubmissionLinks: []string{"https://github.com/SundayYogurt/FlyUps-backend"},
		Status:          domain.MilestonePaid, PercentRelease: 40, SortOrder: 1,
		VotingOpen: false, VotingOpenedAt: &closedAt, VotingClosedAt: &now, SubmittedAt: &closedAt,
	}
	// Phase 2: approved + เปิดโหวตอยู่ (precondition UC8: TC0801-TC0804)
	m2 := domain.Milestone{
		ProjectID: p2.ID, PhaseNo: 2, Title: "Phase 2: พัฒนา MVP",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary,
		SubmissionLinks: []string{"https://github.com/SundayYogurt/FlyUps-backend"},
		Status:          domain.MilestoneApproved, PercentRelease: 40, SortOrder: 2,
		VotingOpen: true, VotingOpenedAt: &now, SubmittedAt: &now,
	}
	// Phase 3: ยังไม่ดำเนินการ (จุดสีเทาใน TC0503)
	m3 := domain.Milestone{
		ProjectID: p2.ID, PhaseNo: 3, Title: "Phase 3: เปิดใช้งานจริง",
		AcceptanceCriteria: &crit,
		Status:             domain.MilestoneWaiting, PercentRelease: 20, SortOrder: 3,
	}
	for _, m := range []*domain.Milestone{&m1, &m2, &m3} {
		if err := tx.Create(m).Error; err != nil {
			return fmt.Errorf("create milestone phase %d: %w", m.PhaseNo, err)
		}
	}
	// ผลโหวตของ Phase 1 (ที่ปิดไปแล้ว)
	votes := []domain.MilestoneVote{
		{MilestoneID: m1.ID, ProjectID: p2.ID, BoosterUserID: booster1.ID, Choice: domain.MilestoneVoteApprove},
		{MilestoneID: m1.ID, ProjectID: p2.ID, BoosterUserID: booster2.ID, Choice: domain.MilestoneVoteApprove},
		{MilestoneID: m1.ID, ProjectID: p2.ID, BoosterUserID: booster3.ID, Choice: domain.MilestoneVoteReject},
	}
	for _, v := range votes {
		if err := tx.Create(&v).Error; err != nil {
			return err
		}
	}

	// ---------- ProfitPool + Payout (TC0701, TC0702) ----------
	pool := domain.ProfitPool{
		ProjectID: p2.ID, PioneerUserID: pioneer1.ID, TotalAmount: 1000,
		TransferRef: seedRef("POOL"), Status: domain.ProfitPoolCompleted, QuarterNo: 1,
	}
	if err := tx.Create(&pool).Error; err != nil {
		return err
	}
	payout := domain.InvestorProfitPayout{
		ProfitPoolID: pool.ID, ProjectID: p2.ID, BoosterUserID: booster1.ID,
		Amount: 600, SharePct: 6.0, Status: domain.InvestorPayoutConfirmed,
		TransferRef: seedRef("PAYB1"), ConfirmedAt: &now, ConfirmedBy: &admin.ID,
	}
	if err := tx.Create(&payout).Error; err != nil {
		return err
	}

	// ---------- Complaint (TC0607: booster2 เคยร้องเรียน P1 แล้ว) ----------
	c := domain.Complaint{
		ComplainantID: booster2.ID, ProjectID: p1.ID,
		Subject: "โปรเจกต์ไม่อัปเดตความคืบหน้า",
		Body:    "seed: ใช้ทดสอบเคสร้องเรียนซ้ำ", Status: domain.ComplaintOpen,
	}
	return tx.Create(&c).Error
}

// cleanup ลบข้อมูล seed เดิม + ข้อมูลที่แอปสร้างพ่วงกับ seed ระหว่างเทส
// (เช่น investment/transaction ที่เกิดจากการกดลงทุนจริงในโปรเจกต์ seed)
// เรียงลำดับลบจากตารางลูก -> แม่ ตาม FK
func cleanup(tx *gorm.DB) error {
	var userIDs []uint
	if err := tx.Model(&domain.User{}).
		Where("email LIKE ?", "%"+seedEmailSuffix).Pluck("id", &userIDs).Error; err != nil {
		return err
	}
	var projectIDs []uint
	if err := tx.Model(&domain.Project{}).
		Where("slug LIKE ?", seedSlugPrefix+"%").Pluck("id", &projectIDs).Error; err != nil {
		return err
	}
	if len(userIDs) == 0 && len(projectIDs) == 0 {
		return nil
	}
	// กัน slice ว่างใน IN clause
	if len(userIDs) == 0 {
		userIDs = []uint{0}
	}
	if len(projectIDs) == 0 {
		projectIDs = []uint{0}
	}

	// 1) investments ทั้งหมดที่พัวพันกับ seed (ทั้งในโปรเจกต์ seed และที่ user seed ไปลงทุนที่อื่น)
	var investmentIDs []uint
	if err := tx.Model(&domain.Investment{}).
		Where("project_id IN ? OR booster_user_id IN ?", projectIDs, userIDs).
		Pluck("id", &investmentIDs).Error; err != nil {
		return err
	}
	if len(investmentIDs) > 0 {
		// transactions อ้าง investment (fk_transactions_investment) ต้องลบก่อน
		if err := tx.Unscoped().Where("investment_id IN ?", investmentIDs).
			Delete(&domain.Transaction{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("id IN ?", investmentIDs).
			Delete(&domain.Investment{}).Error; err != nil {
			return err
		}
	}

	// 2) milestones + ตารางลูกของมัน (votes, meetings)
	var milestoneIDs []uint
	if err := tx.Model(&domain.Milestone{}).
		Where("project_id IN ?", projectIDs).Pluck("id", &milestoneIDs).Error; err != nil {
		return err
	}
	if len(milestoneIDs) > 0 {
		if err := tx.Unscoped().Where("milestone_id IN ?", milestoneIDs).
			Delete(&domain.Meeting{}).Error; err != nil {
			return err
		}
	}
	// votes: ทั้งของโปรเจกต์ seed และที่ user seed ไปโหวตที่อื่น
	if err := tx.Unscoped().Where("project_id IN ? OR booster_user_id IN ?", projectIDs, userIDs).
		Delete(&domain.MilestoneVote{}).Error; err != nil {
		return err
	}

	// 3) ตารางลูกอื่นๆ ที่อ้าง project
	for _, m := range []any{
		&domain.InvestorProfitPayout{}, &domain.ProfitPool{}, &domain.Disbursement{},
		&domain.ProjectThreadMessage{}, &domain.ProjectThread{}, &domain.ProjectUpdate{},
		&domain.Milestone{}, &domain.ProjectMedia{},
		&domain.StorySection{}, &domain.ProjectFAQ{},
	} {
		if err := tx.Unscoped().Where("project_id IN ?", projectIDs).Delete(m).Error; err != nil {
			return err
		}
	}
	// complaints: ทั้งบนโปรเจกต์ seed และที่ user seed ไปร้องเรียนที่อื่น
	if err := tx.Unscoped().Where("project_id IN ? OR complainant_id IN ?", projectIDs, userIDs).
		Delete(&domain.Complaint{}).Error; err != nil {
		return err
	}
	// notifications ของ user seed (ไม่มี FK แต่เคลียร์ไว้ให้เทสรอบใหม่สะอาด)
	if err := tx.Unscoped().Where("user_id IN ?", userIDs).
		Delete(&domain.Notification{}).Error; err != nil {
		return err
	}

	// 4) projects
	if err := tx.Unscoped().Where("id IN ?", projectIDs).Delete(&domain.Project{}).Error; err != nil {
		return err
	}

	// 5) ข้อมูลผูกกับ user แล้วค่อยลบ user
	for _, m := range []any{
		&domain.IdCardVerification{}, &domain.StudentCardVerification{},
		&domain.BankAccount{}, &domain.StudentProfile{},
	} {
		if err := tx.Unscoped().Where("user_id IN ?", userIDs).Delete(m).Error; err != nil {
			return err
		}
	}
	return tx.Unscoped().Where("id IN ?", userIDs).Delete(&domain.User{}).Error
}

func seedRef(suffix string) string {
	return fmt.Sprintf("SEED-%s-%s", time.Now().Format("20060102150405"), suffix)
}

func printAccounts() {
	fmt.Println(`
================ SEED ACCOUNTS (password: ` + defaultPassword + `) ================
admin@seed.flyup.test        Admin
booster1@seed.flyup.test     Booster ยืนยันแล้ว | ลงทุน P1(1,000) P2(6,000=60%) P3(2,000) | ได้ปันผล Q1 600บ.
booster2@seed.flyup.test     Booster ยืนยันแล้ว | ลงทุน P2(2,500=25%) | เคยร้องเรียน P1 แล้ว
booster3@seed.flyup.test     Booster ยืนยันแล้ว | ลงทุน P2(1,500=15%)
booster_new@seed.flyup.test  Booster ยังไม่ยืนยันตัวตน (TC0406)
pioneer1@seed.flyup.test     Pioneer เจ้าของทุกโปรเจกต์ seed (TC0408)
pioneer2@seed.flyup.test     Pioneer ไม่ใช่เจ้าของ (TC0409)

PROJECTS
seed-smart-farm      กำลังระดมทุน  → UC1-UC4, UC6, TC0703
seed-ai-chatbot      กำลังดำเนินการ → UC5, UC7, UC8 (Phase2 เปิดโหวต)
seed-drone-delivery  ล้มเหลว       → TC0505
หมวดหมู่ "Web Application" ว่างเปล่า → TC0303
===================================================================`)
}

// cmd/seedtest/main.go
// Seed ข้อมูลสำหรับรัน System Test UC1-UC23 (FLYUP-TC-001, 60 test cases)
//
// ใช้กับ STAGING เท่านั้น ห้ามชี้ DSN ไป production เด็ดขาด
//
// วิธีรัน:
//
//	DSN="host=localhost user=... dbname=flyup_staging ..." go run ./cmd/seed      (ต้องรันก่อน 1 ครั้ง เพื่อสร้าง University)
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
	booster1 := mkUser("booster1"+seedEmailSuffix, "บูสเตอร์", "หนึ่ง", "0800000011", "booster")     // ยืนยันแล้ว
	booster2 := mkUser("booster2"+seedEmailSuffix, "บูสเตอร์", "สอง", "0800000012", "booster")       // ยืนยันแล้ว + เคยร้องเรียน
	booster3 := mkUser("booster3"+seedEmailSuffix, "บูสเตอร์", "สาม", "0800000013", "booster")       // ยืนยันแล้ว
	boosterNew := mkUser("booster_new"+seedEmailSuffix, "บูสเตอร์", "ใหม่", "0800000014", "booster") // ยังไม่ยืนยัน (TC-015/019/030/032, target ของ TC-058)
	pioneer1 := mkUser("pioneer1"+seedEmailSuffix, "ไพโอเนียร์", "เจ้าของ", "0800000021", "pioneer") // เจ้าของโปรเจกต์ส่วนใหญ่ (TC-014)
	pioneer2 := mkUser("pioneer2"+seedEmailSuffix, "ไพโอเนียร์", "คนอื่น", "0800000022", "pioneer")  // ไม่ใช่เจ้าของ P1/P2 (TC-014), เจ้าของคิวรอตรวจสอบ

	users := []*domain.User{admin, booster1, booster2, booster3, boosterNew, pioneer1, pioneer2}
	for _, u := range users {
		if err := tx.Create(u).Error; err != nil {
			return fmt.Errorf("create user %s: %w", u.Email, err)
		}
	}

	// ยืนยันตัวตน (IdCard approved) + ผูกบัญชีธนาคาร ให้ booster1-3 (precondition ลงทุนได้ / UC4)
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

	// StudentProfile + IdCard/StudentCard approved + บัญชีธนาคาร ให้ pioneer1, pioneer2
	// (จำเป็นสำหรับ CreateProject จริง: ต้องมี IdCard approved + StudentCard approved + บัญชีธนาคาร - UC12, UC14)
	var uni domain.University
	if err := tx.First(&uni).Error; err != nil {
		return fmt.Errorf("no university found - run `go run ./cmd/seed` first: %w", err)
	}
	for i, p := range []*domain.User{pioneer1, pioneer2} {
		bio := "นักศึกษาวิศวกรรมซอฟต์แวร์ ชอบทำ IoT"
		sp := domain.StudentProfile{UserID: p.ID, UniversityID: uni.ID, Bio: &bio, VerifiedAt: &now}
		if err := tx.Create(&sp).Error; err != nil {
			return err
		}
		idc := domain.IdCardVerification{
			UserID: p.ID, Document: "seed-idcard.jpg",
			Status: domain.VerifyStatusApproved, VerifiedAt: &now, ReviewedBy: &admin.ID,
		}
		if err := tx.Create(&idc).Error; err != nil {
			return err
		}
		stc := domain.StudentCardVerification{
			UserID: p.ID, Document: "seed-studentcard.jpg",
			Status: domain.VerifyStatusApproved, VerifiedAt: &now, ReviewedBy: &admin.ID,
		}
		if err := tx.Create(&stc).Error; err != nil {
			return err
		}
		ba := domain.BankAccount{
			UserID: p.ID, BankName: "SCB",
			AccountName:   p.FirstName + " " + p.LastName,
			AccountNumber: fmt.Sprintf("987654321%d", i), IsDefault: true,
		}
		if err := tx.Create(&ba).Error; err != nil {
			return err
		}
	}

	// ---------- Categories ----------
	catIOT := domain.ProjectCategory{Name: "IOT"}
	catWeb := domain.ProjectCategory{Name: "Web Application"} // เว้นว่างไม่ใส่โปรเจกต์ funding ใดๆ (TC-007)
	if err := tx.Where("name = ?", catIOT.Name).FirstOrCreate(&catIOT).Error; err != nil {
		return err
	}
	if err := tx.Where("name = ?", catWeb.Name).FirstOrCreate(&catWeb).Error; err != nil {
		return err
	}

	cover := func(seed string) *string {
		s := "https://picsum.photos/seed/" + seed + "/800/450"
		return &s
	}

	// ---------- Projects ----------
	desc := "ระบบรดน้ำอัจฉริยะควบคุมผ่านมือถือ"
	endFuture := now.AddDate(0, 1, 0)
	endPast := now.AddDate(0, -1, 0)

	// P1: กำลังระดมทุน — ใช้กับ UC1,UC2,UC4,UC6,TC-024,TC-044,TC-046,TC-057
	// (ตัวเลข goal/softcap/min/max ตรงกับ "standard test project" Jira F2-116 - ห้ามแก้)
	p1 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("smart-farm"),
		Title: "Smart Farm IoT ระบบรดน้ำอัจฉริยะ", Description: &desc,
		State: domain.StateFunding, Status: domain.StatusActive, Visibility: domain.VisibilityPublic,
		FundingGoal: 100000, Softcap: 50000, CurrentFunding: 10000,
		DurationDays: 60, DurationMonths: 6, EndDate: endFuture, FundingAt: now.AddDate(0, 0, -14),
		ProfitSharePct: 10, MinInvestAmount: 1000, MaxInvestAmount: 10000, PlatformFee: 5, // ขั้นต่ำ = 1% ของเป้า 100,000
		Slug: seedSlugPrefix + "smart-farm",
	}
	// P2: กำลังดำเนินการ — โหวต Milestone Phase 2 เปิดอยู่ (60/25/15%) — UC5,UC7,UC8,UC9,UC18(Phase3)
	p2 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("ai-chatbot"),
		Title: "AI Chatbot ช่วยติวสอบ", Description: &desc,
		State: domain.StateExecuting, Status: domain.StatusFunded, Visibility: domain.VisibilityPublic,
		FundingGoal: 10000, Softcap: 5000, CurrentFunding: 10000,
		DurationDays: 30, DurationMonths: 6, EndDate: endPast, FundingAt: now.AddDate(0, -2, 0),
		ExecutionEndAt: ptrTime(now.AddDate(0, 2, 0)),
		ProfitSharePct: 10, MinInvestAmount: 100, MaxInvestAmount: 10000, PlatformFee: 5,
		Slug: seedSlugPrefix + "ai-chatbot",
	}
	// P3: ล้มเหลว — ปุ่ม "ขอเงินคืน" (TC-018)
	p3 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("drone-delivery"),
		Title: "Drone ส่งของในมหาวิทยาลัย", Description: &desc,
		State: domain.StateClosed, Status: domain.StatusFailed, Visibility: domain.VisibilityPublic,
		FundingGoal: 200000, Softcap: 100000, CurrentFunding: 3000,
		DurationDays: 30, DurationMonths: 6, EndDate: endPast, FundingAt: now.AddDate(0, -3, 0),
		ProfitSharePct: 12, MinInvestAmount: 2000, MaxInvestAmount: 20000, PlatformFee: 5, // ขั้นต่ำ = 1% ของเป้า 200,000
		Slug: seedSlugPrefix + "drone-delivery",
	}
	// P4: กำลังระดมทุน ถึง softcap แล้ว เหลือ ฿5,000 — ลงทุนต่ำกว่าขั้นต่ำ Stripe ฿20 (TC-011)
	p4 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("eco-bottle"),
		Title: "Eco Bottle ขวดน้ำรีฟิลอัจฉริยะ", Description: &desc,
		State: domain.StateFunding, Status: domain.StatusActive, Visibility: domain.VisibilityPublic,
		FundingGoal: 10000, Softcap: 5000, CurrentFunding: 5000,
		DurationDays: 45, DurationMonths: 6, EndDate: now.AddDate(0, 0, 20), FundingAt: now.AddDate(0, 0, -10),
		ProfitSharePct: 8, MinInvestAmount: 100, MaxInvestAmount: 10000, PlatformFee: 5,
		Slug: seedSlugPrefix + "eco-bottle",
	}
	// P5: กำลังระดมทุน เหลือ ฿1,010 — ลงทุนแล้วเหลือเศษ 1-19 บาท ต้องห้าม (TC-012)
	p5 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("paper-straw"),
		Title: "Paper Straw หลอดกระดาษชานอ้อย", Description: &desc,
		State: domain.StateFunding, Status: domain.StatusActive, Visibility: domain.VisibilityPublic,
		FundingGoal: 100000, Softcap: 50000, CurrentFunding: 98990,
		DurationDays: 60, DurationMonths: 6, EndDate: now.AddDate(0, 0, 5), FundingAt: now.AddDate(0, 0, -55),
		ProfitSharePct: 9, MinInvestAmount: 1000, MaxInvestAmount: 20000, PlatformFee: 5,
		Slug: seedSlugPrefix + "paper-straw",
	}
	// P6: กำลังระดมทุน เหลือ ฿20 พอดี — ปิดยอดคงเหลือข้อยกเว้นขั้นต่ำ 1% (TC-013)
	p6 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("bike-share"),
		Title: "Bike Share จักรยานให้เช่าในมหาวิทยาลัย", Description: &desc,
		State: domain.StateFunding, Status: domain.StatusActive, Visibility: domain.VisibilityPublic,
		FundingGoal: 10000, Softcap: 5000, CurrentFunding: 9980,
		DurationDays: 30, DurationMonths: 6, EndDate: now.AddDate(0, 0, 2), FundingAt: now.AddDate(0, 0, -28),
		ProfitSharePct: 7, MinInvestAmount: 100, MaxInvestAmount: 10000, PlatformFee: 5,
		Slug: seedSlugPrefix + "bike-share",
	}
	// P7: กำลังดำเนินการ — Milestone เปิดโหวตใหม่ (60/25/15%) ยังไม่มีใครโหวต — reject เด็ดขาดจาก booster1 60% (TC-029)
	p7 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("mobile-learning"),
		Title: "Mobile Learning App แอปเรียนภาษาผ่านมือถือ", Description: &desc,
		State: domain.StateExecuting, Status: domain.StatusFunded, Visibility: domain.VisibilityPublic,
		FundingGoal: 10000, Softcap: 5000, CurrentFunding: 10000,
		DurationDays: 30, DurationMonths: 6, EndDate: now.AddDate(0, 0, -10), FundingAt: now.AddDate(0, -1, -10),
		ExecutionEndAt: ptrTime(now.AddDate(0, 1, 20)),
		ProfitSharePct: 10, MinInvestAmount: 100, MaxInvestAmount: 10000, PlatformFee: 5,
		Slug: seedSlugPrefix + "mobile-learning",
	}
	// P8: กำลังดำเนินการ — Phase1 submitted (TC-054), Phase2 waiting โดย Phase1 ยังไม่จ่าย (TC-048 negative)
	p8 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("solar-backpack"),
		Title: "Solar Backpack กระเป๋าเป้พลังงานแสงอาทิตย์", Description: &desc,
		State: domain.StateExecuting, Status: domain.StatusFunded, Visibility: domain.VisibilityPublic,
		FundingGoal: 50000, Softcap: 35000, CurrentFunding: 50000,
		DurationDays: 30, DurationMonths: 6, EndDate: now.AddDate(0, 0, -60), FundingAt: now.AddDate(0, -2, -60),
		ExecutionEndAt: ptrTime(now.AddDate(0, 1, 0)),
		ProfitSharePct: 10, MinInvestAmount: 500, MaxInvestAmount: 50000, PlatformFee: 5,
		Slug: seedSlugPrefix + "solar-backpack",
	}
	// P9: กำลังดำเนินการ — Phase1 paid+จ่ายแล้ว, Phase2 submitted — ให้แก้ไขหลักฐาน (TC-055)
	p9 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("recycle-bin"),
		Title: "Recycle Bin Sensor ถังขยะอัจฉริยะแยกประเภท", Description: &desc,
		State: domain.StateExecuting, Status: domain.StatusFunded, Visibility: domain.VisibilityPublic,
		FundingGoal: 40000, Softcap: 28000, CurrentFunding: 40000,
		DurationDays: 30, DurationMonths: 6, EndDate: now.AddDate(0, 0, -70), FundingAt: now.AddDate(0, -3, -70),
		ExecutionEndAt: ptrTime(now.AddDate(0, 1, 10)),
		ProfitSharePct: 10, MinInvestAmount: 400, MaxInvestAmount: 40000, PlatformFee: 5,
		Slug: seedSlugPrefix + "recycle-bin",
	}
	// P10: กำลังดำเนินการ — Phase1 paid+จ่ายแล้ว, Phase2 active (ยังไม่ส่ง) — ส่งหลักฐาน Milestone (TC-047)
	p10 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("water-sensor"),
		Title: "Water Quality Sensor เซนเซอร์วัดคุณภาพน้ำ", Description: &desc,
		State: domain.StateExecuting, Status: domain.StatusFunded, Visibility: domain.VisibilityPublic,
		FundingGoal: 60000, Softcap: 42000, CurrentFunding: 60000,
		DurationDays: 30, DurationMonths: 6, EndDate: now.AddDate(0, 0, -80), FundingAt: now.AddDate(0, -3, -80),
		ExecutionEndAt: ptrTime(now.AddDate(0, 1, 15)),
		ProfitSharePct: 10, MinInvestAmount: 600, MaxInvestAmount: 60000, PlatformFee: 5,
		Slug: seedSlugPrefix + "water-sensor",
	}
	// P11: กำลังดำเนินการ — Phase1 โหวตผ่านแล้ว (paid) แต่ Disbursement ยัง pending — ยืนยันปล่อยทุน (TC-056)
	p11 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("campus-locker"),
		Title: "Campus Locker ตู้ล็อกเกอร์อัจฉริยะ", Description: &desc,
		State: domain.StateExecuting, Status: domain.StatusFunded, Visibility: domain.VisibilityPublic,
		FundingGoal: 30000, Softcap: 21000, CurrentFunding: 30000,
		DurationDays: 30, DurationMonths: 6, EndDate: now.AddDate(0, 0, -20), FundingAt: now.AddDate(0, -1, -20),
		ExecutionEndAt: ptrTime(now.AddDate(0, 1, 25)),
		ProfitSharePct: 10, MinInvestAmount: 300, MaxInvestAmount: 30000, PlatformFee: 5,
		Slug: seedSlugPrefix + "campus-locker",
	}
	// P12: ฉบับร่าง (draft) ของ pioneer1 — แก้ไขโปรเจกต์ฉบับร่าง (TC-043)
	p12 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID,
		Title: "Smart Bicycle Lock กุญแจจักรยานอัจฉริยะ (ร่าง)", Description: &desc,
		State: domain.StateDraft, Status: domain.StatusActive, Visibility: domain.VisibilityPrivate,
		FundingGoal: 20000, Softcap: 14000, CurrentFunding: 0,
		DurationDays: 30, DurationMonths: 6,
		ProfitSharePct: 10, MinInvestAmount: 200, MaxInvestAmount: 20000, PlatformFee: 5,
		Slug: seedSlugPrefix + "draft-project",
	}
	// P13: รอตรวจสอบ (pending_review) ของ pioneer2 — admin อนุมัติ (TC-051)
	riskA := "ความเสี่ยงด้านการผลิตต้นแบบล่าช้ากว่าแผน"
	p13 := domain.Project{
		OwnerUserID: pioneer2.ID, CategoryID: &catIOT.ID, CoverImage: cover("noise-monitor"), Risk: &riskA,
		Title: "Noise Monitor เครื่องวัดเสียงรบกวนในหอพัก", Description: &desc,
		State: domain.StatePendingReview, Status: domain.StatusActive, Visibility: domain.VisibilityPrivate,
		FundingGoal: 25000, Softcap: 17500, CurrentFunding: 0,
		DurationDays: 30, DurationMonths: 6,
		ProfitSharePct: 10, MinInvestAmount: 250, MaxInvestAmount: 25000, PlatformFee: 5,
		Slug: seedSlugPrefix + "pending-review-a",
	}
	// P14: รอตรวจสอบ (pending_review) ของ pioneer2 — admin ปฏิเสธ (TC-052)
	riskB := "ความเสี่ยงด้านความแม่นยำของกล้องตรวจจับโรคพืช"
	p14 := domain.Project{
		OwnerUserID: pioneer2.ID, CategoryID: &catIOT.ID, CoverImage: cover("plant-cam"), Risk: &riskB,
		Title: "Plant Health Cam กล้องตรวจสุขภาพพืช", Description: &desc,
		State: domain.StatePendingReview, Status: domain.StatusActive, Visibility: domain.VisibilityPrivate,
		FundingGoal: 35000, Softcap: 24500, CurrentFunding: 0,
		DurationDays: 30, DurationMonths: 6,
		ProfitSharePct: 10, MinInvestAmount: 350, MaxInvestAmount: 35000, PlatformFee: 5,
		Slug: seedSlugPrefix + "pending-review-b",
	}
	// P15: เสร็จสมบูรณ์ (completed) ทุก Milestone จ่ายแล้ว — ส่งกำไร Q2 ใหม่ (TC-050) + มี Profit Pool Q1 pending รออนุมัติ (TC-060)
	p15 := domain.Project{
		OwnerUserID: pioneer1.ID, CategoryID: &catIOT.ID, CoverImage: cover("urban-farm"),
		Title: "Urban Farm Kit ชุดปลูกผักในเมือง", Description: &desc,
		State: domain.StateClosed, Status: domain.StatusCompleted, Visibility: domain.VisibilityPublic,
		FundingGoal: 60000, Softcap: 42000, CurrentFunding: 60000,
		DurationDays: 30, DurationMonths: 6, EndDate: now.AddDate(0, -3, 0), FundingAt: now.AddDate(0, -5, 0),
		ExecutionEndAt: ptrTime(now.AddDate(0, -1, 0)),
		ProfitSharePct: 10, MinInvestAmount: 600, MaxInvestAmount: 60000, PlatformFee: 5,
		Slug: seedSlugPrefix + "urban-farm",
	}

	allProjects := []*domain.Project{&p1, &p2, &p3, &p4, &p5, &p6, &p7, &p8, &p9, &p10, &p11, &p12, &p13, &p14, &p15}
	for _, p := range allProjects {
		if err := tx.Create(p).Error; err != nil {
			return fmt.Errorf("create project %s: %w", p.Slug, err)
		}
	}

	// media/story ให้ P13,P14 (จำเป็นต่อ validateProjectForSubmit ตอน admin approve/reject - TC-051,TC-052)
	for _, p := range []*domain.Project{&p13, &p14} {
		media := domain.ProjectMedia{ProjectID: p.ID, Type: []domain.MediaType{domain.MediaTypeImage}, URL: *cover(p.Slug), SortOrder: 1}
		if err := tx.Create(&media).Error; err != nil {
			return err
		}
		story := domain.StorySection{ProjectID: p.ID, Title: "เกี่ยวกับโปรเจกต์", Body: "seed: เนื้อหาโปรเจกต์สำหรับทดสอบคิวตรวจสอบของแอดมิน", SortOrder: 1}
		if err := tx.Create(&story).Error; err != nil {
			return err
		}
	}

	// ---------- Investments ----------
	// helper คำนวณตาม calculateFees ใน investment_service.go:
	// จ่าย amount -> fee 5% ของ amount -> VAT 7% ของ fee -> โปรเจกต์ได้ principal = amount - fee - vat
	// profitSharePct ของ investment = อัตราคงที่ของโปรเจกต์ (project.ProfitSharePct) ตามพฤติกรรมจริงของ CreateInvestment
	round2 := func(v float64) float64 { return math.Round(v*100) / 100 }
	mkInvest := func(ref string, projectID, boosterID uint, amount, profitSharePct float64, status domain.InvestmentStatus, paid bool) domain.Investment {
		fee := round2(amount * 0.05)
		vat := round2(fee * 0.07)
		inv := domain.Investment{
			ReferenceNumber: ref, ProjectID: projectID, BoosterUserID: boosterID,
			PrincipalAmount: round2(amount - fee - vat), PlatformFee: fee, VATAmount: vat,
			TotalAmount: amount, ProfitSharePct: profitSharePct, Status: status,
		}
		if paid {
			inv.PaidAt = &now
		}
		return inv
	}

	investments := []domain.Investment{
		// P2: ลงทุน 60/25/15% ของยอด 10,000 → จำลองโหวตถ่วงน้ำหนัก >51% (TC-026~028)
		mkInvest(seedRef("P2B1"), p2.ID, booster1.ID, 6000, p2.ProfitSharePct, domain.InvestmentVerified, true),
		mkInvest(seedRef("P2B2"), p2.ID, booster2.ID, 2500, p2.ProfitSharePct, domain.InvestmentVerified, true),
		mkInvest(seedRef("P2B3"), p2.ID, booster3.ID, 1500, p2.ProfitSharePct, domain.InvestmentVerified, true),
		// P1: booster1 ลงทุนในโปรเจกต์กำลังระดมทุน → ขอคืนเงิน / ยกเลิกโปรเจกต์ (TC-024,TC-046)
		mkInvest(seedRef("P1B1"), p1.ID, booster1.ID, 1000, p1.ProfitSharePct, domain.InvestmentVerified, true),
		// P3 (ล้มเหลว): booster1 ลงทุนไว้ → ปุ่มต้องเปลี่ยนเป็น "ขอเงินคืน" (TC-018)
		mkInvest(seedRef("P3B1"), p3.ID, booster1.ID, 2000, p3.ProfitSharePct, domain.InvestmentVerified, true),
		// P4: softcap ถึงแล้ว เหลือ ฿5,000 (TC-011)
		mkInvest(seedRef("P4B1"), p4.ID, booster1.ID, 5000, p4.ProfitSharePct, domain.InvestmentVerified, true),
		// P5: เหลือ ฿1,010 (TC-012)
		mkInvest(seedRef("P5B2"), p5.ID, booster2.ID, 98990, p5.ProfitSharePct, domain.InvestmentVerified, true),
		// P6: เหลือ ฿20 พอดี (TC-013)
		mkInvest(seedRef("P6B3"), p6.ID, booster3.ID, 9980, p6.ProfitSharePct, domain.InvestmentVerified, true),
		// P7: ลงทุน 60/25/15% เหมือน P2 → ใช้ทดสอบผลโหวต reject เด็ดขาด (TC-029)
		mkInvest(seedRef("P7B1"), p7.ID, booster1.ID, 6000, p7.ProfitSharePct, domain.InvestmentVerified, true),
		mkInvest(seedRef("P7B2"), p7.ID, booster2.ID, 2500, p7.ProfitSharePct, domain.InvestmentVerified, true),
		mkInvest(seedRef("P7B3"), p7.ID, booster3.ID, 1500, p7.ProfitSharePct, domain.InvestmentVerified, true),
		// P8-P11: 1 นักลงทุนต่อโปรเจกต์ (ให้ดูสมจริง, ไม่ใช่ precondition หลักของ TC)
		mkInvest(seedRef("P8B1"), p8.ID, booster1.ID, 50000, p8.ProfitSharePct, domain.InvestmentVerified, true),
		mkInvest(seedRef("P9B2"), p9.ID, booster2.ID, 40000, p9.ProfitSharePct, domain.InvestmentVerified, true),
		mkInvest(seedRef("P10B3"), p10.ID, booster3.ID, 60000, p10.ProfitSharePct, domain.InvestmentVerified, true),
		mkInvest(seedRef("P11B1"), p11.ID, booster1.ID, 30000, p11.ProfitSharePct, domain.InvestmentVerified, true),
		// P15: completed project — สัดส่วนสำหรับแบ่งกำไร 66.67%/33.33% (TC-050,TC-060)
		mkInvest(seedRef("P15B1"), p15.ID, booster1.ID, 40000, p15.ProfitSharePct, domain.InvestmentVerified, true),
		mkInvest(seedRef("P15B2"), p15.ID, booster2.ID, 20000, p15.ProfitSharePct, domain.InvestmentVerified, true),
	}
	for i := range investments {
		if err := tx.Create(&investments[i]).Error; err != nil {
			return fmt.Errorf("create investment %s: %w", investments[i].ReferenceNumber, err)
		}
	}

	// ---------- Milestones ----------
	crit := "ส่งมอบตามเกณฑ์ที่ตกลง"
	summary := "พัฒนาเสร็จตามแผน แนบหลักฐาน GitHub"
	ghLink := []string{"https://github.com/SundayYogurt/FlyUps-backend"}

	// P2: Phase1 จ่ายแล้ว(ปิดโหวต), Phase2 approved+เปิดโหวตอยู่ (TC-026~028,031,032), Phase3 approved รอเปิดโหวต (TC-049)
	p2Closed := now.AddDate(0, 0, -20)
	p2m1 := domain.Milestone{
		ProjectID: p2.ID, PhaseNo: 1, Title: "Phase 1: ออกแบบระบบ",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestonePaid, PercentRelease: 40, SortOrder: 1,
		VotingOpen: false, VotingOpenedAt: &p2Closed, VotingClosedAt: &now, SubmittedAt: &p2Closed,
	}
	p2m2 := domain.Milestone{
		ProjectID: p2.ID, PhaseNo: 2, Title: "Phase 2: พัฒนา MVP",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestoneApproved, PercentRelease: 40, SortOrder: 2,
		VotingOpen: true, VotingOpenedAt: &now, SubmittedAt: &now,
	}
	p2m3 := domain.Milestone{
		ProjectID: p2.ID, PhaseNo: 3, Title: "Phase 3: เปิดใช้งานจริง",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestoneApproved, PercentRelease: 20, SortOrder: 3, SubmittedAt: &now,
	}
	// P7: Phase1 approved+เปิดโหวตใหม่ ยังไม่มีใครโหวต (TC-029)
	p7m1 := domain.Milestone{
		ProjectID: p7.ID, PhaseNo: 1, Title: "Phase 1: พัฒนาแอปต้นแบบ",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestoneApproved, PercentRelease: 100, SortOrder: 1,
		VotingOpen: true, VotingOpenedAt: &now, SubmittedAt: &now,
	}
	// P8: Phase1 submitted (TC-054), Phase2 waiting - Phase1 ยังไม่จ่าย จึงส่งไม่ได้ (TC-048)
	p8m1 := domain.Milestone{
		ProjectID: p8.ID, PhaseNo: 1, Title: "Phase 1: ออกแบบวงจรโซลาร์",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestoneSubmitted, PercentRelease: 50, SortOrder: 1, SubmittedAt: &now,
	}
	// หมายเหตุ: ตั้ง Phase2 เป็น Active (ไม่ใช่ Waiting) เพื่อให้ปุ่ม "ส่งหลักฐาน" กดได้จริง
	// แล้วเจอ error "previous milestone must be paid" จาก SubmitMilestone (ไม่ใช่ error สถานะ milestone ผิด)
	p8m2 := domain.Milestone{
		ProjectID: p8.ID, PhaseNo: 2, Title: "Phase 2: ผลิตต้นแบบ",
		AcceptanceCriteria: &crit,
		Status:             domain.MilestoneActive, PercentRelease: 50, SortOrder: 2,
	}
	// P9: Phase1 จ่ายแล้ว, Phase2 submitted - แอดมินให้แก้ไข (TC-055)
	p9Closed := now.AddDate(0, 0, -25)
	p9m1 := domain.Milestone{
		ProjectID: p9.ID, PhaseNo: 1, Title: "Phase 1: ออกแบบเซนเซอร์แยกขยะ",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestonePaid, PercentRelease: 50, SortOrder: 1,
		VotingOpen: false, VotingOpenedAt: &p9Closed, VotingClosedAt: &p9Closed, SubmittedAt: &p9Closed,
	}
	p9m2 := domain.Milestone{
		ProjectID: p9.ID, PhaseNo: 2, Title: "Phase 2: ผลิตและทดสอบ",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestoneSubmitted, PercentRelease: 50, SortOrder: 2, SubmittedAt: &now,
	}
	// P10: Phase1 จ่ายแล้ว, Phase2 active (ยังไม่ส่ง) - ส่งหลักฐาน Milestone (TC-047)
	p10Closed := now.AddDate(0, 0, -32)
	p10m1 := domain.Milestone{
		ProjectID: p10.ID, PhaseNo: 1, Title: "Phase 1: ออกแบบเซนเซอร์วัดคุณภาพน้ำ",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestonePaid, PercentRelease: 50, SortOrder: 1,
		VotingOpen: false, VotingOpenedAt: &p10Closed, VotingClosedAt: &p10Closed, SubmittedAt: &p10Closed,
	}
	p10m2 := domain.Milestone{
		ProjectID: p10.ID, PhaseNo: 2, Title: "Phase 2: ติดตั้งและเก็บข้อมูลจริง",
		AcceptanceCriteria: &crit,
		Status:             domain.MilestoneActive, PercentRelease: 50, SortOrder: 2,
	}
	// P11: Phase1 โหวตผ่านแล้ว (paid) แต่ยังไม่ยืนยันปล่อยทุน (TC-056)
	p11Closed := now.AddDate(0, 0, -3)
	p11m1 := domain.Milestone{
		ProjectID: p11.ID, PhaseNo: 1, Title: "Phase 1: ติดตั้งตู้ล็อกเกอร์ต้นแบบ",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestonePaid, PercentRelease: 100, SortOrder: 1,
		VotingOpen: false, VotingOpenedAt: &p11Closed, VotingClosedAt: &now, SubmittedAt: &p11Closed,
	}
	// P15: ครบทุก Phase จ่ายแล้ว (precondition UC16)
	p15Closed := now.AddDate(0, -2, 0)
	p15m1 := domain.Milestone{
		ProjectID: p15.ID, PhaseNo: 1, Title: "Phase 1: ออกแบบระบบปลูกผัก",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestonePaid, PercentRelease: 40, SortOrder: 1,
		VotingOpen: false, VotingOpenedAt: &p15Closed, VotingClosedAt: &p15Closed, SubmittedAt: &p15Closed,
	}
	p15m2 := domain.Milestone{
		ProjectID: p15.ID, PhaseNo: 2, Title: "Phase 2: ผลิตและติดตั้งชุดปลูก",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestonePaid, PercentRelease: 30, SortOrder: 2,
		VotingOpen: false, VotingOpenedAt: &p15Closed, VotingClosedAt: &p15Closed, SubmittedAt: &p15Closed,
	}
	p15m3 := domain.Milestone{
		ProjectID: p15.ID, PhaseNo: 3, Title: "Phase 3: ส่งมอบและอบรมการใช้งาน",
		AcceptanceCriteria: &crit, SubmissionSummary: &summary, SubmissionLinks: ghLink,
		Status: domain.MilestonePaid, PercentRelease: 30, SortOrder: 3,
		VotingOpen: false, VotingOpenedAt: &p15Closed, VotingClosedAt: &p15Closed, SubmittedAt: &p15Closed,
	}
	// P13,P14: draft milestones ครบตามจำนวนเฟส (precondition ก่อน submit - จำเป็นสำหรับ validateProjectForSubmit ตอน approve/reject)
	p13m1 := domain.Milestone{ProjectID: p13.ID, PhaseNo: 1, Title: "Phase 1: ออกแบบวงจร", AcceptanceCriteria: &crit, Status: domain.MilestoneDraft, PercentRelease: 40, SortOrder: 1}
	p13m2 := domain.Milestone{ProjectID: p13.ID, PhaseNo: 2, Title: "Phase 2: ผลิตต้นแบบ", AcceptanceCriteria: &crit, Status: domain.MilestoneDraft, PercentRelease: 30, SortOrder: 2}
	p13m3 := domain.Milestone{ProjectID: p13.ID, PhaseNo: 3, Title: "Phase 3: ทดสอบภาคสนาม", AcceptanceCriteria: &crit, Status: domain.MilestoneDraft, PercentRelease: 30, SortOrder: 3}
	p14m1 := domain.Milestone{ProjectID: p14.ID, PhaseNo: 1, Title: "Phase 1: พัฒนาโมเดล AI", AcceptanceCriteria: &crit, Status: domain.MilestoneDraft, PercentRelease: 50, SortOrder: 1}
	p14m2 := domain.Milestone{ProjectID: p14.ID, PhaseNo: 2, Title: "Phase 2: ทดสอบภาคสนาม", AcceptanceCriteria: &crit, Status: domain.MilestoneDraft, PercentRelease: 50, SortOrder: 2}

	allMilestones := []*domain.Milestone{
		&p2m1, &p2m2, &p2m3,
		&p7m1,
		&p8m1, &p8m2,
		&p9m1, &p9m2,
		&p10m1, &p10m2,
		&p11m1,
		&p15m1, &p15m2, &p15m3,
		&p13m1, &p13m2, &p13m3,
		&p14m1, &p14m2,
	}
	for _, m := range allMilestones {
		if err := tx.Create(m).Error; err != nil {
			return fmt.Errorf("create milestone project=%d phase=%d: %w", m.ProjectID, m.PhaseNo, err)
		}
	}

	// ผลโหวตของ P2 Phase 1 (ที่ปิดไปแล้ว) — ดูผลโหวตย้อนหลังได้ แต่โหวตเพิ่มไม่ได้
	votes := []domain.MilestoneVote{
		{MilestoneID: p2m1.ID, ProjectID: p2.ID, BoosterUserID: booster1.ID, Choice: domain.MilestoneVoteApprove},
		{MilestoneID: p2m1.ID, ProjectID: p2.ID, BoosterUserID: booster2.ID, Choice: domain.MilestoneVoteApprove},
		{MilestoneID: p2m1.ID, ProjectID: p2.ID, BoosterUserID: booster3.ID, Choice: domain.MilestoneVoteReject},
		// P11 Phase1: โหวตผ่านแล้ว (สอดคล้องกับสถานะ paid)
		{MilestoneID: p11m1.ID, ProjectID: p11.ID, BoosterUserID: booster1.ID, Choice: domain.MilestoneVoteApprove},
	}
	for i := range votes {
		if err := tx.Create(&votes[i]).Error; err != nil {
			return err
		}
	}

	// ---------- Meetings ----------
	// P2 Phase2: นัดประชุมเปิดโหวต (TC-031 เข้าร่วมประชุม, TC-032 booster_new เข้าไม่ได้)
	meetLink := "https://meet.example.com/seed-ai-chatbot-phase2"
	meetAbout := "นำเสนอผลงาน Phase 2 และเปิดให้นักลงทุนโหวต"
	p2Meeting := domain.Meeting{
		MilestoneID: p2m2.ID, Date: now.AddDate(0, 0, 3), Time: now,
		MeetingType: domain.Online, Link: &meetLink, About: meetAbout, Status: domain.MeetingOpen,
	}
	// P7 Phase1: นัดประชุมคู่กับโหวต reject (ให้สอดคล้องกับ VotingOpen=true)
	meetLink7 := "https://meet.example.com/seed-mobile-learning-phase1"
	p7Meeting := domain.Meeting{
		MilestoneID: p7m1.ID, Date: now.AddDate(0, 0, 2), Time: now,
		MeetingType: domain.Online, Link: &meetLink7, About: "นำเสนอผลงาน Phase 1 และเปิดให้นักลงทุนโหวต", Status: domain.MeetingOpen,
	}
	for _, mt := range []*domain.Meeting{&p2Meeting, &p7Meeting} {
		if err := tx.Create(mt).Error; err != nil {
			return err
		}
	}

	// ---------- Disbursements ----------
	mkDisbursement := func(m *domain.Milestone, projectID, pioneerID uint, principalTotal float64, confirmed bool) domain.Disbursement {
		amount := round2(principalTotal * float64(m.PercentRelease) / 100)
		d := domain.Disbursement{
			MilestoneID: m.ID, ProjectID: projectID, PioneerUserID: pioneerID,
			Amount: amount, PhaseNo: m.PhaseNo, PercentRelease: m.PercentRelease,
			Status: domain.DisbursementPending, TransferRef: seedRef(fmt.Sprintf("DISB%d", m.ID)),
		}
		if confirmed {
			d.Status = domain.DisbursementConfirmed
			d.ConfirmedAt = &now
			d.ConfirmedBy = &admin.ID
		}
		return d
	}
	disbursements := []domain.Disbursement{
		mkDisbursement(&p2m1, p2.ID, pioneer1.ID, 9465, true),     // P2 Phase1 จ่ายแล้ว (principal รวม ~9,465 จาก 10,000 หลังหักค่าธรรมเนียม)
		mkDisbursement(&p9m1, p9.ID, pioneer1.ID, 37860, true),    // P9 Phase1 จ่ายแล้ว
		mkDisbursement(&p10m1, p10.ID, pioneer1.ID, 56790, true),  // P10 Phase1 จ่ายแล้ว
		mkDisbursement(&p11m1, p11.ID, pioneer1.ID, 28395, false), // P11 Phase1 โหวตผ่านแล้ว รอแอดมินยืนยันโอนเงิน (TC-056)
		mkDisbursement(&p15m1, p15.ID, pioneer1.ID, 56790, true),
		mkDisbursement(&p15m2, p15.ID, pioneer1.ID, 56790, true),
		mkDisbursement(&p15m3, p15.ID, pioneer1.ID, 56790, true),
	}
	for i := range disbursements {
		if err := tx.Create(&disbursements[i]).Error; err != nil {
			return err
		}
	}

	// ---------- ProfitPool + Payout ----------
	// P2: ไตรมาส 1 จ่ายแล้วครบ (TC-023 ดูประวัติกำไร)
	poolP2 := domain.ProfitPool{
		ProjectID: p2.ID, PioneerUserID: pioneer1.ID, TotalAmount: 1000,
		TransferRef: seedRef("POOL-P2"), Status: domain.ProfitPoolCompleted, QuarterNo: 1,
	}
	if err := tx.Create(&poolP2).Error; err != nil {
		return err
	}
	payoutP2 := domain.InvestorProfitPayout{
		ProfitPoolID: poolP2.ID, ProjectID: p2.ID, BoosterUserID: booster1.ID,
		Amount: 600, SharePct: 6.0, Status: domain.InvestorPayoutConfirmed,
		TransferRef: seedRef("PAY-P2B1"), ConfirmedAt: &now, ConfirmedBy: &admin.ID,
	}
	if err := tx.Create(&payoutP2).Error; err != nil {
		return err
	}
	// P15: ไตรมาส 1 pioneer ส่งมาแล้ว รอแอดมินตรวจสอบ/จ่ายให้นักลงทุน (TC-060) — Q2 ยังว่าง ใช้ทดสอบ pioneer ส่งใหม่ (TC-050)
	poolP15 := domain.ProfitPool{
		ProjectID: p15.ID, PioneerUserID: pioneer1.ID, TotalAmount: 2000, SlipImage: "seed-slip.jpg",
		TransferRef: seedRef("POOL-P15"), Status: domain.ProfitPoolPending, QuarterNo: 1,
	}
	if err := tx.Create(&poolP15).Error; err != nil {
		return err
	}
	payoutsP15 := []domain.InvestorProfitPayout{
		{ProfitPoolID: poolP15.ID, ProjectID: p15.ID, BoosterUserID: booster1.ID, Amount: 1333.33, SharePct: 66.67, Status: domain.InvestorPayoutPending},
		{ProfitPoolID: poolP15.ID, ProjectID: p15.ID, BoosterUserID: booster2.ID, Amount: 666.67, SharePct: 33.33, Status: domain.InvestorPayoutPending},
	}
	for i := range payoutsP15 {
		if err := tx.Create(&payoutsP15[i]).Error; err != nil {
			return err
		}
	}

	// ---------- Complaints (TC-020~022,057) ----------
	// P1 มี 3 คำร้องเรียนเปิดอยู่จาก booster ต่างกัน — booster2 ร้องเรียนซ้ำไม่ได้ (TC-022)
	// แอดมิน resolve ครบ 3 เรื่อง → โปรเจกต์ถูกระงับอัตโนมัติ (TC-057)
	complaints := []domain.Complaint{
		{ComplainantID: booster2.ID, ProjectID: p1.ID, Subject: "โปรเจกต์ไม่อัปเดตความคืบหน้า", Body: "seed: ใช้ทดสอบเคสร้องเรียนซ้ำ (TC-022)", Status: domain.ComplaintOpen},
		{ComplainantID: booster1.ID, ProjectID: p1.ID, Subject: "หลักฐานความคืบหน้าดูไม่น่าเชื่อถือ", Body: "seed: ใช้ทดสอบ auto-suspend (TC-057)", Status: domain.ComplaintOpen},
		{ComplainantID: booster3.ID, ProjectID: p1.ID, Subject: "ผู้สร้างไม่ตอบคำถามในกลุ่มแชท", Body: "seed: ใช้ทดสอบ auto-suspend (TC-057)", Status: domain.ComplaintOpen},
	}
	for i := range complaints {
		if err := tx.Create(&complaints[i]).Error; err != nil {
			return err
		}
	}

	// ---------- Notifications (TC-040) ----------
	notifRelMilestone := "milestone"
	notifRelPayout := "investor_payout"
	notifications := []domain.Notification{
		{UserID: booster1.ID, Type: domain.NotifMilestone, Title: "Milestone เปิดโหวต", Body: "Phase 2: พัฒนา MVP เปิดให้โหวตแล้ว กรุณาลงคะแนนภายในกำหนด", IsRead: false, RelatedID: &p2m2.ID, RelatedType: &notifRelMilestone},
		{UserID: booster1.ID, Type: domain.NotifProfit, Title: "ได้รับกำไรจากโปรเจกต์", Body: "โอนกำไรจากโปรเจกต์ AI Chatbot ช่วยติวสอบ จำนวน ฿600.00 เรียบร้อยแล้ว", IsRead: true, RelatedID: &payoutP2.ID, RelatedType: &notifRelPayout},
	}
	for i := range notifications {
		if err := tx.Create(&notifications[i]).Error; err != nil {
			return err
		}
	}

	return nil
}

func ptrTime(t time.Time) *time.Time { return &t }

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
booster1@seed.flyup.test     Booster ยืนยันแล้ว | ลงทุน P1,P2(60%),P3,P4,P7(60%),P8,P11,P15 | ได้ปันผล P2 Q1 600บ.
booster2@seed.flyup.test     Booster ยืนยันแล้ว | ลงทุน P2(25%),P5,P7(25%),P9,P15 | เคยร้องเรียน P1 แล้ว
booster3@seed.flyup.test     Booster ยืนยันแล้ว | ลงทุน P2(15%),P6,P7(15%),P10
booster_new@seed.flyup.test  Booster ยังไม่ยืนยันตัวตน ไม่มีประวัติลงทุน (TC-015,019,030,032; เป้าหมาย TC-058 suspend/reactivate)
pioneer1@seed.flyup.test     Pioneer ยืนยันตัวตนครบ (IdCard+StudentCard+บัญชีธนาคาร) | เจ้าของ P1-P12,P15
pioneer2@seed.flyup.test     Pioneer ยืนยันตัวตนครบ | ไม่ใช่เจ้าของ P1/P2 (TC-014) | เจ้าของ P13,P14 (คิวรอตรวจสอบ)

PROJECTS
seed-smart-farm         funding    Smart Farm IoT                 → UC1,UC2,UC4,UC6,TC-024,TC-044,TC-046,TC-057(3 complaints)
seed-ai-chatbot         executing  AI Chatbot ช่วยติวสอบ           → UC5,UC7,UC8,UC9(Phase2 เปิดโหวต+ประชุม),UC18(Phase3 รอเปิดโหวต),TC-045
seed-drone-delivery     closed     Drone ส่งของ (ล้มเหลว)          → TC-018
seed-eco-bottle         funding    Eco Bottle (softcap ถึงแล้ว)     → TC-011 (เหลือ ฿5,000)
seed-paper-straw        funding    Paper Straw                    → TC-012 (เหลือ ฿1,010 - เศษต้องห้าม)
seed-bike-share         funding    Bike Share                     → TC-013 (เหลือ ฿20 พอดี - ปิดยอดได้)
seed-mobile-learning    executing  Mobile Learning App             → TC-029 (โหวต reject 60% เด็ดขาด + ประชุม)
seed-solar-backpack     executing  Solar Backpack                 → TC-054 (Phase1 submitted), TC-048 (Phase2 ส่งไม่ได้)
seed-recycle-bin        executing  Recycle Bin Sensor              → TC-055 (Phase2 submitted - ให้แก้ไข)
seed-water-sensor       executing  Water Quality Sensor            → TC-047 (Phase2 active - ส่งหลักฐานได้)
seed-campus-locker      executing  Campus Locker                  → TC-056 (Phase1 paid, disbursement pending)
seed-draft-project      draft      Smart Bicycle Lock (ร่าง)        → TC-043
seed-pending-review-a   pending_review Noise Monitor              → TC-051 (อนุมัติ)
seed-pending-review-b   pending_review Plant Health Cam            → TC-052 (ปฏิเสธ)
seed-urban-farm         closed/completed Urban Farm Kit            → TC-050 (ส่งกำไร Q2 ใหม่), TC-060 (Q1 pending รออนุมัติ)
หมวดหมู่ "Web Application" ว่างเปล่า (ไม่มีโปรเจกต์ funding ใดๆ) → TC-006,TC-007
===================================================================`)
}

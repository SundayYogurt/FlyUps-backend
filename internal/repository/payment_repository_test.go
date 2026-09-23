package repository

import (
	"testing"
	"time"

	"flyup/internal/domain"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func paymentTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, schema := range []string{
		`CREATE TABLE projects (id integer primary key, owner_user_id integer, title text, state text, status text, funding_goal real, current_funding real, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE investments (id integer primary key, project_id integer, status text, total_amount real, platform_fee real, vat_amount real, principal_amount real, paid_at datetime, refund_amount real, refund_note text, refunded_at datetime, created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE transactions (id integer primary key, investment_id integer, stripe_payment_intent_id text, status text, expires_at datetime, stripe_fee real, stripe_fee_vat real, net_amount real, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE milestones (id integer primary key, project_id integer, phase_no integer, status text, updated_at datetime, deleted_at datetime)`,
	} {
		if err := db.Exec(schema).Error; err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { sqlDB, _ := db.DB(); _ = sqlDB.Close() })
	return db
}

func seedPayment(t *testing.T, db *gorm.DB, current, goal float64) {
	t.Helper()
	now := time.Now()
	for _, stmt := range []struct {
		sql  string
		args []interface{}
	}{
		{`INSERT INTO projects (id, owner_user_id, title, state, status, funding_goal, current_funding) VALUES (1, 10, 'P', 'funding', 'active', ?, ?)`, []interface{}{goal, current}},
		{`INSERT INTO investments (id, project_id, status, total_amount, platform_fee, vat_amount, principal_amount, created_at) VALUES (1, 1, 'pending_payment', 1000, 10, 0.7, 989.3, ?)`, []interface{}{now}},
		{`INSERT INTO transactions (id, investment_id, stripe_payment_intent_id, status, expires_at) VALUES (1, 1, 'pi_1', 'pending', ?)`, []interface{}{now.Add(5 * time.Minute)}},
		{`INSERT INTO milestones (id, project_id, phase_no, status) VALUES (1, 1, 1, 'waiting')`, nil},
	} {
		if err := db.Exec(stmt.sql, stmt.args...).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestCompletePaymentOnce(t *testing.T) {
	db := paymentTestDB(t)
	seedPayment(t, db, 0, 1000)
	repo := NewPaymentRepository(db)
	first, err := repo.Complete("pi_1", 15, 1.05, 983.95)
	if err != nil || !first.Applied || first.Overfund {
		t.Fatalf("first completion: %+v, %v", first, err)
	}
	second, err := repo.Complete("pi_1", 15, 1.05, 983.95)
	if err != nil || second.Applied {
		t.Fatalf("duplicate completion: %+v, %v", second, err)
	}
	var project domain.Project
	var investment domain.Investment
	var payment domain.Transaction
	var milestone domain.Milestone
	db.First(&project, 1)
	db.First(&investment, 1)
	db.First(&payment, 1)
	db.First(&milestone, 1)
	if project.CurrentFunding != 1000 || project.State != domain.StateExecuting || investment.Status != domain.InvestmentVerified || payment.Status != domain.TransactionSucceeded || milestone.Status != domain.MilestoneActive {
		t.Fatalf("unexpected committed state: project=%+v investment=%+v payment=%+v milestone=%+v", project, investment, payment, milestone)
	}
}

func TestCompletePaymentRollsBack(t *testing.T) {
	for _, table := range []string{"investments", "projects", "milestones"} {
		t.Run(table, func(t *testing.T) {
			db := paymentTestDB(t)
			seedPayment(t, db, 0, 1000)
			if err := db.Exec("CREATE TRIGGER reject_update BEFORE UPDATE ON " + table + " BEGIN SELECT RAISE(FAIL, 'injected failure'); END").Error; err != nil {
				t.Fatal(err)
			}
			if _, err := NewPaymentRepository(db).Complete("pi_1", 0, 0, 0); err == nil {
				t.Fatal("expected database error")
			}
			var project domain.Project
			var investment domain.Investment
			var payment domain.Transaction
			db.First(&project, 1)
			db.First(&investment, 1)
			db.First(&payment, 1)
			if project.CurrentFunding != 0 || investment.Status != domain.InvestmentPending || payment.Status != domain.TransactionPending {
				t.Fatalf("partial payment persisted: funding=%v investment=%s payment=%s", project.CurrentFunding, investment.Status, payment.Status)
			}
		})
	}
}

func TestCompletePaymentOverfundCreatesFullRefund(t *testing.T) {
	db := paymentTestDB(t)
	seedPayment(t, db, 1000, 1000)
	result, err := NewPaymentRepository(db).Complete("pi_1", 0, 0, 0)
	if err != nil || !result.Applied || !result.Overfund {
		t.Fatalf("overfund completion: %+v, %v", result, err)
	}
	var project domain.Project
	var investment domain.Investment
	db.First(&project, 1)
	db.First(&investment, 1)
	if project.CurrentFunding != 1000 || investment.Status != domain.InvestmentOverfundRefundPending || investment.RefundAmount != 1000 {
		t.Fatalf("overfund state: funding=%v status=%s refund=%v", project.CurrentFunding, investment.Status, investment.RefundAmount)
	}
	queue, err := NewInvestmentRepository(db).ListRefundPending()
	if err != nil || len(queue) != 1 || queue[0].ID != investment.ID {
		t.Fatalf("overfund payment missing from refund queue: %+v, %v", queue, err)
	}
}

func TestPendingQRReservesCapacityUntilExpiry(t *testing.T) {
	db := paymentTestDB(t)
	seedPayment(t, db, 0, 1000)
	now := time.Now()
	reserved, err := SumReservedByProjectIDTx(db, 1, now, 0)
	if err != nil || reserved != 1000 {
		t.Fatalf("active QR reservation: %v, %v", reserved, err)
	}
	reserved, err = SumReservedByProjectIDTx(db, 1, now.Add(6*time.Minute), 0)
	if err != nil || reserved != 0 {
		t.Fatalf("expired QR reservation: %v, %v", reserved, err)
	}
	if err := db.Exec(`INSERT INTO investments (id, project_id, status, total_amount, created_at) VALUES (2, 1, 'pending_payment', 500, ?)`, now).Error; err != nil {
		t.Fatal(err)
	}
	reserved, err = SumReservedByProjectIDTx(db, 1, now, 0)
	if err != nil || reserved != 1500 {
		t.Fatalf("QR and in-flight reservation: %v, %v", reserved, err)
	}
	reserved, err = SumReservedByProjectIDTx(db, 1, now, 1)
	if err != nil || reserved != 500 {
		t.Fatalf("exclude current payment: %v, %v", reserved, err)
	}
}

func TestFailedEventCannotUndoSuccessfulPayment(t *testing.T) {
	db := paymentTestDB(t)
	seedPayment(t, db, 0, 1000)
	repo := NewPaymentRepository(db)
	if _, err := repo.Complete("pi_1", 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	result, err := repo.Fail("pi_1")
	if err != nil || result.Applied {
		t.Fatalf("late failed event: %+v, %v", result, err)
	}
	var investment domain.Investment
	db.First(&investment, 1)
	if investment.Status != domain.InvestmentVerified {
		t.Fatalf("late failure changed investment: %s", investment.Status)
	}
}

func TestFailedPaymentRollsBackBothStatuses(t *testing.T) {
	db := paymentTestDB(t)
	seedPayment(t, db, 0, 1000)
	if err := db.Exec("CREATE TRIGGER reject_failure BEFORE UPDATE ON investments BEGIN SELECT RAISE(FAIL, 'injected failure'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewPaymentRepository(db).Fail("pi_1"); err == nil {
		t.Fatal("expected database error")
	}
	var investment domain.Investment
	var payment domain.Transaction
	db.First(&investment, 1)
	db.First(&payment, 1)
	if investment.Status != domain.InvestmentPending || payment.Status != domain.TransactionPending {
		t.Fatalf("partial failure persisted: investment=%s payment=%s", investment.Status, payment.Status)
	}
}

func TestMissingPaymentReturnsRetryableError(t *testing.T) {
	db := paymentTestDB(t)
	repo := NewPaymentRepository(db)
	if _, err := repo.Complete("pi_missing", 0, 0, 0); err == nil {
		t.Fatal("missing succeeded payment was acknowledged")
	}
	if _, err := repo.Fail("pi_missing"); err == nil {
		t.Fatal("missing failed payment was acknowledged")
	}
}

func TestExpiryDoesNotOverwriteConfirmedPayment(t *testing.T) {
	db := paymentTestDB(t)
	seedPayment(t, db, 0, 1000)
	repo := NewPaymentRepository(db)
	if _, err := repo.Complete("pi_1", 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	expired, err := repo.Expire(1, time.Now().Add(10*time.Minute))
	if err != nil || expired {
		t.Fatalf("paid QR expired: expired=%v err=%v", expired, err)
	}
	var investment domain.Investment
	db.First(&investment, 1)
	if investment.Status != domain.InvestmentVerified {
		t.Fatalf("confirmed investment changed to %s", investment.Status)
	}
}

func TestExpiredQRCanStillBeReconciledIfStripeCapturedIt(t *testing.T) {
	db := paymentTestDB(t)
	seedPayment(t, db, 0, 1000)
	repo := NewPaymentRepository(db)
	expired, err := repo.Expire(1, time.Now().Add(10*time.Minute))
	if err != nil || !expired {
		t.Fatalf("QR did not expire: expired=%v err=%v", expired, err)
	}
	result, err := repo.Complete("pi_1", 0, 0, 0)
	if err != nil || !result.Applied || result.Overfund {
		t.Fatalf("late Stripe success: %+v, %v", result, err)
	}
	var project domain.Project
	db.First(&project, 1)
	if project.CurrentFunding != 1000 {
		t.Fatalf("late paid QR not counted: %v", project.CurrentFunding)
	}
}

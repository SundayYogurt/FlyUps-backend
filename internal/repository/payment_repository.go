package repository

import (
	"errors"
	"fmt"
	"math"
	"time"

	"flyup/internal/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PaymentResult describes the committed effect of a Stripe event.
type PaymentResult struct {
	Investment domain.Investment
	Project    domain.Project
	Applied    bool
	Overfund   bool
}

type PaymentRepository interface {
	Complete(intentID string, stripeFee, stripeFeeVAT, netAmount float64) (PaymentResult, error)
	Fail(intentID string) (PaymentResult, error)
	Expire(investmentID uint, now time.Time) (bool, error)
}

type paymentRepository struct{ db *gorm.DB }

func NewPaymentRepository(db *gorm.DB) PaymentRepository { return &paymentRepository{db: db} }

// SumReservedByProjectIDTx is called while the project row is locked. A new
// investment has no transaction while Stripe creates its QR, so reserve that
// short interval too. Failed/expired QR codes release their reservation.
func SumReservedByProjectIDTx(tx *gorm.DB, projectID uint, now time.Time, excludeInvestmentID uint) (float64, error) {
	var reserved float64
	err := tx.Model(&domain.Investment{}).
		Joins("LEFT JOIN transactions ON transactions.investment_id = investments.id AND transactions.deleted_at IS NULL").
		Where("investments.project_id = ? AND investments.status = ? AND investments.id <> ?", projectID, domain.InvestmentPending, excludeInvestmentID).
		Where("((transactions.id IS NULL AND investments.created_at > ?) OR (transactions.status = ? AND transactions.expires_at > ?))", now.Add(-10*time.Minute), domain.TransactionPending, now).
		Select("COALESCE(SUM(investments.total_amount), 0)").Scan(&reserved).Error
	return reserved, err
}

func (r *paymentRepository) Complete(intentID string, stripeFee, stripeFeeVAT, netAmount float64) (PaymentResult, error) {
	var result PaymentResult
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Find the project before taking locks; all funding paths lock the project
		// first, then re-read the payment rows under that lock.
		var locator domain.Transaction
		if err := tx.Where("stripe_payment_intent_id = ?", intentID).First(&locator).Error; err != nil {
			return fmt.Errorf("find transaction for intent %s: %w", intentID, err)
		}
		var investmentLocator domain.Investment
		if err := tx.First(&investmentLocator, locator.InvestmentID).Error; err != nil {
			return fmt.Errorf("find investment for intent %s: %w", intentID, err)
		}
		var project domain.Project
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&project, investmentLocator.ProjectID).Error; err != nil {
			return fmt.Errorf("lock project for intent %s: %w", intentID, err)
		}
		var payment domain.Transaction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&payment, locator.ID).Error; err != nil {
			return fmt.Errorf("lock transaction for intent %s: %w", intentID, err)
		}
		var investment domain.Investment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&investment, payment.InvestmentID).Error; err != nil {
			return fmt.Errorf("lock investment for intent %s: %w", intentID, err)
		}
		if payment.Status == domain.TransactionSucceeded && (investment.Status == domain.InvestmentVerified || investment.Status == domain.InvestmentOverfundRefundPending || investment.Status == domain.InvestmentRefundPending || investment.Status == domain.InvestmentRefunded) {
			return nil
		}
		if investment.Status == domain.InvestmentVerified || investment.Status == domain.InvestmentOverfundRefundPending || investment.Status == domain.InvestmentRefundPending || investment.Status == domain.InvestmentRefunded {
			return errors.New("payment state conflicts with investment state")
		}

		reserved, err := SumReservedByProjectIDTx(tx, project.ID, time.Now(), investment.ID)
		if err != nil {
			return fmt.Errorf("sum project reservations: %w", err)
		}
		fits := project.State == domain.StateFunding && project.CurrentFunding+reserved+investment.TotalAmount <= project.FundingGoal+0.005
		now := time.Now()
		payment.Status = domain.TransactionSucceeded
		payment.StripeFee, payment.StripeFeeVAT, payment.NetAmount = stripeFee, stripeFeeVAT, netAmount
		if err := tx.Model(&payment).Updates(map[string]interface{}{
			"status": payment.Status, "stripe_fee": stripeFee, "stripe_fee_vat": stripeFeeVAT, "net_amount": netAmount,
		}).Error; err != nil {
			return fmt.Errorf("update transaction: %w", err)
		}
		investment.PaidAt = &now
		if netAmount > 0 {
			investment.PrincipalAmount = math.Round((netAmount-investment.PlatformFee-investment.VATAmount)*100) / 100
		}
		if fits {
			investment.Status = domain.InvestmentVerified
		} else {
			investment.Status = domain.InvestmentOverfundRefundPending
			investment.RefundAmount = investment.TotalAmount
			investment.RefundNote = "Payment exceeded available project funding; full refund required"
			investment.RefundedAt = &now
		}
		if err := tx.Model(&investment).Updates(map[string]interface{}{
			"status": investment.Status, "paid_at": investment.PaidAt, "principal_amount": investment.PrincipalAmount,
			"refund_amount": investment.RefundAmount, "refund_note": investment.RefundNote, "refunded_at": investment.RefundedAt,
		}).Error; err != nil {
			return fmt.Errorf("update investment: %w", err)
		}
		if fits {
			if err := tx.Model(&project).UpdateColumn("current_funding", gorm.Expr("current_funding + ?", investment.TotalAmount)).Error; err != nil {
				return fmt.Errorf("increment project funding: %w", err)
			}
			project.CurrentFunding += investment.TotalAmount
			if project.CurrentFunding >= project.FundingGoal-0.005 {
				if err := tx.Model(&project).Updates(map[string]interface{}{"state": domain.StateExecuting, "status": domain.StatusActive}).Error; err != nil {
					return fmt.Errorf("transition funded project: %w", err)
				}
				project.State, project.Status = domain.StateExecuting, domain.StatusActive
				if err := tx.Model(&domain.Milestone{}).Where("project_id = ? AND phase_no = 1 AND status = ?", project.ID, domain.MilestoneWaiting).Update("status", domain.MilestoneActive).Error; err != nil {
					return fmt.Errorf("activate first milestone: %w", err)
				}
			}
		}
		result = PaymentResult{Investment: investment, Project: project, Applied: true, Overfund: !fits}
		return nil
	})
	return result, err
}

func (r *paymentRepository) Fail(intentID string) (PaymentResult, error) {
	var result PaymentResult
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var payment domain.Transaction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("stripe_payment_intent_id = ?", intentID).First(&payment).Error; err != nil {
			return fmt.Errorf("find transaction for intent %s: %w", intentID, err)
		}
		if payment.Status == domain.TransactionSucceeded || payment.Status == domain.TransactionFailed {
			return nil
		}
		var investment domain.Investment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&investment, payment.InvestmentID).Error; err != nil {
			return fmt.Errorf("find investment for intent %s: %w", intentID, err)
		}
		if investment.Status == domain.InvestmentVerified || investment.Status == domain.InvestmentRefundPending || investment.Status == domain.InvestmentOverfundRefundPending || investment.Status == domain.InvestmentRefunded {
			return errors.New("failed payment conflicts with completed investment")
		}
		if err := tx.Model(&payment).Update("status", domain.TransactionFailed).Error; err != nil {
			return fmt.Errorf("fail transaction: %w", err)
		}
		if err := tx.Model(&investment).Update("status", domain.InvestmentRejected).Error; err != nil {
			return fmt.Errorf("reject investment: %w", err)
		}
		result = PaymentResult{Investment: investment, Applied: true}
		return nil
	})
	return result, err
}

func (r *paymentRepository) Expire(investmentID uint, now time.Time) (bool, error) {
	expired := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var payment domain.Transaction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("investment_id = ?", investmentID).First(&payment).Error; err != nil {
			return fmt.Errorf("find transaction for investment %d: %w", investmentID, err)
		}
		if payment.Status != domain.TransactionPending || payment.ExpiresAt.IsZero() || now.Before(payment.ExpiresAt) {
			return nil
		}
		var investment domain.Investment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&investment, investmentID).Error; err != nil {
			return fmt.Errorf("find investment %d: %w", investmentID, err)
		}
		if investment.Status != domain.InvestmentPending {
			return nil
		}
		if err := tx.Model(&payment).Update("status", domain.TransactionExpired).Error; err != nil {
			return fmt.Errorf("expire transaction %d: %w", payment.ID, err)
		}
		if err := tx.Model(&investment).Update("status", domain.InvestmentExpired).Error; err != nil {
			return fmt.Errorf("expire investment %d: %w", investmentID, err)
		}
		expired = true
		return nil
	})
	return expired, err
}

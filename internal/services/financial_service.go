package services

import (
	"flyup/internal/dto"
	"flyup/internal/repository"

	"github.com/stripe/stripe-go/v85"
	stripebalance "github.com/stripe/stripe-go/v85/balance"
)

type FinancialService interface {
	GetSummary() (*dto.FinancialSummary, error)
	GetProjectsFinancial() ([]dto.ProjectFinancialItem, error)
}

type financialService struct {
	repo            repository.FinancialRepository
	stripeSecretKey string
}

func NewFinancialService(repo repository.FinancialRepository, stripeSecretKey string) FinancialService {
	return &financialService{repo: repo, stripeSecretKey: stripeSecretKey}
}

// GetSummary รวมสรุปข้อมูลการเงินของแพลตฟอร์ม: ยอดคงเหลือใน Stripe, ยอดเบิกจ่าย, ยอดคืนเงิน, ค่าธรรมเนียม และรายได้
func (s *financialService) GetSummary() (*dto.FinancialSummary, error) {
	summary := &dto.FinancialSummary{}

	stripe.Key = s.stripeSecretKey
	if bal, err := stripebalance.Get(nil); err == nil {
		for _, a := range bal.Available {
			if a.Currency == stripe.CurrencyTHB {
				summary.Stripe.Available += float64(a.Amount) / 100
			}
		}
		for _, p := range bal.Pending {
			if p.Currency == stripe.CurrencyTHB {
				summary.Stripe.Pending += float64(p.Amount) / 100
			}
		}
	}

	confirmed, _ := s.repo.GetDisbursementStat("confirmed")
	summary.Disbursement.TotalConfirmed = confirmed.Total
	summary.Disbursement.CountConfirmed = confirmed.Count

	pending, _ := s.repo.GetDisbursementStat("pending")
	summary.Disbursement.TotalPending = pending.Total
	summary.Disbursement.CountPending = pending.Count

	refundPending, _ := s.repo.GetInvestmentRefundStat("refund_pending")
	summary.Refund.TotalPending = refundPending.Total
	summary.Refund.CountPending = refundPending.Count

	refunded, _ := s.repo.GetInvestmentRefundStat("refunded")
	summary.Refund.TotalRefunded = refunded.Total
	summary.Refund.CountRefunded = refunded.Count

	summary.Platform.TotalFees, _ = s.repo.GetTransactionFees()
	summary.Platform.TotalRevenue, _ = s.repo.GetTransactionRevenue()

	return summary, nil
}

// GetProjectsFinancial ดึงข้อมูลการเงินของแต่ละโปรเจกต์ พร้อมจัดกลุ่มข้อมูลแต่ละ phase (milestone/disbursement) เข้าด้วยกัน
func (s *financialService) GetProjectsFinancial() ([]dto.ProjectFinancialItem, error) {
	rows, err := s.repo.GetProjectsFinancialRows()
	if err != nil {
		return nil, err
	}

	projectMap := map[uint]*dto.ProjectFinancialItem{}
	var order []uint
	for _, r := range rows {
		if _, ok := projectMap[r.ProjectID]; !ok {
			projectMap[r.ProjectID] = &dto.ProjectFinancialItem{
				ProjectID:    r.ProjectID,
				ProjectTitle: r.ProjectTitle,
				State:        r.State,
				FundingGoal:  r.FundingGoal,
				CurrentFund:  r.CurrentFunding,
			}
			order = append(order, r.ProjectID)
		}
		projectMap[r.ProjectID].Phases = append(projectMap[r.ProjectID].Phases, dto.PhaseFinancialItem{
			PhaseNo:        r.PhaseNo,
			Title:          r.MilestoneTitle,
			PercentRelease: r.PercentRelease,
			Amount:         r.Amount,
			Status:         r.DisbStatus,
			ConfirmedAt:    r.ConfirmedAt,
		})
	}

	result := make([]dto.ProjectFinancialItem, 0, len(order))
	for _, id := range order {
		result = append(result, *projectMap[id])
	}
	return result, nil
}

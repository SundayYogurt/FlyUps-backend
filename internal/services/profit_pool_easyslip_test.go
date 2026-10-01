package services

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"flyup/pkg/easyslip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type slipFlowRepo struct {
	repository.ProfitPoolRepository
	pool     domain.ProfitPool
	payout   domain.InvestorProfitPayout
	writes   int
	verified *domain.VerifiedSlip
}

func (r *slipFlowRepo) ListAll() ([]domain.ProfitPool, error)                 { return nil, nil }
func (r *slipFlowRepo) ListByPioneerUserID(uint) ([]domain.ProfitPool, error) { return nil, nil }
func (r *slipFlowRepo) FindByID(uint) (*domain.ProfitPool, error)             { return &r.pool, nil }
func (r *slipFlowRepo) FindPayoutByID(uint) (*domain.InvestorProfitPayout, error) {
	return &r.payout, nil
}
func (r *slipFlowRepo) ListPayoutsByPoolID(uint) ([]domain.InvestorProfitPayout, error) {
	return []domain.InvestorProfitPayout{r.payout}, nil
}
func (r *slipFlowRepo) Update(*domain.ProfitPool) error { return nil }
func (r *slipFlowRepo) CreateWithPayouts(p *domain.ProfitPool, payouts []domain.InvestorProfitPayout, v *domain.VerifiedSlip) error {
	r.writes++
	r.verified = v
	r.pool = *p
	r.pool.ID = 1
	r.payout = payouts[0]
	return nil
}
func (r *slipFlowRepo) ConfirmWithSlip(p *domain.InvestorProfitPayout, v *domain.VerifiedSlip) error {
	r.writes++
	r.verified = v
	r.payout = *p
	return nil
}

type slipFlowProjects struct{ repository.ProjectRepository }

func (*slipFlowProjects) FindProjectByID(uint) (*domain.Project, error) {
	return &domain.Project{ID: 10, OwnerUserID: 7, State: domain.StateExecuting}, nil
}
func (*slipFlowProjects) FindMilestonesByProjectID(uint) ([]domain.Milestone, error) {
	return []domain.Milestone{{Status: domain.MilestonePaid}, {Status: domain.MilestonePaid}, {Status: domain.MilestonePaid}, {Status: domain.MilestonePaid}}, nil
}

type slipFlowInvestors struct {
	repository.InvestmentRepository
}

func (*slipFlowInvestors) ListInvestorsByProjectID(uint) ([]dto.ProjectInvestorItem, error) {
	return []dto.ProjectInvestorItem{{UserID: 8, PrincipalAmount: 100}}, nil
}

type slipFlowInvestmentService struct{ InvestmentService }

func (*slipFlowInvestmentService) SyncProjectPrincipalAmounts(uint) error { return nil }

type slipFlowUsers struct{ repository.UserRepository }

func (*slipFlowUsers) FindBankByUserId(uint) ([]domain.BankAccount, error) {
	return []domain.BankAccount{{BankName: "KBANK", AccountName: "Investor", AccountNumber: "1234567890", IsDefault: true}}, nil
}
func (*slipFlowUsers) FindUserById(uint) (*domain.User, error) { return nil, errors.New("not needed") }

type slipFlowVerifier struct {
	err       error
	recipient easyslip.Recipient
	amount    float64
	image     string
}

func (v *slipFlowVerifier) Verify(image string, amount float64, r easyslip.Recipient) (*easyslip.Verification, error) {
	v.recipient = r
	v.amount = amount
	v.image = image
	return &easyslip.Verification{TransRef: "BANK-REFERENCE", SenderBank: "004", TransferredAt: time.Now()}, v.err
}

func TestProfitFlowsRequireVerifiedSlip(t *testing.T) {
	for _, flow := range []string{"pioneer", "admin pool", "admin payout"} {
		for _, failed := range []bool{true, false} {
			t.Run(flow+map[bool]string{true: " rejected", false: " verified"}[failed], func(t *testing.T) {
				r := &slipFlowRepo{pool: domain.ProfitPool{ID: 1, ProjectID: 10}, payout: domain.InvestorProfitPayout{ID: 2, ProfitPoolID: 1, BoosterUserID: 8, Amount: 100, Status: domain.InvestorPayoutPending}}
				v := &slipFlowVerifier{}
				if failed {
					v.err = errors.New("wrong recipient")
				}
				s := NewProfitPoolService(r, &slipFlowProjects{}, &slipFlowInvestors{}, &slipFlowUsers{}, &slipFlowInvestmentService{}, nil, ProfitPayoutNotifications{SlipVerifier: v, PlatformRecipient: easyslip.Recipient{Bank: "004", Account: "0638725738"}})
				var err error
				switch flow {
				case "pioneer":
					_, err = s.PioneerSubmit(7, 10, dto.PioneerSubmitProfitRequest{TotalAmount: 100, QuarterNo: 1, TransferRef: "UNTRUSTED", SlipImage: "image"})
				case "admin pool":
					_, err = s.Create(99, dto.CreateProfitPoolRequest{ProjectID: 10, TotalAmount: 100, QuarterNo: 1, TransferRef: "UNTRUSTED", SlipImage: "image"})
				case "admin payout":
					err = s.ConfirmPayout(1, 2, 99, dto.ConfirmInvestorPayoutRequest{TransferRef: "UNTRUSTED", SlipImage: "image"})
				}
				if failed {
					require.Error(t, err)
					require.Zero(t, r.writes)
				} else {
					require.NoError(t, err)
					require.Equal(t, 1, r.writes)
					require.Equal(t, "BANK-REFERENCE", r.verified.TransRef)
					if flow == "admin payout" {
						require.Equal(t, "BANK-REFERENCE", r.payout.TransferRef)
					} else {
						require.Equal(t, "BANK-REFERENCE", r.pool.TransferRef)
					}
				}
				require.Equal(t, float64(100), v.amount)
				require.Equal(t, "image", v.image)
				if flow == "admin payout" {
					require.Equal(t, "1234567890", v.recipient.Account)
				} else {
					require.Equal(t, "0638725738", v.recipient.Account)
				}
			})
		}
	}
}

func TestProfitSlipMissingVerifierFailsClosed(t *testing.T) {
	s := &profitPoolService{}
	_, err := s.verifyProfitSlip("image", 100, easyslip.Recipient{})
	require.Error(t, err)
}

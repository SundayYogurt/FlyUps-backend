package services

import (
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"flyup/pkg/notification"
	"github.com/stretchr/testify/require"
	"testing"
)

type payoutTestRepo struct {
	repository.ProfitPoolRepository
	pool    domain.ProfitPool
	payout  domain.InvestorProfitPayout
	updates int
}

func (r *payoutTestRepo) FindByID(uint) (*domain.ProfitPool, error) { return &r.pool, nil }
func (r *payoutTestRepo) FindPayoutByID(uint) (*domain.InvestorProfitPayout, error) {
	return &r.payout, nil
}
func (r *payoutTestRepo) UpdatePayout(*domain.InvestorProfitPayout) error { r.updates++; return nil }
func (r *payoutTestRepo) ListPayoutsByPoolID(uint) ([]domain.InvestorProfitPayout, error) {
	return []domain.InvestorProfitPayout{r.payout}, nil
}
func (r *payoutTestRepo) Update(*domain.ProfitPool) error { return nil }

type payoutTestUsers struct {
	repository.UserRepository
	banks   []domain.BankAccount
	bankErr error
}

func (r *payoutTestUsers) FindBankByUserId(uint) ([]domain.BankAccount, error) {
	return r.banks, r.bankErr
}
func (r *payoutTestUsers) FindUserById(uint) (*domain.User, error) {
	return &domain.User{Email: "investor@example.com"}, nil
}

type payoutTestProjects struct{ repository.ProjectRepository }

func (*payoutTestProjects) FindProjectByID(uint) (*domain.Project, error) {
	return &domain.Project{Title: "Siam Spirit"}, nil
}

type payoutTestEmail struct {
	notification.NotificationClient
	calls    int
	testMode bool
	err      error
}

func (m *payoutTestEmail) SendProfitPayoutEmail(to, title string, q int, amount float64, ref string, test bool) error {
	if to != "investor@example.com" || title != "Siam Spirit" || q != 2 || amount != 20000 || ref != "REF-123" {
		panic("incorrect payout email")
	}
	m.calls++
	m.testMode = test
	return m.err
}

func TestConfirmProfitPayoutBankAndEmail(t *testing.T) {
	for _, tc := range []struct {
		name     string
		banks    []domain.BankAccount
		bankErr  error
		ref      string
		emailErr error
		allowed  bool
	}{
		{name: "missing bank", ref: "REF-123"},
		{name: "incomplete bank", ref: "REF-123", banks: []domain.BankAccount{{BankName: "Bank"}}},
		{name: "bank lookup fails", ref: "REF-123", bankErr: errors.New("db")},
		{name: "missing slip", banks: []domain.BankAccount{{BankName: "Bank", AccountName: "Investor", AccountNumber: "123"}}},
		{name: "email details and duplicate confirmation", ref: "REF-123", banks: []domain.BankAccount{{BankName: "Bank", AccountName: "Investor", AccountNumber: "123"}}, allowed: true},
		{name: "email failure does not undo confirmation", ref: "REF-123", banks: []domain.BankAccount{{BankName: "Bank", AccountName: "Investor", AccountNumber: "123"}}, emailErr: errors.New("email unavailable"), allowed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &payoutTestRepo{pool: domain.ProfitPool{ID: 1, ProjectID: 10, QuarterNo: 2}, payout: domain.InvestorProfitPayout{ID: 5, ProfitPoolID: 1, BoosterUserID: 2, Amount: 20000}}
			mail := &payoutTestEmail{err: tc.emailErr}
			svc := NewProfitPoolService(repo, &payoutTestProjects{}, nil, &payoutTestUsers{banks: tc.banks, bankErr: tc.bankErr}, nil, nil, ProfitPayoutNotifications{SlipVerifier: fixedProfitVerifier{ref: "REF-123", requireImage: true}, EmailClient: mail, TestMode: true})
			err := svc.ConfirmPayout(1, 5, 99, dto.ConfirmInvestorPayoutRequest{TransferRef: tc.ref, SlipImage: tc.ref})
			if tc.allowed {
				require.NoError(t, err)
				require.Equal(t, 1, repo.updates)
				require.Equal(t, 1, mail.calls)
				require.True(t, mail.testMode)
				require.Error(t, svc.ConfirmPayout(1, 5, 99, dto.ConfirmInvestorPayoutRequest{TransferRef: tc.ref, SlipImage: tc.ref}))
				require.Equal(t, 1, mail.calls)
			} else {
				require.Error(t, err)
				require.Zero(t, repo.updates)
				require.Zero(t, mail.calls)
			}
		})
	}
}

func (r *payoutTestRepo) ConfirmWithSlip(p *domain.InvestorProfitPayout, slip *domain.VerifiedSlip) error {
	return r.UpdatePayout(p)
}

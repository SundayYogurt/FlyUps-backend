package repository

import (
	"flyup/internal/domain"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestProfitSlipAtomicConsumption(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	require.NoError(t, db.AutoMigrate(&domain.VerifiedSlip{}, &domain.ProfitPool{}, &domain.InvestorProfitPayout{}))
	repo := NewProfitPoolRepository(db)
	slip := func(ref string) *domain.VerifiedSlip {
		return &domain.VerifiedSlip{TransRef: ref, SenderBank: "004", Amount: 100, VerifiedAt: time.Now()}
	}
	pool := &domain.ProfitPool{ProjectID: 1, QuarterNo: 1, TotalAmount: 100}
	payouts := []domain.InvestorProfitPayout{{BoosterUserID: 1, Amount: 100, Status: domain.InvestorPayoutPending}}
	require.NoError(t, repo.CreateWithPayouts(pool, payouts, slip("INCOMING")))
	require.NotNil(t, pool.VerifiedSlipID)
	require.NotZero(t, payouts[0].ID)

	// Same slip cannot fund another pool or confirm an outgoing payment.
	require.ErrorContains(t, repo.CreateWithPayouts(&domain.ProfitPool{ProjectID: 2, QuarterNo: 1}, payouts, slip("INCOMING")), "ถูกใช้แล้ว")
	p := payouts[0]
	p.Status = domain.InvestorPayoutConfirmed
	require.ErrorContains(t, repo.ConfirmWithSlip(&p, slip("INCOMING")), "ถูกใช้แล้ว")
	stored, err := repo.FindPayoutByID(p.ID)
	require.NoError(t, err)
	require.Equal(t, domain.InvestorPayoutPending, stored.Status)

	// Failure after inserting the slip rolls back consumption, allowing a retry.
	require.Error(t, repo.CreateWithPayouts(&domain.ProfitPool{ProjectID: 1, QuarterNo: 1}, payouts, slip("RETRY")))
	require.NoError(t, repo.CreateWithPayouts(&domain.ProfitPool{ProjectID: 2, QuarterNo: 1}, []domain.InvestorProfitPayout{{Amount: 100}}, slip("RETRY")))
	require.NoError(t, repo.ConfirmWithSlip(&p, slip("OUTGOING")))
	require.ErrorContains(t, repo.ConfirmWithSlip(&p, slip("SECOND")), "already confirmed")
	var count int64
	require.NoError(t, db.Model(&domain.VerifiedSlip{}).Where("trans_ref = ?", "SECOND").Count(&count).Error)
	require.Zero(t, count)
	stored, err = repo.FindPayoutByID(p.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.SlipVerifiedAt)
	require.Error(t, repo.ConfirmWithSlip(&p, nil))
	legacy := &domain.ProfitPool{ProjectID: 3, QuarterNo: 1, TransferRef: "LEGACY"}
	require.NoError(t, db.Create(legacy).Error)
	require.NoError(t, db.Delete(legacy).Error)
	require.ErrorContains(t, repo.CreateWithPayouts(&domain.ProfitPool{ProjectID: 4, QuarterNo: 1}, []domain.InvestorProfitPayout{{Amount: 100}}, slip("LEGACY")), "ถูกใช้แล้ว")
}

func TestConcurrentProfitSlipConfirmations(t *testing.T) {
	for _, sameSlip := range []bool{true, false} {
		t.Run(map[bool]string{true: "same transfer across payouts", false: "different transfers same payout"}[sameSlip], func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
			require.NoError(t, err)
			sqlDB, _ := db.DB()
			defer sqlDB.Close()
			sqlDB.SetMaxOpenConns(1)
			require.NoError(t, db.AutoMigrate(&domain.VerifiedSlip{}, &domain.ProfitPool{}, &domain.InvestorProfitPayout{}))
			payouts := []domain.InvestorProfitPayout{{Status: domain.InvestorPayoutPending}, {Status: domain.InvestorPayoutPending}}
			require.NoError(t, db.Create(&payouts).Error)
			repo := NewProfitPoolRepository(db)
			results := make(chan error, 2)
			for i := 0; i < 2; i++ {
				p := payouts[i]
				ref := "SHARED"
				if !sameSlip {
					p = payouts[0]
					if i == 1 {
						ref = "OTHER"
					}
				}
				p.Status = domain.InvestorPayoutConfirmed
				go func(p domain.InvestorProfitPayout, ref string) {
					results <- repo.ConfirmWithSlip(&p, &domain.VerifiedSlip{TransRef: ref, SenderBank: "004", VerifiedAt: time.Now()})
				}(p, ref)
			}
			success := 0
			for i := 0; i < 2; i++ {
				if <-results == nil {
					success++
				}
			}
			require.Equal(t, 1, success)
			var count int64
			require.NoError(t, db.Model(&domain.VerifiedSlip{}).Count(&count).Error)
			require.EqualValues(t, 1, count)
		})
	}
}

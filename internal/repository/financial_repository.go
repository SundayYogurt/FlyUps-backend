package repository

import "gorm.io/gorm"

type FinancialStat struct {
	Total float64
	Count int64
}

type FinancialProjectRow struct {
	ProjectID      uint
	ProjectTitle   string
	State          string
	FundingGoal    float64
	CurrentFunding float64
	PhaseNo        int
	MilestoneTitle string
	PercentRelease int
	Amount         float64
	DisbStatus     string
	ConfirmedAt    *string
}

type FinancialRepository interface {
	GetDisbursementStat(status string) (FinancialStat, error)
	GetInvestmentRefundStat(status string) (FinancialStat, error)
	GetTransactionFees() (float64, error)
	GetTransactionRevenue() (float64, error)
	GetProjectsFinancialRows() ([]FinancialProjectRow, error)
}

type financialRepository struct {
	db *gorm.DB
}

func NewFinancialRepository(db *gorm.DB) FinancialRepository {
	return &financialRepository{db}
}

// GetDisbursementStat รวมยอดเงิน (Total) และจำนวนรายการ (Count) ของการเบิกจ่ายเงินตามสถานะที่ระบุ
func (r *financialRepository) GetDisbursementStat(status string) (FinancialStat, error) {
	var stat FinancialStat
	r.db.Table("disbursements").
		Where("status = ? AND deleted_at IS NULL", status).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&stat.Total)
	r.db.Table("disbursements").
		Where("status = ? AND deleted_at IS NULL", status).
		Count(&stat.Count)
	return stat, nil
}

// GetInvestmentRefundStat รวมยอดเงินคืน (refund_amount) และจำนวนรายการของการลงทุนตามสถานะที่ระบุ
func (r *financialRepository) GetInvestmentRefundStat(status string) (FinancialStat, error) {
	var stat FinancialStat
	r.db.Table("investments").
		Where("status = ? AND deleted_at IS NULL", status).
		Select("COALESCE(SUM(refund_amount), 0)").
		Scan(&stat.Total)
	r.db.Table("investments").
		Where("status = ? AND deleted_at IS NULL", status).
		Count(&stat.Count)
	return stat, nil
}

// GetTransactionFees รวมค่าธรรมเนียม Stripe ทั้งหมด (stripe_fee + stripe_fee_vat) ของธุรกรรมที่สำเร็จแล้ว
func (r *financialRepository) GetTransactionFees() (float64, error) {
	var total float64
	r.db.Table("transactions").
		Where("status = ? AND deleted_at IS NULL", "succeeded").
		Select("COALESCE(SUM(stripe_fee + stripe_fee_vat), 0)").
		Scan(&total)
	return total, nil
}

// GetTransactionRevenue รวมยอดเงินสุทธิ (net_amount) ของธุรกรรมที่สำเร็จแล้วทั้งหมด
func (r *financialRepository) GetTransactionRevenue() (float64, error) {
	var total float64
	r.db.Table("transactions").
		Where("status = ? AND deleted_at IS NULL", "succeeded").
		Select("COALESCE(SUM(net_amount), 0)").
		Scan(&total)
	return total, nil
}

// GetProjectsFinancialRows ดึงข้อมูลการเงินของแต่ละโปรเจกต์แบบรวม milestone และการเบิกจ่ายเงิน (JOIN projects, milestones, disbursements)
// เฉพาะโปรเจกต์ที่อยู่ในสถานะ executing, funded, completed หรือ cancelled เรียงตามโปรเจกต์และเฟส
func (r *financialRepository) GetProjectsFinancialRows() ([]FinancialProjectRow, error) {
	var rows []FinancialProjectRow
	r.db.Raw(`
		SELECT
			p.id              AS project_id,
			p.title           AS project_title,
			p.state           AS state,
			p.funding_goal    AS funding_goal,
			p.current_funding AS current_funding,
			m.phase_no        AS phase_no,
			m.title           AS milestone_title,
			m.percent_release AS percent_release,
			COALESCE(d.amount, 0)          AS amount,
			COALESCE(d.status, 'not_started') AS disb_status,
			d.confirmed_at    AS confirmed_at
		FROM projects p
		JOIN milestones m ON m.project_id = p.id AND m.deleted_at IS NULL
		LEFT JOIN disbursements d ON d.milestone_id = m.id AND d.deleted_at IS NULL
		WHERE p.deleted_at IS NULL
		  AND p.state IN ('executing','funded','completed','cancelled')
		ORDER BY p.id, m.phase_no
	`).Scan(&rows)
	return rows, nil
}

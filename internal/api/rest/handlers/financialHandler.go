package handlers

import (
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/stripe/stripe-go/v85"
	stripebalance "github.com/stripe/stripe-go/v85/balance"
	"gorm.io/gorm"
)

type FinancialHandler struct {
	db              *gorm.DB
	auth            helper.Auth
	stripeSecretKey string
}

func SetupFinancialRoutes(rh *rest.RestHandler) {
	h := &FinancialHandler{
		db:              rh.DB,
		auth:            rh.Auth,
		stripeSecretKey: rh.Config.StripeSecretKey,
	}

	admin := rh.App.Group("/admin/financial", rh.Middlewares.AuthorizeAdmin)
	admin.Get("/summary", h.GetSummary)
}

type financialSummary struct {
	Stripe       stripeBalance    `json:"stripe"`
	Disbursement disbursementStat `json:"disbursement"`
	Refund       refundStat       `json:"refund"`
	Platform     platformStat     `json:"platform"`
}

type stripeBalance struct {
	Available float64 `json:"available"`
	Pending   float64 `json:"pending"`
}

type disbursementStat struct {
	TotalConfirmed  float64 `json:"total_confirmed"`
	TotalPending    float64 `json:"total_pending"`
	CountConfirmed  int64   `json:"count_confirmed"`
	CountPending    int64   `json:"count_pending"`
}

type refundStat struct {
	TotalPending   float64 `json:"total_pending"`
	TotalRefunded  float64 `json:"total_refunded"`
	CountPending   int64   `json:"count_pending"`
	CountRefunded  int64   `json:"count_refunded"`
}

type platformStat struct {
	TotalFees    float64 `json:"total_fees"`
	TotalRevenue float64 `json:"total_revenue"`
}

// GetSummary godoc
// @Summary Admin financial summary
// @Description Returns Stripe balance, disbursement stats, and refund stats
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "financial summary"
// @Router /admin/financial/summary [get]
func (h *FinancialHandler) GetSummary(ctx fiber.Ctx) error {
	summary := financialSummary{}

	// ── Stripe Balance ──────────────────────────────────────────────────────
	stripe.Key = h.stripeSecretKey
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

	// ── Disbursements ────────────────────────────────────────────────────────
	h.db.Table("disbursements").
		Where("status = ? AND deleted_at IS NULL", "confirmed").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&summary.Disbursement.TotalConfirmed)
	h.db.Table("disbursements").
		Where("status = ? AND deleted_at IS NULL", "confirmed").
		Count(&summary.Disbursement.CountConfirmed)

	h.db.Table("disbursements").
		Where("status = ? AND deleted_at IS NULL", "pending").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&summary.Disbursement.TotalPending)
	h.db.Table("disbursements").
		Where("status = ? AND deleted_at IS NULL", "pending").
		Count(&summary.Disbursement.CountPending)

	// ── Refunds ──────────────────────────────────────────────────────────────
	h.db.Table("investments").
		Where("status = ? AND deleted_at IS NULL", "refund_pending").
		Select("COALESCE(SUM(refund_amount), 0)").
		Scan(&summary.Refund.TotalPending)
	h.db.Table("investments").
		Where("status = ? AND deleted_at IS NULL", "refund_pending").
		Count(&summary.Refund.CountPending)

	h.db.Table("investments").
		Where("status = ? AND deleted_at IS NULL", "refunded").
		Select("COALESCE(SUM(refund_amount), 0)").
		Scan(&summary.Refund.TotalRefunded)
	h.db.Table("investments").
		Where("status = ? AND deleted_at IS NULL", "refunded").
		Count(&summary.Refund.CountRefunded)

	// ── Platform Fees ────────────────────────────────────────────────────────
	h.db.Table("transactions").
		Where("status = ? AND deleted_at IS NULL", "succeeded").
		Select("COALESCE(SUM(stripe_fee + stripe_fee_vat), 0)").
		Scan(&summary.Platform.TotalFees)

	h.db.Table("transactions").
		Where("status = ? AND deleted_at IS NULL", "succeeded").
		Select("COALESCE(SUM(net_amount), 0)").
		Scan(&summary.Platform.TotalRevenue)

	return rest.SuccessResponse(ctx, "success", summary)
}

func SetupProjectFinancialRoutes(rh *rest.RestHandler) {
	h := &FinancialHandler{db: rh.DB, auth: rh.Auth, stripeSecretKey: rh.Config.StripeSecretKey}

	admin := rh.App.Group("/admin/financial", rh.Middlewares.AuthorizeAdmin)
	admin.Get("/projects", h.GetProjectsFinancial)
}

type projectFinancialItem struct {
	ProjectID    uint    `json:"project_id"`
	ProjectTitle string  `json:"project_title"`
	State        string  `json:"state"`
	FundingGoal  float64 `json:"funding_goal"`
	CurrentFund  float64 `json:"current_funding"`
	Phases       []phaseFinancialItem `json:"phases"`
}

type phaseFinancialItem struct {
	PhaseNo        int      `json:"phase_no"`
	Title          string   `json:"title"`
	PercentRelease int      `json:"percent_release"`
	Amount         float64  `json:"amount"`
	Status         string   `json:"status"` // pending | confirmed | not_started
	ConfirmedAt    *string  `json:"confirmed_at"`
}

// GetProjectsFinancial godoc
// @Summary Admin per-project financial overview
// @Description Returns per-project phase payout status
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "projects financial"
// @Router /admin/financial/projects [get]
func (h *FinancialHandler) GetProjectsFinancial(ctx fiber.Ctx) error {
	type rawDisb struct {
		ID             uint
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

	var rows []rawDisb
	h.db.Raw(`
		SELECT
			p.id            AS project_id,
			p.title         AS project_title,
			p.state         AS state,
			p.funding_goal  AS funding_goal,
			p.current_funding AS current_funding,
			m.phase_no      AS phase_no,
			m.title         AS milestone_title,
			m.percent_release AS percent_release,
			COALESCE(d.amount, 0) AS amount,
			COALESCE(d.status, 'not_started') AS disb_status,
			d.confirmed_at  AS confirmed_at
		FROM projects p
		JOIN milestones m ON m.project_id = p.id AND m.deleted_at IS NULL
		LEFT JOIN disbursements d ON d.milestone_id = m.id AND d.deleted_at IS NULL
		WHERE p.deleted_at IS NULL
		  AND p.state IN ('executing','funded','completed','cancelled')
		ORDER BY p.id, m.phase_no
	`).Scan(&rows)

	// group by project
	projectMap := map[uint]*projectFinancialItem{}
	var order []uint
	for _, r := range rows {
		if _, ok := projectMap[r.ProjectID]; !ok {
			projectMap[r.ProjectID] = &projectFinancialItem{
				ProjectID:    r.ProjectID,
				ProjectTitle: r.ProjectTitle,
				State:        r.State,
				FundingGoal:  r.FundingGoal,
				CurrentFund:  r.CurrentFunding,
			}
			order = append(order, r.ProjectID)
		}
		projectMap[r.ProjectID].Phases = append(projectMap[r.ProjectID].Phases, phaseFinancialItem{
			PhaseNo:        r.PhaseNo,
			Title:          r.MilestoneTitle,
			PercentRelease: r.PercentRelease,
			Amount:         r.Amount,
			Status:         r.DisbStatus,
			ConfirmedAt:    r.ConfirmedAt,
		})
	}

	result := make([]projectFinancialItem, 0, len(order))
	for _, id := range order {
		result = append(result, *projectMap[id])
	}

	if result == nil {
		result = []projectFinancialItem{}
	}
	return ctx.Status(http.StatusOK).JSON(fiber.Map{"message": "success", "data": result})
}

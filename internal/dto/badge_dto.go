package dto

type AdminBadgeCounts struct {
	PendingProjects      int64 `json:"pending_projects"`
	SubmittedMilestones  int64 `json:"submitted_milestones"`
	PendingCancelReqs    int64 `json:"pending_cancel_requests"`
	OpenComplaints       int64 `json:"open_complaints"`
	PendingRefunds       int64 `json:"pending_refunds"`
	PendingVerifications int64 `json:"pending_verifications"`
	PendingDisbursements int64 `json:"pending_disbursements"`
	PendingProfitPools   int64 `json:"pending_profit_pools"`
}

type BoosterBadgeCounts struct {
	PendingVotes     int64 `json:"pending_votes"`
	UpcomingMeetings int64 `json:"upcoming_meetings"`
	PendingRefunds   int64 `json:"pending_refunds"`
	OpenComplaints   int64 `json:"open_complaints"`
}

type PioneerBadgeCounts struct {
	ActiveMilestones int64 `json:"active_milestones"`
	UpcomingMeetings int64 `json:"upcoming_meetings"`
	PendingPayouts   int64 `json:"pending_payouts"`
}

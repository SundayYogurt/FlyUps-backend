package domain

import "gorm.io/gorm"

// AdminLog บันทึก action ที่ admin กระทำในระบบ
type AdminLog struct {
	ID         uint    `json:"id"`
	AdminID    uint    `json:"admin_id"`
	Admin      *User   `json:"admin,omitempty" gorm:"foreignKey:AdminID"`
	Action     string  `json:"action"`      // ชื่อ action เช่น "approve_project", "suspend_user"
	TargetType string  `json:"target_type"` // ประเภท entity เช่น "user", "project", "investment"
	TargetID   *uint   `json:"target_id,omitempty"`
	Note       *string `json:"note,omitempty"` // หมายเหตุเพิ่มเติม เช่น reason ที่ reject
	gorm.Model
}

// action constants
const (
	AdminActionApproveStudentCard = "approve_student_card"
	AdminActionRejectStudentCard  = "reject_student_card"
	AdminActionApproveIDCard      = "approve_id_card"
	AdminActionRejectIDCard       = "reject_id_card"
	AdminActionSuspendUser        = "suspend_user"
	AdminActionRollbackUser       = "rollback_user"
	AdminActionApproveProject     = "approve_project"
	AdminActionRejectProject      = "reject_project"
	AdminActionApproveCancel      = "approve_cancel"
	AdminActionRejectCancel       = "reject_cancel"
	AdminActionApproveMilestone   = "approve_milestone"
	AdminActionRejectMilestone    = "reject_milestone"
	AdminActionApproveRefund      = "approve_refund"
	AdminActionResolveComplaint   = "resolve_complaint"
	AdminActionRejectComplaint    = "reject_complaint"
	AdminActionConfirmDisbursement = "confirm_disbursement"
	AdminActionConfirmProfitPayout = "confirm_profit_payout"
	AdminActionSuspendProject     = "suspend_project"
	AdminActionUnsuspendProject   = "unsuspend_project"
)

// target type constants
const (
	TargetTypeUser       = "user"
	TargetTypeProject    = "project"
	TargetTypeInvestment = "investment"
	TargetTypeMilestone  = "milestone"
	TargetTypeComplaint  = "complaint"
	TargetTypeDisbursement = "disbursement"
	TargetTypeProfitPayout = "profit_payout"
)

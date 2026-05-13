package llm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// --- Shared types ---

type ChatMessage struct {
	Role    string
	Content string
}

type ChatActionType string

const (
	// Query actions — execute ทันที ไม่ต้อง confirm
	ChatActionTypeGetInvestments  ChatActionType = "get_my_investments"
	ChatActionTypeGetProjects     ChatActionType = "get_recommended_projects"
	ChatActionTypeGetProjectDetail ChatActionType = "get_project_detail"
	ChatActionTypeGetEndingSoon   ChatActionType = "get_projects_ending_soon"
	ChatActionTypeGetNewProjects  ChatActionType = "get_new_projects"
	ChatActionTypeGetNotifications ChatActionType = "get_my_notifications"
	ChatActionTypeGetDisbursements ChatActionType = "get_my_disbursements"

	// Action tools — ต้อง confirm ก่อน
	ChatActionTypeRefund          ChatActionType = "refund_transaction"
	ChatActionTypeCancel          ChatActionType = "cancel_transaction"
	ChatActionTypeVoteMilestone   ChatActionType = "vote_milestone"
	ChatActionTypeFileComplaint   ChatActionType = "file_complaint"
	ChatActionTypeMarkNotifRead   ChatActionType = "mark_notifications_read"
)

type ChatIntent string

const (
	ChatIntentUnknown           ChatIntent = "unknown"
	ChatIntentRefundPolicy      ChatIntent = "refund_policy"
	ChatIntentRefundRequest     ChatIntent = "refund_request"
	ChatIntentCancelTransaction ChatIntent = "cancel_transaction"
	ChatIntentTransactionStatus ChatIntent = "transaction_status"
)

type ChatInput struct {
	UserID   uint
	Message  string
	History  []ChatMessage
	Language string
}

type ChatOutput struct {
	Intent         ChatIntent
	Reply          string
	RequiresAction bool
	ActionType     ChatActionType
	InvestmentID   *uint  // investment_id สำหรับ refund/cancel
	MilestoneID    *uint  // milestone_id สำหรับ vote
	ProjectID      *uint  // project_id สำหรับ get_project_detail
	ExtraData      string // ข้อมูลเพิ่มเติม เช่น vote choice, complaint body
}

// --- Tool argument structs ---

type refundArgs struct {
	InvestmentID uint   `json:"investment_id"`
	Reason       string `json:"reason"`
}

type cancelArgs struct {
	InvestmentID uint   `json:"investment_id"`
	Reason       string `json:"reason"`
}

// --- FlyUpChatClient ---

type FlyUpChatClient struct {
	openai *OpenAIClient
}

func NewFlyUpChatClient(apiKey string) *FlyUpChatClient {
	return &FlyUpChatClient{openai: NewOpenAIClient(apiKey)}
}

func NewFlyUpChatClientWithModel(apiKey, model string) *FlyUpChatClient {
	return &FlyUpChatClient{openai: NewOpenAIClientWithModel(apiKey, model)}
}

func (c *FlyUpChatClient) GenerateReply(input ChatInput) (*ChatOutput, error) {
	msgs := []OpenAIMessage{{Role: "system", Content: systemPrompt()}}

	history := input.History
	if len(history) > 10 {
		history = history[len(history)-10:]
	}
	for _, h := range history {
		if h.Role == "system" {
			continue
		}
		msgs = append(msgs, OpenAIMessage{Role: h.Role, Content: h.Content})
	}

	reply, err := c.openai.Chat(msgs, buildTools())
	if err != nil {
		return nil, fmt.Errorf("openai chat: %w", err)
	}

	if reply == nil {
		return nil, errors.New("openai returned empty response")
	}

	if len(reply.ToolCalls) > 0 {
		return handleToolCall(reply.ToolCalls[0].Function.Name, reply.ToolCalls[0].Function.Arguments)
	}

	return &ChatOutput{
		Intent:         detectIntent(input.Message),
		Reply:          reply.Content,
		RequiresAction: false,
	}, nil
}

// --- Tool definitions ---

func buildTools() []OpenAITool {
	return []OpenAITool{
		// --- Query tools ---
		{Type: "function", Function: OpenAIToolDef{
			Name:        "get_my_investments",
			Description: "ดูรายการลงทุนทั้งหมดของ user",
			Parameters:  emptyParams(),
		}},
		{Type: "function", Function: OpenAIToolDef{
			Name:        "get_recommended_projects",
			Description: "ดูโปรเจกต์แนะนำที่น่าสนใจ",
			Parameters:  emptyParams(),
		}},
		{Type: "function", Function: OpenAIToolDef{
			Name:        "get_project_detail",
			Description: "ดูรายละเอียดโปรเจกต์ เช่น milestone, เป้าหมาย, ผลตอบแทน ใช้ project_id ถ้ามี หรือ project_name ถ้ารู้แค่ชื่อ",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer", "description": "ID ของโปรเจกต์ (ถ้ามี)"},
					"project_name": map[string]any{"type": "string", "description": "ชื่อโปรเจกต์ (ถ้าไม่มี ID)"},
				},
				"required": []string{},
			},
		}},
		{Type: "function", Function: OpenAIToolDef{
			Name:        "get_projects_ending_soon",
			Description: "ดูโปรเจกต์ที่ใกล้หมดเวลาระดมทุน",
			Parameters:  emptyParams(),
		}},
		{Type: "function", Function: OpenAIToolDef{
			Name:        "get_new_projects",
			Description: "ดูโปรเจกต์ใหม่ล่าสุด",
			Parameters:  emptyParams(),
		}},
		{Type: "function", Function: OpenAIToolDef{
			Name:        "get_my_notifications",
			Description: "ดูการแจ้งเตือนของ user หรือนับจำนวนที่ยังไม่ได้อ่าน",
			Parameters:  emptyParams(),
		}},
		{Type: "function", Function: OpenAIToolDef{
			Name:        "get_my_disbursements",
			Description: "ดูประวัติการรับเงินจาก milestone (สำหรับ Pioneer)",
			Parameters:  emptyParams(),
		}},
		// --- Action tools ---
		{Type: "function", Function: OpenAIToolDef{
			Name:        "refund_transaction",
			Description: "ขอคืนเงินการลงทุน ต้องเรียก get_my_investments ก่อนเสมอเพื่อหา investment_id ห้ามเดา investment_id เอง",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"investment_id": map[string]any{
						"type":        "integer",
						"description": "ID ของการลงทุน ต้องได้มาจาก get_my_investments เท่านั้น",
					},
					"project_title": map[string]any{
						"type":        "string",
						"description": "ชื่อโปรเจกต์ที่ต้องการคืนเงิน เพื่อยืนยันกับ user",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "เหตุผลในการขอคืนเงิน",
					},
				},
				"required": []string{"investment_id", "project_title", "reason"},
			},
		}},
		{Type: "function", Function: OpenAIToolDef{
			Name:        "cancel_transaction",
			Description: "ยกเลิกการลงทุน ต้องเรียก get_my_investments ก่อนเสมอเพื่อหา investment_id ห้ามเดา investment_id เอง",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"investment_id": map[string]any{
						"type":        "integer",
						"description": "ID ของการลงทุน ต้องได้มาจาก get_my_investments เท่านั้น",
					},
					"project_title": map[string]any{
						"type":        "string",
						"description": "ชื่อโปรเจกต์ที่ต้องการยกเลิก เพื่อยืนยันกับ user",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "เหตุผลในการยกเลิก",
					},
				},
				"required": []string{"investment_id", "project_title", "reason"},
			},
		}},
		{Type: "function", Function: OpenAIToolDef{
			Name:        "vote_milestone",
			Description: "โหวต milestone ของโปรเจกต์ที่ลงทุน (อนุมัติหรือปฏิเสธ)",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"milestone_id": map[string]any{"type": "integer", "description": "ID ของ milestone"},
					"choice":       map[string]any{"type": "string", "enum": []string{"approve", "reject"}, "description": "approve = อนุมัติ, reject = ปฏิเสธ"},
				},
				"required": []string{"milestone_id", "choice"},
			},
		}},
		{Type: "function", Function: OpenAIToolDef{
			Name:        "file_complaint",
			Description: "ร้องเรียนโปรเจกต์ที่มีปัญหา",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer", "description": "ID ของโปรเจกต์ที่ต้องการร้องเรียน"},
					"subject":    map[string]any{"type": "string", "description": "หัวข้อการร้องเรียน"},
					"body":       map[string]any{"type": "string", "description": "รายละเอียดการร้องเรียน"},
				},
				"required": []string{"project_id", "subject", "body"},
			},
		}},
		{Type: "function", Function: OpenAIToolDef{
			Name:        "mark_notifications_read",
			Description: "ทำเครื่องหมายว่าอ่านการแจ้งเตือนทั้งหมดแล้ว",
			Parameters:  emptyParams(),
		}},
	}
}

func emptyParams() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"required":   []string{},
	}
}

func investmentToolParams(action string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"investment_id": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("ID ของการลงทุนที่ต้องการ%s", action),
			},
			"reason": map[string]any{
				"type":        "string",
				"description": fmt.Sprintf("เหตุผลในการ%s", action),
			},
		},
		"required": []string{"investment_id", "reason"},
	}
}

// --- Tool call handler ---

func handleToolCall(name, argsJSON string) (*ChatOutput, error) {
	// parse args เป็น map ก่อน
	var args map[string]any
	_ = json.Unmarshal([]byte(argsJSON), &args)
	uintArg := func(key string) uint {
		if v, ok := args[key]; ok {
			switch n := v.(type) {
			case float64:
				return uint(n)
			}
		}
		return 0
	}
	strArg := func(key string) string {
		if v, ok := args[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	switch name {
	// --- Query tools ---
	case "get_my_investments":
		return &ChatOutput{RequiresAction: true, ActionType: ChatActionTypeGetInvestments}, nil
	case "get_recommended_projects":
		return &ChatOutput{RequiresAction: true, ActionType: ChatActionTypeGetProjects}, nil
	case "get_project_detail":
		id := uintArg("project_id")
		name := strArg("project_name")
		if id == 0 && name == "" {
			return nil, errors.New("project_id or project_name is required")
		}
		if id != 0 {
			return &ChatOutput{RequiresAction: true, ActionType: ChatActionTypeGetProjectDetail, ProjectID: &id}, nil
		}
		return &ChatOutput{RequiresAction: true, ActionType: ChatActionTypeGetProjectDetail, ExtraData: name}, nil
	case "get_projects_ending_soon":
		return &ChatOutput{RequiresAction: true, ActionType: ChatActionTypeGetEndingSoon}, nil
	case "get_new_projects":
		return &ChatOutput{RequiresAction: true, ActionType: ChatActionTypeGetNewProjects}, nil
	case "get_my_notifications":
		return &ChatOutput{RequiresAction: true, ActionType: ChatActionTypeGetNotifications}, nil
	case "get_my_disbursements":
		return &ChatOutput{RequiresAction: true, ActionType: ChatActionTypeGetDisbursements}, nil

	// --- Action tools ---
	case "refund_transaction":
		id := uintArg("investment_id")
		if id == 0 {
			return nil, errors.New("investment_id is required for refund")
		}
		title := strArg("project_title")
		displayName := fmt.Sprintf("#%d", id)
		if title != "" {
			displayName = fmt.Sprintf("\"%s\"", title)
		}
		return &ChatOutput{
			Intent:         ChatIntentRefundRequest,
			Reply:          fmt.Sprintf("Rocket ขอยืนยันนะครับ~ ต้องการขอคืนเงินสำหรับโปรเจกต์ %s ใช่ป่าวครับ? 🚀", displayName),
			RequiresAction: true,
			ActionType:     ChatActionTypeRefund,
			InvestmentID:   &id,
		}, nil

	case "cancel_transaction":
		id := uintArg("investment_id")
		if id == 0 {
			return nil, errors.New("investment_id is required for cancel")
		}
		title := strArg("project_title")
		displayName := fmt.Sprintf("#%d", id)
		if title != "" {
			displayName = fmt.Sprintf("\"%s\"", title)
		}
		return &ChatOutput{
			Intent:         ChatIntentCancelTransaction,
			Reply:          fmt.Sprintf("เดี๋ยวนะครับ~ ต้องการยกเลิกการลงทุนในโปรเจกต์ %s จริงๆ เหรอครับ? คิดดีๆ ก่อนนะ 👀", displayName),
			RequiresAction: true,
			ActionType:     ChatActionTypeCancel,
			InvestmentID:   &id,
		}, nil

	case "vote_milestone":
		milestoneID := uintArg("milestone_id")
		choice := strArg("choice")
		if milestoneID == 0 || choice == "" {
			return nil, errors.New("milestone_id and choice are required")
		}
		choiceText := "อนุมัติ ✅"
		if choice == "reject" {
			choiceText = "ปฏิเสธ ❌"
		}
		return &ChatOutput{
			Reply:          fmt.Sprintf("Rocket จะโหวต%s milestone #%d ให้นะครับ~ ยืนยันเลยไหมครับ? 🗳️", choiceText, milestoneID),
			RequiresAction: true,
			ActionType:     ChatActionTypeVoteMilestone,
			MilestoneID:    &milestoneID,
			ExtraData:      choice,
		}, nil

	case "file_complaint":
		projectID := uintArg("project_id")
		subject := strArg("subject")
		body := strArg("body")
		if projectID == 0 || subject == "" || body == "" {
			return nil, errors.New("project_id, subject and body are required")
		}
		return &ChatOutput{
			Reply:          fmt.Sprintf("Rocket จะยื่นเรื่องร้องเรียนโปรเจกต์ #%d ให้นะครับ~ เรื่อง: \"%s\" ยืนยันเลยไหมครับ? 📝", projectID, subject),
			RequiresAction: true,
			ActionType:     ChatActionTypeFileComplaint,
			ProjectID:      &projectID,
			ExtraData:      subject + "||" + body,
		}, nil

	case "mark_notifications_read":
		return &ChatOutput{
			Reply:          "Rocket จะทำเครื่องหมายว่าอ่านการแจ้งเตือนทั้งหมดแล้วนะครับ~ ยืนยันเลยไหมครับ? 🔔",
			RequiresAction: true,
			ActionType:     ChatActionTypeMarkNotifRead,
		}, nil

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// --- Helpers ---

func detectIntent(message string) ChatIntent {
	msg := strings.ToLower(message)
	switch {
	case containsAny(msg, "คืนเงิน", "refund", "ขอเงินคืน"):
		return ChatIntentRefundRequest
	case containsAny(msg, "ยกเลิก", "cancel"):
		return ChatIntentCancelTransaction
	case containsAny(msg, "สถานะ", "status", "ตรวจสอบ"):
		return ChatIntentTransactionStatus
	case containsAny(msg, "นโยบาย", "policy", "เงื่อนไข"):
		return ChatIntentRefundPolicy
	default:
		return ChatIntentUnknown
	}
}

func containsAny(s string, keywords ...string) bool {
	for _, k := range keywords {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func systemPrompt() string {
	return `You are Rocket, the AI assistant of the FlyUp platform — a crowdfunding platform for student entrepreneurs (Pioneers) and investors (Boosters).

Your personality: friendly, upbeat, slightly playful, but always accurate and helpful. Use emoji sparingly to keep things fun 🚀

=== LANGUAGE RULE (CRITICAL) ===
- Detect the language of the user's message and reply in the SAME language.
- You ONLY speak Thai (ภาษาไทย) and English. These are the only two languages allowed.
- If the user writes in any other language, politely tell them (in English) that you only support Thai and English.
- Never respond in any language other than Thai or English.

=== WHAT ROCKET CAN HELP WITH ===
- All questions about how FlyUp works (investment, project creation, milestones, voting, profit sharing, etc.)
- Refund and cancellation policy
- Investment guidance for Boosters
- Project creation guidance for Pioneers
- Viewing investments, projects, notifications, disbursements
- Requesting refunds / cancellations
- Voting on milestones
- Filing complaints about projects
- Marking notifications as read
- Anything else related to FlyUp

=== ABOUT FLYUP ===
FlyUp is a milestone-based crowdfunding platform connecting student entrepreneurs (Pioneers) with investors (Boosters). Projects raise funding, execute in phases (milestones), and investors vote to approve each milestone before funds are released to the Pioneer.

User Roles:
- Pioneer: A university student who creates and manages crowdfunding projects. Must register with a university email and verify identity with both a student card and national ID.
- Booster: An investor who funds projects, votes on milestones, and receives profit shares. Must verify identity (KYC) with a national ID before investing.
- Admin: FlyUp staff who approve projects, verify identities, manage refunds, and oversee the platform.

=== INVESTMENT PROCESS ===
1. Register as a Booster and complete identity verification (KYC) with national ID + selfie.
2. Browse projects on the Explore page.
3. Check project details: goal, profit share %, duration, milestones.
4. Click Invest and pay via PromptPay QR Code (expires in 15 minutes).
5. Receive email confirmation once payment succeeds.
6. Track progress in your Portfolio page.
7. Vote on milestone submissions as they are submitted.
8. Receive profit shares quarterly if the project generates profit.

Investment conditions:
- Maximum per transaction: 500,000 THB
- Must complete KYC before investing
- Payment via PromptPay only
- Platform fee: 3% of investment amount
- VAT: 7% on the platform fee (not refundable)

=== PROJECT CREATION (PIONEER) ===
1. Register as a Pioneer using a university email that is registered in the FlyUp system. Must be a current student.
2. Verify identity with student card and national ID.
3. Create a project and fill in all required details (title, description, cover image, funding goal, softcap, hardcap, profit share %, duration, milestones).
4. Submit the project for FlyUp admin review.
5. Once approved, the project enters the funding phase and Boosters can invest.

Project rules:
- Must set both a Softcap (minimum goal) and a Hardcap (maximum cap).
- Must have at least 1 Milestone.
- If funding does not reach Softcap by end date → project fails → all investors get a full refund of principal.
- If project is cancelled by Pioneer → investors receive a proportional refund.
- Only current university students may be Pioneers.

=== PROJECT LIFECYCLE ===
draft → (submit) → pending_review → (admin approves) → funding → (softcap reached) → executing → (all milestones paid) → closed
                                                          → (admin rejects) → rejected
                                                          → (end date passed, softcap not met) → failed
funding → (pioneer requests cancel) → pending_cancel → (admin approves) → cancelled
                                                      → (admin rejects) → back to funding

=== MILESTONE SYSTEM ===
Milestones are phased deliverables the Pioneer must complete after funding succeeds.

Milestone lifecycle:
draft → waiting → active → submitted → (admin reviews) → approved → (voting opens) → (>50% boosters approve) → paid
                                                        → rejected → Pioneer retries (up to 3 times)
                                       submitted → admin rejects → retry deadline applies

Steps:
1. Pioneer creates milestones with acceptance criteria and delivery due date.
2. When project funding succeeds, milestones become active.
3. Pioneer submits milestone with summary, evidence, and attachments.
4. FlyUp admin reviews the submission.
5. If approved, admin opens voting for Boosters.
6. Boosters vote approve or reject (majority >50% required).
7. If approved, a disbursement is created (pending).
8. Admin confirms the payout to Pioneer's bank account.
9. Pioneer can retry up to 3 times if rejected.

=== REFUND POLICY ===
1. Refunds are only available for investments with "verified" status while the project is still in the funding phase.
2. Only the principal amount is refunded. Platform fees and VAT are NOT refunded.
3. Refunds are processed within 3–5 business days.
4. Money is returned to the bank account linked to the investor's account.
5. Admin must approve each refund request.

=== CANCELLATION POLICY ===
- An investment can be cancelled (before project succeeds in funding) by the Booster.
- If the whole project is cancelled by the Pioneer, all investors receive a proportional refund.
- Cancellation requires a reason.

=== PROFIT SHARING ===
- Each project defines a profit share percentage for investors.
- Profits are distributed quarterly by the Pioneer submitting a profit pool.
- Admin confirms individual investor payouts based on their investment proportion.
- Investors can track their profit payouts from the platform.

=== MEETINGS ===
- Pioneers can schedule meetings (online, onsite, or hybrid) for Boosters.
- Meetings can be linked to specific milestones.
- Types: online, onsite, hybrid.
- Boosters can view meetings they are invited to.

=== DISCUSSIONS & THREADS ===
- Both Pioneers and Boosters can create discussion threads on a project page.
- Threads can also be linked to specific project updates.
- Thread statuses: open, answered, hidden, locked.
- Anyone can post messages in open threads.

=== NOTIFICATIONS ===
- Users receive notifications for: new investments, milestone events, votes, project status changes, profit payouts, meetings, verification results, and more.
- Notifications can be marked as read individually or all at once.
- Real-time notifications are delivered via SSE (Server-Sent Events).

=== VERIFICATION (KYC) ===
- Boosters must submit a national ID card + selfie for identity verification.
- Pioneers must submit both a student card and a national ID card.
- Verification status: pending → approved / rejected.
- Admin reviews all verification submissions.
- Users cannot invest or receive payouts without approved verification.

=== COMPLAINTS ===
- Any user can file a complaint about a project (1 complaint per user per project).
- Complaint requires a subject and detailed description.
- Admin reviews, resolves, or rejects complaints.

=== CONTACT & SUPPORT ===
Email: contact@flyup.co.th

=== CRITICAL RULES FOR ROCKET ===
- Answer ONLY questions related to FlyUp. If someone asks about unrelated topics, politely say you can only help with FlyUp-related questions.
- LANGUAGE: Reply in Thai if the user writes in Thai. Reply in English if the user writes in English. If any other language is used, say (in English): "I'm sorry, I only support Thai and English! / ขออภัยครับ ผมรองรับเฉพาะภาษาไทยและอังกฤษเท่านั้นครับ~"
- When a user asks for a refund or cancellation, ALWAYS call get_my_investments first to get the investment list, then call refund_transaction or cancel_transaction with the correct investment_id. NEVER guess investment_id.
- If the user has multiple investments and does not specify which one, show the list and ask which one they want to cancel/refund.
- If the user asks to vote or file a complaint and you are missing required data, ask for the missing info before calling the tool.
- When in doubt, recommend contacting support at contact@flyup.co.th`
}

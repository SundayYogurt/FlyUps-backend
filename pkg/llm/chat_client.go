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
	return `คุณคือ Rocket ผู้ช่วย AI ของแพลตฟอร์ม FlyUp — แพลตฟอร์ม Crowdfunding สำหรับนักศึกษาผู้ประกอบการ (Pioneer) และนักลงทุน (Booster)

บุคลิกของ Rocket: เป็นมิตร สดใส มีความสนุกสนานเล็กน้อย แต่ให้ข้อมูลที่ถูกต้องและเป็นประโยชน์เสมอ ใช้ Emoji พอประมาณ 🚀

=== กฎภาษา (สำคัญมาก) ===
- ตรวจจับภาษาของข้อความ user แล้วตอบกลับด้วย**ภาษาเดียวกัน**
- Rocket รองรับเฉพาะ**ภาษาไทย** และ**ภาษาอังกฤษ** เท่านั้น
- หาก user เขียนภาษาอื่น ให้แจ้งสุภาพ (เป็นภาษาอังกฤษ) ว่ารองรับเฉพาะไทยและอังกฤษ
- ห้ามตอบด้วยภาษาอื่นนอกจากไทยหรืออังกฤษเด็ดขาด

=== Rocket ช่วยอะไรได้บ้าง ===
- คำถามทุกอย่างเกี่ยวกับ FlyUp (การลงทุน, การสร้างโปรเจกต์, Milestone, การโหวต, การแบ่งปันผลกำไร ฯลฯ)
- นโยบายการคืนเงินและการยกเลิก
- คำแนะนำการลงทุนสำหรับ Booster
- คำแนะนำการสร้างโปรเจกต์สำหรับ Pioneer
- ดูรายการลงทุน, โปรเจกต์, การแจ้งเตือน, ประวัติรับเงิน
- ขอคืนเงิน / ยกเลิกการลงทุน
- โหวต Milestone
- ยื่นเรื่องร้องเรียนโปรเจกต์
- ทำเครื่องหมายว่าอ่านการแจ้งเตือนแล้ว
- ทุกอย่างที่เกี่ยวกับ FlyUp

=== เกี่ยวกับ FlyUp ===
FlyUp คือแพลตฟอร์ม Crowdfunding แบบ Milestone-based ที่เชื่อมต่อนักศึกษาผู้ประกอบการ (Pioneer) กับนักลงทุน (Booster) โปรเจกต์ระดมทุนและดำเนินงานเป็นระยะ (Milestone) โดยนักลงทุนโหวตอนุมัติแต่ละ Milestone ก่อนที่เงินจะถูกปล่อยให้ Pioneer

บทบาทผู้ใช้:
- Pioneer: นักศึกษามหาวิทยาลัยที่สร้างและจัดการโปรเจกต์ระดมทุน ต้องลงทะเบียนด้วย Email มหาวิทยาลัยและยืนยันตัวตนด้วยบัตรนักศึกษาและบัตรประชาชน
- Booster: นักลงทุนที่ลงทุนในโปรเจกต์, โหวต Milestone, และรับส่วนแบ่งกำไร ต้องยืนยันตัวตน (KYC) ด้วยบัตรประชาชนก่อนลงทุน
- Admin: ทีมงาน FlyUp ที่อนุมัติโปรเจกต์, ตรวจสอบตัวตน, จัดการคืนเงิน, และดูแลแพลตฟอร์ม

=== ขั้นตอนการลงทุน ===
1. สมัครเป็น Booster และยืนยันตัวตน (KYC) ด้วยบัตรประชาชน + Selfie
2. เรียกดูโปรเจกต์ในหน้า Explore
3. ตรวจสอบรายละเอียดโปรเจกต์: เป้าหมาย, % ผลตอบแทน, ระยะเวลา, Milestone
4. กดลงทุนและชำระเงินผ่าน QR Code PromptPay (หมดอายุใน 15 นาที)
5. รับ Email ยืนยันเมื่อชำระเงินสำเร็จ
6. ติดตามความคืบหน้าในหน้า Portfolio
7. โหวต Milestone เมื่อ Pioneer ส่งรายงาน
8. รับส่วนแบ่งกำไรรายไตรมาสหากโปรเจกต์มีกำไร

เงื่อนไขการลงทุน:
- สูงสุดต่อรายการ: 500,000 บาท
- ต้องยืนยันตัวตน (KYC) ก่อนลงทุน
- ชำระเงินผ่าน PromptPay เท่านั้น
- ค่าธรรมเนียมแพลตฟอร์ม: 3% ของยอดลงทุน
- VAT: 7% ของค่าธรรมเนียม (ไม่คืนเงิน)

=== การสร้างโปรเจกต์ (Pioneer) ===
1. สมัครเป็น Pioneer โดยใช้ Email มหาวิทยาลัยที่ลงทะเบียนในระบบ FlyUp ต้องเป็นนักศึกษาปัจจุบัน
2. ยืนยันตัวตนด้วยบัตรนักศึกษาและบัตรประชาชน
3. สร้างโปรเจกต์และกรอกข้อมูลครบถ้วน (ชื่อ, คำอธิบาย, รูปปก, เป้าหมายระดมทุน, Softcap, Hardcap, % ผลตอบแทน, ระยะเวลา, Milestone)
4. ส่งโปรเจกต์ให้ Admin FlyUp ตรวจสอบ
5. เมื่ออนุมัติแล้ว โปรเจกต์เข้าสู่ระยะระดมทุนและ Booster สามารถลงทุนได้

กฎโปรเจกต์:
- ต้องกำหนดทั้ง Softcap (เป้าหมายขั้นต่ำ) และ Hardcap (เพดานสูงสุด)
- ต้องมีอย่างน้อย 1 Milestone
- หากระดมทุนไม่ถึง Softcap ภายในวันที่กำหนด → โปรเจกต์ล้มเหลว → นักลงทุนทุกคนได้รับเงินต้นคืนเต็มจำนวน
- หาก Pioneer ยกเลิกโปรเจกต์ → นักลงทุนได้รับเงินคืนตามสัดส่วน
- เฉพาะนักศึกษามหาวิทยาลัยปัจจุบันเท่านั้นที่เป็น Pioneer ได้

=== วงจรชีวิตโปรเจกต์ ===
draft → (ส่งตรวจ) → pending_review → (Admin อนุมัติ) → funding → (ถึง Softcap) → executing → (จ่าย Milestone ครบ) → closed
                                                           → (Admin ปฏิเสธ) → rejected
                                                           → (หมดเวลา, ไม่ถึง Softcap) → failed
funding → (Pioneer ขอยกเลิก) → pending_cancel → (Admin อนุมัติ) → cancelled
                                               → (Admin ปฏิเสธ) → กลับไป funding

=== ระบบ Milestone ===
Milestone คือผลงานที่ต้องส่งมอบเป็นระยะหลังระดมทุนสำเร็จ

วงจรชีวิต Milestone:
draft → waiting → active → submitted → (Admin ตรวจสอบ) → approved → (เปิดโหวต) → (>50% Booster อนุมัติ) → paid
                                                         → rejected → Pioneer ลองใหม่ (สูงสุด 3 ครั้ง)

ขั้นตอน:
1. Pioneer สร้าง Milestone พร้อมเกณฑ์การยอมรับและกำหนดส่ง
2. เมื่อระดมทุนสำเร็จ Milestone จะเริ่ม Active
3. Pioneer ส่ง Milestone พร้อมสรุป, หลักฐาน, และเอกสารแนบ
4. Admin FlyUp ตรวจสอบ
5. หากอนุมัติ Admin เปิดให้ Booster โหวต
6. Booster โหวตอนุมัติหรือปฏิเสธ (ต้องการเสียงข้างมาก >50%)
7. หากผ่าน ระบบสร้าง Disbursement (pending)
8. Admin ยืนยันการจ่ายเงินให้ Pioneer
9. Pioneer ลองใหม่ได้สูงสุด 3 ครั้งหากถูกปฏิเสธ

=== นโยบายการคืนเงิน ===
1. คืนเงินได้เฉพาะการลงทุนที่มีสถานะ "verified" และโปรเจกต์ยังอยู่ในระยะ Funding เท่านั้น
2. คืนเฉพาะเงินต้น ค่าธรรมเนียมแพลตฟอร์มและ VAT ไม่คืน
3. ดำเนินการภายใน 3–5 วันทำการ
4. เงินกลับเข้าบัญชีธนาคารที่ผูกกับบัญชีนักลงทุน
5. Admin ต้องอนุมัติคำขอคืนเงินทุกรายการ

=== นโยบายการยกเลิก ===
- Booster ยกเลิกการลงทุนได้ (ก่อนโปรเจกต์ระดมทุนสำเร็จ)
- หาก Pioneer ยกเลิกโปรเจกต์ทั้งหมด นักลงทุนทุกคนได้รับเงินคืนตามสัดส่วน
- การยกเลิกต้องระบุเหตุผล

=== การแบ่งปันผลกำไร ===
- แต่ละโปรเจกต์กำหนด % ผลตอบแทนสำหรับนักลงทุน
- กำไรแจกจ่ายรายไตรมาสโดย Pioneer ส่ง Profit Pool
- Admin ยืนยันการจ่ายเงินให้นักลงทุนแต่ละรายตามสัดส่วนการลงทุน
- นักลงทุนติดตามประวัติรับผลกำไรได้ในแพลตฟอร์ม

=== การประชุม (Meeting) ===
- Pioneer กำหนดการประชุม (ออนไลน์, ออนไซต์, หรือผสม) สำหรับ Booster
- Meeting สามารถเชื่อมกับ Milestone ที่เฉพาะเจาะจงได้
- ประเภท: online, onsite, hybrid
- Booster ดูการประชุมที่ได้รับเชิญได้

=== การสนทนา / Thread ===
- ทั้ง Pioneer และ Booster สร้าง Thread ในหน้าโปรเจกต์ได้
- Thread เชื่อมกับ Project Update ที่เฉพาะเจาะจงได้
- สถานะ Thread: open, answered, hidden, locked
- ทุกคนโพสต์ข้อความใน Thread ที่เปิดอยู่ได้

=== การแจ้งเตือน ===
- ผู้ใช้รับการแจ้งเตือนสำหรับ: การลงทุนใหม่, เหตุการณ์ Milestone, การโหวต, การเปลี่ยนสถานะโปรเจกต์, การรับกำไร, การประชุม, ผลการยืนยันตัวตน และอื่นๆ
- ทำเครื่องหมายว่าอ่านแล้วทีละรายการหรือทั้งหมดพร้อมกันได้
- การแจ้งเตือน Real-time ผ่าน SSE (Server-Sent Events)

=== การยืนยันตัวตน (KYC) ===
- Booster ต้องส่งบัตรประชาชน + Selfie สำหรับยืนยันตัวตน
- Pioneer ต้องส่งทั้งบัตรนักศึกษาและบัตรประชาชน
- สถานะการยืนยัน: pending → approved / rejected
- Admin ตรวจสอบทุกรายการ
- ผู้ใช้ที่ยังไม่ผ่านการยืนยันไม่สามารถลงทุนหรือรับเงินได้

=== การร้องเรียน ===
- ผู้ใช้ทุกคนยื่นเรื่องร้องเรียนโปรเจกต์ได้ (1 เรื่องต่อ 1 โปรเจกต์ต่อ 1 ผู้ใช้)
- ต้องระบุหัวข้อและรายละเอียด
- Admin ตรวจสอบ, แก้ไข, หรือปฏิเสธเรื่องร้องเรียน

=== ติดต่อและสนับสนุน ===
Email: contact@flyup.co.th

=== กฎสำคัญสำหรับ Rocket ===
- ตอบเฉพาะคำถามที่เกี่ยวกับ FlyUp เท่านั้น หากมีคำถามนอกเรื่อง ให้แจ้งสุภาพว่าช่วยได้เฉพาะเรื่อง FlyUp
- ภาษา: ตอบภาษาไทยเมื่อ user พิมพ์ไทย ตอบภาษาอังกฤษเมื่อ user พิมพ์อังกฤษ หากใช้ภาษาอื่น ให้ตอบ (เป็นภาษาอังกฤษ): "I'm sorry, I only support Thai and English! / ขออภัยครับ ผมรองรับเฉพาะภาษาไทยและอังกฤษเท่านั้นครับ~"
- เมื่อ user ขอคืนเงินหรือยกเลิก ต้องเรียก get_my_investments ก่อนเสมอเพื่อดึงรายการลงทุน จากนั้นเรียก refund_transaction หรือ cancel_transaction ด้วย investment_id ที่ถูกต้อง ห้ามเดา investment_id เด็ดขาด
- หาก user มีการลงทุนหลายรายการและไม่ระบุว่าต้องการยกเลิก/คืนเงินรายการไหน ให้แสดงรายการและถามว่าต้องการเลือกรายการใด
- หาก user ขอโหวตหรือร้องเรียนและข้อมูลไม่ครบ ให้ถามข้อมูลที่ขาดก่อนเรียก Tool
- หากไม่แน่ใจ แนะนำให้ติดต่อ Support ที่ contact@flyup.co.th`
}

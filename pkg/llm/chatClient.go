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
	InvestmentID   *uint
	ExtraData      string // ข้อมูลเพิ่มเติมสำหรับ action เช่น vote choice, complaint body
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
			Description: "ดูรายละเอียดโปรเจกต์ เช่น milestone, เป้าหมาย, ผลตอบแทน",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{"type": "integer", "description": "ID ของโปรเจกต์"},
				},
				"required": []string{"project_id"},
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
		if id == 0 {
			return nil, errors.New("project_id is required")
		}
		return &ChatOutput{RequiresAction: true, ActionType: ChatActionTypeGetProjectDetail, InvestmentID: &id}, nil
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
			InvestmentID:   &milestoneID, // reuse field เก็บ milestoneID
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
			InvestmentID:   &projectID, // reuse field เก็บ projectID
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
	return `สวัสดีครับ! ผม Rocket น้องชายสุดน่ารักกวนๆ ของแพลตฟอร์ม FlyUp
ผมพูดภาษาไทยเสมอ มีบุคลิกสดใส ร่าเริง กวนนิดๆ แต่ให้ข้อมูลถูกต้องและเป็นประโยชน์เสมอ
ใช้ emoji ประกอบบ้างให้ดูสนุก แต่ไม่มากเกินไปนะครับ~

สิ่งที่ Rocket ช่วยได้:
- ตอบคำถามนโยบายการคืนเงิน
- แนะนำวิธีลงทุนและสร้างโปรเจกต์
- ดูรายการลงทุน, โปรเจกต์แนะนำ, โปรเจกต์ใหม่, โปรเจกต์ใกล้หมดเวลา
- ดูรายละเอียดโปรเจกต์และ milestone
- ดูการแจ้งเตือน และทำเครื่องหมายว่าอ่านแล้ว
- ดูประวัติรับเงิน (สำหรับ Pioneer)
- ขอคืนเงิน / ยกเลิกการลงทุน
- โหวต milestone
- ร้องเรียนโปรเจกต์

=== วิธีการลงทุนบน FlyUp ===
1. สมัครสมาชิกและยืนยันตัวตนด้วยบัตรประชาชน
2. เลือกโปรเจกต์ที่สนใจจากหน้า Explore
3. ตรวจสอบรายละเอียด เช่น เป้าหมาย ผลตอบแทน ระยะเวลา และ Milestone
4. กดลงทุนและชำระเงินผ่าน PromptPay QR Code (หมดอายุใน 5 นาที)
5. รับการยืนยันทางอีเมล
6. ติดตามความคืบหน้าผ่านหน้า Portfolio
7. รับผลตอบแทนเมื่อโปรเจกต์ดำเนินการสำเร็จตาม Milestone

เงื่อนไขการลงทุน:
- ยอดลงทุนสูงสุดต่อรายการไม่เกิน 500,000 บาท
- ต้องยืนยันตัวตน (KYC) ก่อนลงทุน
- ชำระเงินผ่าน PromptPay เท่านั้น

=== วิธีสร้างโปรเจกต์ระดมทุน ===
1. สมัครในฐานะ Pioneer (ต้องใช้อีเมลมหาวิทยาลัยที่ลงทะเบียนในระบบ)
2. ยืนยันตัวตนด้วยบัตรนักศึกษาและบัตรประชาชน
3. สร้างโปรเจกต์และกรอกรายละเอียดให้ครบ
4. ส่งให้ทีมงาน FlyUp ตรวจสอบ
5. เมื่อผ่านการอนุมัติ โปรเจกต์จะเริ่มระดมทุน

กฎเกณฑ์โปรเจกต์:
- ต้องกำหนด Softcap และ Hardcap
- ต้องมี Milestone อย่างน้อย 1 รายการ
- ถ้าระดมทุนไม่ถึง Softcap โปรเจกต์ล้มเหลว นักลงทุนได้เงินคืนทั้งหมด
- ถ้าโปรเจกต์ถูกยกเลิก นักลงทุนได้รับเงินคืนตามสัดส่วน

=== นโยบายการคืนเงิน ===
1. คืนได้เฉพาะ investment ที่ verified และโปรเจกต์อยู่ใน funding เท่านั้น
2. ได้คืนแค่ยอดหลัก ค่าธรรมเนียมและ VAT ไม่ได้คืน
3. รอรับเงินคืนภายใน 3-5 วันทำการ
4. โอนกลับบัญชีธนาคารที่ผูกไว้

ติดต่อ support: email contact@flyup.co.th

กฎสำคัญ:
- ตอบเป็นภาษาไทยเสมอ พูดแบบ Rocket น่ารักกวนๆ
- ตอบเฉพาะเรื่องที่เกี่ยวกับ FlyUp เท่านั้น ถ้าถามนอกเรื่องให้บอกว่าช่วยไม่ได้แบบน่ารักๆ
- ถ้า user ขอคืนเงินหรือยกเลิกการลงทุน ให้เรียก get_my_investments ก่อนเสมอ เพื่อดูว่า user มีการลงทุนอะไรบ้าง แล้วค่อยเรียก refund_transaction หรือ cancel_transaction พร้อม investment_id ที่ถูกต้อง ห้ามเดา investment_id เด็ดขาด
- ถ้า user มีการลงทุนหลายรายการและไม่ได้ระบุว่าอยากยกเลิกอันไหน ให้แสดงรายการและถามว่าอยากยกเลิกอันไหน
- ถ้า user ขอโหวต/ร้องเรียน ให้ถามข้อมูลที่ขาดก่อน แล้วค่อยเรียก tool
- ถ้าไม่แน่ใจ ให้แนะนำให้ติดต่อ support`
}

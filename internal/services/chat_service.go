package services

import (
	"context"
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/repository"
	"flyup/pkg/llm"
	"fmt"
	"strings"
)

type ChatService interface {
	SendMessage(userID uint, req dto.SendChatMessageRequest) (*dto.SendChatMessageResponse, error)
	ConfirmAction(userID uint, req dto.ConfirmChatActionRequest) (*dto.ConfirmChatActionResponse, error)
}

// ChatAIClient เป็น interface ที่ services รู้จัก
// ทำให้ swap implementation หรือ mock ใน test ได้
type ChatAIClient interface {
	GenerateReply(input ChatAIInput) (*ChatAIOutput, error)
}

type ChatAIInput struct {
	UserID   uint
	Message  string
	History  []domain.ChatMessage
	Language string
}

type ChatAIOutput struct {
	Intent         domain.ChatIntent
	Reply          string
	RequiresAction bool
	ActionType     domain.ChatActionType
	InvestmentID   *uint
	MilestoneID    *uint
	ProjectID      *uint
	ExtraData      string
}

// flyUpChatAIAdapter แปลง llm.FlyUpChatClient ให้ implement ChatAIClient
// เป็น adapter layer ที่แปลง pkg/llm types ↔ domain types
type flyUpChatAIAdapter struct {
	client *llm.FlyUpChatClient
}

func NewChatAIClient(apiKey, model string) ChatAIClient {
	return &flyUpChatAIAdapter{
		client: llm.NewFlyUpChatClientWithModel(apiKey, model),
	}
}

func (a *flyUpChatAIAdapter) GenerateReply(input ChatAIInput) (*ChatAIOutput, error) {
	// แปลง domain.ChatMessage → llm.ChatMessage
	history := make([]llm.ChatMessage, 0, len(input.History))
	for _, h := range input.History {
		history = append(history, llm.ChatMessage{
			Role:    string(h.Role),
			Content: h.Content,
		})
	}

	out, err := a.client.GenerateReply(llm.ChatInput{
		UserID:   input.UserID,
		Message:  input.Message,
		History:  history,
		Language: input.Language,
	})
	if err != nil {
		return nil, err
	}

	// แปลง llm types → domain types
	return &ChatAIOutput{
		Intent:         domain.ChatIntent(out.Intent),
		Reply:          out.Reply,
		RequiresAction: out.RequiresAction,
		ActionType:     domain.ChatActionType(out.ActionType),
		InvestmentID:   out.InvestmentID,
		MilestoneID:    out.MilestoneID,
		ProjectID:      out.ProjectID,
		ExtraData:      out.ExtraData,
	}, nil
}

type chatService struct {
	chatRepo      repository.ChatRepository
	aiClient      ChatAIClient
	investmentSvc InvestmentService
	projectSvc    ProjectService
	notifSvc      NotificationService
	disburseSvc   DisbursementService
	complaintSvc  ComplaintService
}

func NewChatService(
	chatRepo repository.ChatRepository,
	aiClient ChatAIClient,
	investmentSvc InvestmentService,
	projectSvc ProjectService,
	notifSvc NotificationService,
	disburseSvc DisbursementService,
	complaintSvc ComplaintService,
) ChatService {
	return &chatService{
		chatRepo:      chatRepo,
		aiClient:      aiClient,
		investmentSvc: investmentSvc,
		projectSvc:    projectSvc,
		notifSvc:      notifSvc,
		disburseSvc:   disburseSvc,
		complaintSvc:  complaintSvc,
	}
}

func (s *chatService) SendMessage(userID uint, req dto.SendChatMessageRequest) (*dto.SendChatMessageResponse, error) {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, errors.New("message is required")
	}

	if s.aiClient == nil {
		return nil, errors.New("chat ai client is required")
	}

	session, err := s.getOrCreateSession(userID, req.SessionID)
	if err != nil {
		return nil, err
	}

	userMessage := &domain.ChatMessage{
		SessionID: session.ID,
		Role:      domain.ChatMessageRoleUser,
		Content:   message,
		Intent:    domain.ChatIntentUnknown,
	}

	if err := s.chatRepo.CreateMessage(userMessage); err != nil {
		return nil, errors.New("failed to save message")
	}

	// ดึง history ก่อน save userMessage ไม่ได้ เพราะ userMessage เพิ่งถูก save
	// ดึงหลัง save เพื่อให้ history รวม message ปัจจุบันด้วย
	history, err := s.chatRepo.GetMessagesBySessionID(session.ID)
	if err != nil {
		return nil, errors.New("failed to load chat history")
	}

	aiOutput, err := s.aiClient.GenerateReply(ChatAIInput{
		UserID:   userID,
		Message:  message,
		History:  history,
		Language: "th",
	})
	if err != nil {
		return nil, fmt.Errorf("ai error: %w", err)
	}

	if err := s.validateAIOutput(aiOutput); err != nil {
		return nil, err
	}

	userMessage.Intent = aiOutput.Intent
	if err := s.chatRepo.UpdateMessage(userMessage); err != nil {
		return nil, errors.New("failed to update message intent")
	}

	var assistantMessage *domain.ChatMessage
	var action *domain.ChatAction

	if aiOutput.RequiresAction {
		// query actions — execute ทันที ไม่ต้องรอ confirm
		if aiOutput.ActionType == domain.ChatActionTypeGetInvestments ||
			aiOutput.ActionType == domain.ChatActionTypeGetProjects ||
			aiOutput.ActionType == domain.ChatActionTypeGetProjectDetail ||
			aiOutput.ActionType == domain.ChatActionTypeGetEndingSoon ||
			aiOutput.ActionType == domain.ChatActionTypeGetNewProjects ||
			aiOutput.ActionType == domain.ChatActionTypeGetNotifications ||
			aiOutput.ActionType == domain.ChatActionTypeGetDisbursements {

			replyContent, err := s.executeQueryAction(userID, aiOutput)
			if err != nil {
				replyContent = "ไม่สามารถดึงข้อมูลได้ในขณะนี้ครับ~ ลองใหม่อีกทีนะครับ 😅"
			}

			assistantMessage = &domain.ChatMessage{
				SessionID: session.ID,
				Role:      domain.ChatMessageRoleAssistant,
				Content:   replyContent,
				Intent:    aiOutput.Intent,
			}
			if err := s.chatRepo.CreateMessage(assistantMessage); err != nil {
				return nil, errors.New("failed to save assistant message")
			}
		} else {
			// refund/cancel/vote/complaint/notif — สร้าง action รอ confirm
			assistantMessage = &domain.ChatMessage{
				SessionID: session.ID,
				Role:      domain.ChatMessageRoleAssistant,
				Content:   aiOutput.Reply,
				Intent:    aiOutput.Intent,
			}
			if err := s.chatRepo.CreateMessage(assistantMessage); err != nil {
				return nil, errors.New("failed to save assistant message")
			}

			action = &domain.ChatAction{
				SessionID:    session.ID,
				UserID:       userID,
				Type:         aiOutput.ActionType,
				Status:       domain.ChatActionStatusPending,
				InvestmentID: aiOutput.InvestmentID,
				MilestoneID:  aiOutput.MilestoneID,
				ProjectID:    aiOutput.ProjectID,
			}
			if aiOutput.ExtraData != "" {
				action.Payload = &aiOutput.ExtraData
			}
			if err := s.chatRepo.CreateAction(action); err != nil {
				return nil, errors.New("failed to create action")
			}
		}
	} else {
		// text reply ปกติ
		assistantMessage = &domain.ChatMessage{
			SessionID: session.ID,
			Role:      domain.ChatMessageRoleAssistant,
			Content:   aiOutput.Reply,
			Intent:    aiOutput.Intent,
		}
		if err := s.chatRepo.CreateMessage(assistantMessage); err != nil {
			return nil, errors.New("failed to save assistant message")
		}
	}

	return &dto.SendChatMessageResponse{
		Session: s.toChatSessionResponse(session),
		Message: s.toChatMessageResponse(userMessage),
		Action:  s.toChatActionResponsePointer(action),
		Reply:   s.toChatMessageResponse(assistantMessage),
	}, nil
}

func (s *chatService) ConfirmAction(userID uint, req dto.ConfirmChatActionRequest) (*dto.ConfirmChatActionResponse, error) {
	action, err := s.chatRepo.GetActionByIDAndUserID(req.ActionID, userID)
	if err != nil {
		return nil, errors.New("action not found")
	}

	if action.Status != domain.ChatActionStatusPending {
		return nil, errors.New("action is not pending")
	}

	// user ปฏิเสธ → จบเลย
	if !req.Confirm {
		action.Status = domain.ChatActionStatusRejected
		if err := s.chatRepo.UpdateAction(action); err != nil {
			return nil, errors.New("failed to update action")
		}
		return s.buildConfirmResponse(action, s.rejectReply(action.Type))
	}

	// user ยืนยัน → execute จริง
	replyContent, execErr := s.executeAction(userID, action)

	if execErr != nil {
		// execute ล้มเหลว → mark failed + แจ้ง user
		action.Status = domain.ChatActionStatusFailed
		_ = s.chatRepo.UpdateAction(action)
		return s.buildConfirmResponse(action, fmt.Sprintf("ไม่สามารถดำเนินการได้: %s", execErr.Error()))
	}

	action.Status = domain.ChatActionStatusCompleted
	if err := s.chatRepo.UpdateAction(action); err != nil {
		return nil, errors.New("failed to update action")
	}

	return s.buildConfirmResponse(action, replyContent)
}

// executeQueryAction execute action ที่เป็น read-only ไม่ต้อง confirm
func (s *chatService) executeQueryAction(userID uint, aiOutput *ChatAIOutput) (string, error) {
	switch aiOutput.ActionType {

	case domain.ChatActionTypeGetInvestments:
		// ใช้ ListUserInvestments เพื่อได้ investment ID ครบ
		investments, err := s.investmentSvc.ListUserInvestments(userID)
		if err != nil {
			return "", err
		}
		// filter เฉพาะที่ verified (คืนเงินได้)
		var active []domain.Investment
		for _, inv := range investments {
			if inv.Status == domain.InvestmentVerified || inv.Status == domain.InvestmentRefundPending {
				active = append(active, inv)
			}
		}
		if len(active) == 0 {
			return "ยังไม่มีรายการลงทุนที่ active เลยนะครับ~ 😊", nil
		}

		// ดึงชื่อโปรเจกต์
		projectNames := map[uint]string{}
		if projects, err := s.investmentSvc.ListInvestedProjects(userID); err == nil {
			for _, p := range projects {
				projectNames[p.ProjectID] = p.Title
			}
		}

		reply := fmt.Sprintf("รายการลงทุนของคุณทั้งหมด %d รายการครับ~ 📊\n\n", len(active))
		for i, inv := range active {
			title := projectNames[inv.ProjectID]
			if title == "" {
				title = fmt.Sprintf("Project #%d", inv.ProjectID)
			}
			reply += fmt.Sprintf("%d. %s (Investment ID: %d)\n   • ยอดลงทุน: %.2f บาท\n   • สถานะ: %s\n\n",
				i+1, title, inv.ID, inv.TotalAmount, string(inv.Status))
		}
		reply += "ถ้าอยากขอคืนเงินรายการไหน บอกชื่อโปรเจกต์หรือหมายเลขในลิสต์ได้เลยนะครับ~ 😊"
		return reply, nil

	case domain.ChatActionTypeGetProjects:
		projects, err := s.projectSvc.GetProjectRecommendations()
		if err != nil {
			return "", err
		}
		if len(projects) == 0 {
			return "ตอนนี้ยังไม่มีโปรเจกต์แนะนำเลยครับ~ รอติดตามได้นะครับ 👀", nil
		}
		reply := fmt.Sprintf("โปรเจกต์แนะนำสุดเจ๋ง %d รายการครับ~ ✨\n\n", len(projects))
		for i, p := range projects {
			progress := 0.0
			if p.FundingGoal > 0 {
				progress = (p.CurrentFunding / p.FundingGoal) * 100
			}
			reply += fmt.Sprintf("%d. %s\n   • เป้าหมาย: %.0f บาท\n   • ระดมทุนแล้ว: %.0f%% (%.0f บาท)\n   • ผลตอบแทน: %.1f%%\n   • ลงทุนขั้นต่ำ: %.0f บาท\n\n",
				i+1, p.Title, p.FundingGoal, progress, p.CurrentFunding, p.ProfitSharePct, p.MinInvestAmount)
		}
		reply += "สนใจโปรเจกต์ไหน ถามได้เลยนะครับ~ 🚀"
		return reply, nil

	case domain.ChatActionTypeGetProjectDetail:
		var p *domain.Project
		var err error

		if aiOutput.ProjectID != nil && *aiOutput.ProjectID != 0 {
			p, err = s.projectSvc.GetPublicProjectByID(context.Background(), *aiOutput.ProjectID)
		} else if aiOutput.ExtraData != "" {
			// มีแค่ชื่อ → ค้นหาจาก public projects ด้วย search
			name := aiOutput.ExtraData
			all, searchErr := s.projectSvc.GetPublicProjects(dto.PublicProjectFilter{Search: name})
			if searchErr != nil {
				return "", errors.New("ไม่สามารถค้นหาโปรเจกต์ได้ครับ~ 😅")
			}
			if len(all) == 0 {
				return fmt.Sprintf("ไม่พบโปรเจกต์ชื่อ \"%s\" ครับ~ ลองพิมพ์ชื่อใหม่อีกทีนะ 🔍", name), nil
			}
			p = &all[0]
		} else {
			return "บอกชื่อหรือ ID โปรเจกต์มาด้วยนะครับ~ 😅", nil
		}

		if err != nil {
			return "", errors.New("ไม่พบโปรเจกต์นี้ครับ~ ลองเช็คอีกทีนะ 🔍")
		}

		progress := 0.0
		if p.FundingGoal > 0 {
			progress = (p.CurrentFunding / p.FundingGoal) * 100
		}
		reply := fmt.Sprintf("รายละเอียดโปรเจกต์ \"%s\" ครับ~ 📋\n\n", p.Title)
		reply += fmt.Sprintf("• เป้าหมาย: %.0f บาท\n• ระดมทุนแล้ว: %.0f%% (%.0f บาท)\n• ผลตอบแทน: %.1f%%\n• ลงทุนขั้นต่ำ: %.0f บาท\n• สถานะ: %s\n",
			p.FundingGoal, progress, p.CurrentFunding, p.ProfitSharePct, p.MinInvestAmount, string(p.State))
		if len(p.Milestones) > 0 {
			reply += fmt.Sprintf("\nMilestone ทั้งหมด %d รายการ:\n", len(p.Milestones))
			for _, m := range p.Milestones {
				reply += fmt.Sprintf("  • Phase %d: %s (%.0f%%)\n", m.PhaseNo, m.Title, float64(m.PercentRelease))
			}
		}
		return reply, nil

	case domain.ChatActionTypeGetEndingSoon:
		projects, err := s.projectSvc.GetProjectEndingSoon()
		if err != nil {
			return "", err
		}
		if len(projects) == 0 {
			return "ตอนนี้ไม่มีโปรเจกต์ที่ใกล้หมดเวลาครับ~ 😌", nil
		}
		reply := fmt.Sprintf("โปรเจกต์ที่ใกล้หมดเวลาระดมทุน %d รายการครับ~ รีบๆ หน่อยนะ! ⏰\n\n", len(projects))
		for i, p := range projects {
			progress := 0.0
			if p.FundingGoal > 0 {
				progress = (p.CurrentFunding / p.FundingGoal) * 100
			}
			reply += fmt.Sprintf("%d. %s\n   • ระดมทุนแล้ว: %.0f%%\n   • หมดเวลา: %s\n\n",
				i+1, p.Title, progress, p.EndDate.Format("02/01/2006"))
		}
		return reply, nil

	case domain.ChatActionTypeGetNewProjects:
		projects, err := s.projectSvc.GetNewProjects()
		if err != nil {
			return "", err
		}
		if len(projects) == 0 {
			return "ยังไม่มีโปรเจกต์ใหม่เลยครับ~ คอยติดตามได้นะ 👀", nil
		}
		reply := fmt.Sprintf("โปรเจกต์ใหม่มาแรง %d รายการครับ~ 🆕\n\n", len(projects))
		for i, p := range projects {
			reply += fmt.Sprintf("%d. %s\n   • เป้าหมาย: %.0f บาท\n   • ผลตอบแทน: %.1f%%\n   • ลงทุนขั้นต่ำ: %.0f บาท\n\n",
				i+1, p.Title, p.FundingGoal, p.ProfitSharePct, p.MinInvestAmount)
		}
		return reply, nil

	case domain.ChatActionTypeGetNotifications:
		notifs, _, err := s.notifSvc.GetNotifications(userID, 1, 10)
		if err != nil {
			return "", err
		}
		unread, _ := s.notifSvc.CountUnread(userID)
		if len(notifs) == 0 {
			return "ยังไม่มีการแจ้งเตือนเลยครับ~ 🔕", nil
		}
		reply := fmt.Sprintf("การแจ้งเตือนล่าสุด (ยังไม่อ่าน %d รายการ) ครับ~ 🔔\n\n", unread)
		for i, n := range notifs {
			read := "✅"
			if !n.IsRead {
				read = "🔴"
			}
			reply += fmt.Sprintf("%s %d. %s\n   %s\n\n", read, i+1, n.Title, n.Body)
		}
		return reply, nil

	case domain.ChatActionTypeGetDisbursements:
		payouts, err := s.disburseSvc.ListMyPayouts(userID)
		if err != nil {
			return "", err
		}
		if len(payouts) == 0 {
			return "ยังไม่มีประวัติรับเงินเลยครับ~ 💸", nil
		}
		reply := fmt.Sprintf("ประวัติรับเงินจาก Milestone ทั้งหมด %d รายการครับ~ 💰\n\n", len(payouts))
		for i, p := range payouts {
			reply += fmt.Sprintf("%d. %s — Phase %d\n   • ยอด: %.2f บาท\n   • สถานะ: %s\n\n",
				i+1, p.ProjectTitle, p.PhaseNo, p.Amount, p.Status)
		}
		return reply, nil

	default:
		return "", fmt.Errorf("unknown query action: %s", aiOutput.ActionType)
	}
}

// executeAction เรียก services จริงตาม action type
func (s *chatService) executeAction(userID uint, action *domain.ChatAction) (string, error) {
	switch action.Type {

	case domain.ChatActionTypeRefundTransaction, domain.ChatActionTypeCancelTransaction:
		if action.InvestmentID == nil {
			return "", errors.New("ไม่พบข้อมูล investment กรุณาระบุหมายเลขใหม่อีกครั้งนะครับ~ 😅")
		}
		note := "ขอคืนเงินผ่าน Rocket AI Chat"
		if action.Type == domain.ChatActionTypeCancelTransaction {
			note = "ยกเลิกการลงทุนผ่าน Rocket AI Chat"
		}
		res, err := s.investmentSvc.RefundInvestment(userID, *action.InvestmentID, note)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(
			"เรียบร้อยแล้วครับ~ 🎉\n• หมายเลขอ้างอิง: %s\n• ยอดที่จะได้รับคืน: %.2f บาท\n• ค่าธรรมเนียมที่หัก: %.2f บาท\n\nรอรับเงินคืนภายใน 3-5 วันทำการนะครับ~ 💸",
			res.ReferenceNumber, res.RefundAmount, res.FeesDeducted,
		), nil

	case domain.ChatActionTypeVoteMilestone:
		if action.MilestoneID == nil {
			return "", errors.New("ไม่พบ milestone_id ครับ~ 😅")
		}
		choice := domain.MilestoneVoteChoice("")
		if action.Payload != nil {
			choice = domain.MilestoneVoteChoice(*action.Payload)
		}
		_, err := s.investmentSvc.VoteMilestone(userID, *action.MilestoneID, choice)
		if err != nil {
			return "", err
		}
		choiceText := "อนุมัติ ✅"
		if choice == "reject" {
			choiceText = "ปฏิเสธ ❌"
		}
		return fmt.Sprintf("โหวต%s milestone #%d เรียบร้อยแล้วครับ~ 🗳️", choiceText, *action.MilestoneID), nil

	case domain.ChatActionTypeFileComplaint:
		if action.ProjectID == nil || action.Payload == nil || *action.Payload == "" {
			return "", errors.New("ข้อมูลไม่ครบครับ~ 😅")
		}
		parts := strings.SplitN(*action.Payload, "||", 2)
		if len(parts) != 2 {
			return "", errors.New("รูปแบบข้อมูลไม่ถูกต้องครับ~")
		}
		_, err := s.complaintSvc.Create(userID, dto.CreateComplaintRequest{
			ProjectID: *action.ProjectID,
			Subject:   parts[0],
			Body:      parts[1],
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("ยื่นเรื่องร้องเรียนโปรเจกต์ #%d เรียบร้อยแล้วครับ~ ทีมงานจะตรวจสอบให้เร็วๆ นี้นะ 📝", *action.ProjectID), nil

	case domain.ChatActionTypeMarkNotifRead:
		if err := s.notifSvc.MarkAllAsRead(userID); err != nil {
			return "", err
		}
		return "ทำเครื่องหมายว่าอ่านการแจ้งเตือนทั้งหมดแล้วครับ~ 🔔✅", nil

	default:
		return "", fmt.Errorf("ไม่รองรับ action ประเภท: %s", action.Type)
	}
}

// buildConfirmResponse สร้าง response พร้อม save reply message
func (s *chatService) buildConfirmResponse(action *domain.ChatAction, replyContent string) (*dto.ConfirmChatActionResponse, error) {
	reply := &domain.ChatMessage{
		SessionID: action.SessionID,
		Role:      domain.ChatMessageRoleAssistant,
		Content:   replyContent,
		Intent:    s.intentFromActionType(action.Type),
	}

	if err := s.chatRepo.CreateMessage(reply); err != nil {
		return nil, errors.New("failed to save reply message")
	}

	return &dto.ConfirmChatActionResponse{
		Action: s.toChatActionResponse(action),
		Reply:  s.toChatMessageResponse(reply),
	}, nil
}

func (s *chatService) rejectReply(actionType domain.ChatActionType) string {
	switch actionType {
	case domain.ChatActionTypeRefundTransaction:
		return "ยกเลิกคำขอคืนเงินเรียบร้อยแล้วครับ หากต้องการดำเนินการในภายหลัง สามารถแจ้งได้เลยครับ"
	case domain.ChatActionTypeCancelTransaction:
		return "ยกเลิกคำขอยกเลิกธุรกรรมเรียบร้อยแล้วครับ การลงทุนของคุณยังคงอยู่ตามเดิมครับ"
	default:
		return "ยกเลิกคำขอเรียบร้อยแล้วครับ"
	}
}

func (s *chatService) getOrCreateSession(userID uint, sessionID *uint) (*domain.ChatSession, error) {
	if sessionID != nil {
		return s.chatRepo.GetSessionByIDAndUserID(*sessionID, userID)
	}

	session := &domain.ChatSession{
		UserID: userID,
		Status: domain.ChatSessionStatusActive,
	}

	if err := s.chatRepo.CreateSession(session); err != nil {
		return nil, errors.New("failed to create session")
	}

	return session, nil
}

func (s *chatService) validateAIOutput(output *ChatAIOutput) error {
	if output == nil {
		return errors.New("ai output is required")
	}

	if output.Intent == "" {
		output.Intent = domain.ChatIntentUnknown
	}

	if output.RequiresAction && output.ActionType == "" {
		return errors.New("ai action type is required")
	}

	if !output.RequiresAction && output.ActionType != "" {
		output.ActionType = ""
	}

	// Query actions ไม่มี reply จาก AI — services จะ fill เอง
	isQueryAction := output.RequiresAction && (output.ActionType == domain.ChatActionTypeGetInvestments ||
		output.ActionType == domain.ChatActionTypeGetProjects ||
		output.ActionType == domain.ChatActionTypeGetProjectDetail ||
		output.ActionType == domain.ChatActionTypeGetEndingSoon ||
		output.ActionType == domain.ChatActionTypeGetNewProjects ||
		output.ActionType == domain.ChatActionTypeGetNotifications ||
		output.ActionType == domain.ChatActionTypeGetDisbursements)
	if !isQueryAction && strings.TrimSpace(output.Reply) == "" {
		return errors.New("ai reply is required")
	}

	return nil
}

func (s *chatService) intentFromActionType(actionType domain.ChatActionType) domain.ChatIntent {
	switch actionType {
	case domain.ChatActionTypeRefundTransaction:
		return domain.ChatIntentRefundRequest
	case domain.ChatActionTypeCancelTransaction:
		return domain.ChatIntentCancelTransaction
	default:
		return domain.ChatIntentUnknown
	}
}

// --- mapper helpers ---

func (s *chatService) toChatSessionResponse(session *domain.ChatSession) dto.ChatSessionResponse {
	return dto.ChatSessionResponse{
		ID:     session.ID,
		UserID: session.UserID,
		Status: session.Status,
	}
}

func (s *chatService) toChatMessageResponse(message *domain.ChatMessage) dto.ChatMessageResponse {
	return dto.ChatMessageResponse{
		ID:        message.ID,
		SessionID: message.SessionID,
		Role:      message.Role,
		Content:   message.Content,
		Intent:    message.Intent,
	}
}

func (s *chatService) toChatActionResponse(action *domain.ChatAction) dto.ChatActionResponse {
	return dto.ChatActionResponse{
		ID:           action.ID,
		SessionID:    action.SessionID,
		Type:         action.Type,
		Status:       action.Status,
		InvestmentID: action.InvestmentID,
	}
}

func (s *chatService) toChatActionResponsePointer(action *domain.ChatAction) *dto.ChatActionResponse {
	if action == nil {
		return nil
	}
	res := s.toChatActionResponse(action)
	return &res
}

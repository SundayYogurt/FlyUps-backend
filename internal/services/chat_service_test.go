package services

import (
	"context"
	"errors"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"mime/multipart"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock: ChatRepository ---

type mockChatRepo struct{ mock.Mock }

func (m *mockChatRepo) CreateSession(session *domain.ChatSession) error {
	return m.Called(session).Error(0)
}

func (m *mockChatRepo) GetSessionByIDAndUserID(sessionID uint, userID uint) (*domain.ChatSession, error) {
	args := m.Called(sessionID, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.ChatSession), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatRepo) CreateMessage(message *domain.ChatMessage) error {
	return m.Called(message).Error(0)
}

func (m *mockChatRepo) GetMessagesBySessionID(sessionID uint) ([]domain.ChatMessage, error) {
	args := m.Called(sessionID)
	if v := args.Get(0); v != nil {
		return v.([]domain.ChatMessage), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatRepo) UpdateMessage(message *domain.ChatMessage) error {
	return m.Called(message).Error(0)
}

func (m *mockChatRepo) CreateAction(action *domain.ChatAction) error {
	return m.Called(action).Error(0)
}

func (m *mockChatRepo) GetActionByIDAndUserID(actionID uint, userID uint) (*domain.ChatAction, error) {
	args := m.Called(actionID, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.ChatAction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatRepo) UpdateAction(action *domain.ChatAction) error {
	return m.Called(action).Error(0)
}

// --- Mock: ChatAIClient ---

type mockChatAI struct{ mock.Mock }

func (m *mockChatAI) GenerateReply(input ChatAIInput) (*ChatAIOutput, error) {
	args := m.Called(input)
	if v := args.Get(0); v != nil {
		return v.(*ChatAIOutput), args.Error(1)
	}
	return nil, args.Error(1)
}

// --- Mock: InvestmentService (for chatService) ---

type mockChatInvestSvc struct{ mock.Mock }

func (m *mockChatInvestSvc) ListUserInvestments(boosterUserID uint) ([]domain.Investment, error) {
	args := m.Called(boosterUserID)
	if v := args.Get(0); v != nil {
		return v.([]domain.Investment), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatInvestSvc) ListInvestedProjects(boosterUserID uint) ([]dto.InvestedProjectItem, error) {
	args := m.Called(boosterUserID)
	if v := args.Get(0); v != nil {
		return v.([]dto.InvestedProjectItem), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatInvestSvc) RefundInvestment(boosterUserID uint, investmentID uint, note string) (*dto.RefundResponse, error) {
	args := m.Called(boosterUserID, investmentID, note)
	if v := args.Get(0); v != nil {
		return v.(*dto.RefundResponse), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatInvestSvc) VoteMilestone(boosterUserID uint, milestoneID uint, choice domain.MilestoneVoteChoice) (*domain.MilestoneVote, error) {
	args := m.Called(boosterUserID, milestoneID, choice)
	if v := args.Get(0); v != nil {
		return v.(*domain.MilestoneVote), args.Error(1)
	}
	return nil, args.Error(1)
}

// stubs
func (m *mockChatInvestSvc) GetInvestment(boosterUserID, investmentID uint) (*domain.Investment, *domain.Transaction, error) {
	return nil, nil, nil
}
func (m *mockChatInvestSvc) GenerateContractHTML(boosterUserID, investmentID uint) ([]byte, error) {
	return nil, nil
}
func (m *mockChatInvestSvc) CreateInvestment(boosterUserID uint, boosterEmail string, req dto.CreateInvestmentRequest) (*dto.InvestmentResponse, error) {
	return nil, nil
}
func (m *mockChatInvestSvc) HandleStripeWebhook(payload []byte, sigHeader string) error { return nil }
func (m *mockChatInvestSvc) ApproveRefund(investmentID uint) error                       { return nil }
func (m *mockChatInvestSvc) ListRefundRequests() ([]dto.RefundRequestItem, error)         { return nil, nil }
func (m *mockChatInvestSvc) GetProjectInvestors(projectID uint) ([]dto.ProjectInvestorItem, error) {
	return nil, nil
}
func (m *mockChatInvestSvc) GetMyVote(boosterUserID, milestoneID uint) (*domain.MilestoneVote, error) {
	return nil, nil
}
func (m *mockChatInvestSvc) GetMilestoneVoters(pioneerUserID, milestoneID uint) ([]dto.MilestoneVoterItem, error) {
	return nil, nil
}
func (m *mockChatInvestSvc) RefundProjectInvestments(project domain.Project) {}
func (m *mockChatInvestSvc) GetCancelPreview(projectID uint) (*dto.CancelPreviewResponse, error) {
	return nil, nil
}
func (m *mockChatInvestSvc) FinalizeVotingIfExpired(milestoneID uint) error { return nil }
func (m *mockChatInvestSvc) GetTotalFunding() (float64, error)              { return 0, nil }
func (m *mockChatInvestSvc) GetUniqueBoostersCount() (int64, error)         { return 0, nil }
func (m *mockChatInvestSvc) SyncProjectPrincipalAmounts(projectID uint) error { return nil }

// --- Mock: ProjectService (for chatService) ---

type mockChatProjectSvc struct{ mock.Mock }

func (m *mockChatProjectSvc) GetProjectRecommendations() ([]domain.Project, error) {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.([]domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatProjectSvc) GetNewProjects() ([]domain.Project, error) {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.([]domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatProjectSvc) GetProjectEndingSoon() ([]domain.Project, error) {
	args := m.Called()
	if v := args.Get(0); v != nil {
		return v.([]domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatProjectSvc) GetPublicProjectByID(ctx context.Context, id uint) (*domain.Project, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChatProjectSvc) GetPublicProjects(filter dto.PublicProjectFilter) ([]domain.Project, error) {
	args := m.Called(filter)
	if v := args.Get(0); v != nil {
		return v.([]domain.Project), args.Error(1)
	}
	return nil, args.Error(1)
}

// stubs — project core
func (m *mockChatProjectSvc) CreateProject(ownerID uint) (*domain.Project, error) { return nil, nil }
func (m *mockChatProjectSvc) GetProjectDetailByID(id uint) (*domain.Project, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) UpdateProject(projectID uint, input dto.UpdateProjectRequest, user domain.User) (*domain.Project, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) DeleteProject(projectID uint, user domain.User) error  { return nil }
func (m *mockChatProjectSvc) GetMyProjects(ownerID uint) ([]domain.Project, error)  { return nil, nil }
func (m *mockChatProjectSvc) GetOwnerProjectByID(id uint, ownerID uint) (*domain.Project, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) GetProjectsByCategory(categoryID uint) ([]domain.Project, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) UpdateProjectStatus(projectID uint, newState domain.ProjectState, newStatus domain.ProjectStatus) error {
	return nil
}
func (m *mockChatProjectSvc) GetExecutingProjects() ([]domain.Project, error) { return nil, nil }
func (m *mockChatProjectSvc) GetPublicProjectBySlug(ctx context.Context, slug string) (*domain.Project, error) {
	return nil, nil
}

// stubs — media
func (m *mockChatProjectSvc) AttachProjectMedia(ctx context.Context, projectID uint, url string, mediaTypes []domain.MediaType, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) GetProjectMedia(projectID uint) ([]domain.ProjectMedia, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) UpdateProjectMedia(mediaID uint, input *domain.ProjectMedia, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) DeleteProjectMedia(mediaID uint, user domain.User) error { return nil }

// stubs — milestone
func (m *mockChatProjectSvc) CreateMilestone(projectID uint, input dto.CreateMilestoneRequest, user domain.User) (*domain.Milestone, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) UpdateMilestone(milestoneID uint, input dto.UpdateMilestoneRequest, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) DeleteMilestone(milestoneID uint, user domain.User) error { return nil }
func (m *mockChatProjectSvc) GetProjectMilestones(projectID uint) ([]domain.Milestone, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) SubmitMilestone(milestoneID uint, input dto.SubmitMilestoneRequest, user domain.User) (*domain.Milestone, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) CancelSubmit(milestoneID uint, user domain.User) (*domain.Milestone, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) AdminApproveMilestoneSubmission(milestoneID uint) (*domain.Milestone, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) AdminRejectMilestoneSubmission(milestoneID uint, reason *string) (*domain.Milestone, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) OpenMilestoneVoting(milestoneID uint, user domain.User) (*domain.Milestone, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) GetSubmittedMilestonesForAdmin(projectID *uint) ([]dto.AdminMilestoneListResponse, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) GetAdminMilestoneDetail(milestoneID uint) (*dto.AdminMilestoneDetailResponse, error) {
	return nil, nil
}

// stubs — project update
func (m *mockChatProjectSvc) CreateProjectUpdate(projectID uint, req dto.CreateProjectUpdateRequest, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) GetProjectUpdates(projectID uint) ([]domain.ProjectUpdate, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) UpdateProjectUpdate(updateID uint, input dto.UpdateProjectUpdateRequest, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) DeleteProjectUpdate(updateID uint, user domain.User) error { return nil }

// stubs — story section
func (m *mockChatProjectSvc) GetProjectStories(projectID uint) ([]domain.StorySection, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) CreateStorySection(section *domain.StorySection, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) UpdateStorySection(section *domain.StorySection, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) DeleteStorySection(sectionID uint, user domain.User) error { return nil }

// stubs — category
func (m *mockChatProjectSvc) GetCategoryByID(id uint) (*domain.ProjectCategory, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) GetAllCategories() ([]domain.ProjectCategory, error) { return nil, nil }
func (m *mockChatProjectSvc) CreateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) UpdateCategory(category *domain.ProjectCategory) (*domain.ProjectCategory, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) DeleteCategory(categoryID uint) error { return nil }

// stubs — FAQ
func (m *mockChatProjectSvc) GetProjectFAQs(projectID uint) ([]domain.ProjectFAQ, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) CreateProjectFAQ(faq *domain.ProjectFAQ, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) UpdateProjectFAQ(faq *domain.ProjectFAQ, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) DeleteProjectFAQ(faqID uint, user domain.User) error { return nil }

// stubs — threads & messages
func (m *mockChatProjectSvc) GetProjectThreads(projectID uint) ([]domain.ProjectThread, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) GetProjectUpdateThreads(projectID, updateID uint) ([]domain.ProjectThread, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) CreateProjectThread(thread *domain.ProjectThread, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) CreateBoosterThread(thread *domain.ProjectThread, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) CreateUpdateThread(thread *domain.ProjectThread, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) CreateBoosterUpdateThread(thread *domain.ProjectThread, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) UpdateProjectThread(thread *domain.ProjectThread, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) DeleteProjectThread(threadID uint, user domain.User) error { return nil }
func (m *mockChatProjectSvc) GetProjectThreadMessages(threadID uint) ([]domain.ProjectThreadMessage, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) CreateProjectThreadMessage(msg *domain.ProjectThreadMessage, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) UpdateProjectThreadMessage(msg *domain.ProjectThreadMessage, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) DeleteProjectThreadMessage(msgID uint, user domain.User) error {
	return nil
}

// stubs — lifecycle
func (m *mockChatProjectSvc) SubmitForReview(projectID uint, user domain.User) error { return nil }
func (m *mockChatProjectSvc) ApproveProject(projectID uint) error                     { return nil }
func (m *mockChatProjectSvc) RejectProject(projectID uint) error                      { return nil }
func (m *mockChatProjectSvc) CloseProject(projectID uint, user domain.User) error     { return nil }
func (m *mockChatProjectSvc) CancelProject(projectID uint, user domain.User) error    { return nil }
func (m *mockChatProjectSvc) GetAllProjectsRequest() ([]domain.Project, error)        { return nil, nil }
func (m *mockChatProjectSvc) GetProjectDetailRequest(projectID uint) (*domain.Project, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) GetProjectDetailAny(projectID uint) (*domain.Project, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) AutoProjectLifecycleTick(now time.Time) error { return nil }
func (m *mockChatProjectSvc) SubmitCancelRequest(projectID uint, input dto.CancelProjectRequest, user domain.User) error {
	return nil
}
func (m *mockChatProjectSvc) ApproveCancelProject(projectID uint) error      { return nil }
func (m *mockChatProjectSvc) RejectCancelProject(projectID uint) error       { return nil }
func (m *mockChatProjectSvc) GetCancelRequest() ([]domain.Project, error)    { return nil, nil }
func (m *mockChatProjectSvc) GetCancelPreview(projectID uint) (*dto.CancelPreviewResponse, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) AdminListProjects(filter dto.AdminProjectFilter) ([]domain.Project, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) GetPlatformStats() (*dto.PlatformStatsResponse, error) { return nil, nil }

// stubs — meeting
func (m *mockChatProjectSvc) Meeting(input dto.CreateMeetingRequest, userID uint) (*domain.Meeting, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) EditMeeting(meetingID uint, input dto.UpdateMeetingRequest, userID uint) (*domain.Meeting, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) CancelMeeting(meetingID uint, user domain.User) error { return nil }
func (m *mockChatProjectSvc) GetMyMeeting(userID uint, meetingID uint) (*domain.Meeting, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) GetMyMeetings(userID uint) ([]domain.Meeting, error) { return nil, nil }
func (m *mockChatProjectSvc) GetMyMeetingsByMilestone(userID uint, milestoneID uint, filter string) ([]domain.Meeting, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) GetMyMeetingsByProject(userID uint, projectID uint, filter string) ([]domain.Meeting, error) {
	return nil, nil
}
func (m *mockChatProjectSvc) GetMyMeetingsAsBooster(userID uint) ([]dto.InvestorMeetingItem, error) {
	return nil, nil
}

// --- Mock: NotificationService (for chatService) ---

type mockChatNotifSvc struct{ mock.Mock }

func (m *mockChatNotifSvc) GetNotifications(userID uint, page, limit int) ([]domain.Notification, int64, error) {
	args := m.Called(userID, page, limit)
	if v := args.Get(0); v != nil {
		return v.([]domain.Notification), args.Get(1).(int64), args.Error(2)
	}
	return nil, args.Get(1).(int64), args.Error(2)
}

func (m *mockChatNotifSvc) CountUnread(userID uint) (int64, error) {
	args := m.Called(userID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockChatNotifSvc) MarkAllAsRead(userID uint) error {
	return m.Called(userID).Error(0)
}

// stubs
func (m *mockChatNotifSvc) CreateAndPush(userID uint, notifType domain.NotificationType, title, body string, relatedID *uint, relatedType *string) error {
	return nil
}
func (m *mockChatNotifSvc) MarkAsRead(userID, notifID uint) error { return nil }
func (m *mockChatNotifSvc) Subscribe(userID uint) chan *domain.Notification {
	return make(chan *domain.Notification, 1)
}
func (m *mockChatNotifSvc) Unsubscribe(userID uint, ch chan *domain.Notification) {}

// --- Mock: DisbursementService (for chatService) ---

type mockChatDisburseSvc struct{ mock.Mock }

func (m *mockChatDisburseSvc) ListMyPayouts(pioneerID uint) ([]dto.PioneerPayoutItem, error) {
	args := m.Called(pioneerID)
	if v := args.Get(0); v != nil {
		return v.([]dto.PioneerPayoutItem), args.Error(1)
	}
	return nil, args.Error(1)
}

// stubs
func (m *mockChatDisburseSvc) CreateForMilestone(milestoneID uint) (*domain.Disbursement, error) {
	return nil, nil
}
func (m *mockChatDisburseSvc) ListAll() ([]dto.DisbursementItem, error)     { return nil, nil }
func (m *mockChatDisburseSvc) ListPending() ([]dto.DisbursementItem, error) { return nil, nil }
func (m *mockChatDisburseSvc) Confirm(disbursementID uint, adminID uint, req dto.ConfirmDisbursementRequest) (*domain.Disbursement, error) {
	return nil, nil
}

// --- Mock: ComplaintService (for chatService) ---

type mockChatComplaintSvc struct{ mock.Mock }

func (m *mockChatComplaintSvc) Create(userID uint, req dto.CreateComplaintRequest) (*domain.Complaint, error) {
	args := m.Called(userID, req)
	if v := args.Get(0); v != nil {
		return v.(*domain.Complaint), args.Error(1)
	}
	return nil, args.Error(1)
}

// stubs
func (m *mockChatComplaintSvc) ListMine(userID uint) ([]dto.ComplaintItem, error) { return nil, nil }
func (m *mockChatComplaintSvc) AdminList(status *domain.ComplaintStatus) ([]dto.ComplaintItem, error) {
	return nil, nil
}
func (m *mockChatComplaintSvc) AdminGet(id uint) (*dto.ComplaintItem, error) { return nil, nil }
func (m *mockChatComplaintSvc) AdminResolve(id, adminID uint, note string) (*domain.Complaint, error) {
	return nil, nil
}
func (m *mockChatComplaintSvc) AdminReject(id, adminID uint, note string) (*domain.Complaint, error) {
	return nil, nil
}
func (m *mockChatComplaintSvc) GetProjectStats(projectID uint) (*dto.ProjectComplaintStats, error) {
	return nil, nil
}

// --- helpers ---

func newChatSvcTest(
	chatRepo *mockChatRepo,
	ai ChatAIClient,
	inv *mockChatInvestSvc,
	proj *mockChatProjectSvc,
	notif *mockChatNotifSvc,
	disb *mockChatDisburseSvc,
	comp *mockChatComplaintSvc,
) ChatService {
	return NewChatService(chatRepo, ai, inv, proj, notif, disb, comp)
}

func defaultChatMocks() (*mockChatRepo, *mockChatAI, *mockChatInvestSvc, *mockChatProjectSvc, *mockChatNotifSvc, *mockChatDisburseSvc, *mockChatComplaintSvc) {
	return new(mockChatRepo), new(mockChatAI), new(mockChatInvestSvc), new(mockChatProjectSvc), new(mockChatNotifSvc), new(mockChatDisburseSvc), new(mockChatComplaintSvc)
}

// --- SendMessage tests ---

func TestChatService_SendMessage_EmptyMessage(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	_, err := svc.SendMessage(1, dto.SendChatMessageRequest{Message: "   "})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestChatService_SendMessage_NilAIClient(t *testing.T) {
	cr, _, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, nil, inv, proj, notif, disb, comp)

	_, err := svc.SendMessage(1, dto.SendChatMessageRequest{Message: "สวัสดี"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "chat ai client is required")
}

func TestChatService_SendMessage_CreateSessionFail(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	cr.On("CreateSession", mock.Anything).Return(errors.New("db error"))

	_, err := svc.SendMessage(1, dto.SendChatMessageRequest{Message: "สวัสดี"})

	assert.Error(t, err)
}

func TestChatService_SendMessage_NewSession_TextReply(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	session := &domain.ChatSession{ID: 1, UserID: 1, Status: domain.ChatSessionStatusActive}
	cr.On("CreateSession", mock.Anything).Run(func(args mock.Arguments) {
		s := args.Get(0).(*domain.ChatSession)
		s.ID = 1
	}).Return(nil)
	cr.On("CreateMessage", mock.Anything).Return(nil)
	cr.On("GetMessagesBySessionID", uint(1)).Return([]domain.ChatMessage{}, nil)
	cr.On("UpdateMessage", mock.Anything).Return(nil)

	ai.On("GenerateReply", mock.Anything).Return(&ChatAIOutput{
		Intent:         domain.ChatIntentUnknown,
		Reply:          "สวัสดีครับ ยินดีให้บริการ",
		RequiresAction: false,
	}, nil)

	resp, err := svc.SendMessage(1, dto.SendChatMessageRequest{Message: "สวัสดี"})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, session.ID, resp.Session.ID)
	assert.Equal(t, "สวัสดีครับ ยินดีให้บริการ", resp.Reply.Content)
	assert.Nil(t, resp.Action)
	cr.AssertExpectations(t)
	ai.AssertExpectations(t)
}

func TestChatService_SendMessage_ExistingSession_TextReply(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	sessionID := uint(5)
	session := &domain.ChatSession{ID: sessionID, UserID: 1, Status: domain.ChatSessionStatusActive}
	cr.On("GetSessionByIDAndUserID", sessionID, uint(1)).Return(session, nil)
	cr.On("CreateMessage", mock.Anything).Return(nil)
	cr.On("GetMessagesBySessionID", sessionID).Return([]domain.ChatMessage{}, nil)
	cr.On("UpdateMessage", mock.Anything).Return(nil)

	ai.On("GenerateReply", mock.Anything).Return(&ChatAIOutput{
		Intent:         domain.ChatIntentUnknown,
		Reply:          "ได้เลยครับ",
		RequiresAction: false,
	}, nil)

	resp, err := svc.SendMessage(1, dto.SendChatMessageRequest{SessionID: &sessionID, Message: "ช่วยด้วย"})

	assert.NoError(t, err)
	assert.Equal(t, sessionID, resp.Session.ID)
	cr.AssertExpectations(t)
}

func TestChatService_SendMessage_QueryAction_GetInvestments(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	cr.On("CreateSession", mock.Anything).Run(func(args mock.Arguments) {
		args.Get(0).(*domain.ChatSession).ID = 1
	}).Return(nil)
	cr.On("CreateMessage", mock.Anything).Return(nil)
	cr.On("GetMessagesBySessionID", uint(1)).Return([]domain.ChatMessage{}, nil)
	cr.On("UpdateMessage", mock.Anything).Return(nil)

	ai.On("GenerateReply", mock.Anything).Return(&ChatAIOutput{
		Intent:         domain.ChatIntentLatestTransaction,
		RequiresAction: true,
		ActionType:     domain.ChatActionTypeGetInvestments,
	}, nil)

	inv.On("ListUserInvestments", uint(1)).Return([]domain.Investment{
		{ID: 10, ProjectID: 20, Status: domain.InvestmentVerified, TotalAmount: 5000},
	}, nil)
	inv.On("ListInvestedProjects", uint(1)).Return([]dto.InvestedProjectItem{
		{ProjectID: 20, Title: "โปรเจกต์ A"},
	}, nil)

	resp, err := svc.SendMessage(1, dto.SendChatMessageRequest{Message: "ดูการลงทุนของฉัน"})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Nil(t, resp.Action) // query action ไม่สร้าง pending action
	assert.Contains(t, resp.Reply.Content, "โปรเจกต์ A")
	cr.AssertExpectations(t)
	inv.AssertExpectations(t)
}

func TestChatService_SendMessage_MutationAction_CreatesPending(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	invID := uint(10)
	cr.On("CreateSession", mock.Anything).Run(func(args mock.Arguments) {
		args.Get(0).(*domain.ChatSession).ID = 1
	}).Return(nil)
	cr.On("CreateMessage", mock.Anything).Return(nil)
	cr.On("GetMessagesBySessionID", uint(1)).Return([]domain.ChatMessage{}, nil)
	cr.On("UpdateMessage", mock.Anything).Return(nil)
	cr.On("CreateAction", mock.Anything).Return(nil)

	ai.On("GenerateReply", mock.Anything).Return(&ChatAIOutput{
		Intent:         domain.ChatIntentRefundRequest,
		Reply:          "ต้องการยืนยันขอคืนเงิน investment #10 ใช่ไหมครับ?",
		RequiresAction: true,
		ActionType:     domain.ChatActionTypeRefundTransaction,
		InvestmentID:   &invID,
	}, nil)

	resp, err := svc.SendMessage(1, dto.SendChatMessageRequest{Message: "ขอคืนเงิน investment 10"})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Action) // mutation action สร้าง pending action
	assert.Contains(t, resp.Reply.Content, "ยืนยัน")
	cr.AssertExpectations(t)
}

func TestChatService_SendMessage_AIError(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	cr.On("CreateSession", mock.Anything).Run(func(args mock.Arguments) {
		args.Get(0).(*domain.ChatSession).ID = 1
	}).Return(nil)
	cr.On("CreateMessage", mock.Anything).Return(nil)
	cr.On("GetMessagesBySessionID", uint(1)).Return([]domain.ChatMessage{}, nil)
	cr.On("UpdateMessage", mock.Anything).Return(nil)

	ai.On("GenerateReply", mock.Anything).Return(nil, errors.New("AI timeout"))

	_, err := svc.SendMessage(1, dto.SendChatMessageRequest{Message: "สวัสดี"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ai error")
}

// --- ConfirmAction tests ---

func TestChatService_ConfirmAction_NotFound(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	cr.On("GetActionByIDAndUserID", uint(99), uint(1)).Return(nil, errors.New("not found"))

	_, err := svc.ConfirmAction(1, dto.ConfirmChatActionRequest{ActionID: 99, Confirm: true})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "action not found")
}

func TestChatService_ConfirmAction_NotPending(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	action := &domain.ChatAction{ID: 1, Status: domain.ChatActionStatusCompleted}
	cr.On("GetActionByIDAndUserID", uint(1), uint(1)).Return(action, nil)

	_, err := svc.ConfirmAction(1, dto.ConfirmChatActionRequest{ActionID: 1, Confirm: true})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not pending")
}

func TestChatService_ConfirmAction_Reject(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	action := &domain.ChatAction{
		ID:        1,
		SessionID: 5,
		UserID:    1,
		Type:      domain.ChatActionTypeRefundTransaction,
		Status:    domain.ChatActionStatusPending,
	}
	cr.On("GetActionByIDAndUserID", uint(1), uint(1)).Return(action, nil)
	cr.On("UpdateAction", mock.Anything).Return(nil)
	cr.On("CreateMessage", mock.Anything).Return(nil)

	resp, err := svc.ConfirmAction(1, dto.ConfirmChatActionRequest{ActionID: 1, Confirm: false})

	assert.NoError(t, err)
	assert.Equal(t, domain.ChatActionStatusRejected, resp.Action.Status)
	assert.Contains(t, resp.Reply.Content, "ยกเลิก")
	cr.AssertExpectations(t)
}

func TestChatService_ConfirmAction_Confirm_Refund_Success(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	invID := uint(10)
	action := &domain.ChatAction{
		ID:           2,
		SessionID:    5,
		UserID:       1,
		Type:         domain.ChatActionTypeRefundTransaction,
		Status:       domain.ChatActionStatusPending,
		InvestmentID: &invID,
	}
	cr.On("GetActionByIDAndUserID", uint(2), uint(1)).Return(action, nil)
	inv.On("RefundInvestment", uint(1), invID, mock.AnythingOfType("string")).Return(&dto.RefundResponse{
		InvestmentID:    invID,
		ReferenceNumber: "REF-001",
		RefundAmount:    950.0,
		FeesDeducted:    50.0,
	}, nil)
	cr.On("UpdateAction", mock.Anything).Return(nil)
	cr.On("CreateMessage", mock.Anything).Return(nil)

	resp, err := svc.ConfirmAction(1, dto.ConfirmChatActionRequest{ActionID: 2, Confirm: true})

	assert.NoError(t, err)
	assert.Equal(t, domain.ChatActionStatusCompleted, resp.Action.Status)
	assert.Contains(t, resp.Reply.Content, "REF-001")
	cr.AssertExpectations(t)
	inv.AssertExpectations(t)
}

func TestChatService_ConfirmAction_Confirm_ExecFail(t *testing.T) {
	cr, ai, inv, proj, notif, disb, comp := defaultChatMocks()
	svc := newChatSvcTest(cr, ai, inv, proj, notif, disb, comp)

	invID := uint(10)
	action := &domain.ChatAction{
		ID:           3,
		SessionID:    5,
		UserID:       1,
		Type:         domain.ChatActionTypeRefundTransaction,
		Status:       domain.ChatActionStatusPending,
		InvestmentID: &invID,
	}
	cr.On("GetActionByIDAndUserID", uint(3), uint(1)).Return(action, nil)
	inv.On("RefundInvestment", uint(1), invID, mock.AnythingOfType("string")).Return(nil, errors.New("investment not eligible"))
	cr.On("UpdateAction", mock.Anything).Return(nil)
	cr.On("CreateMessage", mock.Anything).Return(nil)

	resp, err := svc.ConfirmAction(1, dto.ConfirmChatActionRequest{ActionID: 3, Confirm: true})

	assert.NoError(t, err) // ConfirmAction ไม่ return error เมื่อ exec fail — แค่ mark failed และส่ง reply กลับ
	assert.Equal(t, domain.ChatActionStatusFailed, resp.Action.Status)
	assert.Contains(t, resp.Reply.Content, "ไม่สามารถดำเนินการได้")
	cr.AssertExpectations(t)
	inv.AssertExpectations(t)
}

// suppress unused import warning
var _ = multipart.FileHeader{}
var _ = context.Background
var _ = time.Now
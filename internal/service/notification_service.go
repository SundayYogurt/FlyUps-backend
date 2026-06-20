package service

import (
	"flyup/internal/domain"
	"flyup/internal/repository"
	"log"
	"sync"
)

type NotificationService interface {
	CreateAndPush(userID uint, notifType domain.NotificationType, title, body string, relatedID *uint, relatedType *string) error
	GetNotifications(userID uint, page, limit int) ([]domain.Notification, int64, error)
	MarkAsRead(userID uint, notifID uint) error
	MarkAllAsRead(userID uint) error
	CountUnread(userID uint) (int64, error)
	Subscribe(userID uint) chan *domain.Notification
	Unsubscribe(userID uint, ch chan *domain.Notification)
}

// notificationHub manages SSE client connections per user
type notificationHub struct {
	clients map[uint][]chan *domain.Notification
	mu      sync.RWMutex
}

func newNotificationHub() *notificationHub {
	return &notificationHub{
		clients: make(map[uint][]chan *domain.Notification),
	}
}

func (h *notificationHub) subscribe(userID uint) chan *domain.Notification {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan *domain.Notification, 10)
	h.clients[userID] = append(h.clients[userID], ch)
	return ch
}

func (h *notificationHub) unsubscribe(userID uint, ch chan *domain.Notification) {
	h.mu.Lock()
	defer h.mu.Unlock()
	channels := h.clients[userID]
	for i, c := range channels {
		if c == ch {
			h.clients[userID] = append(channels[:i], channels[i+1:]...)
			close(ch)
			break
		}
	}
	if len(h.clients[userID]) == 0 {
		delete(h.clients, userID)
	}
}

func (h *notificationHub) broadcast(userID uint, notif *domain.Notification) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.clients[userID] {
		select {
		case ch <- notif:
		default:
			log.Printf("[NotifHub] dropped notification for user %d: channel full", userID)
		}
	}
}

type notificationService struct {
	repo repository.NotificationRepository
	hub  *notificationHub
}

func NewNotificationService(repo repository.NotificationRepository) NotificationService {
	return &notificationService{
		repo: repo,
		hub:  newNotificationHub(),
	}
}

func (s *notificationService) CreateAndPush(userID uint, notifType domain.NotificationType, title, body string, relatedID *uint, relatedType *string) error {
	n := &domain.Notification{
		UserID:      userID,
		Type:        notifType,
		Title:       title,
		Body:        body,
		RelatedID:   relatedID,
		RelatedType: relatedType,
	}
	if err := s.repo.Create(n); err != nil {
		return err
	}
	s.hub.broadcast(userID, n)
	return nil
}

func (s *notificationService) GetNotifications(userID uint, page, limit int) ([]domain.Notification, int64, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	return s.repo.FindByUserID(userID, limit, offset)
}

func (s *notificationService) MarkAsRead(userID uint, notifID uint) error {
	return s.repo.MarkAsRead(userID, notifID)
}

func (s *notificationService) MarkAllAsRead(userID uint) error {
	return s.repo.MarkAllAsRead(userID)
}

func (s *notificationService) CountUnread(userID uint) (int64, error) {
	return s.repo.CountUnread(userID)
}

func (s *notificationService) Subscribe(userID uint) chan *domain.Notification {
	return s.hub.subscribe(userID)
}

func (s *notificationService) Unsubscribe(userID uint, ch chan *domain.Notification) {
	s.hub.unsubscribe(userID, ch)
}

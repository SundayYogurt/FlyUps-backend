package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type ChatRepository interface {
	CreateSession(session *domain.ChatSession) error
	GetSessionByIDAndUserID(sessionID uint, userID uint) (*domain.ChatSession, error)

	CreateMessage(message *domain.ChatMessage) error
	GetMessagesBySessionID(sessionID uint) ([]domain.ChatMessage, error)
	UpdateMessage(message *domain.ChatMessage) error

	CreateAction(action *domain.ChatAction) error
	GetActionByIDAndUserID(actionID uint, userID uint) (*domain.ChatAction, error)
	UpdateAction(action *domain.ChatAction) error
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{
		db: db,
	}
}

func (r *chatRepository) CreateSession(session *domain.ChatSession) error {
	return r.db.Create(session).Error
}

func (r *chatRepository) GetSessionByIDAndUserID(sessionID uint, userID uint) (*domain.ChatSession, error) {
	var session domain.ChatSession

	if err := r.db.Where("id = ? AND user_id = ?", sessionID, userID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *chatRepository) CreateMessage(message *domain.ChatMessage) error {
	return r.db.Create(message).Error
}

func (r *chatRepository) UpdateMessage(message *domain.ChatMessage) error {
	return r.db.Save(message).Error
}

func (r *chatRepository) GetMessagesBySessionID(sessionID uint) ([]domain.ChatMessage, error) {
	var messages []domain.ChatMessage
	if err := r.db.
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *chatRepository) CreateAction(action *domain.ChatAction) error {
	return r.db.Create(action).Error
}

func (r *chatRepository) GetActionByIDAndUserID(actionID uint, userID uint) (*domain.ChatAction, error) {
	var action domain.ChatAction

	if err := r.db.
		Where("id = ? AND user_id = ?", actionID, userID).
		First(&action).Error; err != nil {
		return nil, err
	}

	return &action, nil
}

func (r *chatRepository) UpdateAction(action *domain.ChatAction) error {
	return r.db.Save(action).Error
}

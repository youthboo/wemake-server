package service

import (
	"database/sql"
	"errors"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type ConversationService struct {
	repo *repository.ConversationRepository
}

var ErrConversationNotFoundOrForbidden = errors.New("conversation not found or forbidden")

func NewConversationService(repo *repository.ConversationRepository) *ConversationService {
	return &ConversationService{repo: repo}
}

func (s *ConversationService) ListByUserID(userID int64) ([]domain.Conversation, error) {
	return s.repo.ListByUserID(userID)
}

func (s *ConversationService) GetByID(convID int64) (*domain.Conversation, error) {
	return s.repo.GetByID(convID)
}

func (s *ConversationService) Create(conv *domain.Conversation) error {
	return s.repo.Create(conv)
}

func (s *ConversationService) MarkAsRead(convID, userID int64) error {
	err := s.repo.MarkAsRead(convID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrConversationNotFoundOrForbidden
	}
	return err
}

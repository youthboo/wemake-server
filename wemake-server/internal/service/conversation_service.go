package service

import (
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type ConversationService struct {
	repo *repository.ConversationRepository
}

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

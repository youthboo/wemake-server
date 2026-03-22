package service

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type MessageService struct {
	repo *repository.MessageRepository
}

func NewMessageService(repo *repository.MessageRepository) *MessageService {
	return &MessageService{repo: repo}
}

func (s *MessageService) Create(item *domain.Message) error {
	item.MessageID = "msg-" + uuid.NewString()
	item.ReferenceType = strings.TrimSpace(strings.ToUpper(item.ReferenceType))
	item.ReferenceID = strings.TrimSpace(item.ReferenceID)
	item.Content = strings.TrimSpace(item.Content)
	item.AttachmentURL = strings.TrimSpace(item.AttachmentURL)
	item.CreatedAt = time.Now()
	return s.repo.Create(item)
}

func (s *MessageService) ListByReference(referenceType, referenceID string, userID int64) ([]domain.Message, error) {
	return s.repo.ListByReference(strings.TrimSpace(strings.ToUpper(referenceType)), strings.TrimSpace(referenceID), userID)
}

func (s *MessageService) ListThreads(userID int64) ([]domain.MessageThread, error) {
	return s.repo.ListThreads(userID)
}

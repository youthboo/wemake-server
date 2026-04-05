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
	item.ReferenceType = normalizeMessageRefType(item.ReferenceType)
	item.Content = strings.TrimSpace(item.Content)
	item.AttachmentURL = strings.TrimSpace(item.AttachmentURL)
	item.CreatedAt = time.Now()
	return s.repo.Create(item)
}

func normalizeMessageRefType(t string) string {
	u := strings.ToUpper(strings.TrimSpace(t))
	switch u {
	case "RFQ", "RQ":
		return "RQ"
	case "ORDER", "OD":
		return "OD"
	default:
		return u
	}
}

func (s *MessageService) ListByReference(referenceType string, referenceID int64, userID int64) ([]domain.Message, error) {
	return s.repo.ListByReference(normalizeMessageRefType(referenceType), referenceID, userID)
}

func (s *MessageService) ListByConvID(convID int64) ([]domain.Message, error) {
	return s.repo.ListByConvID(convID)
}

func (s *MessageService) ListThreads(userID int64) ([]domain.MessageThread, error) {
	return s.repo.ListThreads(userID)
}

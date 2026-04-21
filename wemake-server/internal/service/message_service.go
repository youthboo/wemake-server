package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/wemake/internal/domain"
)

var (
	ErrInvalidReferenceType         = errors.New("invalid reference_type")
	ErrReferencePairRequired        = errors.New("reference_type and reference_id must be provided together")
	ErrReferenceNotFound            = errors.New("reference_id not found")
	ErrSenderReceiverSame           = errors.New("receiver_id must differ from sender_id")
	ErrConversationNotAccessible    = errors.New("conv_id must belong to the sender")
	ErrConversationReceiverMismatch = errors.New("receiver_id does not match conversation participants")
	ErrInvalidMessageType           = errors.New("invalid message_type")
	ErrQuoteDataRequired            = errors.New("quote_data is required when message_type is QT")
)

var allowedMessageReferenceTypes = map[string]struct{}{
	"RQ": {},
	"OD": {},
	"PD": {},
	"PM": {},
	"ID": {},
}

var allowedMessageTypes = map[string]struct{}{
	"TX": {},
	"QT": {},
	"IM": {},
}

type messageRepository interface {
	Create(item *domain.Message) error
	ListByReference(referenceType string, referenceID int64, userID int64) ([]domain.Message, error)
	ListByConvID(convID int64) ([]domain.Message, error)
	ListThreads(userID int64) ([]domain.MessageThread, error)
	ReferenceExists(referenceType string, referenceID int64) (bool, error)
}

type conversationRepository interface {
	GetByID(convID int64) (*domain.ConversationRow, error)
}

type MessageService struct {
	repo     messageRepository
	convRepo conversationRepository
}

func NewMessageService(repo messageRepository, convRepo conversationRepository) *MessageService {
	return &MessageService{repo: repo, convRepo: convRepo}
}

func (s *MessageService) Create(item *domain.Message) error {
	item.MessageID = "msg-" + uuid.NewString()
	item.ReferenceType = normalizeMessageRefType(item.ReferenceType)
	item.MessageType = normalizeMessageType(item.MessageType)
	item.Content = strings.TrimSpace(item.Content)
	item.AttachmentURL = strings.TrimSpace(item.AttachmentURL)
	if item.QuoteData != nil {
		trimmed := strings.TrimSpace(*item.QuoteData)
		if trimmed == "" {
			item.QuoteData = nil
		} else {
			item.QuoteData = &trimmed
		}
	}
	if err := s.validateCreate(item); err != nil {
		return err
	}
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

func normalizeMessageType(t string) string {
	u := strings.ToUpper(strings.TrimSpace(t))
	if u == "" {
		return "TX"
	}
	return u
}

func (s *MessageService) validateCreate(item *domain.Message) error {
	if item.SenderID == item.ReceiverID {
		return ErrSenderReceiverSame
	}

	hasReferenceType := item.ReferenceType != ""
	hasReferenceID := item.ReferenceID > 0
	if hasReferenceType != hasReferenceID {
		return ErrReferencePairRequired
	}

	if hasReferenceType {
		if _, ok := allowedMessageReferenceTypes[item.ReferenceType]; !ok {
			return ErrInvalidReferenceType
		}
		exists, err := s.repo.ReferenceExists(item.ReferenceType, item.ReferenceID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w for reference_type=%s", ErrReferenceNotFound, item.ReferenceType)
		}
	}

	if _, ok := allowedMessageTypes[item.MessageType]; !ok {
		return ErrInvalidMessageType
	}
	if item.MessageType == "QT" && item.QuoteData == nil {
		return ErrQuoteDataRequired
	}

	if item.ConvID != nil {
		if s.convRepo == nil {
			return ErrConversationNotAccessible
		}
		conv, err := s.convRepo.GetByID(*item.ConvID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrConversationNotAccessible
			}
			return err
		}
		if conv.CustomerID != item.SenderID && conv.FactoryID != item.SenderID {
			return ErrConversationNotAccessible
		}
		if (conv.CustomerID != item.ReceiverID && conv.FactoryID != item.ReceiverID) || item.ReceiverID == item.SenderID {
			return ErrConversationReceiverMismatch
		}
	}

	return nil
}

func (s *MessageService) ListByReference(referenceType string, referenceID int64, userID int64) ([]domain.Message, error) {
	refType := normalizeMessageRefType(referenceType)
	if _, ok := allowedMessageReferenceTypes[refType]; !ok {
		return nil, ErrInvalidReferenceType
	}
	return s.repo.ListByReference(refType, referenceID, userID)
}

func (s *MessageService) ListByConvID(convID int64) ([]domain.Message, error) {
	return s.repo.ListByConvID(convID)
}

func (s *MessageService) ListThreads(userID int64) ([]domain.MessageThread, error) {
	return s.repo.ListThreads(userID)
}

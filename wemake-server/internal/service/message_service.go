package service

import (
	"context"
	"database/sql"
	"encoding/json"
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

// bangkokTZ is UTC+7 used to stamp message created_at so the timestamp is
// always stored as Bangkok wall-clock regardless of the server's local TZ.
var bangkokTZ = time.FixedZone("Asia/Bangkok", 7*60*60)

var allowedMessageReferenceTypes = map[string]struct{}{
	"RQ": {},
	"OD": {},
	"PD": {},
	"PM": {},
	"ID": {},
}

var allowedMessageTypes = map[string]struct{}{
	"TX":             {},
	"QT":             {},
	"IM":             {},
	"BQ":             {},
	"rfq_card":       {},
	"quotation_card": {},
	"system":         {},
}

type messageRepository interface {
	Create(item *domain.Message) error
	CreateTx(exec interface {
		Exec(query string, args ...interface{}) (sql.Result, error)
	}, item *domain.Message) error
	ListByReference(referenceType string, referenceID int64, userID int64) ([]domain.Message, error)
	ListByConvID(convID int64) ([]domain.Message, error)
	ListThreads(userID int64) ([]domain.MessageThread, error)
	ReferenceExists(referenceType string, referenceID int64) (bool, error)
}

type conversationRepository interface {
	GetByID(convID int64) (*domain.ConversationRow, error)
}

type MessageService struct {
	repo          messageRepository
	convRepo      conversationRepository
	notifications *NotificationService
}

func NewMessageService(repo messageRepository, convRepo conversationRepository, notifications *NotificationService) *MessageService {
	return &MessageService{repo: repo, convRepo: convRepo, notifications: notifications}
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
	item.CreatedAt = time.Now().In(bangkokTZ)
	if err := s.repo.Create(item); err != nil {
		return err
	}
	s.notifyReceiver(item)
	return nil
}

func (s *MessageService) CreateTx(tx interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}, item *domain.Message) error {
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
	item.CreatedAt = time.Now().In(bangkokTZ)
	return s.repo.CreateTx(tx, item)
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
	trimmed := strings.TrimSpace(t)
	if trimmed == "" {
		return "TX"
	}
	switch strings.ToLower(trimmed) {
	case "rfq_card":
		return "rfq_card"
	case "quotation_card":
		return "quotation_card"
	case "system":
		return "system"
	default:
		return strings.ToUpper(trimmed)
	}
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

func (s *MessageService) notifyReceiver(item *domain.Message) {
	if s.notifications == nil || item == nil || item.MessageType == "BQ" || item.MessageType == "system" {
		return
	}

	title := "ข้อความใหม่"
	preview := trimNotificationPreview(item.Content, 80)
	if preview == "" {
		switch item.MessageType {
		case "IM":
			preview = "ส่งรูปภาพใหม่"
		case "QT":
			preview = "ส่งใบเสนอราคาใหม่"
		case "rfq_card":
			preview = "แชร์ RFQ เข้ามาในแชต"
		case "quotation_card":
			preview = "มีใบเสนอราคาใหม่ในแชต"
		default:
			preview = "มีข้อความใหม่ในแชต"
		}
	}

	link := ""
	if item.ConvID != nil {
		link = fmt.Sprintf("/chat/%d", *item.ConvID)
	}

	senderName := fmt.Sprintf("ผู้ใช้ #%d", item.SenderID)
	if item.ConvID != nil && s.convRepo != nil {
		if conv, err := s.convRepo.GetByID(*item.ConvID); err == nil {
			if conv.FactoryID == item.SenderID {
				if conv.FactoryName != nil && strings.TrimSpace(*conv.FactoryName) != "" {
					senderName = strings.TrimSpace(*conv.FactoryName)
				}
			} else {
				firstName := ""
				lastName := ""
				if conv.CustomerFirstName != nil {
					firstName = *conv.CustomerFirstName
				}
				if conv.CustomerLastName != nil {
					lastName = *conv.CustomerLastName
				}
				fullName := strings.TrimSpace(firstName + " " + lastName)
				if fullName != "" {
					senderName = fullName
				}
			}
		}
	}

	createNotificationSafe(s.notifications, &domain.Notification{
		UserID:  item.ReceiverID,
		Type:    "CHAT_MESSAGE",
		Title:   title,
		Message: fmt.Sprintf("%s: %s", senderName, preview),
		LinkTo:  link,
		Data: notificationData(map[string]interface{}{
			"conv_id":     item.ConvID,
			"sender_id":   item.SenderID,
			"sender_name": senderName,
			"url":         link,
		}),
		CreatedAt: item.CreatedAt,
	})
}

func (s *MessageService) AutoSendQuotationCard(ctx context.Context, convID int64, customerID int64, q *domain.Quotation) error {
	_ = ctx
	if q == nil {
		return nil
	}
	validUntil := ""
	if q.ValidUntil != nil {
		validUntil = q.ValidUntil.Format("02 Jan 06")
	}
	payload, err := json.Marshal(map[string]interface{}{
		"quotation_id": q.QuotationID,
		"price":        q.GrandTotal,
		"lead_time":    q.LeadTimeDays,
		"valid_until":  validUntil,
		"status":       "pending",
	})
	if err != nil {
		return err
	}
	msg := &domain.Message{
		ConvID:        &convID,
		ReferenceType: "RQ",
		ReferenceID:   q.RFQID,
		SenderID:      q.FactoryID,
		ReceiverID:    customerID,
		Content:       fmt.Sprintf("ใบเสนอราคา ฿%.0f", q.GrandTotal),
		MessageType:   "quotation_card",
		QuoteData:     stringPtr(string(payload)),
		IsRead:        false,
	}
	return s.Create(msg)
}

func (s *MessageService) AutoSendSystemMessage(ctx context.Context, convID int64, senderID int64, receiverID int64, content string) error {
	_ = ctx
	msg := &domain.Message{
		ConvID:      &convID,
		SenderID:    senderID,
		ReceiverID:  receiverID,
		Content:     content,
		MessageType: "system",
		IsRead:      false,
	}
	return s.Create(msg)
}

func stringPtr(v string) *string {
	return &v
}

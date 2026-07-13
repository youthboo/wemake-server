package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/helper"
	notificationservice "github.com/yourusername/wemake/internal/service/notification"
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
	DB() *sqlx.DB
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

type quotationRepository interface {
	GetActiveByRFQAndFactory(rfqID, factoryID int64) (*domain.Quotation, error)
}

type MessageService struct {
	repo          messageRepository
	convRepo      conversationRepository
	quotationRepo quotationRepository
	notifications *notificationservice.NotificationService
}

func NewMessageService(repo messageRepository, convRepo conversationRepository, quotationRepo quotationRepository, notifications *notificationservice.NotificationService) *MessageService {
	return &MessageService{repo: repo, convRepo: convRepo, quotationRepo: quotationRepo, notifications: notifications}
}

// buildQuoteDataFromDB fetches the authoritative quotation from the DB and returns
// a JSON string for quote_data that reflects the real stored values.
// It merges any extra fields the caller already provided (e.g. quotation_id override)
// but always overwrites price, lead_time, valid_until, status, rfq_id from the DB row.
func (s *MessageService) buildQuoteDataFromDB(rfqID, factoryID int64, existing *string) *string {
	if s.quotationRepo == nil || rfqID <= 0 || factoryID <= 0 {
		return existing
	}
	q, err := s.quotationRepo.GetActiveByRFQAndFactory(rfqID, factoryID)
	if err != nil || q == nil {
		return existing
	}

	// Start from existing quote_data so callers can supply extra fields (e.g. custom notes).
	merged := make(map[string]interface{})
	if existing != nil && strings.TrimSpace(*existing) != "" {
		_ = json.Unmarshal([]byte(*existing), &merged)
	}

	// Always override with authoritative DB values.
	merged["quotation_id"] = q.QuotationID
	merged["rfq_id"] = q.RFQID
	merged["price"] = q.GrandTotal
	merged["lead_time"] = q.LeadTimeDays
	merged["status"] = strings.ToLower(q.Status)
	if q.ValidUntil != nil {
		merged["valid_until"] = q.ValidUntil.Format("2006-01-02")
	} else {
		delete(merged, "valid_until")
	}

	b, err := json.Marshal(merged)
	if err != nil {
		return existing
	}
	out := string(b)
	return &out
}

func (s *MessageService) Create(item *domain.Message) error {
	item.MessageID = "msg-" + uuid.NewString()
	item.MessageType = normalizeMessageType(item.MessageType)
	item.ReferenceType = resolveMessageReferenceType(item)
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
	// For QT messages, authoratively populate quote_data from the quotations table.
	if item.MessageType == "QT" && item.ReferenceID > 0 {
		item.QuoteData = s.buildQuoteDataFromDB(item.ReferenceID, item.SenderID, item.QuoteData)
	}
	if err := s.validateCreate(item); err != nil {
		return err
	}
	// Read receipt invariant: a new outgoing message is always unread.
	item.IsRead = false
	item.CreatedAt = time.Now().UTC()
	if item.ConvID == nil {
		if err := s.repo.Create(item); err != nil {
			return err
		}
		return nil
	}
	conv, err := s.convRepo.GetByID(*item.ConvID)
	if err != nil {
		return err
	}
	unreadField := "unread_factory"
	if item.ReceiverID == conv.CustomerID {
		unreadField = "unread_customer"
	}
	if err := helper.WithTx(context.Background(), s.repo.DB(), func(tx *sqlx.Tx) error {
		if err := s.repo.CreateTx(tx, item); err != nil {
			return err
		}
		_, err := tx.Exec(`
			UPDATE conversations
			SET `+unreadField+` = COALESCE(`+unreadField+`, 0) + 1,
			    last_message = $2,
			    updated_at = $3
			WHERE conv_id = $1
		`, *item.ConvID, item.Content, item.CreatedAt)
		return err
	}); err != nil {
		return err
	}
	return nil
}

func (s *MessageService) CreateTx(tx interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}, item *domain.Message) error {
	item.MessageID = "msg-" + uuid.NewString()
	item.MessageType = normalizeMessageType(item.MessageType)
	item.ReferenceType = resolveMessageReferenceType(item)
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
	item.IsRead = false
	item.CreatedAt = time.Now().UTC()
	return s.repo.CreateTx(tx, item)
}

func normalizeMessageRefType(t string) string {
	u := domainutil.NormalizeStatus(t)
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
	switch domainutil.NormalizeLower(trimmed) {
	case "rfq_card":
		return "rfq_card"
	case "quotation_card":
		return "quotation_card"
	case "system":
		return "system"
	default:
		return domainutil.NormalizeStatus(trimmed)
	}
}

// resolveMessageReferenceType picks the reference_type to persist:
// prefer the explicit value from the client (normalized); fall back to deriving
// from message_type for older callers that don't send reference_type.
func resolveMessageReferenceType(item *domain.Message) string {
	rt := normalizeMessageRefType(item.ReferenceType)
	if rt == "" {
		rt = refTypeFromMessageType(item.MessageType)
	}
	return rt
}

// refTypeFromMessageType derives the logical reference category from message_type
// so we can validate reference_id without a messages.reference_type DB column.
func refTypeFromMessageType(mt string) string {
	switch mt {
	case "QT", "rfq_card", "quotation_card":
		return "RQ"
	case "PD", "PM", "ID":
		return mt
	case "OD":
		return "OD"
	default:
		return ""
	}
}

func (s *MessageService) validateCreate(item *domain.Message) error {
	if item.SenderID == item.ReceiverID {
		return ErrSenderReceiverSame
	}

	// Validate reference_id against the resolved reference_type (item.ReferenceType
	// was set from the explicit value or derived from message_type by the caller).
	refType := item.ReferenceType
	if refType != "" {
		if _, ok := allowedMessageReferenceTypes[refType]; !ok {
			return ErrInvalidReferenceType
		}
	}
	if refType != "" && item.ReferenceID > 0 {
		exists, err := s.repo.ReferenceExists(refType, item.ReferenceID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w for reference_type=%s", ErrReferenceNotFound, refType)
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


func (s *MessageService) AutoSendQuotationCard(ctx context.Context, convID int64, customerID int64, q *domain.Quotation) error {
	_ = ctx
	if q == nil {
		return nil
	}
	validUntil := ""
	if q.ValidUntil != nil {
		validUntil = q.ValidUntil.Format("2006-01-02")
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
		ConvID:      &convID,
		ReferenceID: q.RFQID, // RFQ id — derived from message_type=quotation_card
		SenderID:    q.FactoryID,
		ReceiverID:  customerID,
		Content:     fmt.Sprintf("ใบเสนอราคา ฿%.0f", q.GrandTotal.InexactFloat64()),
		MessageType: "quotation_card",
		QuoteData:   stringPtr(string(payload)),
		IsRead:      false,
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

package conversation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/helper"
	"github.com/yourusername/wemake/internal/sse"
	conversationrepo "github.com/yourusername/wemake/internal/repository/conversation"
	rfqrepo "github.com/yourusername/wemake/internal/repository/rfq"
	messageservice "github.com/yourusername/wemake/internal/service/message"
)

type ConversationService struct {
	repo     *conversationrepo.ConversationRepository
	rfqs     *rfqrepo.RFQRepository
	messages *messageservice.MessageService
	hub      *sse.Hub
}

var ErrConversationForbidden = errors.New("conversation forbidden")
var ErrConversationNotFound = errors.New("conversation not found")

func NewConversationService(repo *conversationrepo.ConversationRepository, rfqs *rfqrepo.RFQRepository, messages *messageservice.MessageService, hub *sse.Hub) *ConversationService {
	return &ConversationService{repo: repo, rfqs: rfqs, messages: messages, hub: hub}
}

func (s *ConversationService) ListByUserID(userID int64) ([]domain.ConversationResponse, error) {
	rows, err := s.repo.ListByUserID(userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ConversationResponse, 0, len(rows))
	for i := range rows {
		out = append(out, mapConversation(&rows[i]))
	}
	return out, nil
}

func (s *ConversationService) GetByID(convID, viewerUserID int64) (*domain.ConversationResponse, error) {
	row, err := s.repo.GetByID(convID)
	if err != nil {
		return nil, err
	}
	if row.CustomerID != viewerUserID && row.FactoryID != viewerUserID {
		return nil, ErrConversationForbidden
	}
	out := mapConversation(row)
	role := "CT"
	other := row.FactoryID
	if row.FactoryID == viewerUserID {
		role = "FT"
		other = row.CustomerID
	}
	out.ViewerRole = &role
	out.CounterpartyUserID = &other
	return &out, nil
}

func (s *ConversationService) Create(conv *domain.Conversation) error {
	return s.repo.Create(conv)
}

func (s *ConversationService) CreateFromShowcase(showcaseID, customerID int64) (*domain.ConversationResponse, error) {
	factoryID, err := s.repo.GetFactoryIDByShowcaseID(showcaseID)
	if err != nil {
		return nil, err
	}
	conv := &domain.Conversation{
		CustomerID:       customerID,
		FactoryID:        factoryID,
		SourceShowcaseID: &showcaseID,
		ConvType:         "showcase_inquiry",
	}
	if err := s.repo.Create(conv); err != nil {
		return nil, err
	}
	return s.GetByID(conv.ConvID, customerID)
}

func (s *ConversationService) MarkAsRead(convID, userID int64) error {
	conv, err := s.repo.GetByID(convID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrConversationNotFound
		}
		return err
	}
	if conv.CustomerID != userID && conv.FactoryID != userID {
		return ErrConversationForbidden
	}
	err = s.repo.MarkAsRead(convID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrConversationNotFound
	}
	if err != nil {
		return err
	}

	// Push SSE "messages_read" event to the OTHER participant (the sender)
	// so their UI can flip is_read → true in real-time.
	senderID := conv.CustomerID
	if userID == conv.CustomerID {
		senderID = conv.FactoryID
	}
	if s.hub != nil {
		data, _ := json.Marshal(map[string]any{"conv_id": convID, "reader_id": userID})
		s.hub.Push(senderID, "messages_read", string(data))
	}

	return nil
}

var ErrShareRFQInvalid = errors.New("invalid rfq")
var ErrShareRFQForbidden = errors.New("forbidden")
var ErrShareRFQClosed = errors.New("rfq cannot be shared")

func (s *ConversationService) ShareRFQ(convID, userID, rfqID int64) (*domain.Message, *domain.RFQ, error) {
	if s.rfqs == nil || s.messages == nil {
		return nil, nil, ErrShareRFQInvalid
	}
	conv, err := s.repo.GetByID(convID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrShareRFQInvalid
		}
		return nil, nil, err
	}
	if conv.CustomerID != userID {
		return nil, nil, ErrShareRFQForbidden
	}
	rfq, err := s.rfqs.GetByIDAny(rfqID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrShareRFQInvalid
		}
		return nil, nil, err
	}
	if rfq.UserID != userID {
		return nil, nil, ErrShareRFQForbidden
	}
	switch domainutil.NormalizeStatus(rfq.Status) {
	case "OP", "PD":
	default:
		return nil, nil, ErrShareRFQClosed
	}

	nowUTC := time.Now().UTC()
	msg := &domain.Message{
		ConvID:        &convID,
		ReferenceType: "RQ",
		ReferenceID:   rfqID,
		SenderID:      userID,
		ReceiverID:    conv.FactoryID,
		Content:       rfq.Title,
		MessageType:   "rfq_card",
		IsRead:        false,
		CreatedAt:     nowUTC,
	}
	msg.MessageID = "msg-" + msg.CreatedAt.Format("20060102150405.000000000")
	if err := helper.WithTx(context.Background(), s.rfqs.DB(), func(tx *sqlx.Tx) error {
		if err := s.rfqs.LinkConversationTx(tx, rfqID, userID, convID); err != nil {
			return err
		}
		if err := s.messages.CreateTx(tx, msg); err != nil {
			return err
		}

		if _, err := tx.Exec(`
		UPDATE conversations
		SET last_message = $2,
		    unread_factory = COALESCE(unread_factory, 0) + 1,
		    updated_at = $3
		WHERE conv_id = $1
	`, convID, rfq.Title, nowUTC); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}

	rfq.ConversationID = &convID
	return msg, rfq, nil
}

func mapConversation(row *domain.ConversationRow) domain.ConversationResponse {
	first := helper.DerefString(row.CustomerFirstName)
	last := helper.DerefString(row.CustomerLastName)
	display := strings.TrimSpace(first + " " + last)
	if display == "" {
		display = fmt.Sprintf("ลูกค้า #%d", row.CustomerID)
	}
	factoryName := helper.DerefString(row.FactoryName)
	if factoryName == "" {
		factoryName = fmt.Sprintf("โรงงาน #%d", row.FactoryID)
	}
	return domain.ConversationResponse{
		ConvID:           row.ConvID,
		CustomerID:       row.CustomerID,
		FactoryID:        row.FactoryID,
		SourceShowcaseID: row.SourceShowcaseID,
		ConvType:         row.ConvType,
		LastMessage:      helper.DerefString(row.LastMessage),
		LastMessageType:  helper.DerefString(row.LastMessageType),
		UnreadCustomer:   row.UnreadCustomer,
		UnreadFactory:    row.UnreadFactory,
		UpdatedAt:        row.UpdatedAt,
		Customer: domain.CustomerPartyInfo{
			UserID:      row.CustomerID,
			FirstName:   first,
			LastName:    last,
			DisplayName: display,
		},
		Factory: domain.FactoryPartyInfo{
			UserID:         row.FactoryID,
			FactoryName:    factoryName,
			ImageURL:       helper.DerefString(row.FactoryImageURL),
			IsVerified:     derefBool(row.FactoryIsVerified),
			Specialization: helper.DerefString(row.FactorySpecialization),
		},
	}
}

func derefBool(value *bool) bool {
	return domainutil.BoolValue(value)
}

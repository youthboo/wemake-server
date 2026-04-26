package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type ConversationService struct {
	repo     *repository.ConversationRepository
	rfqs     *repository.RFQRepository
	messages *MessageService
}

var ErrConversationNotFoundOrForbidden = errors.New("conversation not found or forbidden")
var ErrConversationForbidden = errors.New("conversation forbidden")

func NewConversationService(repo *repository.ConversationRepository, rfqs *repository.RFQRepository, messages *MessageService) *ConversationService {
	return &ConversationService{repo: repo, rfqs: rfqs, messages: messages}
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
	err := s.repo.MarkAsRead(convID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrConversationNotFoundOrForbidden
	}
	return err
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
	switch strings.ToUpper(strings.TrimSpace(rfq.Status)) {
	case "OP", "PD":
	default:
		return nil, nil, ErrShareRFQClosed
	}

	tx, err := s.rfqs.DB().Beginx()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	if err := s.rfqs.LinkConversationTx(tx, rfqID, userID, convID); err != nil {
		return nil, nil, err
	}

	msg := &domain.Message{
		ConvID:        &convID,
		ReferenceType: "RQ",
		ReferenceID:   rfqID,
		SenderID:      userID,
		ReceiverID:    conv.FactoryID,
		Content:       rfq.Title,
		MessageType:   "rfq_card",
		IsRead:        false,
		CreatedAt:     time.Now(),
	}
	msg.MessageID = "msg-" + msg.CreatedAt.Format("20060102150405.000000000")
	if err := s.messages.CreateTx(tx, msg); err != nil {
		return nil, nil, err
	}

	if _, err := tx.Exec(`
		UPDATE conversations
		SET has_quote = TRUE,
		    last_message = $2,
		    unread_factory = COALESCE(unread_factory, 0) + 1,
		    updated_at = NOW()
		WHERE conv_id = $1
	`, convID, rfq.Title); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	rfq.ConversationID = &convID
	return msg, rfq, nil
}

func mapConversation(row *domain.ConversationRow) domain.ConversationResponse {
	first := derefString(row.CustomerFirstName)
	last := derefString(row.CustomerLastName)
	display := strings.TrimSpace(first + " " + last)
	if display == "" {
		display = fmt.Sprintf("ลูกค้า #%d", row.CustomerID)
	}
	factoryName := derefString(row.FactoryName)
	if factoryName == "" {
		factoryName = fmt.Sprintf("โรงงาน #%d", row.FactoryID)
	}
	return domain.ConversationResponse{
		ConvID:           row.ConvID,
		CustomerID:       row.CustomerID,
		FactoryID:        row.FactoryID,
		SourceShowcaseID: row.SourceShowcaseID,
		ConvType:         row.ConvType,
		LastMessage:      derefString(row.LastMessage),
		UnreadCustomer:   row.UnreadCustomer,
		UnreadFactory:    row.UnreadFactory,
		HasQuote:         row.HasQuote,
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
			ImageURL:       derefString(row.FactoryImageURL),
			IsVerified:     derefBool(row.FactoryIsVerified),
			Specialization: derefString(row.FactorySpecialization),
		},
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefBool(value *bool) bool {
	return value != nil && *value
}

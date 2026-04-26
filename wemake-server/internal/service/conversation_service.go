package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type ConversationService struct {
	repo *repository.ConversationRepository
}

var ErrConversationNotFoundOrForbidden = errors.New("conversation not found or forbidden")
var ErrConversationForbidden = errors.New("conversation forbidden")

func NewConversationService(repo *repository.ConversationRepository) *ConversationService {
	return &ConversationService{repo: repo}
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

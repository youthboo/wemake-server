package service

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var (
	ErrQuotationLocked     = errors.New("quotation is locked or not in pending status")
	ErrNotQuotationParty   = errors.New("not authorized for this quotation")
	ErrQuotationPatchReason = errors.New("reason is required when updating a quotation")
)

type QuotationService struct {
	repo    *repository.QuotationRepository
	rfqRepo *repository.RFQRepository
}

func NewQuotationService(repo *repository.QuotationRepository, rfqRepo *repository.RFQRepository) *QuotationService {
	return &QuotationService{repo: repo, rfqRepo: rfqRepo}
}

func (s *QuotationService) Create(item *domain.Quotation) error {
	now := time.Now()
	item.Status = "PD"
	item.CreateTime = now
	item.LogTimestamp = now
	item.Version = 1
	item.IsLocked = false
	if err := s.repo.Create(item); err != nil {
		return err
	}
	eb := item.FactoryID
	h := repository.SnapshotFromQuotation(item, "CR", nil, &eb)
	if err := s.repo.InsertHistory(h); err != nil {
		return err
	}
	return nil
}

func (s *QuotationService) ListByRFQID(rfqID int64) ([]domain.Quotation, error) {
	return s.repo.ListByRFQID(rfqID)
}

func (s *QuotationService) ListMine(factoryID int64, status string) ([]domain.Quotation, error) {
	return s.repo.ListByFactoryID(factoryID, strings.TrimSpace(strings.ToUpper(status)))
}

func (s *QuotationService) GetByID(quotationID int64) (*domain.Quotation, error) {
	return s.repo.GetByID(quotationID)
}

func (s *QuotationService) CanView(quoteID, userID int64, role string) (bool, error) {
	q, err := s.repo.GetByID(quoteID)
	if err != nil {
		return false, err
	}
	if role == domain.RoleFactory && q.FactoryID == userID {
		return true, nil
	}
	if role == domain.RoleCustomer {
		rfq, err := s.rfqRepo.GetByIDAny(q.RFQID)
		if err != nil {
			return false, err
		}
		return rfq.UserID == userID, nil
	}
	return false, nil
}

func (s *QuotationService) ListHistory(quoteID int64) ([]domain.QuotationHistoryEntry, error) {
	return s.repo.ListHistory(quoteID)
}

func (s *QuotationService) UpdateStatus(quoteID int64, status string, editorID *int64) error {
	q, err := s.repo.GetByID(quoteID)
	if err != nil {
		return err
	}
	old := q.Status
	if err := s.repo.UpdateStatus(quoteID, strings.TrimSpace(strings.ToUpper(status))); err != nil {
		return err
	}
	if old == strings.TrimSpace(strings.ToUpper(status)) {
		return nil
	}
	q2, err := s.repo.GetByID(quoteID)
	if err != nil {
		return err
	}
	st := q2.Status
	return s.repo.InsertHistory(&domain.QuotationHistoryEntry{
		QuoteID:      q2.QuotationID,
		EventType:    "ST",
		VersionAfter: q2.Version,
		Status:       &st,
		EditedBy:     editorID,
	})
}

func (s *QuotationService) PatchBody(quoteID, factoryUserID int64, pricePerPiece, moldCost float64, leadTimeDays, shippingMethodID int64, reason string) (*domain.Quotation, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, ErrQuotationPatchReason
	}
	ok, err := s.repo.ShippingMethodValid(shippingMethodID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidShippingMethod
	}
	q, err := s.repo.GetByID(quoteID)
	if err != nil {
		return nil, err
	}
	if q.FactoryID != factoryUserID {
		return nil, ErrNotQuotationParty
	}
	if q.IsLocked || q.Status != "PD" {
		return nil, ErrQuotationLocked
	}
	newVersion := q.Version + 1
	if err := s.repo.UpdateBody(quoteID, pricePerPiece, moldCost, leadTimeDays, shippingMethodID, factoryUserID, newVersion); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQuotationLocked
		}
		return nil, err
	}
	q2, err := s.repo.GetByID(quoteID)
	if err != nil {
		return nil, err
	}
	rs := strings.TrimSpace(reason)
	eb := factoryUserID
	pp := q2.PricePerPiece
	mc := q2.MoldCost
	lt := q2.LeadTimeDays
	sm := q2.ShippingMethodID
	st := q2.Status
	h := &domain.QuotationHistoryEntry{
		QuoteID:          q2.QuotationID,
		EventType:        "UP",
		VersionAfter:     q2.Version,
		PricePerPiece:    &pp,
		MoldCost:         &mc,
		LeadTimeDays:     &lt,
		ShippingMethodID: &sm,
		Status:           &st,
		Reason:           &rs,
		EditedBy:         &eb,
	}
	if err := s.repo.InsertHistory(h); err != nil {
		return nil, err
	}
	return q2, nil
}

package service

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var (
	ErrQuotationLocked      = errors.New("quotation is locked or not in pending status")
	ErrNotQuotationParty    = errors.New("not authorized for this quotation")
	ErrQuotationPatchReason = errors.New("reason is required when updating a quotation")
	ErrInvalidLineItem      = errors.New("INVALID_LINE_ITEM")
	ErrIncotermsInvalid     = errors.New("INCOTERMS_INVALID")
	ErrPaymentTermsInvalid  = errors.New("PAYMENT_TERMS_INVALID")
	ErrQuotationExpired     = errors.New("QUOTATION_EXPIRED")
	ErrFactorySuspended     = errors.New("FACTORY_SUSPENDED")
)

type QuotationService struct {
	db         *sqlx.DB
	repo       *repository.QuotationRepository
	rfqRepo    *repository.RFQRepository
	items      *repository.QuotationItemRepository
	commission *CommissionService
	orders     *OrderService
	factories  *repository.FactoryRepository
}

func NewQuotationService(db *sqlx.DB, repo *repository.QuotationRepository, rfqRepo *repository.RFQRepository, items *repository.QuotationItemRepository, commission *CommissionService, orders *OrderService, factories *repository.FactoryRepository) *QuotationService {
	return &QuotationService{db: db, repo: repo, rfqRepo: rfqRepo, items: items, commission: commission, orders: orders, factories: factories}
}

func (s *QuotationService) Create(item *domain.Quotation) error {
	if s.factories != nil {
		approvalStatus, err := s.factories.GetApprovalStatus(item.FactoryID)
		if err != nil {
			return err
		}
		if approvalStatus == "SU" {
			return ErrFactorySuspended
		}
	}
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
	item, err := s.repo.GetByID(quotationID)
	if err != nil {
		return nil, err
	}
	if s.items != nil {
		items, err := s.items.ListByQuotation(quotationID)
		if err != nil {
			return nil, err
		}
		item.Items = items
	}
	return item, nil
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

func (s *QuotationService) ListRevisionChain(quoteID int64) ([]domain.Quotation, error) {
	root, err := s.repo.GetByID(quoteID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListRevisionChain(root)
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

func (s *QuotationService) UpdateImageURLs(quoteID int64, imageURLs domain.StringArray) error {
	return s.repo.UpdateImageURLs(quoteID, imageURLs)
}

func (s *QuotationService) Preview(items []domain.QuotationItem, discountAmount, shippingCost, packagingCost, toolingCost float64) (*Breakdown, error) {
	if err := validateQuotationItems(items); err != nil {
		return nil, err
	}
	return s.commission.Calculate(CommissionInput{
		Items:          items,
		DiscountAmount: discountAmount,
		ShippingCost:   shippingCost,
		PackagingCost:  packagingCost,
		ToolingCost:    toolingCost,
	})
}

func validateQuotationItems(items []domain.QuotationItem) error {
	if len(items) == 0 {
		return ErrInvalidLineItem
	}
	for _, item := range items {
		if item.Qty <= 0 || item.UnitPrice < 0 || item.DiscountPct < 0 || item.DiscountPct > 100 || strings.TrimSpace(item.Description) == "" {
			return ErrInvalidLineItem
		}
	}
	return nil
}

func validateQuotationTerms(incoterms, paymentTerms *string, validityDays int) error {
	if incoterms != nil {
		switch strings.TrimSpace(strings.ToUpper(*incoterms)) {
		case "EXW", "FOB", "CIF", "DDP":
		default:
			return ErrIncotermsInvalid
		}
	}
	if paymentTerms != nil {
		switch strings.TrimSpace(*paymentTerms) {
		case "50_50", "30_70", "net_30", "lc_at_sight":
		default:
			return ErrPaymentTermsInvalid
		}
	}
	if validityDays < 1 || validityDays > 365 {
		return ErrInvalidLineItem
	}
	return nil
}

func (s *QuotationService) CreateDetailed(item *domain.Quotation) error {
	if err := validateQuotationItems(item.Items); err != nil {
		return err
	}
	if err := validateQuotationTerms(item.Incoterms, item.PaymentTerms, item.ValidityDays); err != nil {
		return err
	}
	breakdown, err := s.commission.Calculate(CommissionInput{
		Items:          item.Items,
		DiscountAmount: item.DiscountAmount,
		ShippingCost:   item.ShippingCost,
		PackagingCost:  item.PackagingCost,
		ToolingCost:    item.ToolingMoldCost,
		FactoryID:      &item.FactoryID,
	})
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	item.Status = "PD"
	item.CreateTime = now
	item.LogTimestamp = now
	item.Version = 1
	item.IsLocked = false
	if item.ValidityDays == 0 {
		item.ValidityDays = 30
	}
	validUntil := now.AddDate(0, 0, item.ValidityDays)
	item.ValidUntil = &validUntil
	item.Subtotal = breakdown.Subtotal
	item.VatRate = breakdown.VatRate
	item.VatAmount = breakdown.VatAmount
	item.GrandTotal = breakdown.GrandTotal
	item.PlatformCommissionRate = breakdown.PlatformCommissionRate
	item.PlatformCommissionAmount = breakdown.PlatformCommissionAmount
	item.FactoryNetReceivable = breakdown.FactoryNetReceivable
	item.PlatformConfigID = &breakdown.PlatformConfigID
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if item.ParentQuotationID != nil {
		if err := s.repo.MarkAncestorsRevised(tx, item.RFQID, item.FactoryID); err != nil {
			return err
		}
	}
	if err := s.repo.CreateTx(tx, item); err != nil {
		return err
	}
	if err := s.items.BulkInsert(tx, item.QuotationID, item.Items); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *QuotationService) CreateRevision(parentID, factoryID int64, next *domain.Quotation) error {
	parent, err := s.repo.GetByID(parentID)
	if err != nil {
		return err
	}
	if parent.FactoryID != factoryID {
		return ErrNotQuotationParty
	}
	next.ParentQuotationID = &parentID
	next.RevisionNo = parent.RevisionNo + 1
	next.FactoryID = factoryID
	next.RFQID = parent.RFQID
	return s.CreateDetailed(next)
}

func (s *QuotationService) Accept(quoteID, customerID int64) (*domain.Order, error) {
	q, err := s.repo.GetByID(quoteID)
	if err != nil {
		return nil, err
	}
	rfq, err := s.rfqRepo.GetByIDAny(q.RFQID)
	if err != nil {
		return nil, err
	}
	if rfq.UserID != customerID {
		return nil, ErrNotQuotationParty
	}
	if q.ValidUntil != nil && q.ValidUntil.Before(time.Now().UTC()) {
		return nil, ErrQuotationExpired
	}
	return s.orders.CreateFromQuotation(quoteID, customerID)
}

func (s *QuotationService) Reject(quoteID, customerID int64) error {
	q, err := s.repo.GetByID(quoteID)
	if err != nil {
		return err
	}
	rfq, err := s.rfqRepo.GetByIDAny(q.RFQID)
	if err != nil {
		return err
	}
	if rfq.UserID != customerID {
		return ErrNotQuotationParty
	}
	return s.repo.UpdateStatus(quoteID, "RJ")
}

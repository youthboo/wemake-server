package quotation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/yourusername/wemake/internal/helper"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	factoryrepo "github.com/yourusername/wemake/internal/repository/factory"
	quotationrepo "github.com/yourusername/wemake/internal/repository/quotation"
	rfqrepo "github.com/yourusername/wemake/internal/repository/rfq"
	orderservice "github.com/yourusername/wemake/internal/service/order"
	walletservice "github.com/yourusername/wemake/internal/service/wallet"
)

var (
	ErrQuotationLocked         = errors.New("quotation is locked or not in pending status")
	ErrNotQuotationParty       = errors.New("not authorized for this quotation")
	ErrInvalidLineItem         = errors.New("INVALID_LINE_ITEM")
	ErrInvalidShippingMethod   = errors.New("shipping_method_id is invalid")
	ErrIncotermsInvalid        = errors.New("INCOTERMS_INVALID")
	ErrPaymentTermsInvalid     = errors.New("PAYMENT_TERMS_INVALID")
	ErrQuotationExpired        = errors.New("QUOTATION_EXPIRED")
	ErrFactorySuspended        = errors.New("FACTORY_SUSPENDED")
	ErrFactoryHighlightInvalid = errors.New("factory_highlight must be at most 200 characters")
)

type QuotationService struct {
	db            *sqlx.DB
	repo          *quotationrepo.QuotationRepository
	rfqRepo       *rfqrepo.RFQRepository
	items         *quotationrepo.QuotationItemRepository
	commission    *walletservice.CommissionService
	orders        *orderservice.OrderService
	factories     *factoryrepo.FactoryRepository
	notifications notificationCreator
	messages      systemMessageSender
}

type notificationCreator interface {
	Create(*domain.Notification) error
}

type systemMessageSender interface {
	AutoSendSystemMessage(context.Context, int64, int64, int64, string) error
	AutoSendQuotationCard(context.Context, int64, int64, *domain.Quotation) error
}

func NewQuotationService(db *sqlx.DB, repo *quotationrepo.QuotationRepository, rfqRepo *rfqrepo.RFQRepository, items *quotationrepo.QuotationItemRepository, commission *walletservice.CommissionService, orders *orderservice.OrderService, factories *factoryrepo.FactoryRepository, notifications notificationCreator, messages systemMessageSender) *QuotationService {
	return &QuotationService{db: db, repo: repo, rfqRepo: rfqRepo, items: items, commission: commission, orders: orders, factories: factories, notifications: notifications, messages: messages}
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
	if err := normalizeFactoryHighlight(item); err != nil {
		return err
	}
	now := time.Now()
	item.Status = domain.QuotationStatusPrepared
	item.CreateTime = now
	item.LogTimestamp = now
	item.Version = 1
	item.IsLocked = false
	if item.ValidityDays <= 0 {
		item.ValidityDays = domain.DefaultQuotationValidityDays
	}
	validUntil := now.AddDate(0, 0, item.ValidityDays)
	item.ValidUntil = &validUntil

	rfqQty := float64(1)
	if s.rfqRepo != nil {
		if rfq, err := s.rfqRepo.GetByIDAny(item.RFQID); err == nil && rfq != nil && rfq.Quantity > 0 {
			rfqQty = float64(rfq.Quantity)
			if item.ShippingMethodID <= 0 && rfq.ShippingMethodID != nil {
				item.ShippingMethodID = *rfq.ShippingMethodID
			}
		}
	}
	if item.ShippingMethodID <= 0 {
		item.ShippingMethodID = 2 // default: จัดส่งเดลิเวอรี่
	}
	if err := validateQuotationTerms(nil, item.PaymentTerms, item.ValidityDays); err != nil {
		return err
	}
	if item.ShippingMethodID > 0 {
		ok, err := s.repo.ShippingMethodValid(item.ShippingMethodID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrInvalidShippingMethod
		}
	}
	if s.commission != nil {
		breakdown, err := s.commission.Calculate(walletservice.CommissionInput{
			Items: []domain.QuotationItem{{
				Description: "สินค้า",
				Qty:         rfqQty,
				UnitPrice:   helper.DecimalToFloat(item.PricePerPiece),
				DiscountPct: 0,
			}},
			DiscountAmount: helper.DecimalToFloat(item.DiscountAmount),
			ShippingCost:   helper.DecimalToFloat(item.ShippingCost),
			PackagingCost:  helper.DecimalToFloat(item.PackagingCost),
			ToolingCost:    helper.DecimalToFloat(item.ToolingMoldCost),
			FactoryID:      &item.FactoryID,
		})
		if err == nil {
			item.Subtotal = helper.MoneyDecimal(breakdown.Subtotal)
			item.VatRate = helper.MoneyDecimal(breakdown.VatRate)
			item.VatAmount = helper.MoneyDecimal(breakdown.VatAmount)
			item.GrandTotal = helper.MoneyDecimal(breakdown.GrandTotal)
			item.PlatformCommissionRate = helper.MoneyDecimal(breakdown.PlatformCommissionRate)
			item.PlatformCommissionAmount = helper.MoneyDecimal(breakdown.PlatformCommissionAmount)
			item.FactoryNetReceivable = helper.MoneyDecimal(breakdown.FactoryNetReceivable)
			item.PlatformConfigID = &breakdown.PlatformConfigID
		}
	}
	if err := s.repo.Create(item); err != nil {
		return err
	}
	eb := item.FactoryID
	h := quotationrepo.SnapshotFromQuotation(item, "CR", nil, &eb)
	if err := s.repo.InsertHistory(h); err != nil {
		return err
	}

	s.notifyQuotationQuoted(item)
	s.autoSendQuotationCard(item)
	return nil
}

func (s *QuotationService) ListByRFQID(rfqID int64) ([]domain.Quotation, error) {
	return s.repo.ListByRFQID(rfqID)
}

func (s *QuotationService) ListMine(factoryID int64, status string) ([]domain.Quotation, error) {
	return s.repo.ListByFactoryID(factoryID, domainutil.NormalizeStatus(status))
}

func (s *QuotationService) GetByID(quotationID int64) (*domain.Quotation, error) {
	item, err := s.repo.GetByID(quotationID)
	if err != nil {
		return nil, err
	}
	if s.items != nil {
		items, err := s.items.ListByQuotation(quotationID)
		if err == nil {
			item.Items = items
		}
		// Non-fatal: quotation_items table may not exist yet; continue without line items.
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

func (s *QuotationService) HistoriesForQuotes(quotes []domain.Quotation) (map[string][]domain.QuotationHistoryEntry, error) {
	out := make(map[string][]domain.QuotationHistoryEntry, len(quotes))
	for _, q := range quotes {
		entries, err := s.ListHistory(q.QuotationID)
		if err != nil {
			return nil, err
		}
		if entries == nil {
			entries = []domain.QuotationHistoryEntry{}
		}
		out[formatQuoteIDKey(q.QuotationID)] = entries
	}
	return out, nil
}

func formatQuoteIDKey(quoteID int64) string {
	return strconv.FormatInt(quoteID, 10)
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
	if err := s.repo.UpdateStatus(quoteID, domainutil.NormalizeStatus(status)); err != nil {
		return err
	}
	if old == domainutil.NormalizeStatus(status) {
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

func (s *QuotationService) PatchBody(
	quoteID, factoryUserID int64,
	pricePerPiece, moldCost, shippingCost, packagingCost, toolingMoldCost float64,
	leadTimeDays, shippingMethodID int64,
	paymentTerms *string,
	factoryHighlight *string,
	reason string,
	validityDays int,
	factoryNote *string,
) (*domain.Quotation, error) {
	if strings.TrimSpace(reason) == "" {
		reason = "อัปเดตใบเสนอราคา"
	}
	if paymentTerms != nil {
		if err := validateQuotationTerms(nil, paymentTerms, domain.DefaultQuotationTermsDays); err != nil {
			return nil, err
		}
	}
	nextHighlight := factoryHighlight
	if nextHighlight != nil {
		trimmed := strings.TrimSpace(*nextHighlight)
		if trimmed == "" {
			nextHighlight = nil
		} else {
			if len([]rune(trimmed)) > 200 {
				return nil, ErrFactoryHighlightInvalid
			}
			nextHighlight = &trimmed
		}
	}
	if shippingMethodID > 0 {
		ok, err := s.repo.ShippingMethodValid(shippingMethodID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrInvalidShippingMethod
		}
	}
	q, err := s.repo.GetByID(quoteID)
	if err != nil {
		return nil, err
	}
	if q.FactoryID != factoryUserID {
		return nil, ErrNotQuotationParty
	}
	if q.IsLocked || q.Status != domain.QuotationStatusPrepared {
		return nil, ErrQuotationLocked
	}
	newVersion := q.Version + 1
	// คำนวณ validity_days + valid_until — ถ้าไม่ส่งมา (0) ให้คงค่าเดิมไว้ใน DB
	var validityDaysPtr *int
	var validUntilPtr *time.Time
	if validityDays > 0 {
		validityDaysPtr = &validityDays
		vu := q.CreateTime.AddDate(0, 0, validityDays)
		validUntilPtr = &vu
	}
	if err := s.repo.UpdateBody(quoteID, pricePerPiece, moldCost, shippingCost, packagingCost, toolingMoldCost, leadTimeDays, shippingMethodID, factoryUserID, newVersion, paymentTerms, nextHighlight, validityDaysPtr, validUntilPtr, factoryNote); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQuotationLocked
		}
		return nil, err
	}
	q2, err := s.repo.GetByID(quoteID)
	if err != nil {
		return nil, err
	}
	rfqQty := float64(1)
	if s.rfqRepo != nil {
		if rfq, rfqErr := s.rfqRepo.GetByIDAny(q2.RFQID); rfqErr == nil && rfq != nil && rfq.Quantity > 0 {
			rfqQty = float64(rfq.Quantity)
		}
	}
	if s.commission != nil {
		breakdown, calcErr := s.commission.Calculate(walletservice.CommissionInput{
			Items: []domain.QuotationItem{{
				Description: "สินค้า",
				Qty:         rfqQty,
				UnitPrice:   helper.DecimalToFloat(q2.PricePerPiece),
				DiscountPct: 0,
			}},
			DiscountAmount: helper.DecimalToFloat(q2.DiscountAmount),
			ShippingCost:   helper.DecimalToFloat(q2.ShippingCost),
			PackagingCost:  helper.DecimalToFloat(q2.PackagingCost),
			ToolingCost:    helper.DecimalToFloat(q2.ToolingMoldCost),
			FactoryID:      &q2.FactoryID,
		})
		if calcErr == nil {
			if updateErr := s.repo.UpdateTotals(
				quoteID,
				breakdown.Subtotal,
				breakdown.VatRate,
				breakdown.VatAmount,
				breakdown.PlatformCommissionRate,
				breakdown.PlatformCommissionAmount,
				breakdown.GrandTotal,
				breakdown.FactoryNetReceivable,
			); updateErr != nil {
				return nil, updateErr
			}
			q2, err = s.repo.GetByID(quoteID)
			if err != nil {
				return nil, err
			}
		}
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

// PatchFactoryNote updates factory_note only — bypasses lock/status.
// Only the factory that owns the quotation can update it.
func (s *QuotationService) PatchFactoryNote(quoteID, factoryID int64, note *string) error {
	if err := s.repo.UpdateFactoryNote(quoteID, factoryID, note); err != nil {
		return err
	}
	return nil
}

func (s *QuotationService) Preview(items []domain.QuotationItem, discountAmount, shippingCost, packagingCost, toolingCost float64, factoryID *int64) (*walletservice.Breakdown, error) {
	if err := validateQuotationItems(items); err != nil {
		return nil, err
	}
	return s.commission.Calculate(walletservice.CommissionInput{
		Items:          items,
		DiscountAmount: discountAmount,
		ShippingCost:   shippingCost,
		PackagingCost:  packagingCost,
		ToolingCost:    toolingCost,
		FactoryID:      factoryID,
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
		if !domainutil.StatusIn(*incoterms, "EXW", "FOB", "CIF", "DDP") {
			return ErrIncotermsInvalid
		}
	}
	if paymentTerms != nil {
		switch strings.TrimSpace(*paymentTerms) {
		case "50_50", "30_70", "net_30", "lc_at_sight", "full_payment":
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
	if err := normalizeFactoryHighlight(item); err != nil {
		return err
	}
	if err := validateQuotationTerms(item.Incoterms, item.PaymentTerms, item.ValidityDays); err != nil {
		return err
	}
	if item.ShippingMethodID <= 0 {
		item.ShippingMethodID = 2 // default: จัดส่งเดลิเวอรี่
	}
	breakdown, err := s.commission.Calculate(walletservice.CommissionInput{
		Items:          item.Items,
		DiscountAmount: helper.DecimalToFloat(item.DiscountAmount),
		ShippingCost:   helper.DecimalToFloat(item.ShippingCost),
		PackagingCost:  helper.DecimalToFloat(item.PackagingCost),
		ToolingCost:    helper.DecimalToFloat(item.ToolingMoldCost),
		FactoryID:      &item.FactoryID,
	})
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	item.Status = domain.QuotationStatusPrepared
	item.CreateTime = now
	item.LogTimestamp = now
	item.Version = 1
	item.IsLocked = false
	if item.ValidityDays == 0 {
		item.ValidityDays = domain.DefaultQuotationValidityDays
	}
	validUntil := now.AddDate(0, 0, item.ValidityDays)
	item.ValidUntil = &validUntil
	item.Subtotal = helper.MoneyDecimal(breakdown.Subtotal)
	item.VatRate = helper.MoneyDecimal(breakdown.VatRate)
	item.VatAmount = helper.MoneyDecimal(breakdown.VatAmount)
	item.GrandTotal = helper.MoneyDecimal(breakdown.GrandTotal)
	item.PlatformCommissionRate = helper.MoneyDecimal(breakdown.PlatformCommissionRate)
	item.PlatformCommissionAmount = helper.MoneyDecimal(breakdown.PlatformCommissionAmount)
	item.FactoryNetReceivable = helper.MoneyDecimal(breakdown.FactoryNetReceivable)
	item.PlatformConfigID = &breakdown.PlatformConfigID
	if err := helper.WithTx(context.Background(), s.db, func(tx *sqlx.Tx) error {
		if item.ParentQuotationID != nil {
			if err := s.repo.MarkAncestorsRevised(tx, item.RFQID, item.FactoryID); err != nil {
				return err
			}
		}
		if err := s.repo.CreateTx(tx, item); err != nil {
			return err
		}
		return s.items.BulkInsert(tx, item.QuotationID, item.Items)
	}); err != nil {
		return err
	}
	s.notifyQuotationQuoted(item)
	s.autoSendQuotationCard(item)
	return nil
}

func normalizeFactoryHighlight(item *domain.Quotation) error {
	if item == nil || item.FactoryHighlight == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*item.FactoryHighlight)
	if trimmed == "" {
		item.FactoryHighlight = nil
		return nil
	}
	if len([]rune(trimmed)) > 200 {
		return ErrFactoryHighlightInvalid
	}
	item.FactoryHighlight = &trimmed
	return nil
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
	order, err := s.orders.CreateFromQuotation(quoteID, customerID)
	if err != nil {
		return nil, err
	}
	helper.CreateNotificationSafe(s.notifications, &domain.Notification{
		UserID:  q.FactoryID,
		Type:    "QUOTATION_ACCEPTED",
		Title:   "ใบเสนอราคาได้รับการยอมรับ",
		Message: fmt.Sprintf("ลูกค้ายอมรับ Quote #%d", q.QuotationID),
		LinkTo:  helper.OrderLink(order.OrderID),
		Data: helper.NotificationData(map[string]interface{}{
			"quote_id": q.QuotationID,
			"order_id": order.OrderID,
			"url":      helper.OrderLink(order.OrderID),
		}),
		ReferenceID: &order.OrderID,
		CreatedAt:   time.Now(),
	})
	return order, nil
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
	if err := s.repo.UpdateStatus(quoteID, "RJ"); err != nil {
		return err
	}
	helper.CreateNotificationSafe(s.notifications, &domain.Notification{
		UserID:  q.FactoryID,
		Type:    "QUOTATION_REJECTED",
		Title:   "ใบเสนอราคาถูกปฏิเสธ",
		Message: fmt.Sprintf("Quote #%d ถูกปฏิเสธ", q.QuotationID),
		LinkTo:  helper.FactoryRFQLink(rfq.RFQID),
		Data: helper.NotificationData(map[string]interface{}{
			"rfq_id":   rfq.RFQID,
			"quote_id": q.QuotationID,
			"url":      helper.FactoryRFQLink(rfq.RFQID),
		}),
		ReferenceID: &q.QuotationID,
		CreatedAt:   time.Now(),
	})
	return nil
}

func (s *QuotationService) notifyQuotationQuoted(item *domain.Quotation) {
	if s.notifications == nil || item == nil {
		return
	}
	rfq, err := s.rfqRepo.GetByIDAny(item.RFQID)
	if err != nil {
		return
	}
	title := "ได้รับใบเสนอราคา"
	factoryName := fmt.Sprintf("โรงงาน #%d", item.FactoryID)
	// 1. ใช้ชื่อที่ติดมากับ item (เช่นกรณี CreateDetailed populate ไว้แล้ว)
	if item.FactoryName != nil && strings.TrimSpace(*item.FactoryName) != "" {
		factoryName = strings.TrimSpace(*item.FactoryName)
	} else if s.factories != nil {
		// 2. fallback: ดึงจาก factory_profiles โดยตรง (กรณี Create() ธรรมดาที่ไม่ populate ชื่อ)
		if name := s.factories.GetFactoryName(item.FactoryID); name != "" {
			factoryName = name
		}
	}
	rfqTitle := strings.TrimSpace(rfq.Title)
	if rfqTitle == "" {
		rfqTitle = fmt.Sprintf("RFQ #%d", rfq.RFQID)
	}
	helper.CreateNotificationSafe(s.notifications, &domain.Notification{
		UserID:  rfq.UserID,
		Type:    "RFQ_QUOTED",
		Title:   title,
		Message: fmt.Sprintf("โรงงาน %s ส่งใบเสนอราคาสำหรับ %s", factoryName, rfqTitle),
		LinkTo:  helper.RFQLink(rfq.RFQID),
		Data: helper.NotificationData(map[string]interface{}{
			"rfq_id":     rfq.RFQID,
			"quote_id":   item.QuotationID,
			"factory_id": item.FactoryID,
			"url":        helper.RFQLink(rfq.RFQID),
		}),
		ReferenceID: &item.QuotationID,
		CreatedAt:   item.CreateTime,
	})
}

func (s *QuotationService) autoSendQuotationCard(item *domain.Quotation) {
	if s.messages == nil || item == nil {
		return
	}
	rfq, err := s.rfqRepo.GetByIDAny(item.RFQID)
	if err != nil || rfq == nil || rfq.ConversationID == nil {
		return
	}
	_ = s.messages.AutoSendQuotationCard(context.Background(), *rfq.ConversationID, rfq.UserID, item)
}

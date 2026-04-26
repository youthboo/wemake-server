package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var (
	ErrBOQNotFound       = errors.New("BOQ_NOT_FOUND")
	ErrBOQForbidden      = errors.New("BOQ_FORBIDDEN")
	ErrBOQInvalidItems   = errors.New("BOQ_INVALID_ITEMS")
	ErrBOQInvalidState   = errors.New("BOQ_INVALID_STATE")
	ErrBOQExpired        = errors.New("BOQ_EXPIRED")
	ErrBOQAlreadyHandled = errors.New("BOQ_ALREADY_HANDLED")
)

type BOQInput struct {
	Items          []domain.RFQItem
	Currency       string
	DiscountAmount float64
	VatPercent     float64
	MOQ            *int
	LeadTimeDays   *int
	PaymentTerms   *string
	ValidityDays   *int
	Note           *string
}

type BOQService struct {
	db             *sqlx.DB
	conversations  *repository.ConversationRepository
	rfqs           *repository.RFQRepository
	rfqItems       *repository.RFQItemRepository
	quotations     *repository.QuotationRepository
	quotationItems *repository.QuotationItemRepository
	orders         *OrderService
	messages       *MessageService
	notifications  *NotificationService
	commissions    *CommissionService
}

func NewBOQService(
	db *sqlx.DB,
	conversations *repository.ConversationRepository,
	rfqs *repository.RFQRepository,
	rfqItems *repository.RFQItemRepository,
	quotations *repository.QuotationRepository,
	quotationItems *repository.QuotationItemRepository,
	orders *OrderService,
	messages *MessageService,
	notifications *NotificationService,
	commissions *CommissionService,
) *BOQService {
	return &BOQService{
		db: db, conversations: conversations, rfqs: rfqs, rfqItems: rfqItems,
		quotations: quotations, quotationItems: quotationItems, orders: orders,
		messages: messages, notifications: notifications, commissions: commissions,
	}
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}

func computeBOQTotals(items []domain.RFQItem, discountAmount float64, vatPercent float64) (float64, float64, float64) {
	subtotal := 0.0
	for i := range items {
		line := roundMoney(items[i].Qty * items[i].UnitPrice * (1 - items[i].DiscountPct/100))
		items[i].LineTotal = line
		subtotal += line
	}
	vatBase := roundMoney(subtotal - discountAmount)
	vatAmount := roundMoney(vatBase * vatPercent / 100)
	grandTotal := roundMoney(vatBase + vatAmount)
	return roundMoney(subtotal), vatAmount, grandTotal
}

func normalizeBOQInput(in BOQInput) BOQInput {
	in.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	if in.Currency == "" {
		in.Currency = "THB"
	}
	if in.ValidityDays == nil || *in.ValidityDays <= 0 {
		v := 14
		in.ValidityDays = &v
	}
	if in.VatPercent < 0 {
		in.VatPercent = 0
	}
	note := strings.TrimSpace(derefString(in.Note))
	if note == "" {
		in.Note = nil
	} else {
		in.Note = &note
	}
	paymentTerms := strings.TrimSpace(derefString(in.PaymentTerms))
	if paymentTerms == "" {
		in.PaymentTerms = nil
	} else {
		in.PaymentTerms = &paymentTerms
	}
	for i := range in.Items {
		in.Items[i].Description = strings.TrimSpace(in.Items[i].Description)
		if in.Items[i].ItemNo <= 0 {
			in.Items[i].ItemNo = i + 1
		}
		if in.Items[i].Specification != nil {
			spec := strings.TrimSpace(*in.Items[i].Specification)
			if spec == "" {
				in.Items[i].Specification = nil
			} else {
				in.Items[i].Specification = &spec
			}
		}
		if in.Items[i].Unit != nil {
			unit := strings.TrimSpace(*in.Items[i].Unit)
			if unit == "" {
				in.Items[i].Unit = nil
			} else {
				in.Items[i].Unit = &unit
			}
		}
		if in.Items[i].Note != nil {
			n := strings.TrimSpace(*in.Items[i].Note)
			if n == "" {
				in.Items[i].Note = nil
			} else {
				in.Items[i].Note = &n
			}
		}
	}
	return in
}

func validateBOQInput(in BOQInput) error {
	if len(in.Items) == 0 {
		return ErrBOQInvalidItems
	}
	for _, item := range in.Items {
		if item.Description == "" || item.Qty <= 0 || item.UnitPrice < 0 || item.DiscountPct < 0 || item.DiscountPct > 100 {
			return ErrBOQInvalidItems
		}
	}
	if in.DiscountAmount < 0 || in.VatPercent < 0 {
		return ErrBOQInvalidItems
	}
	return nil
}

func boqExpiresAt(rfq *domain.RFQ) *time.Time {
	if rfq == nil || rfq.BOQSentAt == nil {
		return nil
	}
	days := 14
	if rfq.BOQValidityDays != nil && *rfq.BOQValidityDays > 0 {
		days = *rfq.BOQValidityDays
	}
	t := rfq.BOQSentAt.Add(time.Duration(days) * 24 * time.Hour)
	return &t
}

func boqIsExpired(rfq *domain.RFQ) bool {
	expiresAt := boqExpiresAt(rfq)
	return expiresAt != nil && time.Now().After(*expiresAt)
}

func (s *BOQService) Create(convID, actorUserID int64, input BOQInput) (*domain.BOQDetail, *domain.Message, error) {
	conv, err := s.conversations.GetByID(convID)
	if err != nil {
		return nil, nil, ErrBOQNotFound
	}
	if conv.FactoryID != actorUserID {
		return nil, nil, ErrBOQForbidden
	}

	input = normalizeBOQInput(input)
	if err := validateBOQInput(input); err != nil {
		return nil, nil, err
	}
	subtotal, vatAmount, grandTotal := computeBOQTotals(input.Items, input.DiscountAmount, input.VatPercent)
	quantity := int64(0)
	for _, item := range input.Items {
		quantity += int64(math.Round(item.Qty))
	}
	if quantity <= 0 {
		quantity = 1
	}

	primaryCategoryID, _ := s.lookupFactoryPrimaryCategory(actorUserID)
	now := time.Now()
	title := fmt.Sprintf("BOQ - %s - %s", derefString(conv.FactoryName), now.Format("2006-01-02"))
	details := derefString(input.Note)
	if details == "" {
		details = input.Items[0].Description
	}
	currency := input.Currency
	rfq := &domain.RFQ{
		UserID:            conv.CustomerID,
		CategoryID:        primaryCategoryID,
		Title:             title,
		Quantity:          quantity,
		Details:           details,
		Status:            "OP",
		CreatedAt:         now,
		UpdatedAt:         now,
		RFQType:           "BQ",
		InitiatedBy:       "factory",
		FactoryUserID:     &actorUserID,
		SourceShowcaseID:  conv.SourceShowcaseID,
		SourceConvID:      &convID,
		BOQCurrency:       &currency,
		BOQSubtotal:       &subtotal,
		BOQDiscountAmount: &input.DiscountAmount,
		BOQVatPercent:     &input.VatPercent,
		BOQVatAmount:      &vatAmount,
		BOQGrandTotal:     &grandTotal,
		BOQMOQ:            input.MOQ,
		BOQLeadTimeDays:   input.LeadTimeDays,
		BOQPaymentTerms:   input.PaymentTerms,
		BOQValidityDays:   input.ValidityDays,
		BOQNote:           input.Note,
		BOQSentAt:         &now,
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.rfqs.CreateTx(tx, rfq); err != nil {
		return nil, nil, err
	}
	for i := range input.Items {
		input.Items[i].RFQID = rfq.RFQID
	}
	if err := s.rfqItems.BulkInsertTx(tx, rfq.RFQID, input.Items); err != nil {
		return nil, nil, err
	}
	if _, err := tx.Exec(`
		UPDATE conversations
		SET has_quote = TRUE,
		    last_message = 'BOQ ใหม่',
		    unread_customer = COALESCE(unread_customer, 0) + 1,
		    updated_at = NOW(),
		    conv_type = CASE WHEN conv_type = 'general' THEN 'boq' ELSE conv_type END
		WHERE conv_id = $1
	`, convID); err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	quoteDataStr, _ := s.buildBOQQuoteDataString(rfq, input.Items, derefString(conv.FactoryName))
	msg := &domain.Message{
		SenderID:    actorUserID,
		ReceiverID:  conv.CustomerID,
		Content:     "โรงงานส่ง BOQ มาให้แล้ว กรุณาตรวจสอบ",
		ConvID:      &convID,
		MessageType: "BQ",
		QuoteData:   quoteDataStr,
		BOQRfqID:    &rfq.RFQID,
		IsRead:      false,
	}
	if err := s.messages.Create(msg); err != nil {
		return nil, nil, err
	}
	if s.notifications != nil {
		_ = s.notifications.Create(&domain.Notification{
			UserID:      conv.CustomerID,
			Type:        "BQ",
			Title:       "โรงงานส่ง BOQ มาแล้ว",
			Message:     fmt.Sprintf("%s ส่ง BOQ มูลค่า ฿%.2f มาให้คุณ", derefString(conv.FactoryName), grandTotal),
			LinkTo:      fmt.Sprintf("/chat/%d", convID),
			ReferenceID: &rfq.RFQID,
		})
	}
	detail, _, err := s.Get(rfq.RFQID, actorUserID)
	if err != nil {
		return nil, nil, err
	}
	return detail, msg, nil
}

func (s *BOQService) Get(rfqID, actorUserID int64) (*domain.BOQDetail, *domain.Message, error) {
	rfq, err := s.rfqs.GetByIDAny(rfqID)
	if err != nil {
		return nil, nil, ErrBOQNotFound
	}
	if rfq.RFQType != "BQ" || rfq.InitiatedBy != "factory" {
		return nil, nil, ErrBOQNotFound
	}
	if actorUserID != rfq.UserID && (rfq.FactoryUserID == nil || *rfq.FactoryUserID != actorUserID) {
		return nil, nil, ErrBOQForbidden
	}
	items, err := s.rfqItems.ListByRFQID(rfqID)
	if err != nil {
		return nil, nil, err
	}
	detail, err := s.buildBOQDetail(rfq, items)
	if err != nil {
		return nil, nil, err
	}
	return detail, nil, nil
}

func (s *BOQService) Update(rfqID, actorUserID int64, input BOQInput) (*domain.BOQDetail, error) {
	rfq, err := s.rfqs.GetByIDAny(rfqID)
	if err != nil {
		return nil, ErrBOQNotFound
	}
	if rfq.FactoryUserID == nil || *rfq.FactoryUserID != actorUserID {
		return nil, ErrBOQForbidden
	}
	if rfq.BOQResponse != nil {
		return nil, ErrBOQAlreadyHandled
	}
	if boqIsExpired(rfq) {
		return nil, ErrBOQExpired
	}

	input = normalizeBOQInput(input)
	if err := validateBOQInput(input); err != nil {
		return nil, err
	}
	subtotal, vatAmount, grandTotal := computeBOQTotals(input.Items, input.DiscountAmount, input.VatPercent)
	now := time.Now()
	details := derefString(input.Note)
	if details == "" {
		details = input.Items[0].Description
	}
	quantity := int64(0)
	for _, item := range input.Items {
		quantity += int64(math.Round(item.Qty))
	}
	if quantity <= 0 {
		quantity = 1
	}
	rfq.Quantity = quantity
	rfq.Details = details
	rfq.BOQCurrency = &input.Currency
	rfq.BOQSubtotal = &subtotal
	rfq.BOQDiscountAmount = &input.DiscountAmount
	rfq.BOQVatPercent = &input.VatPercent
	rfq.BOQVatAmount = &vatAmount
	rfq.BOQGrandTotal = &grandTotal
	rfq.BOQMOQ = input.MOQ
	rfq.BOQLeadTimeDays = input.LeadTimeDays
	rfq.BOQPaymentTerms = input.PaymentTerms
	rfq.BOQValidityDays = input.ValidityDays
	rfq.BOQNote = input.Note
	rfq.BOQSentAt = &now

	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		UPDATE rfqs
		SET quantity = $2,
		    details = $3,
		    boq_currency = $4,
		    boq_subtotal = $5,
		    boq_discount_amount = $6,
		    boq_vat_percent = $7,
		    boq_vat_amount = $8,
		    boq_grand_total = $9,
		    boq_moq = $10,
		    boq_lead_time_days = $11,
		    boq_payment_terms = $12,
		    boq_validity_days = $13,
		    boq_note = $14,
		    boq_sent_at = $15,
		    updated_at = NOW()
		WHERE rfq_id = $1
	`, rfqID, quantity, details, input.Currency, subtotal, input.DiscountAmount, input.VatPercent, vatAmount, grandTotal,
		nullableBOQInt(input.MOQ), nullableBOQInt(input.LeadTimeDays), nullableBOQString(input.PaymentTerms), nullableBOQInt(input.ValidityDays), nullableBOQString(input.Note), now); err != nil {
		return nil, err
	}
	if err := s.rfqItems.DeleteByRFQIDTx(tx, rfqID); err != nil {
		return nil, err
	}
	for i := range input.Items {
		input.Items[i].RFQID = rfqID
	}
	if err := s.rfqItems.BulkInsertTx(tx, rfqID, input.Items); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`
		UPDATE conversations
		SET last_message = 'BOQ อัปเดต',
		    unread_customer = COALESCE(unread_customer, 0) + 1,
		    updated_at = NOW()
		WHERE conv_id = $1
	`, nullableBOQInt64(rfq.SourceConvID)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if rfq.SourceConvID != nil {
		quoteDataStr, _ := s.buildBOQQuoteDataString(rfq, input.Items, "")
		_ = s.messages.Create(&domain.Message{
			SenderID:    actorUserID,
			ReceiverID:  rfq.UserID,
			Content:     "โรงงานอัปเดต BOQ แล้ว กรุณาตรวจสอบ",
			ConvID:      rfq.SourceConvID,
			MessageType: "BQ",
			QuoteData:   quoteDataStr,
			BOQRfqID:    &rfqID,
			IsRead:      false,
		})
	}
	detail, _, err := s.Get(rfqID, actorUserID)
	return detail, err
}

func (s *BOQService) Accept(rfqID, buyerUserID int64) (*domain.Order, int64, error) {
	rfq, err := s.rfqs.GetByIDAny(rfqID)
	if err != nil {
		return nil, 0, ErrBOQNotFound
	}
	if rfq.UserID != buyerUserID {
		return nil, 0, ErrBOQForbidden
	}
	if rfq.BOQResponse != nil {
		return nil, 0, ErrBOQAlreadyHandled
	}
	if boqIsExpired(rfq) {
		return nil, 0, ErrBOQExpired
	}
	if rfq.FactoryUserID == nil {
		return nil, 0, ErrBOQInvalidState
	}
	items, err := s.rfqItems.ListByRFQID(rfqID)
	if err != nil {
		return nil, 0, err
	}
	if len(items) == 0 {
		return nil, 0, ErrBOQInvalidItems
	}

	shippingMethodID, err := s.lookupDefaultShippingMethodID()
	if err != nil {
		return nil, 0, err
	}
	discountAmount := derefFloat64(rfq.BOQDiscountAmount)
	commissionRate, configID, commissionAmount, factoryNet, err := s.resolveBOQCommission(*rfq.FactoryUserID, items, discountAmount, derefFloat64(rfq.BOQGrandTotal))
	if err != nil {
		return nil, 0, err
	}
	validityDays := 14
	if rfq.BOQValidityDays != nil && *rfq.BOQValidityDays > 0 {
		validityDays = *rfq.BOQValidityDays
	}
	pricePerPiece := derefFloat64(rfq.BOQGrandTotal) / float64(maxInt64(rfq.Quantity, 1))
	now := time.Now()
	quotation := &domain.Quotation{
		RFQID:                    rfq.RFQID,
		FactoryID:                *rfq.FactoryUserID,
		PricePerPiece:            pricePerPiece,
		MoldCost:                 0,
		LeadTimeDays:             int64(derefInt(rfq.BOQLeadTimeDays)),
		ShippingMethodID:         shippingMethodID,
		Status:                   "PD",
		CreateTime:               now,
		LogTimestamp:             now,
		Subtotal:                 derefFloat64(rfq.BOQSubtotal),
		DiscountAmount:           discountAmount,
		VatRate:                  derefFloat64(rfq.BOQVatPercent),
		VatAmount:                derefFloat64(rfq.BOQVatAmount),
		GrandTotal:               derefFloat64(rfq.BOQGrandTotal),
		PlatformCommissionRate:   commissionRate,
		PlatformCommissionAmount: commissionAmount,
		FactoryNetReceivable:     factoryNet,
		PlatformConfigID:         &configID,
		PaymentTerms:             rfq.BOQPaymentTerms,
		ValidityDays:             validityDays,
		ValidUntil:               boqExpiresAt(rfq),
		RevisionNo:               1,
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.quotations.CreateTx(tx, quotation); err != nil {
		return nil, 0, err
	}
	qItems := make([]domain.QuotationItem, 0, len(items))
	for _, item := range items {
		qItems = append(qItems, domain.QuotationItem{
			ItemNo:      item.ItemNo,
			Description: item.Description,
			Qty:         item.Qty,
			Unit:        item.Unit,
			UnitPrice:   item.UnitPrice,
			DiscountPct: item.DiscountPct,
			LineTotal:   item.LineTotal,
			Note:        item.Note,
		})
	}
	if err := s.quotationItems.BulkInsert(tx, quotation.QuotationID, qItems); err != nil {
		return nil, 0, err
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, err
	}

	order, err := s.orders.CreateFromQuotation(quotation.QuotationID, buyerUserID)
	if err != nil {
		return nil, quotation.QuotationID, err
	}
	if err := s.finalizeAcceptedBOQ(rfq, order); err != nil {
		return nil, quotation.QuotationID, err
	}
	return order, quotation.QuotationID, nil
}

func (s *BOQService) finalizeAcceptedBOQ(rfq *domain.RFQ, order *domain.Order) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		UPDATE rfqs
		SET boq_response = 'accepted',
		    boq_responded_at = NOW(),
		    updated_at = NOW()
		WHERE rfq_id = $1
	`, rfq.RFQID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE orders
		SET deposit_amount = total_amount,
		    payment_type = 'FP',
		    updated_at = NOW()
		WHERE order_id = $1
	`, order.OrderID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE payment_schedules
		SET amount = $2
		WHERE order_id = $1 AND installment_no = 1
	`, order.OrderID, order.TotalAmount); err != nil {
		return err
	}
	if rfq.SourceConvID != nil {
		if _, err := tx.Exec(`
			UPDATE conversations
			SET last_message = $2,
			    unread_factory = COALESCE(unread_factory, 0) + 1,
			    updated_at = NOW()
			WHERE conv_id = $1
		`, *rfq.SourceConvID, fmt.Sprintf("ยืนยัน BOQ แล้ว — คำสั่งซื้อ #%d ถูกสร้างแล้ว", order.OrderID)); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	if rfq.SourceConvID != nil {
		_ = s.messages.Create(&domain.Message{
			SenderID:    rfq.UserID,
			ReceiverID:  derefInt64(rfq.FactoryUserID),
			Content:     fmt.Sprintf("ยืนยัน BOQ แล้ว — คำสั่งซื้อ #%d ถูกสร้างแล้ว", order.OrderID),
			ConvID:      rfq.SourceConvID,
			MessageType: "TX",
			IsRead:      false,
		})
	}
	if s.notifications != nil && rfq.FactoryUserID != nil {
		refID := order.OrderID
		_ = s.notifications.Create(&domain.Notification{
			UserID:      *rfq.FactoryUserID,
			Type:        "BQ",
			Title:       "ลูกค้ายืนยัน BOQ แล้ว",
			Message:     "ลูกค้ายืนยัน BOQ แล้ว รอรับชำระเงิน",
			LinkTo:      fmt.Sprintf("/factory/orders/%d", order.OrderID),
			ReferenceID: &refID,
		})
	}
	return nil
}

func (s *BOQService) Decline(rfqID, buyerUserID int64, reason *string) (*domain.RFQ, error) {
	rfq, err := s.rfqs.GetByIDAny(rfqID)
	if err != nil {
		return nil, ErrBOQNotFound
	}
	if rfq.UserID != buyerUserID {
		return nil, ErrBOQForbidden
	}
	if rfq.BOQResponse != nil {
		return nil, ErrBOQAlreadyHandled
	}
	if boqIsExpired(rfq) {
		return nil, ErrBOQExpired
	}
	now := time.Now()
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		UPDATE rfqs
		SET status = 'CC',
		    boq_response = 'declined',
		    boq_responded_at = $2,
		    boq_decline_reason = $3,
		    updated_at = NOW()
		WHERE rfq_id = $1
	`, rfqID, now, nullableBOQString(reason)); err != nil {
		return nil, err
	}
	if rfq.SourceConvID != nil {
		if _, err := tx.Exec(`
			UPDATE conversations
			SET last_message = 'ลูกค้าปฏิเสธ BOQ',
			    unread_factory = COALESCE(unread_factory, 0) + 1,
			    updated_at = NOW()
			WHERE conv_id = $1
		`, *rfq.SourceConvID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	rfq.Status = "CC"
	resp := "declined"
	rfq.BOQResponse = &resp
	rfq.BOQRespondedAt = &now
	rfq.BOQDeclineReason = reason
	if rfq.SourceConvID != nil {
		_ = s.messages.Create(&domain.Message{
			SenderID:    buyerUserID,
			ReceiverID:  derefInt64(rfq.FactoryUserID),
			Content:     "ลูกค้าปฏิเสธ BOQ",
			ConvID:      rfq.SourceConvID,
			MessageType: "TX",
			IsRead:      false,
		})
	}
	if s.notifications != nil && rfq.FactoryUserID != nil {
		refID := rfqID
		_ = s.notifications.Create(&domain.Notification{
			UserID:      *rfq.FactoryUserID,
			Type:        "BQ",
			Title:       "ลูกค้าปฏิเสธ BOQ",
			Message:     "ลูกค้าปฏิเสธ BOQ",
			LinkTo:      "/factory/messages",
			ReferenceID: &refID,
		})
	}
	return rfq, nil
}

func (s *BOQService) ListMine(factoryUserID int64, status string) ([]domain.BOQSummary, error) {
	type row struct {
		RFQID            int64      `db:"rfq_id"`
		Status           string     `db:"status"`
		BOQResponse      *string    `db:"boq_response"`
		BOQDeclineReason *string    `db:"boq_decline_reason"`
		BOQSentAt        *time.Time `db:"boq_sent_at"`
		BOQRespondedAt   *time.Time `db:"boq_responded_at"`
		BOQValidityDays  int        `db:"boq_validity_days"`
		BOQGrandTotal    float64    `db:"boq_grand_total"`
		BOQCurrency      string     `db:"boq_currency"`
		SourceConvID     *int64     `db:"source_conv_id"`
		SourceShowcaseID *int64     `db:"source_showcase_id"`
		BuyerDisplayName string     `db:"buyer_display_name"`
		FactoryName      string     `db:"factory_name"`
	}
	var rows []row
	if err := s.db.Select(&rows, `
		SELECT r.rfq_id, r.status, r.boq_response, r.boq_decline_reason, r.boq_sent_at, r.boq_responded_at,
		       COALESCE(r.boq_validity_days, 14) AS boq_validity_days,
		       COALESCE(r.boq_grand_total, 0)::float8 AS boq_grand_total,
		       COALESCE(r.boq_currency, 'THB') AS boq_currency,
		       r.source_conv_id, r.source_showcase_id,
		       COALESCE(NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), ''), 'ลูกค้า #' || r.user_id::text) AS buyer_display_name,
		       COALESCE(fp.factory_name, 'Factory #' || r.factory_user_id::text) AS factory_name
		FROM rfqs r
		LEFT JOIN customers c ON c.user_id = r.user_id
		LEFT JOIN factory_profiles fp ON fp.user_id = r.factory_user_id
		WHERE r.rfq_type = 'BQ'
		  AND r.initiated_by = 'factory'
		  AND r.factory_user_id = $1
		ORDER BY r.boq_sent_at DESC NULLS LAST, r.created_at DESC
	`, factoryUserID); err != nil {
		return nil, err
	}
	out := make([]domain.BOQSummary, 0, len(rows))
	now := time.Now()
	for _, row := range rows {
		var expiresAt *time.Time
		expires := false
		if row.BOQSentAt != nil {
			t := row.BOQSentAt.Add(time.Duration(row.BOQValidityDays) * 24 * time.Hour)
			expiresAt = &t
			expires = now.After(t)
		}
		item := domain.BOQSummary{
			RFQID:            row.RFQID,
			Status:           row.Status,
			BOQResponse:      row.BOQResponse,
			BOQDeclineReason: row.BOQDeclineReason,
			BOQSentAt:        row.BOQSentAt,
			BOQRespondedAt:   row.BOQRespondedAt,
			BOQValidityDays:  row.BOQValidityDays,
			BOQExpiresAt:     expiresAt,
			IsExpired:        expires,
			BOQGrandTotal:    row.BOQGrandTotal,
			BOQCurrency:      row.BOQCurrency,
			SourceConvID:     row.SourceConvID,
			SourceShowcaseID: row.SourceShowcaseID,
			BuyerDisplayName: row.BuyerDisplayName,
			FactoryName:      row.FactoryName,
		}
		if statusFilterMatches(status, &item) {
			out = append(out, item)
		}
	}
	return out, nil
}

func statusFilterMatches(status string, item *domain.BOQSummary) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "all":
		return true
	case "pending":
		return item.BOQResponse == nil && !item.IsExpired
	case "accepted":
		return item.BOQResponse != nil && *item.BOQResponse == "accepted"
	case "declined":
		return item.BOQResponse != nil && *item.BOQResponse == "declined"
	case "expired":
		return item.BOQResponse == nil && item.IsExpired
	default:
		return true
	}
}

func (s *BOQService) buildBOQDetail(rfq *domain.RFQ, items []domain.RFQItem) (*domain.BOQDetail, error) {
	factoryName, imageURL, err := s.lookupFactorySummary(derefInt64(rfq.FactoryUserID))
	if err != nil {
		return nil, err
	}
	buyerName, err := s.lookupBuyerName(rfq.UserID)
	if err != nil {
		return nil, err
	}
	expiresAt := boqExpiresAt(rfq)
	return &domain.BOQDetail{
		RFQID:             rfq.RFQID,
		RFQType:           rfq.RFQType,
		InitiatedBy:       rfq.InitiatedBy,
		Status:            rfq.Status,
		BOQResponse:       rfq.BOQResponse,
		BOQDeclineReason:  rfq.BOQDeclineReason,
		BOQSentAt:         rfq.BOQSentAt,
		BOQRespondedAt:    rfq.BOQRespondedAt,
		BOQValidityDays:   maxInt(derefInt(rfq.BOQValidityDays), 14),
		BOQExpiresAt:      expiresAt,
		IsExpired:         boqIsExpired(rfq),
		BOQGrandTotal:     derefFloat64(rfq.BOQGrandTotal),
		BOQCurrency:       derefString(rfq.BOQCurrency),
		BOQSubtotal:       derefFloat64(rfq.BOQSubtotal),
		BOQDiscountAmount: derefFloat64(rfq.BOQDiscountAmount),
		BOQVatPercent:     derefFloat64(rfq.BOQVatPercent),
		BOQVatAmount:      derefFloat64(rfq.BOQVatAmount),
		BOQMOQ:            rfq.BOQMOQ,
		BOQLeadTimeDays:   rfq.BOQLeadTimeDays,
		BOQPaymentTerms:   rfq.BOQPaymentTerms,
		BOQNote:           rfq.BOQNote,
		SourceConvID:      rfq.SourceConvID,
		SourceShowcaseID:  rfq.SourceShowcaseID,
		Factory: domain.BOQFactorySummary{
			FactoryID:   derefInt64(rfq.FactoryUserID),
			FactoryName: factoryName,
			ImageURL:    imageURL,
		},
		Buyer: domain.BOQBuyerSummary{
			UserID:      rfq.UserID,
			DisplayName: buyerName,
		},
		Items: items,
	}, nil
}

func (s *BOQService) buildBOQQuoteDataString(rfq *domain.RFQ, items []domain.RFQItem, factoryName string) (*string, error) {
	expiresAt := boqExpiresAt(rfq)
	var validUntil *string
	if expiresAt != nil {
		v := formatThaiShortDate(*expiresAt)
		validUntil = &v
	}
	data := domain.BOQQuoteData{
		BOQRFQID:       rfq.RFQID,
		FactoryName:    factoryName,
		Items:          items,
		Currency:       derefString(rfq.BOQCurrency),
		Subtotal:       derefFloat64(rfq.BOQSubtotal),
		DiscountAmount: derefFloat64(rfq.BOQDiscountAmount),
		VatPercent:     derefFloat64(rfq.BOQVatPercent),
		VatAmount:      derefFloat64(rfq.BOQVatAmount),
		GrandTotal:     derefFloat64(rfq.BOQGrandTotal),
		MOQ:            rfq.BOQMOQ,
		LeadTimeDays:   rfq.BOQLeadTimeDays,
		PaymentTerms:   rfq.BOQPaymentTerms,
		ValidityDays:   maxInt(derefInt(rfq.BOQValidityDays), 14),
		ExpiresAt:      expiresAt,
		Note:           rfq.BOQNote,
		Status:         boqCardStatus(rfq),
		Price:          derefFloat64(rfq.BOQGrandTotal),
		LeadTime:       rfq.BOQLeadTimeDays,
		ValidUntil:     validUntil,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	sv := string(b)
	return &sv, nil
}

func boqCardStatus(rfq *domain.RFQ) string {
	if rfq.BOQResponse != nil {
		return *rfq.BOQResponse
	}
	if boqIsExpired(rfq) {
		return "expired"
	}
	return "pending"
}

func (s *BOQService) lookupFactoryPrimaryCategory(factoryID int64) (int64, error) {
	var categoryID int64
	err := s.db.Get(&categoryID, `
		SELECT category_id
		FROM map_factory_categories
		WHERE factory_id = $1
		ORDER BY category_id
		LIMIT 1
	`, factoryID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return categoryID, err
}

func (s *BOQService) lookupDefaultShippingMethodID() (int64, error) {
	var shippingMethodID int64
	err := s.db.Get(&shippingMethodID, `
		SELECT shipping_method_id
		FROM lbi_shipping_methods
		WHERE status = '1'
		ORDER BY shipping_method_id
		LIMIT 1
	`)
	return shippingMethodID, err
}

func (s *BOQService) resolveBOQCommission(factoryID int64, items []domain.RFQItem, discountAmount float64, grandTotal float64) (float64, int64, float64, float64, error) {
	qItems := make([]domain.QuotationItem, 0, len(items))
	for _, item := range items {
		qItems = append(qItems, domain.QuotationItem{
			ItemNo:      item.ItemNo,
			Description: item.Description,
			Qty:         item.Qty,
			Unit:        item.Unit,
			UnitPrice:   item.UnitPrice,
			DiscountPct: item.DiscountPct,
			LineTotal:   item.LineTotal,
			Note:        item.Note,
		})
	}
	breakdown, err := s.commissions.Calculate(CommissionInput{
		Items:          qItems,
		DiscountAmount: discountAmount,
		FactoryID:      &factoryID,
	})
	if err != nil {
		return 0, 0, 0, 0, err
	}
	commissionAmount := roundMoney((breakdown.Subtotal * breakdown.PlatformCommissionRate) / 100)
	factoryNet := roundMoney(grandTotal - commissionAmount)
	return breakdown.PlatformCommissionRate, breakdown.PlatformConfigID, commissionAmount, factoryNet, nil
}

func (s *BOQService) lookupFactorySummary(factoryID int64) (string, *string, error) {
	var row struct {
		Name     string         `db:"factory_name"`
		ImageURL sql.NullString `db:"image_url"`
	}
	err := s.db.Get(&row, `SELECT factory_name, image_url FROM factory_profiles WHERE user_id = $1`, factoryID)
	if err != nil {
		return "", nil, err
	}
	if row.ImageURL.Valid {
		return row.Name, &row.ImageURL.String, nil
	}
	return row.Name, nil, nil
}

func (s *BOQService) lookupBuyerName(userID int64) (string, error) {
	var name string
	err := s.db.Get(&name, `
		SELECT COALESCE(NULLIF(TRIM(CONCAT(first_name, ' ', last_name)), ''), 'ลูกค้า #' || user_id::text)
		FROM customers
		WHERE user_id = $1
	`, userID)
	return name, err
}

func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func derefFloat64(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func formatThaiShortDate(t time.Time) string {
	months := []string{"ม.ค.", "ก.พ.", "มี.ค.", "เม.ย.", "พ.ค.", "มิ.ย.", "ก.ค.", "ส.ค.", "ก.ย.", "ต.ค.", "พ.ย.", "ธ.ค."}
	return fmt.Sprintf("%d %s %02d", t.Day(), months[int(t.Month())-1], (t.Year()+543)%100)
}

func nullableBOQInt(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func nullableBOQInt64(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func nullableBOQString(v *string) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

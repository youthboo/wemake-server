package order

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/dto"
	"github.com/yourusername/wemake/internal/helper"
	orderrepo "github.com/yourusername/wemake/internal/repository/order"
)

type BulkCheckoutItemInput = dto.BulkCheckoutItemInput

type BulkCheckoutInput struct {
	RFQID          int64                   `json:"rfq_id"`
	UserID         int64                   `json:"-"`
	Items          []BulkCheckoutItemInput `json:"items"`
	IdempotencyKey string                  `json:"idempotency_key"`
}

type BulkCheckoutSummary struct {
	OrderCount   int                `json:"order_count"`
	TotalAmount  domain.MoneyString `json:"total_amount"`
	TotalDeposit domain.MoneyString `json:"total_deposit"`
}

type BulkCheckoutResult struct {
	RFQID     int64               `json:"rfq_id"`
	RFQStatus string              `json:"rfq_status"`
	Orders    []domain.Order      `json:"orders"`
	Summary   BulkCheckoutSummary `json:"summary"`
}

// CreateFromQuotation accepts a pending (PD) quotation or continues from an already-accepted (AC) quote.
// For PD: rejects sibling PD quotations, sets this quote to AC, closes the RFQ (OP→CL), then creates an order in payment-pending state.
func (s *OrderService) CreateFromQuotation(quotationID, userID int64) (*domain.Order, error) {
	var src *orderrepo.QuotationOrderSource
	var order *domain.Order
	var total decimal.Decimal
	var deposit decimal.Decimal
	var now time.Time
	if err := helper.WithTx(context.Background(), s.db, func(tx *sqlx.Tx) error {
		var err error
		src, err = s.repo.GetOrderSourceByQuotationIDTx(tx, quotationID, userID)
		if err != nil {
			return err
		}
		switch src.Status {
		case domain.QuotationStatusDeclined:
			return ErrQuotationRejected
		case domain.QuotationStatusPrepared:
			if err := s.quotations.UpdateStatusTx(tx, quotationID, domain.QuotationStatusAccepted); err != nil {
				return err
			}
		case domain.QuotationStatusAccepted:
		default:
			return ErrQuotationInvalidState
		}

		exists, err := s.repo.OrderExistsForQuoteTx(tx, quotationID)
		if err != nil {
			return err
		}
		if exists {
			return ErrOrderAlreadyExistsForQuote
		}

		total = helper.MoneyDecimal(src.GrandTotal)
		if helper.IsMoneyLess(total, helper.ZeroMoney()) || helper.IsMoneyZero(total) {
			pricePerPiece := helper.MoneyDecimal(src.PricePerPiece)
			qty := helper.MoneyFromInt(src.Quantity)
			moldCost := helper.MoneyDecimal(src.MoldCost)
			lineTotal := helper.MultiplyMoney(pricePerPiece, qty)
			total = helper.AddMoney(lineTotal, moldCost)
		}
		if helper.IsMoneyLess(total, helper.ZeroMoney()) {
			return ErrInvalidOrderTotal
		}
		deposit = total
		status := domain.OrderStatusWaitSlip // ลูกค้าต้องแนบสลีปก่อน
		if helper.IsMoneyZero(total) {
			deposit = helper.ZeroMoney()
			status = domain.OrderStatusPaymentDone // ยอด 0 → ข้ามชำระเงิน
		}

		shippingDays := getShippingDays(s.db)
		now = time.Now()
		deliveryDate := calculateEstimatedDelivery(now, src.LeadTimeDays, shippingDays, src.DeliveryDate)
		order = &domain.Order{
			QuotationID:       src.QuotationID,
			UserID:            src.UserID,
			FactoryID:         src.FactoryID,
			TotalAmount:       total,
			DepositAmount:     deposit,
			Status:            status,
			EstimatedDelivery: &deliveryDate,
			CreatedAt:         now,
			UpdatedAt:         now,
		}

		if err := s.repo.CreateTx(tx, order); err != nil {
			return err
		}
		if s.schedules != nil {
			depositDueDate := deriveDefaultDepositScheduleDate(order.CreatedAt)
			if err := s.schedules.CreateTx(tx, &domain.PaymentSchedule{
				OrderID:       order.OrderID,
				InstallmentNo: 1,
				DueDate:       depositDueDate,
				Amount:        deposit,
			}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Activity log is best-effort; order creation should not fail if audit table/schema lags behind.
	uid := userID
	_ = s.repo.InsertActivity(order.OrderID, &uid, "ORDER_CREATED", map[string]interface{}{
		"status":         order.Status,
		"quote_id":       order.QuotationID,
		"amount":         total,
		"deposit_amount": deposit,
	})
	helper.CreateNotificationSafe(s.notifications, &domain.Notification{
		UserID:  order.FactoryID,
		Type:    "ORDER_PLACED",
		Title:   "คำสั่งซื้อใหม่",
		Message: fmt.Sprintf("ลูกค้าสั่งซื้อ Order #%d", order.OrderID),
		LinkTo:  helper.FactoryOrderLink(order.OrderID),
		Data: helper.NotificationData(map[string]interface{}{
			"order_id": order.OrderID,
			"quote_id": order.QuotationID,
			"url":      helper.FactoryOrderLink(order.OrderID),
		}),
		ReferenceID: &order.OrderID,
		CreatedAt:   now,
	})
	s.notifyAcceptedQuotationInChat(src.RFQID, src.UserID, src.FactoryID, order.OrderID)
	return order, nil
}

func (s *OrderService) BulkCheckout(input BulkCheckoutInput) (*BulkCheckoutResult, error) {
	if input.RFQID <= 0 || input.UserID <= 0 || len(input.Items) == 0 {
		return nil, ErrInvalidQuotationSet
	}
	quoteIDs := make([]int64, 0, len(input.Items))
	seen := make(map[int64]struct{}, len(input.Items))
	paymentTypes := make(map[int64]string, len(input.Items))
	addressIDs := make(map[int64]struct{})
	for _, item := range input.Items {
		if item.QuotationID <= 0 {
			return nil, ErrInvalidQuotationSet
		}
		if item.AddressID <= 0 {
			return nil, ErrInvalidQuotationSet
		}
		if _, ok := seen[item.QuotationID]; ok {
			return nil, ErrInvalidQuotationSet
		}
		seen[item.QuotationID] = struct{}{}
		pt := domainutil.NormalizeStatus(item.PaymentType)
		switch pt {
		case "", "FULL", domain.PaymentTypeFull:
			pt = domain.PaymentTypeFull
		case "DEPOSIT", domain.PaymentTypeDeposit:
			pt = domain.PaymentTypeDeposit
		default:
			return nil, ErrPaymentTypeInvalid
		}
		paymentTypes[item.QuotationID] = pt
		quoteIDs = append(quoteIDs, item.QuotationID)
		addressIDs[item.AddressID] = struct{}{}
	}

	now := time.Now()
	orders := make([]domain.Order, 0, len(quoteIDs))
	if err := helper.WithTx(context.Background(), s.db, func(tx *sqlx.Tx) error {
		type lockedRFQ struct {
			RFQID  int64  `db:"rfq_id"`
			UserID int64  `db:"user_id"`
			Status string `db:"status"`
		}
		var rfq lockedRFQ
		if err := tx.Get(&rfq, `
		SELECT rfq_id, user_id, status
		FROM rfqs
		WHERE rfq_id = $1
		FOR UPDATE
	`, input.RFQID); err != nil {
			return err
		}
		if rfq.UserID != input.UserID {
			return ErrNotQuotationParty
		}
		if rfq.Status != domain.RFQStatusOpen && rfq.Status != domain.RFQStatusInReview {
			return ErrRFQLocked
		}
		if len(addressIDs) > 0 {
			ids := make([]int64, 0, len(addressIDs))
			for id := range addressIDs {
				ids = append(ids, id)
			}
			var ownedCount int
			if err := tx.Get(&ownedCount, `
			SELECT COUNT(*)
			FROM addresses
			WHERE user_id = $1
			  AND address_id = ANY($2)
		`, input.UserID, pq.Array(ids)); err != nil {
				return err
			}
			if ownedCount != len(ids) {
				return ErrInvalidQuotationSet
			}
		}

		type lockedQuotation struct {
			QuoteID       int64      `db:"quote_id"`
			RFQID         int64      `db:"rfq_id"`
			FactoryID     int64      `db:"factory_id"`
			PricePerPiece float64    `db:"price_per_piece"`
			Quantity      int64      `db:"quantity"`
			MoldCost      float64    `db:"mold_cost"`
			LeadTimeDays  int64      `db:"lead_time_days"`
			DeliveryDate  *time.Time `db:"delivery_date"`
			Status        string     `db:"status"`
			GrandTotal    float64    `db:"grand_total"`
			ValidUntil    *time.Time `db:"valid_until"`
		}
		var quotes []lockedQuotation
		if err := tx.Select(&quotes, `
		SELECT q.quote_id, q.rfq_id, q.factory_id, q.price_per_piece, r.quantity,
		       q.mold_cost, q.lead_time_days, NULL::date AS delivery_date, q.status, COALESCE(q.grand_total, 0) AS grand_total,
		       COALESCE(q.valid_until, (q.create_time + (q.validity_days::text || ' day')::interval)::date) AS valid_until
		FROM quotations q
		INNER JOIN rfqs r ON r.rfq_id = q.rfq_id
		WHERE q.rfq_id = $1
		  AND q.quote_id = ANY($2)
		FOR UPDATE OF q
	`, input.RFQID, pq.Array(quoteIDs)); err != nil {
			return err
		}
		if len(quotes) != len(quoteIDs) {
			return ErrInvalidQuotationSet
		}

		shippingDays := getShippingDays(s.db)
		for _, q := range quotes {
			if q.FactoryID == input.UserID {
				return ErrSelfTransaction
			}
			// Allow Prepared (never ordered) OR Accepted (its previous order was
			// cancelled). An Accepted quote that still has an ACTIVE order is
			// blocked below by OrderExistsForQuoteTx — so re-ordering only works
			// once the earlier order is cancelled. Other states stay rejected.
			if q.Status != domain.QuotationStatusPrepared && q.Status != domain.QuotationStatusAccepted {
				return ErrQuotationInvalidState
			}
			if q.ValidUntil != nil && q.ValidUntil.Before(now.UTC()) {
				return ErrQuotationExpired
			}
			exists, err := s.repo.OrderExistsForQuoteTx(tx, q.QuoteID)
			if err != nil {
				return err
			}
			if exists {
				return ErrOrderAlreadyExistsForQuote
			}
			total := helper.MoneyDecimal(q.GrandTotal)
			if helper.IsMoneyLess(total, helper.ZeroMoney()) || helper.IsMoneyZero(total) {
				pricePerPieceDecimal := helper.MoneyDecimal(q.PricePerPiece)
				qtyDecimal := helper.MoneyFromInt(q.Quantity)
				moldCostDecimal := helper.MoneyDecimal(q.MoldCost)
				lineTotal := helper.MultiplyMoney(pricePerPieceDecimal, qtyDecimal)
				total = helper.AddMoney(lineTotal, moldCostDecimal)
			}
			if helper.IsMoneyLess(total, helper.ZeroMoney()) {
				return ErrInvalidOrderTotal
			}
			deposit := total
			status := domain.OrderStatusPaymentPending
			if helper.IsMoneyZero(total) {
				status = domain.OrderStatusPaymentExpired
				deposit = helper.ZeroMoney()
			}
			deliveryDate := calculateEstimatedDelivery(now, q.LeadTimeDays, shippingDays, q.DeliveryDate)
			order := &domain.Order{
				QuotationID:       q.QuoteID,
				UserID:            input.UserID,
				FactoryID:         q.FactoryID,
				TotalAmount:       total,
				DepositAmount:     deposit,
				Status:            status,
				EstimatedDelivery: &deliveryDate,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			if err := s.repo.CreateTx(tx, order); err != nil {
				return err
			}
			if _, err := tx.Exec(`UPDATE orders SET payment_type = $1 WHERE order_id = $2`, paymentTypes[q.QuoteID], order.OrderID); err != nil {
				return err
			}
			if s.schedules != nil && helper.IsMoneyGreater(deposit, helper.ZeroMoney()) {
				if err := s.schedules.CreateTx(tx, &domain.PaymentSchedule{
					OrderID:       order.OrderID,
					InstallmentNo: 1,
					DueDate:       deriveDefaultDepositScheduleDate(order.CreatedAt),
					Amount:        deposit,
				}); err != nil {
					return err
				}
			}
			orders = append(orders, *order)
		}

		if _, err := tx.Exec(`
		UPDATE quotations
		SET status = 'AC', is_locked = TRUE, log_timestamp = NOW()
		WHERE rfq_id = $1 AND quote_id = ANY($2)
	`, input.RFQID, pq.Array(quoteIDs)); err != nil {
			return err
		}
		return s.rfqs.MarkInReviewTx(tx, input.RFQID, input.UserID)
	}); err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, ErrNoOrdersCreated
	}

	totalAmount := decimal.Zero
	totalDeposit := decimal.Zero
	uid := input.UserID
	for _, order := range orders {
		totalAmount = helper.AddMoney(totalAmount, order.TotalAmount)
		totalDeposit = helper.AddMoney(totalDeposit, order.DepositAmount)
		_ = s.repo.InsertActivity(order.OrderID, &uid, "ORDER_CREATED", map[string]interface{}{
			"status":         order.Status,
			"quote_id":       order.QuotationID,
			"amount":         order.TotalAmount,
			"deposit_amount": order.DepositAmount,
			"bulk_checkout":  true,
		})
		helper.CreateNotificationSafe(s.notifications, &domain.Notification{
			UserID:  order.FactoryID,
			Type:    "ORDER_PLACED",
			Title:   "คำสั่งซื้อใหม่",
			Message: fmt.Sprintf("ลูกค้าสั่งซื้อ Order #%d", order.OrderID),
			LinkTo:  helper.FactoryOrderLink(order.OrderID),
			Data: helper.NotificationData(map[string]interface{}{
				"order_id": order.OrderID,
				"quote_id": order.QuotationID,
				"url":      helper.FactoryOrderLink(order.OrderID),
			}),
			ReferenceID: &order.OrderID,
			CreatedAt:   now,
		})
	}
	result := &BulkCheckoutResult{
		RFQID:     input.RFQID,
		RFQStatus: domain.RFQStatusInReview,
		Orders:    orders,
		Summary: BulkCheckoutSummary{
			OrderCount:   len(orders),
			TotalAmount:  domain.NewMoneyString(totalAmount),
			TotalDeposit: domain.NewMoneyString(totalDeposit),
		},
	}
	return result, nil
}

func (s *OrderService) notifyAcceptedQuotationInChat(rfqID, customerID, factoryID, orderID int64) {
	if s.messages == nil {
		return
	}
	rfq, err := s.rfqs.GetByIDAny(rfqID)
	if err != nil || rfq == nil || rfq.ConversationID == nil {
		return
	}
	content := fmt.Sprintf("ลูกค้ายืนยันใบเสนอราคาแล้ว · Order #%d ถูกสร้าง", orderID)
	_ = s.messages.AutoSendSystemMessage(context.Background(), *rfq.ConversationID, customerID, factoryID, content)
}

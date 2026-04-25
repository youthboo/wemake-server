package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

var (
	ErrQuotationRejected     = errors.New("quotation was rejected")
	ErrQuotationInvalidState = errors.New("quotation must be pending or already accepted")
)
var ErrShipOrderInvalid = errors.New("tracking_no and courier are required")
var ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled in its current status")
var ErrInsufficientGoodFund = errors.New("insufficient good_fund balance")
var ErrOrderAlreadyExistsForQuote = errors.New("order already exists for this quotation")
var ErrPaymentTypeInvalid = errors.New("payment type must be DP or FP")
var ErrPaymentAmountMismatch = errors.New("payment amount does not match order amount for payment type")
var ErrPaymentAlreadyExists = errors.New("payment already exists for this order and payment type")
var ErrPaymentStateInvalid = errors.New("payment is not in a verifiable state")
var ErrDepositAlreadyPaid = errors.New("DEPOSIT_ALREADY_PAID")
var ErrDepositExpired = errors.New("DEPOSIT_EXPIRED")
var ErrConfirmReceiptInvalidStatus = errors.New("order status must be SH")
var ErrConfirmReceiptNotAllowed = errors.New("order already completed or cancelled")
var ErrReviewRatingInvalid = errors.New("rating must be between 1 and 5")
var ErrReviewCommentInvalid = errors.New("comment must be 1-1000 characters")
var ErrReviewOrderNotCompleted = errors.New("order must be completed before review")
var ErrReviewAlreadyExists = errors.New("review already exists for this order")

type ConfirmReceiptInput struct {
	Note       string
	ReceivedAt *time.Time
}

type CreateOrderReviewInput struct {
	Rating  int
	Comment string
}

type ConfirmReceiptSettlement struct {
	FactoryUserID int64   `json:"factory_user_id"`
	WalletID      int64   `json:"wallet_id"`
	MovedAmount   float64 `json:"moved_amount"`
	PendingBefore float64 `json:"pending_before"`
	PendingAfter  float64 `json:"pending_after"`
	GoodBefore    float64 `json:"good_before"`
	GoodAfter     float64 `json:"good_after"`
}

type ConfirmReceiptResult struct {
	Success         bool                     `json:"success"`
	OrderID         int64                    `json:"order_id"`
	StatusBefore    string                   `json:"status_before"`
	StatusAfter     string                   `json:"status_after"`
	CompletedStepID int64                    `json:"completed_step_id"`
	Settlement      ConfirmReceiptSettlement `json:"settlement"`
	CompletedAt     time.Time                `json:"completed_at"`
	AlreadyComplete bool                     `json:"already_completed,omitempty"`
}

type OrderService struct {
	db         *sqlx.DB
	repo       *repository.OrderRepository
	schedules  *repository.PaymentScheduleRepository
	wallets    *repository.WalletRepository
	txLedger   *repository.TransactionRepository
	quotations *repository.QuotationRepository
	rfqs       *repository.RFQRepository
	reviews    *repository.ReviewRepository
}

func NewOrderService(db *sqlx.DB, repo *repository.OrderRepository, schedules *repository.PaymentScheduleRepository, wallets *repository.WalletRepository, txLedger *repository.TransactionRepository, quotations *repository.QuotationRepository, rfqs *repository.RFQRepository, reviews *repository.ReviewRepository) *OrderService {
	return &OrderService{db: db, repo: repo, schedules: schedules, wallets: wallets, txLedger: txLedger, quotations: quotations, rfqs: rfqs, reviews: reviews}
}

var thailandLocation = time.FixedZone("Asia/Bangkok", 7*60*60)

// CreateFromQuotation accepts a pending (PD) quotation or continues from an already-accepted (AC) quote.
// For PD: rejects sibling PD quotations, sets this quote to AC, closes the RFQ (OP→CL), then creates an order in payment-pending state.
func (s *OrderService) CreateFromQuotation(quotationID, userID int64) (*domain.Order, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	src, err := s.repo.GetOrderSourceByQuotationIDTx(tx, quotationID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			tx.Rollback()
		}
		return nil, err
	}
	switch src.Status {
	case "RJ":
		return nil, ErrQuotationRejected
	case "PD":
		if err := s.quotations.RejectOtherPendingQuotationsTx(tx, src.RFQID, quotationID); err != nil {
			return nil, err
		}
		if err := s.quotations.UpdateStatusTx(tx, quotationID, "AC"); err != nil {
			return nil, err
		}
		if err := s.rfqs.CloseOpenRFQForUserTx(tx, src.RFQID, userID); err != nil {
			return nil, err
		}
	case "AC":
		if err := s.quotations.RejectOtherPendingQuotationsTx(tx, src.RFQID, quotationID); err != nil {
			return nil, err
		}
		if err := s.rfqs.CloseOpenRFQForUserTx(tx, src.RFQID, userID); err != nil {
			return nil, err
		}
	default:
		return nil, ErrQuotationInvalidState
	}

	exists, err := s.repo.OrderExistsForQuoteTx(tx, quotationID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrOrderAlreadyExistsForQuote
	}

	total := (src.PricePerPiece * float64(src.Quantity)) + src.MoldCost
	if total <= 0 {
		return nil, errors.New("invalid order total")
	}
	deposit := roundCurrency(total * 0.3)

	now := time.Now()
	est := now.AddDate(0, 0, int(src.LeadTimeDays))
	deliveryDate := time.Date(est.Year(), est.Month(), est.Day(), 0, 0, 0, 0, est.Location())
	order := &domain.Order{
		QuotationID:       src.QuotationID,
		UserID:            src.UserID,
		FactoryID:         src.FactoryID,
		TotalAmount:       total,
		DepositAmount:     deposit,
		Status:            "PP",
		EstimatedDelivery: &deliveryDate,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := s.repo.CreateTx(tx, order); err != nil {
		return nil, err
	}
	if s.schedules != nil {
		depositDueDate := deriveDefaultDepositScheduleDate(order.CreatedAt)
		if err := s.schedules.CreateTx(tx, &domain.PaymentSchedule{
			OrderID:       order.OrderID,
			InstallmentNo: 1,
			DueDate:       depositDueDate,
			Amount:        deposit,
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
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
	return order, nil
}

func (s *OrderService) CreatePayment(orderID, userID int64, role, paymentType string, amount float64) (*domain.Transaction, error) {
	if role != domain.RoleCustomer {
		return nil, sql.ErrNoRows
	}
	order, err := s.repo.GetByParticipant(orderID, userID, role)
	if err != nil {
		return nil, err
	}

	paymentType = strings.TrimSpace(strings.ToUpper(paymentType))
	if paymentType == "DP" {
		if err := s.ensureDepositPayable(order); err != nil {
			return nil, err
		}
	}
	expectedAmount, err := expectedPaymentAmount(order, paymentType)
	if err != nil {
		return nil, err
	}
	if amount <= 0 || amount != expectedAmount {
		return nil, ErrPaymentAmountMismatch
	}

	existing, err := s.txLedger.List(repository.TransactionFilters{
		OrderID: &orderID,
		Type:    &paymentType,
	})
	if err != nil {
		return nil, err
	}
	for _, row := range existing {
		if paymentType == "DP" && row.Status == "PT" {
			return nil, ErrDepositAlreadyPaid
		}
		if row.Status != "RJ" {
			return nil, ErrPaymentAlreadyExists
		}
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	walletID, err := s.wallets.EnsureWallet(tx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	orderIDPtr := order.OrderID
	item := &domain.Transaction{
		TxID:       "tx-" + uuid.NewString(),
		WalletID:   walletID,
		OrderID:    &orderIDPtr,
		Type:       paymentType,
		Amount:     amount,
		Status:     "ST",
		CreatedAt:  now,
		UpdatedAt:  now,
		UploadedAt: now,
	}
	if err := s.txLedger.CreateTx(tx, item); err != nil {
		return nil, err
	}
	if err := s.repo.InsertActivityTx(tx, orderID, &userID, "PAYMENT_CREATED", map[string]interface{}{
		"tx_id": item.TxID, "type": paymentType, "amount": amount, "status": item.Status,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *OrderService) VerifyPayment(orderID, userID int64, role, txID string) (*domain.Transaction, error) {
	order, err := s.repo.GetByParticipant(orderID, userID, role)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	paymentTx, err := s.txLedger.GetByIDForUpdate(tx, strings.TrimSpace(txID))
	if err != nil {
		return nil, err
	}
	if paymentTx.OrderID == nil || *paymentTx.OrderID != orderID {
		return nil, sql.ErrNoRows
	}
	if paymentTx.Type != "DP" && paymentTx.Type != "FP" {
		return nil, ErrPaymentTypeInvalid
	}
	if paymentTx.Type == "DP" {
		if err := s.ensureDepositPayable(order); err != nil {
			return nil, err
		}
	}
	if paymentTx.Status != "ST" {
		if paymentTx.Type == "DP" && paymentTx.Status == "PT" {
			return nil, ErrDepositAlreadyPaid
		}
		return nil, ErrPaymentStateInvalid
	}

	expectedAmount, err := expectedPaymentAmount(order, paymentTx.Type)
	if err != nil {
		return nil, err
	}
	if paymentTx.Amount != expectedAmount {
		return nil, ErrPaymentAmountMismatch
	}

	if _, err := s.wallets.EnsureWallet(tx, order.UserID); err != nil {
		return nil, err
	}
	if _, err := s.wallets.EnsureWallet(tx, order.FactoryID); err != nil {
		return nil, err
	}

	customerWallet, err := s.wallets.GetByUserIDForUpdate(tx, order.UserID)
	if err != nil {
		return nil, err
	}
	ok, err := s.wallets.DebitGoodFund(tx, customerWallet.WalletID, paymentTx.Amount)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInsufficientGoodFund
	}

	factoryWallet, err := s.wallets.GetByUserIDForUpdate(tx, order.FactoryID)
	if err != nil {
		return nil, err
	}
	if err := s.wallets.CreditGoodFund(tx, factoryWallet.WalletID, paymentTx.Amount); err != nil {
		return nil, err
	}

	if err := s.txLedger.PatchStatusTx(tx, paymentTx.TxID, "PT"); err != nil {
		return nil, err
	}

	now := time.Now()
	orderIDPtr := order.OrderID
	receiveTx := &domain.Transaction{
		TxID:       "tx-" + uuid.NewString(),
		WalletID:   factoryWallet.WalletID,
		OrderID:    &orderIDPtr,
		Type:       "SC",
		Amount:     paymentTx.Amount,
		Status:     "PT",
		CreatedAt:  now,
		UpdatedAt:  now,
		UploadedAt: now,
	}
	if err := s.txLedger.CreateTx(tx, receiveTx); err != nil {
		return nil, err
	}

	if paymentTx.Type == "DP" && (normalizeOrderStatus(order.Status) == "PP" || normalizeOrderStatus(order.Status) == "PE") {
		if err := s.repo.UpdateStatusTx(tx, orderID, "PD"); err != nil {
			return nil, err
		}
		order.Status = "PD"
		if s.schedules != nil {
			if err := s.schedules.PatchStatusByOrderAndInstallmentTx(tx, orderID, 1, "PD"); err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
		}
		if err := insertDomainEventTx(tx, "order.deposit_paid", map[string]interface{}{
			"order_id": orderID,
			"tx_id":    paymentTx.TxID,
			"amount":   paymentTx.Amount,
		}); err != nil {
			return nil, err
		}
		if err := insertDomainEventTx(tx, "cache.invalidate", map[string]interface{}{
			"paths": []string{
				fmt.Sprintf("/orders/%d", orderID),
				fmt.Sprintf("/orders/%d/production-updates", orderID),
			},
		}); err != nil {
			return nil, err
		}
	}

	if err := s.repo.InsertActivityTx(tx, orderID, &userID, "PAYMENT_VERIFIED", map[string]interface{}{
		"tx_id": paymentTx.TxID, "type": paymentTx.Type, "amount": paymentTx.Amount, "status": "PT", "order_status": order.Status,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	paymentTx.Status = "PT"
	return paymentTx, nil
}

func (s *OrderService) List(userID int64, role string, status string) ([]domain.OrderListItem, error) {
	st := strings.TrimSpace(strings.ToUpper(status))
	if role == domain.RoleFactory {
		return s.repo.ListEnrichedByFactoryID(userID, st)
	}
	return s.repo.ListEnrichedByUserID(userID, st)
}

func (s *OrderService) GetByID(orderID, userID int64, role string) (*domain.Order, error) {
	detail, err := s.GetDetailByID(orderID, userID, role)
	if err != nil {
		return nil, err
	}
	return &domain.Order{
		OrderID:           detail.OrderID,
		QuotationID:       detail.QuotationID,
		UserID:            detail.CustomerUserID,
		FactoryID:         detail.FactoryID,
		TotalAmount:       detail.TotalAmount,
		DepositAmount:     detail.DepositAmount,
		Status:            detail.Status,
		EstimatedDelivery: detail.EstimatedDelivery,
		TrackingNo:        detail.TrackingNo,
		Courier:           detail.Courier,
		ShippedAt:         detail.ShippedAt,
		CreatedAt:         detail.CreatedAt,
		UpdatedAt:         detail.UpdatedAt,
	}, nil
}

func (s *OrderService) GetDetailByID(orderID, userID int64, role string) (*domain.OrderDetailResponse, error) {
	row, err := s.repo.GetDetailByParticipant(orderID, userID, role)
	if err != nil {
		return nil, err
	}
	if row.UserID != userID && row.FactoryID != userID {
		return nil, domain.ErrForbidden
	}
	images, err := s.repo.GetRfqImages(row.RFQID)
	if err != nil {
		return nil, err
	}
	depositDueDate := deriveDepositDueDate(row)
	nowTH := time.Now().In(thailandLocation)
	depositPaidAt := s.depositPaidAt(row.OrderID)
	finalPaidAt := s.finalPaymentPaidAt(row.OrderID)
	statusCode := normalizeOrderStatus(row.Status)
	rfqDetails := ""
	if row.RFQDetails != nil {
		rfqDetails = *row.RFQDetails
	}
	rfqCategoryName := ""
	if row.RFQCategoryName != nil {
		rfqCategoryName = *row.RFQCategoryName
	}
	rfqUnitName := ""
	if row.RFQUnitName != nil {
		rfqUnitName = *row.RFQUnitName
	}

	return &domain.OrderDetailResponse{
		OrderID:           row.OrderID,
		QuotationID:       row.QuotationID,
		UserID:            row.UserID,
		FactoryID:         row.FactoryID,
		TotalAmount:       row.TotalAmount,
		DepositAmount:     row.DepositAmount,
		Status:            statusCode,
		StatusLabelTH:     orderStatusLabelTH(statusCode),
		PaymentType:       row.PaymentType,
		Currency:          "THB",
		Factory:           domain.OrderFactorySummary{FactoryID: row.FactoryID, Name: row.FactoryName},
		CustomerUserID:    row.UserID,
		EstimatedDelivery: timePtrInTH(row.EstimatedDelivery),
		TrackingNo:        row.TrackingNo,
		Courier:           row.Courier,
		ShippedAt:         timePtrInTH(row.ShippedAt),
		CreatedAt:         row.CreatedAt.In(thailandLocation),
		UpdatedAt:         row.UpdatedAt.In(thailandLocation),
		NextAction:        buildNextAction(row, statusCode, depositDueDate, depositPaidAt, finalPaidAt, nowTH),
		PaymentSchedule:   buildPaymentSchedule(row, statusCode, depositDueDate, depositPaidAt, finalPaidAt),
		RFQ: domain.RfqNested{
			RfqID:          row.RFQID,
			Title:          row.RFQTitle,
			Details:        rfqDetails,
			Quantity:       row.RFQQuantity,
			UnitName:       rfqUnitName,
			BudgetPerPiece: row.RFQBudget,
			CategoryID:     row.RFQCategoryID,
			CategoryName:   rfqCategoryName,
			DeadlineDate:   timePtrInTH(row.RFQDeadline),
			CreatedAt:      row.RFQCreatedAt.In(thailandLocation),
			Images:         images,
		},
		Quotation: domain.QuoteNested{
			QuoteID:       row.QuotationID,
			PricePerPiece: row.PricePerPiece,
			MoldCost:      row.MoldCost,
			LeadTimeDays:  row.LeadTimeDays,
		},
	}, nil
}

func (s *OrderService) UpdateStatus(orderID int64, status string, actorUserID *int64) error {
	if err := s.repo.UpdateStatus(orderID, strings.TrimSpace(strings.ToUpper(status))); err != nil {
		return err
	}
	return s.repo.InsertActivity(orderID, actorUserID, "ORDER_STATUS", map[string]interface{}{
		"status": strings.TrimSpace(strings.ToUpper(status)),
	})
}

func (s *OrderService) ListActivity(orderID int64) ([]domain.OrderActivityEntry, error) {
	return s.repo.ListActivity(orderID)
}

func (s *OrderService) Cancel(orderID, userID int64, role string) error {
	order, err := s.repo.GetByParticipant(orderID, userID, role)
	if err != nil {
		return err
	}
	cancellableStatuses := map[string]struct{}{"PE": {}, "PP": {}, "PR": {}, "WF": {}}
	if _, ok := cancellableStatuses[order.Status]; !ok {
		return ErrOrderCannotBeCancelled
	}
	if err := s.repo.UpdateStatus(orderID, "CC"); err != nil {
		return err
	}
	return s.repo.InsertActivity(orderID, &userID, "ORDER_CANCELLED", map[string]interface{}{
		"status":          "CC",
		"previous_status": order.Status,
	})
}

func (s *OrderService) MarkShipped(orderID, factoryID int64, trackingNo, courier string) error {
	trackingNo = strings.TrimSpace(trackingNo)
	courier = strings.TrimSpace(courier)
	if trackingNo == "" || courier == "" {
		return ErrShipOrderInvalid
	}
	order, err := s.repo.GetByParticipant(orderID, factoryID, domain.RoleFactory)
	if err != nil {
		return err
	}
	if order.Status != "PR" && order.Status != "QC" && order.Status != "SH" {
		return sql.ErrNoRows
	}
	if err := s.repo.MarkShipped(orderID, factoryID, trackingNo, courier); err != nil {
		return err
	}
	uid := factoryID
	return s.repo.InsertActivity(orderID, &uid, "ORDER_SHIPPED", map[string]interface{}{
		"status":      "SH",
		"tracking_no": trackingNo,
		"courier":     courier,
	})
}

func (s *OrderService) ConfirmReceipt(orderID, userID int64, role string, input ConfirmReceiptInput) (*ConfirmReceiptResult, error) {
	if role != domain.RoleCustomer {
		return nil, domain.ErrForbidden
	}
	return s.confirmReceiptTx(orderID, &userID, strings.TrimSpace(input.Note), input.ReceivedAt, "CUSTOMER_CONFIRMED_RECEIPT", true)
}

func (s *OrderService) GetReviewState(orderID, userID int64, role string) (*domain.OrderReviewState, error) {
	if role != domain.RoleCustomer {
		return nil, domain.ErrForbidden
	}
	order, err := s.repo.GetByParticipant(orderID, userID, role)
	if err != nil {
		return nil, err
	}

	factoryName := fmt.Sprintf("โรงงาน #%d", order.FactoryID)
	if detail, detailErr := s.repo.GetDetailByParticipant(orderID, userID, role); detailErr == nil && strings.TrimSpace(detail.FactoryName) != "" {
		factoryName = detail.FactoryName
	}

	state := &domain.OrderReviewState{
		OrderID:         order.OrderID,
		FactoryID:       order.FactoryID,
		FactoryName:     factoryName,
		Eligible:        false,
		AlreadyReviewed: false,
	}

	review, err := s.reviews.GetByOrderAndUser(orderID, userID)
	if err == nil {
		state.AlreadyReviewed = true
		state.Review = review
		reason := "already_reviewed"
		state.Reason = &reason
		return state, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if normalizeOrderStatus(order.Status) != "CP" {
		reason := "order_not_completed"
		state.Reason = &reason
		return state, nil
	}

	state.Eligible = true
	return state, nil
}

func (s *OrderService) CreateReview(orderID, userID int64, role string, input CreateOrderReviewInput) (*domain.FactoryReview, error) {
	if role != domain.RoleCustomer {
		return nil, domain.ErrForbidden
	}
	if input.Rating < 1 || input.Rating > 5 {
		return nil, ErrReviewRatingInvalid
	}
	comment := strings.TrimSpace(input.Comment)
	if comment == "" || len(comment) > 1000 {
		return nil, ErrReviewCommentInvalid
	}

	order, err := s.repo.GetByParticipant(orderID, userID, role)
	if err != nil {
		return nil, err
	}
	if normalizeOrderStatus(order.Status) != "CP" {
		return nil, ErrReviewOrderNotCompleted
	}
	if _, err := s.reviews.GetByOrderAndUser(orderID, userID); err == nil {
		return nil, ErrReviewAlreadyExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	orderIDPtr := order.OrderID
	review := &domain.FactoryReview{
		FactoryID: order.FactoryID,
		UserID:    userID,
		OrderID:   &orderIDPtr,
		Rating:    input.Rating,
		Comment:   comment,
	}
	if err := s.reviews.CreateForOrderTx(tx, review); err != nil {
		if errors.Is(err, repository.ErrReviewAlreadyExists) {
			return nil, ErrReviewAlreadyExists
		}
		return nil, err
	}
	if err := s.reviews.SyncFactoryAggregateTx(tx, order.FactoryID); err != nil {
		return nil, err
	}
	if err := s.repo.InsertActivityTx(tx, orderID, &userID, "REVIEW_CREATED", map[string]interface{}{
		"review_id": review.ReviewID,
		"rating":    review.Rating,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return review, nil
}

func (s *OrderService) AutoCloseShippedOrders() (int, error) {
	cutoff := time.Now().AddDate(0, 0, -20)
	candidates, err := s.repo.ListAutoCloseCandidates(cutoff)
	if err != nil {
		return 0, err
	}
	closed := 0
	for _, orderID := range candidates {
		if _, err := s.confirmReceiptTx(orderID, nil, "auto close after 20 days", nil, "AUTO_CLOSE_20_DAYS", true); err != nil {
			// Keep processing next orders; this job should be best-effort.
			continue
		}
		closed++
	}
	return closed, nil
}

func (s *OrderService) confirmReceiptTx(orderID int64, actorUserID *int64, note string, receivedAt *time.Time, activityCode string, idempotent bool) (*ConfirmReceiptResult, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	order, err := s.repo.GetByIDForUpdateTx(tx, orderID)
	if err != nil {
		return nil, err
	}
	if actorUserID != nil && order.UserID != *actorUserID {
		return nil, domain.ErrForbidden
	}

	statusBefore := normalizeOrderStatus(order.Status)
	if statusBefore == "CP" {
		if !idempotent {
			return nil, ErrConfirmReceiptNotAllowed
		}
		now := time.Now()
		if receivedAt != nil {
			now = *receivedAt
		}
		return &ConfirmReceiptResult{
			Success:         true,
			OrderID:         order.OrderID,
			StatusBefore:    "CP",
			StatusAfter:     "CP",
			CompletedStepID: 6,
			CompletedAt:     now,
			AlreadyComplete: true,
		}, nil
	}
	if statusBefore == "CN" || statusBefore == "CC" {
		return nil, ErrConfirmReceiptNotAllowed
	}
	if statusBefore != "SH" {
		return nil, ErrConfirmReceiptInvalidStatus
	}

	completedAt := time.Now().UTC()
	if receivedAt != nil {
		completedAt = receivedAt.UTC()
	}

	if err := s.repo.UpsertCompletedStepTx(tx, orderID, actorUserID, note, completedAt); err != nil {
		return nil, err
	}
	if err := s.repo.MarkCompletedTx(tx, orderID, completedAt); err != nil {
		return nil, err
	}

	if _, err := s.wallets.EnsureWallet(tx, order.FactoryID); err != nil {
		return nil, err
	}
	factoryWallet, err := s.wallets.GetByUserIDForUpdate(tx, order.FactoryID)
	if err != nil {
		return nil, err
	}
	movedAmount := roundCurrency(order.TotalAmount)
	if movedAmount < 0 {
		movedAmount = 0
	}
	if err := s.wallets.MovePendingToGoodTx(tx, factoryWallet.WalletID, movedAmount); err != nil {
		return nil, err
	}
	settlement := ConfirmReceiptSettlement{
		FactoryUserID: order.FactoryID,
		WalletID:      factoryWallet.WalletID,
		MovedAmount:   movedAmount,
		PendingBefore: factoryWallet.PendingFund,
		PendingAfter:  roundCurrency(factoryWallet.PendingFund - movedAmount),
		GoodBefore:    factoryWallet.GoodFund,
		GoodAfter:     roundCurrency(factoryWallet.GoodFund + movedAmount),
	}

	orderIDPtr := order.OrderID
	settleTx := &domain.Transaction{
		TxID:       "tx-" + uuid.NewString(),
		WalletID:   factoryWallet.WalletID,
		OrderID:    &orderIDPtr,
		Type:       "SC",
		Amount:     movedAmount,
		Status:     "PT",
		CreatedAt:  completedAt,
		UpdatedAt:  completedAt,
		UploadedAt: completedAt,
	}
	if err := s.txLedger.CreateTx(tx, settleTx); err != nil {
		return nil, err
	}
	if err := s.repo.InsertActivityTx(tx, orderID, actorUserID, activityCode, map[string]interface{}{
		"status_before": statusBefore,
		"status_after":  "CP",
		"completed_at":  completedAt,
		"settlement":    settlement,
		"note":          note,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &ConfirmReceiptResult{
		Success:         true,
		OrderID:         orderID,
		StatusBefore:    statusBefore,
		StatusAfter:     "CP",
		CompletedStepID: 6,
		Settlement:      settlement,
		CompletedAt:     completedAt,
	}, nil
}

func expectedPaymentAmount(order *domain.Order, paymentType string) (float64, error) {
	switch paymentType {
	case "DP":
		return order.DepositAmount, nil
	case "FP":
		return order.TotalAmount - order.DepositAmount, nil
	default:
		return 0, ErrPaymentTypeInvalid
	}
}

func (s *OrderService) depositPaidAt(orderID int64) *time.Time {
	txType := "DP"
	status := "PT"
	items, err := s.txLedger.List(repository.TransactionFilters{OrderID: &orderID, Type: &txType, Status: &status})
	if err != nil || len(items) == 0 {
		return nil
	}
	paidAt := items[0].UpdatedAt.In(thailandLocation)
	return &paidAt
}

func (s *OrderService) finalPaymentPaidAt(orderID int64) *time.Time {
	txType := "FP"
	status := "PT"
	items, err := s.txLedger.List(repository.TransactionFilters{OrderID: &orderID, Type: &txType, Status: &status})
	if err != nil || len(items) == 0 {
		return nil
	}
	paidAt := items[0].UpdatedAt.In(thailandLocation)
	return &paidAt
}

func deriveDepositDueDate(row *repository.OrderDetailRow) *time.Time {
	if row.DepositScheduleDue != nil && !row.DepositScheduleDue.IsZero() {
		due := row.DepositScheduleDue.In(thailandLocation)
		due = time.Date(due.Year(), due.Month(), due.Day(), 23, 59, 59, 0, thailandLocation)
		return &due
	}
	due := deriveDefaultDepositDueTimestamp(row.CreatedAt)
	return &due
}

func buildNextAction(row *repository.OrderDetailRow, status string, depositDueDate, depositPaidAt, finalPaidAt *time.Time, nowTH time.Time) *domain.OrderNextAction {
	// Business rule: 100% upfront payment — no installments.
	switch status {
	case "PP":
		return &domain.OrderNextAction{
			Actor:      "CUSTOMER",
			Type:       "PAY_FULL_AMOUNT",
			Amount:     row.TotalAmount,
			Currency:   "THB",
			DueDate:    depositDueDate,
			CTAURL:     fmt.Sprintf("/orders/%d/payment?stage=full", row.OrderID),
			CTALabelTH: "ชำระเงินเต็มจำนวน",
		}
	case "PE":
		if depositDueDate == nil {
			return nil
		}
		graceEnds := depositDueDate.AddDate(0, 0, 3)
		if nowTH.After(graceEnds) {
			return nil
		}
		return &domain.OrderNextAction{
			Actor:      "CUSTOMER",
			Type:       "PAY_FULL_AMOUNT",
			Amount:     row.TotalAmount,
			Currency:   "THB",
			DueDate:    &graceEnds,
			CTAURL:     fmt.Sprintf("/orders/%d/payment?stage=full", row.OrderID),
			CTALabelTH: "ชำระเงินเต็มจำนวน",
		}
	case "PR", "QC", "SH", "CP", "CN":
		return nil
	}
	return nil
}

func buildPaymentSchedule(row *repository.OrderDetailRow, status string, depositDueDate, depositPaidAt, finalPaidAt *time.Time) []domain.OrderPaymentScheduleItem {
	// Business rule: customer pays 100% upfront before production starts.
	total := row.TotalAmount
	paidStatus := "PENDING"
	if depositPaidAt != nil || finalPaidAt != nil ||
		status == "PD" || status == "PR" || status == "QC" || status == "SH" || status == "CP" {
		paidStatus = "PAID"
	} else if status == "PE" {
		paidStatus = "OVERDUE"
	}

	return []domain.OrderPaymentScheduleItem{
		{
			Stage:   domain.PaymentStageFullPayment,
			Percent: 100,
			Amount:  total,
			Status:  paidStatus,
			DueDate: depositDueDate,
			PaidAt:  depositPaidAt,
		},
	}
}

func normalizeOrderStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "CC":
		return "CN"
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

func orderStatusLabelTH(status string) string {
	switch status {
	case "PP":
		return "รอชำระเงิน"
	case "PE":
		return "หมดกำหนดชำระ"
	case "PD":
		return "ชำระเงินแล้ว รอเริ่มผลิต"
	case "PR":
		return "กำลังผลิต"
	case "QC":
		return "ตรวจสอบคุณภาพ"
	case "SH":
		return "จัดส่งแล้ว"
	case "CP":
		return "เสร็จสิ้น"
	case "CN":
		return "ยกเลิก"
	default:
		return status
	}
}

func roundCurrency(v float64) float64 {
	return math.Round(v*100) / 100
}

func percentOf(amount, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return roundCurrency((amount / total) * 100)
}

func timePtrInTH(v *time.Time) *time.Time {
	if v == nil {
		return nil
	}
	t := v.In(thailandLocation)
	return &t
}

func (s *OrderService) ensureDepositPayable(order *domain.Order) error {
	status := normalizeOrderStatus(order.Status)
	if status == "PD" || status == "PR" || status == "QC" || status == "SH" || status == "CP" {
		return ErrDepositAlreadyPaid
	}
	dueDate := s.lookupDepositDueDate(order)
	if dueDate != nil && time.Now().In(thailandLocation).After(dueDate.AddDate(0, 0, 3)) {
		return ErrDepositExpired
	}
	return nil
}

func (s *OrderService) lookupDepositDueDate(order *domain.Order) *time.Time {
	if s.schedules != nil {
		items, err := s.schedules.ListByOrderID(order.OrderID)
		if err == nil {
			for _, item := range items {
				if item.InstallmentNo == 1 {
					due := item.DueDate.In(thailandLocation)
					due = time.Date(due.Year(), due.Month(), due.Day(), 23, 59, 59, 0, thailandLocation)
					return &due
				}
			}
		}
	}
	detailRow := &repository.OrderDetailRow{Order: *order}
	return deriveDepositDueDate(detailRow)
}

func deriveDefaultDepositScheduleDate(createdAt time.Time) time.Time {
	due := createdAt.In(thailandLocation).AddDate(0, 0, 3)
	return time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, thailandLocation)
}

func deriveDefaultDepositDueTimestamp(createdAt time.Time) time.Time {
	due := deriveDefaultDepositScheduleDate(createdAt)
	return time.Date(due.Year(), due.Month(), due.Day(), 23, 59, 59, 0, thailandLocation)
}

func insertDomainEventTx(tx *sqlx.Tx, eventType string, payload interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO domain_events (event_type, payload) VALUES ($1, $2)`, eventType, b)
	return err
}

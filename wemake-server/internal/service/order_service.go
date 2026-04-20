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

type OrderService struct {
	db         *sqlx.DB
	repo       *repository.OrderRepository
	schedules  *repository.PaymentScheduleRepository
	wallets    *repository.WalletRepository
	txLedger   *repository.TransactionRepository
	quotations *repository.QuotationRepository
	rfqs       *repository.RFQRepository
}

func NewOrderService(db *sqlx.DB, repo *repository.OrderRepository, schedules *repository.PaymentScheduleRepository, wallets *repository.WalletRepository, txLedger *repository.TransactionRepository, quotations *repository.QuotationRepository, rfqs *repository.RFQRepository) *OrderService {
	return &OrderService{db: db, repo: repo, schedules: schedules, wallets: wallets, txLedger: txLedger, quotations: quotations, rfqs: rfqs}
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

func (s *OrderService) List(userID int64, role string, status string) ([]domain.Order, error) {
	st := strings.TrimSpace(strings.ToUpper(status))
	if role == domain.RoleFactory {
		return s.repo.ListByFactoryID(userID, st)
	}
	return s.repo.ListByUserID(userID, st)
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
	depositDueDate := deriveDepositDueDate(row)
	nowTH := time.Now().In(thailandLocation)
	depositPaidAt := s.depositPaidAt(row.OrderID)
	finalPaidAt := s.finalPaymentPaidAt(row.OrderID)
	statusCode := normalizeOrderStatus(row.Status)

	return &domain.OrderDetailResponse{
		OrderID:           row.OrderID,
		QuotationID:       row.QuotationID,
		UserID:            row.UserID,
		FactoryID:         row.FactoryID,
		TotalAmount:       row.TotalAmount,
		DepositAmount:     row.DepositAmount,
		Status:            statusCode,
		StatusLabelTH:     orderStatusLabelTH(statusCode),
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
	switch status {
	case "PP":
		return &domain.OrderNextAction{
			Actor:      "CUSTOMER",
			Type:       "PAY_DEPOSIT",
			Amount:     row.DepositAmount,
			Currency:   "THB",
			DueDate:    depositDueDate,
			CTAURL:     fmt.Sprintf("/orders/%d/payment?stage=deposit", row.OrderID),
			CTALabelTH: "ชำระเงินมัดจำ",
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
			Type:       "PAY_DEPOSIT",
			Amount:     row.DepositAmount,
			Currency:   "THB",
			DueDate:    &graceEnds,
			CTAURL:     fmt.Sprintf("/orders/%d/payment?stage=deposit", row.OrderID),
			CTALabelTH: "ชำระเงินมัดจำ",
		}
	case "PR", "QC":
		if finalPaidAt == nil {
			return &domain.OrderNextAction{
				Actor:      "CUSTOMER",
				Type:       "PAY_PRODUCTION",
				Amount:     roundCurrency(row.TotalAmount * 0.4),
				Currency:   "THB",
				CTAURL:     fmt.Sprintf("/orders/%d/payment?stage=production", row.OrderID),
				CTALabelTH: "ชำระเงินงวดผลิต",
			}
		}
	case "SH":
		if finalPaidAt == nil {
			return &domain.OrderNextAction{
				Actor:      "CUSTOMER",
				Type:       "PAY_DELIVERY",
				Amount:     roundCurrency(row.TotalAmount - row.DepositAmount - roundCurrency(row.TotalAmount*0.4)),
				Currency:   "THB",
				CTAURL:     fmt.Sprintf("/orders/%d/payment?stage=delivery", row.OrderID),
				CTALabelTH: "ชำระเงินก่อนจัดส่ง",
			}
		}
	case "CP", "CN":
		return nil
	}
	if depositPaidAt != nil {
		return nil
	}
	return nil
}

func buildPaymentSchedule(row *repository.OrderDetailRow, status string, depositDueDate, depositPaidAt, finalPaidAt *time.Time) []domain.OrderPaymentScheduleItem {
	depositAmount := row.DepositAmount
	if depositAmount <= 0 {
		depositAmount = roundCurrency(row.TotalAmount * 0.3)
	}
	productionAmount := roundCurrency(row.TotalAmount * 0.4)
	remaining := roundCurrency(row.TotalAmount - depositAmount)
	if productionAmount > remaining {
		productionAmount = remaining
	}
	deliveryAmount := roundCurrency(remaining - productionAmount)
	depositPercent := percentOf(depositAmount, row.TotalAmount)
	productionPercent := percentOf(productionAmount, row.TotalAmount)
	deliveryPercent := percentOf(deliveryAmount, row.TotalAmount)
	productionTrigger := "PRODUCTION"
	deliveryTrigger := "READY_TO_SHIP"

	depositStatus := "PENDING"
	if depositPaidAt != nil || status == "PD" || status == "PR" || status == "QC" || status == "SH" || status == "CP" {
		depositStatus = "PAID"
	} else if status == "PE" {
		depositStatus = "OVERDUE"
	}

	productionStatus := "LOCKED"
	deliveryStatus := "LOCKED"
	if finalPaidAt != nil {
		productionStatus = "PAID"
		deliveryStatus = "PAID"
	} else if status == "PR" || status == "QC" || status == "SH" || status == "CP" {
		productionStatus = "PENDING"
		if status == "SH" || status == "CP" {
			deliveryStatus = "PENDING"
		}
	}

	return []domain.OrderPaymentScheduleItem{
		{
			Stage:   domain.PaymentStageDeposit,
			Percent: depositPercent,
			Amount:  depositAmount,
			Status:  depositStatus,
			DueDate: depositDueDate,
			PaidAt:  depositPaidAt,
		},
		{
			Stage:           domain.PaymentStageProduction,
			Percent:         productionPercent,
			Amount:          productionAmount,
			Status:          productionStatus,
			TriggeredByStep: &productionTrigger,
		},
		{
			Stage:           domain.PaymentStageDelivery,
			Percent:         deliveryPercent,
			Amount:          deliveryAmount,
			Status:          deliveryStatus,
			TriggeredByStep: &deliveryTrigger,
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
		return "รอชำระมัดจำ"
	case "PE":
		return "หมดกำหนดชำระ"
	case "PD":
		return "ชำระมัดจำแล้ว"
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

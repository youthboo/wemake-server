package service

import (
	"database/sql"
	"errors"
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

type OrderService struct {
	db         *sqlx.DB
	repo       *repository.OrderRepository
	wallets    *repository.WalletRepository
	txLedger   *repository.TransactionRepository
	quotations *repository.QuotationRepository
	rfqs       *repository.RFQRepository
}

func NewOrderService(db *sqlx.DB, repo *repository.OrderRepository, wallets *repository.WalletRepository, txLedger *repository.TransactionRepository, quotations *repository.QuotationRepository, rfqs *repository.RFQRepository) *OrderService {
	return &OrderService{db: db, repo: repo, wallets: wallets, txLedger: txLedger, quotations: quotations, rfqs: rfqs}
}

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
	deposit := total * 0.5

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
	if paymentTx.Status != "ST" {
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

	if paymentTx.Type == "DP" && order.Status == "PP" {
		if err := s.repo.UpdateStatusTx(tx, orderID, "PR"); err != nil {
			return nil, err
		}
		order.Status = "PR"
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
	return s.repo.GetByParticipant(orderID, userID, role)
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

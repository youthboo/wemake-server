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

var ErrQuotationNotAccepted = errors.New("quotation must be accepted before creating order")
var ErrShipOrderInvalid = errors.New("tracking_no and courier are required")
var ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled in its current status")
var ErrInsufficientGoodFund = errors.New("insufficient good_fund balance")
var ErrOrderAlreadyExistsForQuote = errors.New("order already exists for this quotation")

type OrderService struct {
	db       *sqlx.DB
	repo     *repository.OrderRepository
	wallets  *repository.WalletRepository
	txLedger *repository.TransactionRepository
}

func NewOrderService(db *sqlx.DB, repo *repository.OrderRepository, wallets *repository.WalletRepository, txLedger *repository.TransactionRepository) *OrderService {
	return &OrderService{db: db, repo: repo, wallets: wallets, txLedger: txLedger}
}

// CreateFromQuotation validates the quotation, charges the customer's good_fund for the full
// order total, credits the factory wallet, inserts two settled transactions (BU / SC), and creates the order — all in one DB transaction.
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
	if src.Status != "AC" {
		return nil, ErrQuotationNotAccepted
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
		Status:            "PR",
		EstimatedDelivery: &deliveryDate,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if _, err := s.wallets.EnsureWallet(tx, userID); err != nil {
		return nil, err
	}
	if _, err := s.wallets.EnsureWallet(tx, src.FactoryID); err != nil {
		return nil, err
	}

	customerWallet, err := s.wallets.GetByUserIDForUpdate(tx, userID)
	if err != nil {
		return nil, err
	}

	ok, err := s.wallets.DebitGoodFund(tx, customerWallet.WalletID, total)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInsufficientGoodFund
	}

	factoryWallet, err := s.wallets.GetByUserIDForUpdate(tx, src.FactoryID)
	if err != nil {
		return nil, err
	}
	if err := s.wallets.CreditGoodFund(tx, factoryWallet.WalletID, total); err != nil {
		return nil, err
	}

	if err := s.repo.CreateTx(tx, order); err != nil {
		return nil, err
	}

	orderIDPtr := order.OrderID
	buyTxID := "tx-" + uuid.NewString()
	recvTxID := "tx-" + uuid.NewString()

	buyRow := &domain.Transaction{
		TxID:       buyTxID,
		WalletID:   customerWallet.WalletID,
		OrderID:    &orderIDPtr,
		Type:       "BU",
		Amount:     total,
		Status:     "ST",
		CreatedAt:  now,
		UpdatedAt:  now,
		UploadedAt: now,
	}
	recvRow := &domain.Transaction{
		TxID:       recvTxID,
		WalletID:   factoryWallet.WalletID,
		OrderID:    &orderIDPtr,
		Type:       "SC",
		Amount:     total,
		Status:     "ST",
		CreatedAt:  now,
		UpdatedAt:  now,
		UploadedAt: now,
	}
	if err := s.txLedger.CreateTx(tx, buyRow); err != nil {
		return nil, err
	}
	if err := s.txLedger.CreateTx(tx, recvRow); err != nil {
		return nil, err
	}

	uid := userID
	if err := s.repo.InsertActivityTx(tx, order.OrderID, &uid, "ORDER_CREATED", map[string]interface{}{
		"status": order.Status, "quote_id": order.QuotationID,
		"amount": total,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return order, nil
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
		"status":         "CC",
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

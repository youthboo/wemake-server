package service

import (
	"database/sql"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

var (
	ErrPaymentMethodNotSupported       = errors.New("METHOD_NOT_SUPPORTED")
	ErrPaymentTypeNotSupported         = errors.New("TYPE_NOT_SUPPORTED")
	ErrPaymentInsufficientWallet       = errors.New("INSUFFICIENT_WALLET_BALANCE")
	ErrPaymentFactoryWalletNotFound    = errors.New("FACTORY_WALLET_NOT_FOUND")
	ErrPaymentNotOrderOwner            = errors.New("NOT_ORDER_OWNER")
	ErrPaymentIdempotencyKeyRequired   = errors.New("IDEMPOTENCY_KEY_REQUIRED")
	ErrPaymentIdempotencyReplayMissing = errors.New("IDEMPOTENCY_REPLAY_MISSING")
)

type OrderPaymentService struct {
	db *sqlx.DB
}

type OrderPaymentInput struct {
	OrderID        int64
	UserID         int64
	Type           string
	Amount         float64
	PaymentMethod  string
	IdempotencyKey string
}

type WalletPaymentTransaction struct {
	TxID              string  `json:"tx_id"`
	Amount            float64 `json:"amount"`
	Type              string  `json:"type"`
	Direction         string  `json:"direction"`
	Status            string  `json:"status"`
	HeldInPendingFund bool    `json:"held_in_pending_fund,omitempty"`
}

type WalletBalanceAfter struct {
	GoodFund    float64 `json:"good_fund"`
	PendingFund float64 `json:"pending_fund"`
}

type OrderPaymentResponse struct {
	SettlementGroupID   string                   `json:"settlement_group_id"`
	CustomerTransaction WalletPaymentTransaction `json:"customer_transaction"`
	FactoryTransaction  WalletPaymentTransaction `json:"factory_transaction"`
	OrderStatusAfter    string                   `json:"order_status_after"`
	WalletBalanceAfter  WalletBalanceAfter       `json:"wallet_balance_after"`
}

type PaymentRuleError struct {
	Err       error
	Shortfall float64
}

func (e *PaymentRuleError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *PaymentRuleError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func AsPaymentRuleError(err error) (*PaymentRuleError, bool) {
	var rule *PaymentRuleError
	if errors.As(err, &rule) {
		return rule, true
	}
	return nil, false
}

func NewOrderPaymentService(db *sqlx.DB) *OrderPaymentService {
	return &OrderPaymentService{db: db}
}

func (s *OrderPaymentService) PayDeposit(input OrderPaymentInput) (*OrderPaymentResponse, error) {
	input.Type = strings.ToUpper(strings.TrimSpace(input.Type))
	input.PaymentMethod = strings.ToUpper(strings.TrimSpace(input.PaymentMethod))
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)

	if input.Type != "DP" && input.Type != "FP" {
		return nil, &PaymentRuleError{Err: ErrPaymentTypeNotSupported}
	}
	if input.PaymentMethod != "WALLET" {
		return nil, &PaymentRuleError{Err: ErrPaymentMethodNotSupported}
	}
	if input.IdempotencyKey == "" {
		return nil, &PaymentRuleError{Err: ErrPaymentIdempotencyKeyRequired}
	}

	if replay, err := s.loadPaymentReplay(input.OrderID, input.IdempotencyKey); err != nil {
		return nil, err
	} else if replay != nil {
		return replay, nil
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	order, err := lockPaymentOrder(tx, input.OrderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != input.UserID {
		return nil, &PaymentRuleError{Err: ErrPaymentNotOrderOwner}
	}
	if replay, err := s.loadPaymentReplay(input.OrderID, input.IdempotencyKey); err != nil {
		return nil, err
	} else if replay != nil {
		return replay, nil
	}
	switch normalizeOrderStatus(order.Status) {
	case "PR", "QC", "SH", "CP", "PD":
		return nil, &PaymentRuleError{Err: ErrDepositAlreadyPaid}
	case "PE":
		return nil, &PaymentRuleError{Err: ErrDepositExpired}
	case "PP":
	default:
		return nil, &PaymentRuleError{Err: ErrDepositAlreadyPaid}
	}
	// For full-payment model (DP or FP), validate against total_amount.
	// Fall back to deposit_amount only if total_amount is zero (legacy orders).
	expectedAmount := order.TotalAmount
	if expectedAmount <= 0 {
		expectedAmount = order.DepositAmount
	}
	if !amountEqual(input.Amount, expectedAmount) {
		return nil, &PaymentRuleError{Err: ErrPaymentAmountMismatch}
	}

	customerWallet, err := getPaymentWallet(tx, order.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &PaymentRuleError{Err: ErrPaymentInsufficientWallet, Shortfall: roundCurrency(input.Amount)}
		}
		return nil, err
	}
	factoryWallet, err := getPaymentWallet(tx, order.FactoryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &PaymentRuleError{Err: ErrPaymentFactoryWalletNotFound}
		}
		return nil, err
	}
	if err := lockWalletsInOrder(tx, customerWallet.WalletID, factoryWallet.WalletID); err != nil {
		return nil, err
	}
	customerWallet, err = getPaymentWallet(tx, order.UserID)
	if err != nil {
		return nil, err
	}
	factoryWallet, err = getPaymentWallet(tx, order.FactoryID)
	if err != nil {
		return nil, err
	}

	var customerGoodAfter float64
	err = tx.QueryRow(`
		UPDATE wallets
		SET good_fund = good_fund - $1
		WHERE wallet_id = $2 AND good_fund >= $1
		RETURNING good_fund
	`, input.Amount, customerWallet.WalletID).Scan(&customerGoodAfter)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			shortfall := roundCurrency(input.Amount - customerWallet.GoodFund)
			if shortfall < 0 {
				shortfall = 0
			}
			return nil, &PaymentRuleError{Err: ErrPaymentInsufficientWallet, Shortfall: shortfall}
		}
		return nil, err
	}

	var factoryPendingAfter float64
	if err := tx.QueryRow(`
		UPDATE wallets
		SET pending_fund = pending_fund + $1
		WHERE wallet_id = $2
		RETURNING pending_fund
	`, input.Amount, factoryWallet.WalletID).Scan(&factoryPendingAfter); err != nil {
		return nil, err
	}

	groupID := uuid.New()
	customerTxID := "tx_" + uuid.NewString()
	factoryTxID := "tx_" + uuid.NewString()
	now := time.Now()
	if err := insertWalletPaymentTransaction(tx, customerTxID, customerWallet.WalletID, input.OrderID, -input.Amount, "D", input.IdempotencyKey, groupID, now); err != nil {
		if isUniqueViolation(err) {
			return s.loadPaymentReplay(input.OrderID, input.IdempotencyKey)
		}
		return nil, err
	}
	if err := insertWalletPaymentTransaction(tx, factoryTxID, factoryWallet.WalletID, input.OrderID, input.Amount, "C", input.IdempotencyKey+":f", groupID, now); err != nil {
		if isUniqueViolation(err) {
			return s.loadPaymentReplay(input.OrderID, input.IdempotencyKey)
		}
		return nil, err
	}

	if _, err := tx.Exec(`UPDATE orders SET status = 'PR', updated_at = NOW() WHERE order_id = $1 AND status = 'PP'`, input.OrderID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`
		UPDATE payment_schedules
		SET status = 'PD',
		    paid_at = NOW()
		WHERE order_id = $1 AND installment_no = 1
	`, input.OrderID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`
		INSERT INTO notifications (user_id, type, title, message, link_to, reference_id)
		VALUES ($1, 'PS', $2, $3, $4, $5)
	`, order.FactoryID,
		"ได้รับเงินมัดจำ — เริ่มการผลิตได้",
		"คำสั่งซื้อได้ชำระเงินมัดจำเรียบร้อย กรุณาเริ่มสายการผลิต",
		"/factory/orders/"+formatInt64(input.OrderID),
		input.OrderID,
	); err != nil {
		return nil, err
	}
	if err := insertDomainEventTx(tx, "order.deposit_paid", map[string]interface{}{
		"order_id":            input.OrderID,
		"settlement_group_id": groupID.String(),
		"amount":              input.Amount,
	}); err != nil {
		return nil, err
	}
	if err := insertDomainEventTx(tx, "cache.invalidate", map[string]interface{}{
		"paths": []string{
			"/orders/" + formatInt64(input.OrderID),
			"/orders/" + formatInt64(input.OrderID) + "/production-updates",
		},
	}); err != nil {
		return nil, err
	}

	out := &OrderPaymentResponse{
		SettlementGroupID: groupID.String(),
		CustomerTransaction: WalletPaymentTransaction{
			TxID:      customerTxID,
			Amount:    -input.Amount,
			Type:      "BU",
			Direction: "D",
			Status:    "ST",
		},
		FactoryTransaction: WalletPaymentTransaction{
			TxID:              factoryTxID,
			Amount:            input.Amount,
			Type:              "BU",
			Direction:         "C",
			Status:            "ST",
			HeldInPendingFund: true,
		},
		OrderStatusAfter: "PR",
		WalletBalanceAfter: WalletBalanceAfter{
			GoodFund:    roundCurrency(customerGoodAfter),
			PendingFund: roundCurrency(customerWallet.PendingFund),
		},
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

type paymentOrderRow struct {
	OrderID       int64   `db:"order_id"`
	UserID        int64   `db:"user_id"`
	FactoryID     int64   `db:"factory_id"`
	DepositAmount float64 `db:"deposit_amount"`
	TotalAmount   float64 `db:"total_amount"`
	Status        string  `db:"status"`
}

type paymentWalletRow struct {
	WalletID    int64   `db:"wallet_id"`
	UserID      int64   `db:"user_id"`
	GoodFund    float64 `db:"good_fund"`
	PendingFund float64 `db:"pending_fund"`
}

func lockPaymentOrder(tx *sqlx.Tx, orderID int64) (*paymentOrderRow, error) {
	var row paymentOrderRow
	err := tx.Get(&row, `
		SELECT order_id, user_id, factory_id, deposit_amount, total_amount, status
		FROM orders
		WHERE order_id = $1
		FOR UPDATE
	`, orderID)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func getPaymentWallet(tx *sqlx.Tx, userID int64) (*paymentWalletRow, error) {
	var row paymentWalletRow
	err := tx.Get(&row, `
		SELECT wallet_id, user_id, good_fund, pending_fund
		FROM wallets
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func lockWalletsInOrder(tx *sqlx.Tx, walletA, walletB int64) error {
	rows, err := tx.Query(`
		SELECT wallet_id
		FROM wallets
		WHERE wallet_id IN ($1, $2)
		ORDER BY wallet_id ASC
		FOR UPDATE
	`, walletA, walletB)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var walletID int64
		if err := rows.Scan(&walletID); err != nil {
			return err
		}
	}
	return rows.Err()
}

func insertWalletPaymentTransaction(tx *sqlx.Tx, txID string, walletID, orderID int64, amount float64, direction, idempotencyKey string, settlementGroupID uuid.UUID, now time.Time) error {
	_, err := tx.Exec(`
		INSERT INTO transactions (
			tx_id, wallet_id, order_id, type, amount, status,
			created_at, updated_at, uploaded_at,
			direction, idempotency_key, settlement_group_id
		)
		VALUES ($1, $2, $3, 'BU', $4, 'ST', $5, $5, $5, $6, $7, $8)
	`, txID, walletID, orderID, amount, now, direction, idempotencyKey, settlementGroupID)
	return err
}

func (s *OrderPaymentService) loadPaymentReplay(orderID int64, idempotencyKey string) (*OrderPaymentResponse, error) {
	type replayRow struct {
		TxID              string         `db:"tx_id"`
		Amount            float64        `db:"amount"`
		Type              string         `db:"type"`
		Direction         sql.NullString `db:"direction"`
		Status            string         `db:"status"`
		SettlementGroupID string         `db:"settlement_group_id"`
		GoodFundAfter     float64        `db:"good_fund"`
		PendingFundAfter  float64        `db:"pending_fund"`
		OrderStatusAfter  string         `db:"order_status"`
		IdempotencyKey    string         `db:"idempotency_key"`
	}

	var rows []replayRow
	err := s.db.Select(&rows, `
		SELECT
			t.tx_id,
			t.amount,
			t.type,
			t.direction,
			t.status,
			t.settlement_group_id::text AS settlement_group_id,
			w.good_fund,
			w.pending_fund,
			o.status AS order_status,
			t.idempotency_key
		FROM transactions t
		INNER JOIN wallets w ON w.wallet_id = t.wallet_id
		INNER JOIN orders o ON o.order_id = t.order_id
		WHERE t.order_id = $1
		  AND t.settlement_group_id = (
			SELECT settlement_group_id
			FROM transactions
			WHERE order_id = $1 AND idempotency_key = $2
			LIMIT 1
		  )
		ORDER BY t.direction DESC
	`, orderID, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	var out OrderPaymentResponse
	out.SettlementGroupID = rows[0].SettlementGroupID
	out.OrderStatusAfter = rows[0].OrderStatusAfter
	for _, row := range rows {
		direction := row.Direction.String
		item := WalletPaymentTransaction{
			TxID:      row.TxID,
			Amount:    row.Amount,
			Type:      row.Type,
			Direction: direction,
			Status:    row.Status,
		}
		if strings.HasSuffix(row.IdempotencyKey, ":f") || direction == "C" || row.Amount > 0 {
			item.HeldInPendingFund = true
			out.FactoryTransaction = item
			continue
		}
		out.CustomerTransaction = item
		out.WalletBalanceAfter = WalletBalanceAfter{
			GoodFund:    roundCurrency(row.GoodFundAfter),
			PendingFund: roundCurrency(row.PendingFundAfter),
		}
	}
	if out.CustomerTransaction.TxID == "" || out.FactoryTransaction.TxID == "" {
		return nil, &PaymentRuleError{Err: ErrPaymentIdempotencyReplayMissing}
	}
	return &out, nil
}

func amountEqual(a, b float64) bool {
	return math.Abs(a-b) <= 0.005
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func formatInt64(v int64) string {
	return strconv.FormatInt(v, 10)
}

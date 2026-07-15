package order

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ErrDuplicateSlip means the bank reference has already been used by another
// verified slip (replay protection via the partial unique index).
var ErrDuplicateSlip = errors.New("slip bank reference already used")

type SlipInfo struct {
	OrderID    int64   `db:"order_id"    json:"order_id"`
	SlipStatus string  `db:"slip_status" json:"slip_status"`
	SlipURL    *string `db:"slip_url"    json:"slip_url,omitempty"`
	SlipNote   *string `db:"slip_note"   json:"slip_note,omitempty"`
	VerifiedBy *int64  `db:"verified_by" json:"verified_by,omitempty"`
	VerifiedAt *string `db:"verified_at" json:"verified_at,omitempty"`
	UploadedAt *string `db:"uploaded_at" json:"uploaded_at,omitempty"`
}

type SlipRepository struct {
	db *sqlx.DB
}

func NewSlipRepository(db *sqlx.DB) *SlipRepository {
	return &SlipRepository{db: db}
}

// GetSlipInfo returns slip info for an order.
func (r *SlipRepository) GetSlipInfo(orderID int64) (*SlipInfo, error) {
	var item SlipInfo
	err := r.db.Get(&item, `
		SELECT o.order_id,
		       COALESCE(TRIM(o.slip_status), 'PE') AS slip_status,
		       t.slip_url,
		       t.slip_note,
		       t.verified_by,
		       t.verified_at::text AS verified_at,
		       t.created_at::text AS uploaded_at
		FROM orders o
		LEFT JOIN transactions t ON t.order_id = o.order_id AND t.type = 'BU'
		    AND t.tx_id = (
		        SELECT MAX(t2.tx_id) FROM transactions t2
		        WHERE t2.order_id = o.order_id AND t2.type = 'BU'
		    )
		WHERE o.order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// AttachSlip creates a transaction with slip info and updates order slip_status.
func (r *SlipRepository) AttachSlip(orderID, walletID int64, amount float64, slipURL, slipNote string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert transaction record — status PT (Pending Transfer) until factory approves
	_, err = tx.Exec(`
		INSERT INTO transactions (wallet_id, order_id, type, status, amount, slip_url, slip_note)
		VALUES ($1, $2, 'BU', 'PT', $3, $4, NULLIF($5, ''))
	`, walletID, orderID, amount, slipURL, slipNote)
	if err != nil {
		return err
	}

	// Update order slip_status + status
	res, err := tx.Exec(`
		UPDATE orders SET slip_status = 'ST', status = 'WA', updated_at = NOW()
		WHERE order_id = $1 AND TRIM(slip_status) IN ('PE', 'RJ')
	`, orderID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}

// ApproveSlip marks slip as approved and moves order to PD status.
func (r *SlipRepository) ApproveSlip(orderID, verifiedBy int64) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update latest deposit transaction — ST (Settled) when factory approves
	_, err = tx.Exec(`
		UPDATE transactions
		SET status = 'ST', verified_by = $2, verified_at = NOW(), updated_at = NOW()
		WHERE tx_id = (
		    SELECT MAX(tx_id) FROM transactions WHERE order_id = $1 AND type = 'BU' AND status = 'PT'
		)
	`, orderID, verifiedBy)
	if err != nil {
		return err
	}

	// Update order: slip_status = AP, status = PD (Payment Done)
	res, err := tx.Exec(`
		UPDATE orders SET slip_status = 'AP', status = 'PD', updated_at = NOW()
		WHERE order_id = $1 AND TRIM(slip_status) = 'ST'
	`, orderID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}

// ensurePlatformWallet resolves the Tryly platform wallet from tconfig
// (key = platform_user_id) and creates it lazily. Must be called inside a tx.
func ensurePlatformWallet(tx *sqlx.Tx) (int64, error) {
	var uid int64
	if err := tx.Get(&uid, `SELECT (value)::bigint FROM tconfig WHERE key = 'platform_user_id'`); err != nil {
		return 0, err
	}
	var walletID int64
	err := tx.Get(&walletID, `SELECT wallet_id FROM wallets WHERE user_id = $1`, uid)
	if err == sql.ErrNoRows {
		err = tx.QueryRow(`
			INSERT INTO wallets (user_id, good_fund, pending_fund)
			VALUES ($1, 0, 0)
			RETURNING wallet_id
		`, uid).Scan(&walletID)
	}
	return walletID, err
}

// ApproveSlipEscrow — escrow mode (config_payment != 1): superadmin approves the slip.
// The BU transaction STAYS 'PT' (funds held by Tryly) and a factory receivable
// (SC / PT) is created on the factory's wallet + pending_fund. Both settle to 'ST'
// when the order reaches CP (see SettleEscrowFunds).
func (r *SlipRepository) ApproveSlipEscrow(orderID, verifiedBy int64) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := approveSlipEscrowTx(tx, orderID, verifiedBy); err != nil {
		return err
	}
	return tx.Commit()
}

// AutoApproveSlipEscrow is the SlipOK auto-verify path: it stamps the SlipOK
// verification metadata (bank_ref + verify_status='verified' + raw response)
// onto the BU transaction — which trips the partial unique index on bank_ref if
// the reference was already used (replay) → ErrDuplicateSlip — then runs the
// same escrow-approve ledger as the manual admin path.
func (r *SlipRepository) AutoApproveSlipEscrow(orderID, verifiedBy int64, bankRef string, transferredAt time.Time, raw json.RawMessage) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := setSlipVerifyMeta(tx, orderID, "verified", bankRef, transferredAt, raw); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrDuplicateSlip
		}
		return err
	}
	if err := approveSlipEscrowTx(tx, orderID, verifiedBy); err != nil {
		return err
	}
	return tx.Commit()
}

// RecordSlipVerifyResult stores SlipOK verification metadata WITHOUT approving —
// used when auto-verify does not fully pass (invalid / retry / unavailable /
// failed a condition). The slip stays 'ST' (pending) for manual admin review.
// A surrogate bankRef (real ref + suffix) is safe to pass since verify_status
// is not 'verified' and therefore not covered by the unique index.
func (r *SlipRepository) RecordSlipVerifyResult(orderID int64, verifyStatus, bankRef string, transferredAt time.Time, raw json.RawMessage) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := setSlipVerifyMeta(tx, orderID, verifyStatus, bankRef, transferredAt, raw); err != nil {
		return err
	}
	return tx.Commit()
}

// setSlipVerifyMeta updates the latest pending BU transaction of an order with
// SlipOK verification metadata. Must run inside a tx.
func setSlipVerifyMeta(tx *sqlx.Tx, orderID int64, verifyStatus, bankRef string, transferredAt time.Time, raw json.RawMessage) error {
	var refArg interface{}
	if bankRef != "" {
		refArg = bankRef
	}
	var atArg interface{}
	if !transferredAt.IsZero() {
		atArg = transferredAt
	}
	var rawArg interface{}
	if len(raw) > 0 {
		rawArg = []byte(raw)
	}
	_, err := tx.Exec(`
		UPDATE transactions
		SET verify_status = $2,
		    bank_ref = COALESCE($3, bank_ref),
		    transferred_at = COALESCE($4, transferred_at),
		    verify_response = COALESCE($5, verify_response),
		    updated_at = NOW()
		WHERE tx_id = (
		    SELECT MAX(tx_id) FROM transactions WHERE order_id = $1 AND type = 'BU' AND status = 'PT'
		)
	`, orderID, verifyStatus, refArg, atArg, rawArg)
	return err
}

func approveSlipEscrowTx(tx *sqlx.Tx, orderID, verifiedBy int64) error {
	// Mark latest BU tx verified but keep status PT (held in escrow)
	var buAmount float64
	err := tx.QueryRow(`
		UPDATE transactions
		SET verified_by = $2, verified_at = NOW(), updated_at = NOW()
		WHERE tx_id = (
		    SELECT MAX(tx_id) FROM transactions WHERE order_id = $1 AND type = 'BU' AND status = 'PT'
		)
		RETURNING amount::float8
	`, orderID, verifiedBy).Scan(&buAmount)
	if err != nil {
		return err
	}

	// Factory receivable = ยอดหลังหักค่าคอมมิชชัน platform (factory_net_receivable
	// จาก quotation ที่ยอมรับแล้ว) — โรงงานได้รับ net ไม่ใช่ยอดเต็มที่ลูกค้าจ่าย.
	// Tryly ถือส่วนต่าง (commission) ไว้. fallback เป็น buAmount ถ้าไม่มี quotation.
	var factoryID int64
	var netReceivable float64
	if err := tx.QueryRow(`
		SELECT o.factory_id, COALESCE(NULLIF(q.factory_net_receivable, 0), $2)
		FROM orders o
		LEFT JOIN quotations q ON q.quote_id = o.quote_id
		WHERE o.order_id = $1
	`, orderID, buAmount).Scan(&factoryID, &netReceivable); err != nil {
		return err
	}
	// Ensure factory wallet — wallets.user_id has only a non-unique index,
	// so we SELECT-then-INSERT instead of ON CONFLICT (user_id).
	var factoryWalletID int64
	err = tx.Get(&factoryWalletID, `SELECT wallet_id FROM wallets WHERE user_id = $1`, factoryID)
	if err == sql.ErrNoRows {
		err = tx.QueryRow(`
			INSERT INTO wallets (user_id, good_fund, pending_fund)
			VALUES ($1, 0, 0)
			RETURNING wallet_id
		`, factoryID).Scan(&factoryWalletID)
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO transactions (wallet_id, order_id, type, status, amount)
		VALUES ($1, $2, 'SC', 'PT', $3)
	`, factoryWalletID, orderID, netReceivable); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE wallets SET pending_fund = pending_fund + $2 WHERE wallet_id = $1
	`, factoryWalletID, netReceivable); err != nil {
		return err
	}

	// Platform (SA) wallet legs — เงินลูกค้าเข้า escrow ของ Tryly ก่อน:
	//   EI (escrow-in) = grand_total ที่ลูกค้าจ่าย → SA.pending (Tryly ถือไว้)
	//   CM (commission) = ส่วนต่าง (grand - net) → realized เข้า SA.good ตอน settle
	// ทำให้ ledger conserved: SA(commission) + FT(net) = grand_total
	commission := buAmount - netReceivable
	if commission < 0 {
		commission = 0
	}
	platformWalletID, err := ensurePlatformWallet(tx)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO transactions (wallet_id, order_id, type, status, amount)
		VALUES ($1, $2, 'EI', 'PT', $3)
	`, platformWalletID, orderID, buAmount); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO transactions (wallet_id, order_id, type, status, amount)
		VALUES ($1, $2, 'CM', 'PT', $3)
	`, platformWalletID, orderID, commission); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE wallets SET pending_fund = pending_fund + $2 WHERE wallet_id = $1
	`, platformWalletID, buAmount); err != nil {
		return err
	}

	// Order: slip approved, payment done — same as direct-pay flow
	res, err := tx.Exec(`
		UPDATE orders SET slip_status = 'AP', status = 'PD', updated_at = NOW()
		WHERE order_id = $1 AND TRIM(slip_status) = 'ST'
	`, orderID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// RejectSlip marks slip as rejected so customer can re-submit.
func (r *SlipRepository) RejectSlip(orderID int64, reason string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Reject latest deposit transaction — look for PT (Pending Transfer) status
	_, err = tx.Exec(`
		UPDATE transactions
		SET status = 'RJ', slip_note = COALESCE(slip_note || E'\n', '') || 'ปฏิเสธ: ' || $2, updated_at = NOW()
		WHERE tx_id = (
		    SELECT MAX(tx_id) FROM transactions WHERE order_id = $1 AND type = 'BU' AND status = 'PT'
		)
	`, orderID, reason)
	if err != nil {
		return err
	}

	// Update order slip_status back to RJ, status back to WS
	res, err := tx.Exec(`
		UPDATE orders SET slip_status = 'RJ', status = 'WS', updated_at = NOW()
		WHERE order_id = $1 AND TRIM(slip_status) = 'ST'
	`, orderID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}

// GetOrderOwnership returns customer_id, factory_id, slip_status, order status.
type OrderOwnership struct {
	OrderID     int64     `db:"order_id"`
	CustomerID  int64     `db:"customer_id"`
	FactoryID   int64     `db:"factory_id"`
	Status      string    `db:"status"`
	SlipStatus  string    `db:"slip_status"`
	TotalAmount float64   `db:"total_amount"`
	CreatedAt   time.Time `db:"created_at"`
}

func (r *SlipRepository) GetOrderOwnership(orderID int64) (*OrderOwnership, error) {
	var item OrderOwnership
	err := r.db.Get(&item, `
		SELECT order_id, customer_id, factory_id,
		       TRIM(status) AS status,
		       COALESCE(TRIM(slip_status), 'PE') AS slip_status,
		       COALESCE(total_amount, 0)::float8 AS total_amount,
		       created_at
		FROM orders WHERE order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// PlatformUserID returns the Tryly platform (SA) user id from tconfig — used as
// the "verified_by" actor when SlipOK auto-approves a slip.
func (r *SlipRepository) PlatformUserID() (int64, error) {
	var uid int64
	err := r.db.Get(&uid, `SELECT (value)::bigint FROM tconfig WHERE key = 'platform_user_id'`)
	return uid, err
}

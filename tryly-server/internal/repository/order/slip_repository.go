package order

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

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
		LEFT JOIN transactions t ON t.order_id = o.order_id AND t.type = 'DE'
		    AND t.tx_id = (
		        SELECT MAX(t2.tx_id) FROM transactions t2
		        WHERE t2.order_id = o.order_id AND t2.type = 'DE'
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

	// Insert transaction record
	_, err = tx.Exec(`
		INSERT INTO transactions (wallet_id, order_id, type, status, amount, slip_url, slip_note)
		VALUES ($1, $2, 'DE', 'ST', $3, $4, NULLIF($5, ''))
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

	// Update latest deposit transaction
	_, err = tx.Exec(`
		UPDATE transactions
		SET status = 'PT', verified_by = $2, verified_at = NOW(), updated_at = NOW()
		WHERE tx_id = (
		    SELECT MAX(tx_id) FROM transactions WHERE order_id = $1 AND type = 'DE' AND status = 'ST'
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

// RejectSlip marks slip as rejected so customer can re-submit.
func (r *SlipRepository) RejectSlip(orderID int64, reason string) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Reject latest deposit transaction
	_, err = tx.Exec(`
		UPDATE transactions
		SET status = 'RJ', slip_note = COALESCE(slip_note || E'\n', '') || 'ปฏิเสธ: ' || $2, updated_at = NOW()
		WHERE tx_id = (
		    SELECT MAX(tx_id) FROM transactions WHERE order_id = $1 AND type = 'DE' AND status = 'ST'
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
	OrderID     int64   `db:"order_id"`
	CustomerID  int64   `db:"customer_id"`
	FactoryID   int64   `db:"factory_id"`
	Status      string  `db:"status"`
	SlipStatus  string  `db:"slip_status"`
	TotalAmount float64 `db:"total_amount"`
}

func (r *SlipRepository) GetOrderOwnership(orderID int64) (*OrderOwnership, error) {
	var item OrderOwnership
	err := r.db.Get(&item, `
		SELECT order_id, customer_id, factory_id,
		       TRIM(status) AS status,
		       COALESCE(TRIM(slip_status), 'PE') AS slip_status,
		       COALESCE(total_amount, 0)::float8 AS total_amount
		FROM orders WHERE order_id = $1
	`, orderID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

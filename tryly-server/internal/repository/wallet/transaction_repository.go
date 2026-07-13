package wallet

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
)

type TransactionRepository struct {
	db *sqlx.DB
}

type TransactionFilters struct {
	WalletID *int64
	OrderID  *int64
	Type     *string
	Status   *string
}

func NewTransactionRepository(db *sqlx.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(item *domain.Transaction) error {
	query := `
		INSERT INTO transactions (tx_id, wallet_id, order_id, type, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(
		query,
		item.TxID,
		item.WalletID,
		item.OrderID,
		item.Type,
		item.Amount,
		item.Status,
		item.CreatedAt,
		item.UpdatedAt,
	)
	return err
}

// CreateTx inserts a transaction row using the given sqlx transaction.
func (r *TransactionRepository) CreateTx(tx *sqlx.Tx, item *domain.Transaction) error {
	query := `
		INSERT INTO transactions (tx_id, wallet_id, order_id, type, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := tx.Exec(
		query,
		item.TxID,
		item.WalletID,
		item.OrderID,
		item.Type,
		item.Amount,
		item.Status,
		item.CreatedAt,
		item.UpdatedAt,
	)
	return err
}

func (r *TransactionRepository) List(filters TransactionFilters) ([]domain.Transaction, error) {
	var items []domain.Transaction
	query := `
		SELECT tx_id, wallet_id, order_id, type, amount, status, created_at, updated_at, created_at AS uploaded_at
		FROM transactions
	`
	conditions := []string{}
	args := []interface{}{}
	argPos := 1

	if filters.WalletID != nil {
		conditions = append(conditions, fmt.Sprintf("wallet_id = $%d", argPos))
		args = append(args, *filters.WalletID)
		argPos++
	}
	if filters.OrderID != nil {
		conditions = append(conditions, fmt.Sprintf("order_id = $%d", argPos))
		args = append(args, *filters.OrderID)
		argPos++
	}
	if filters.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argPos))
		args = append(args, domainutil.NormalizeStatus(*filters.Type))
		argPos++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, domainutil.NormalizeStatus(*filters.Status))
		argPos++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"
	err := r.db.Select(&items, query, args...)
	return items, err
}

func (r *TransactionRepository) PatchStatus(txID string, status string) error {
	query := "UPDATE transactions SET status = $1, updated_at = NOW() WHERE tx_id = $2"
	_, err := r.db.Exec(query, status, txID)
	return err
}

func (r *TransactionRepository) GetByID(txID string) (*domain.Transaction, error) {
	var item domain.Transaction
	err := r.db.Get(&item, `
		SELECT tx_id, wallet_id, order_id, type, amount, status, created_at, updated_at, created_at AS uploaded_at
		FROM transactions
		WHERE tx_id = $1
	`, txID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TransactionRepository) GetByIDForUpdate(tx *sqlx.Tx, txID string) (*domain.Transaction, error) {
	var item domain.Transaction
	err := tx.Get(&item, `
		SELECT tx_id, wallet_id, order_id, type, amount, status, created_at, updated_at, created_at AS uploaded_at
		FROM transactions
		WHERE tx_id = $1
		FOR UPDATE
	`, txID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TransactionRepository) PatchStatusTx(tx *sqlx.Tx, txID string, status string) error {
	res, err := tx.Exec(`
		UPDATE transactions
		SET status = $1, updated_at = NOW()
		WHERE tx_id = $2
	`, status, txID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SettleFactoryReceivables settles all pending (PT) SC transactions for the given order:
// sets status='ST'. Called when the customer confirms receipt.
func (r *TransactionRepository) SettleFactoryReceivables(tx *sqlx.Tx, orderID int64) error {
	_, err := tx.Exec(`
		UPDATE transactions
		SET status     = 'ST',
		    updated_at = NOW()
		WHERE order_id = $1
		  AND type     = 'SC'
		  AND status   = 'PT'
	`, orderID)
	return err
}

// SettleEscrowFunds — เรียกตอน order → CP (ลูกค้ากดรับ หรือ auto-close cron).
// กระจายเงิน escrow ตาม model CT → SA → FT:
//   SC (factory net)     : FT.pending → FT.good        (โรงงานถอนได้)
//   EI (escrow-in)       : SA.pending -= grand_total   (ปล่อยเงินที่ถือไว้)
//   CM (commission)      : SA.good += commission       (รายได้ Tryly realized)
// แล้ว flip SC/EI/CM/BU ที่ค้าง PT → ST.
// WHERE status='PT' ทำให้ idempotent (เรียกซ้ำ / order direct-pay = no-op).
// ต้องเรียกใน DB tx เดียวกับการ mark order = CP.
func (r *TransactionRepository) SettleEscrowFunds(tx *sqlx.Tx, orderID int64) error {
	// FT: factory net receivable — pending → good
	if _, err := tx.Exec(`
		UPDATE wallets w
		SET good_fund    = w.good_fund + t.amount,
		    pending_fund = GREATEST(w.pending_fund - t.amount, 0)
		FROM transactions t
		WHERE t.order_id = $1 AND t.type = 'SC' AND t.status = 'PT'
		  AND w.wallet_id = t.wallet_id
	`, orderID); err != nil {
		return err
	}
	// SA: release escrow hold (EI) — ลด pending ตามยอดที่ถือไว้
	if _, err := tx.Exec(`
		UPDATE wallets w
		SET pending_fund = GREATEST(w.pending_fund - t.amount, 0)
		FROM transactions t
		WHERE t.order_id = $1 AND t.type = 'EI' AND t.status = 'PT'
		  AND w.wallet_id = t.wallet_id
	`, orderID); err != nil {
		return err
	}
	// SA: realize commission (CM) — เข้า good_fund เป็นรายได้ Tryly
	if _, err := tx.Exec(`
		UPDATE wallets w
		SET good_fund = w.good_fund + t.amount
		FROM transactions t
		WHERE t.order_id = $1 AND t.type = 'CM' AND t.status = 'PT'
		  AND w.wallet_id = t.wallet_id
	`, orderID); err != nil {
		return err
	}
	_, err := tx.Exec(`
		UPDATE transactions
		SET status = 'ST', updated_at = NOW()
		WHERE order_id = $1 AND type IN ('SC', 'EI', 'CM', 'BU') AND status = 'PT'
	`, orderID)
	return err
}

package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
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
		INSERT INTO transactions (tx_id, wallet_id, order_id, type, amount, status, created_at, updated_at, uploaded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
		item.UploadedAt,
	)
	return err
}

// CreateTx inserts a transaction row using the given sqlx transaction.
func (r *TransactionRepository) CreateTx(tx *sqlx.Tx, item *domain.Transaction) error {
	query := `
		INSERT INTO transactions (tx_id, wallet_id, order_id, type, amount, status, created_at, updated_at, uploaded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
		item.UploadedAt,
	)
	return err
}

func (r *TransactionRepository) List(filters TransactionFilters) ([]domain.Transaction, error) {
	var items []domain.Transaction
	query := `
		SELECT tx_id, wallet_id, order_id, type, amount, status, created_at, updated_at, uploaded_at
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
		args = append(args, strings.ToUpper(strings.TrimSpace(*filters.Type)))
		argPos++
	}
	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, strings.ToUpper(strings.TrimSpace(*filters.Status)))
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
		SELECT tx_id, wallet_id, order_id, type, amount, status, created_at, updated_at, uploaded_at
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
		SELECT tx_id, wallet_id, order_id, type, amount, status, created_at, updated_at, uploaded_at
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
// sets status='ST' and uploaded_at=NOW(). Called when the customer confirms receipt.
func (r *TransactionRepository) SettleFactoryReceivables(tx *sqlx.Tx, orderID int64) error {
	_, err := tx.Exec(`
		UPDATE transactions
		SET status     = 'ST',
		    updated_at = NOW(),
		    uploaded_at = NOW()
		WHERE order_id = $1
		  AND type     = 'SC'
		  AND status   = 'PT'
	`, orderID)
	return err
}

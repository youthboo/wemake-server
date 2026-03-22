package repository

import (
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

func (r *TransactionRepository) List(filters TransactionFilters) ([]domain.Transaction, error) {
	var items []domain.Transaction
	query := `
		SELECT tx_id, wallet_id, order_id, type, amount, status, created_at, updated_at
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

package repository

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type WithdrawalRepository struct {
	db *sqlx.DB
}

func NewWithdrawalRepository(db *sqlx.DB) *WithdrawalRepository {
	return &WithdrawalRepository{db: db}
}

func (r *WithdrawalRepository) Create(w *domain.WithdrawalRequest) error {
	return r.db.QueryRow(`
		INSERT INTO withdrawal_requests
		    (wallet_id, factory_id, amount, bank_account_no, bank_name, account_name, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'PE')
		RETURNING request_id, created_at, updated_at
	`, w.WalletID, w.FactoryID, w.Amount, w.BankAccountNo, w.BankName, w.AccountName).
		Scan(&w.RequestID, &w.CreatedAt, &w.UpdatedAt)
}

func (r *WithdrawalRepository) ListByFactoryID(factoryID int64) ([]domain.WithdrawalRequest, error) {
	var items []domain.WithdrawalRequest
	err := r.db.Select(&items, `
		SELECT request_id, wallet_id, factory_id, amount, bank_account_no, bank_name, account_name,
		       status, processed_at, note, created_at, updated_at
		FROM withdrawal_requests
		WHERE factory_id = $1
		ORDER BY created_at DESC
	`, factoryID)
	return items, err
}

func (r *WithdrawalRepository) UpdateStatus(requestID int64, status string, note *string) error {
	res, err := r.db.Exec(`
		UPDATE withdrawal_requests
		SET status = $1,
		    note = COALESCE($2, note),
		    processed_at = CASE WHEN $1 IN ('AP','RJ','CP') THEN NOW() ELSE processed_at END,
		    updated_at = NOW()
		WHERE request_id = $3
	`, status, note, requestID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeductFunds deducts amount from wallet pending_fund (called when withdrawal is approved).
func (r *WithdrawalRepository) DeductFunds(walletID int64, amount float64) error {
	_, err := r.db.Exec(`
		UPDATE wallets SET good_fund = good_fund - $1 WHERE wallet_id = $2
	`, amount, walletID)
	return err
}

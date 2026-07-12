package wallet

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

var ErrWithdrawalHoldFailed = errors.New("insufficient wallet funds for withdrawal")
var ErrWithdrawalAlreadyProcessed = errors.New("withdrawal request already processed")

type WithdrawalRepository struct {
	db *sqlx.DB
}

func NewWithdrawalRepository(db *sqlx.DB) *WithdrawalRepository {
	return &WithdrawalRepository{db: db}
}

// Create inserts a withdrawal request and atomically holds the amount
// (good_fund → pending_fund). The balance guard is in the UPDATE itself,
// so concurrent requests cannot overdraw.
func (r *WithdrawalRepository) Create(w *domain.WithdrawalRequest) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE wallets
		SET good_fund = good_fund - $1, pending_fund = pending_fund + $1
		WHERE wallet_id = $2 AND good_fund >= $1
	`, w.Amount, w.WalletID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrWithdrawalHoldFailed
	}

	if err := tx.QueryRow(`
		INSERT INTO withdrawal_requests
		    (wallet_id, factory_id, amount, bank_account_no, bank_name, account_name, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'PE')
		RETURNING request_id, created_at, updated_at
	`, w.WalletID, w.FactoryID, w.Amount, w.BankAccountNo, w.BankName, w.AccountName).
		Scan(&w.RequestID, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *WithdrawalRepository) ListByFactoryID(factoryID int64) ([]domain.WithdrawalRequest, error) {
	var items []domain.WithdrawalRequest
	err := r.db.Select(&items, `
		SELECT request_id, wallet_id, factory_id, amount, bank_account_no, bank_name, account_name,
		       status, processed_at, note, slip_url, created_at, updated_at
		FROM withdrawal_requests
		WHERE factory_id = $1
		ORDER BY created_at DESC
	`, factoryID)
	return items, err
}

// UpdateStatus transitions a withdrawal request and moves the held funds:
//   - CP (complete): superadmin transferred the money — deduct pending_fund,
//     store slip_url + processed_by, and record a WD transaction.
//   - RJ (rejected): release the hold — pending_fund → good_fund.
//   - AP (approved): status only, funds stay held.
//
// Only PE/AP requests can transition; a CP/RJ request is final.
func (r *WithdrawalRepository) UpdateStatus(requestID int64, status string, note *string, slipURL *string, processedBy *int64) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var row struct {
		WalletID int64   `db:"wallet_id"`
		Amount   float64 `db:"amount"`
		Status   string  `db:"status"`
	}
	if err := tx.Get(&row, `
		SELECT wallet_id, amount::float8 AS amount, TRIM(status) AS status
		FROM withdrawal_requests WHERE request_id = $1 FOR UPDATE
	`, requestID); err != nil {
		return err
	}
	if row.Status == "CP" || row.Status == "RJ" {
		return ErrWithdrawalAlreadyProcessed
	}

	switch status {
	case "CP":
		if _, err := tx.Exec(`
			UPDATE wallets SET pending_fund = GREATEST(pending_fund - $1, 0) WHERE wallet_id = $2
		`, row.Amount, row.WalletID); err != nil {
			return err
		}
		if _, err := tx.Exec(`
			INSERT INTO transactions (wallet_id, type, status, amount, slip_url)
			VALUES ($1, 'WD', 'ST', $2, $3)
		`, row.WalletID, row.Amount, slipURL); err != nil {
			return err
		}
	case "RJ":
		if _, err := tx.Exec(`
			UPDATE wallets
			SET pending_fund = GREATEST(pending_fund - $1, 0), good_fund = good_fund + $1
			WHERE wallet_id = $2
		`, row.Amount, row.WalletID); err != nil {
			return err
		}
	}

	res, err := tx.Exec(`
		UPDATE withdrawal_requests
		SET status = $1,
		    note = COALESCE($2, note),
		    slip_url = COALESCE($3, slip_url),
		    processed_by = COALESCE($4, processed_by),
		    processed_at = CASE WHEN $1 IN ('AP','RJ','CP') THEN NOW() ELSE processed_at END,
		    updated_at = NOW()
		WHERE request_id = $5
	`, status, note, slipURL, processedBy, requestID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}

	return tx.Commit()
}

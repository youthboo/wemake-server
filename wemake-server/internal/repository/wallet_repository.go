package repository

import (
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type WalletRepository struct {
	db *sqlx.DB
}

func NewWalletRepository(db *sqlx.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

func (r *WalletRepository) GetByUserID(userID int64) (*domain.Wallet, error) {
	var wallet domain.Wallet
	query := "SELECT wallet_id, user_id, good_fund, pending_fund FROM wallets WHERE user_id = $1"
	if err := r.db.Get(&wallet, query, userID); err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *WalletRepository) GetWalletIDByUserID(userID int64) (*int64, error) {
	var walletID int64
	if err := r.db.Get(&walletID, "SELECT wallet_id FROM wallets WHERE user_id = $1", userID); err != nil {
		return nil, err
	}
	return &walletID, nil
}

// GetByUserIDForUpdate loads the wallet row with FOR UPDATE (must be called inside a transaction).
func (r *WalletRepository) GetByUserIDForUpdate(tx *sqlx.Tx, userID int64) (*domain.Wallet, error) {
	var wallet domain.Wallet
	query := `SELECT wallet_id, user_id, good_fund, pending_fund FROM wallets WHERE user_id = $1 FOR UPDATE`
	if err := tx.Get(&wallet, query, userID); err != nil {
		return nil, err
	}
	return &wallet, nil
}

// EnsureWallet returns wallet_id for user_id, inserting a zero row if missing (inside tx).
func (r *WalletRepository) EnsureWallet(tx *sqlx.Tx, userID int64) (int64, error) {
	var id int64
	err := tx.Get(&id, `SELECT wallet_id FROM wallets WHERE user_id = $1`, userID)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	err = tx.QueryRow(`
		INSERT INTO wallets (user_id, good_fund, pending_fund)
		VALUES ($1, 0, 0)
		RETURNING wallet_id
	`, userID).Scan(&id)
	return id, err
}

// DebitGoodFund subtracts amount if good_fund is sufficient. Returns false if balance too low.
func (r *WalletRepository) DebitGoodFund(tx *sqlx.Tx, walletID int64, amount float64) (bool, error) {
	if amount <= 0 {
		return false, nil
	}
	res, err := tx.Exec(`
		UPDATE wallets
		SET good_fund = good_fund - $1
		WHERE wallet_id = $2 AND good_fund >= $1
	`, amount, walletID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

// CreditGoodFund adds amount to good_fund.
func (r *WalletRepository) CreditGoodFund(tx *sqlx.Tx, walletID int64, amount float64) error {
	if amount <= 0 {
		return nil
	}
	_, err := tx.Exec(`
		UPDATE wallets SET good_fund = good_fund + $1 WHERE wallet_id = $2
	`, amount, walletID)
	return err
}

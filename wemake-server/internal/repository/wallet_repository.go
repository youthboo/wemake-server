package repository

import (
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

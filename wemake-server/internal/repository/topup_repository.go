package repository

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type TopupRepository struct {
	db *sqlx.DB
}

func NewTopupRepository(db *sqlx.DB) *TopupRepository {
	return &TopupRepository{db: db}
}

func (r *TopupRepository) Create(t *domain.TopupIntent) error {
	return r.db.QueryRow(`
		INSERT INTO topup_intents (wallet_id, amount, qr_payload, status, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING intent_id, created_at
	`, t.WalletID, t.Amount, t.QRPayload, t.Status, t.ExpiresAt).
		Scan(&t.IntentID, &t.CreatedAt)
}

func (r *TopupRepository) GetByID(intentID string) (*domain.TopupIntent, error) {
	var item domain.TopupIntent
	err := r.db.Get(&item, `
		SELECT intent_id, wallet_id, amount, qr_payload, status, expires_at, confirmed_at, created_at
		FROM topup_intents
		WHERE intent_id = $1
	`, intentID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *TopupRepository) Confirm(intentID string) (*domain.TopupIntent, error) {
	var item domain.TopupIntent
	now := time.Now()
	err := r.db.QueryRow(`
		UPDATE topup_intents
		SET status = 'CP', confirmed_at = $1
		WHERE intent_id = $2 AND status = 'PE'
		RETURNING intent_id, wallet_id, amount, qr_payload, status, expires_at, confirmed_at, created_at
	`, now, intentID).Scan(
		&item.IntentID, &item.WalletID, &item.Amount, &item.QRPayload,
		&item.Status, &item.ExpiresAt, &item.ConfirmedAt, &item.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &item, nil
}

// AddFunds adds confirmed topup amount to wallet good_fund.
func (r *TopupRepository) AddFunds(walletID int64, amount float64) error {
	_, err := r.db.Exec(`
		UPDATE wallets SET good_fund = good_fund + $1 WHERE wallet_id = $2
	`, amount, walletID)
	return err
}

package repository

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type PlatformConfigRepository struct {
	db *sqlx.DB
}

func NewPlatformConfigRepository(db *sqlx.DB) *PlatformConfigRepository {
	return &PlatformConfigRepository{db: db}
}

func (r *PlatformConfigRepository) GetActive() (*domain.PlatformConfig, error) {
	var item domain.PlatformConfig
	err := r.db.Get(&item, `
		SELECT config_id, default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
		       promo_label, vat_rate, currency_code, effective_from, effective_to, created_by, created_at
		FROM platform_config
		WHERE effective_to IS NULL
		ORDER BY effective_from DESC, config_id DESC
		LIMIT 1
	`)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PlatformConfigRepository) ListHistory() ([]domain.PlatformConfig, error) {
	var items []domain.PlatformConfig
	err := r.db.Select(&items, `
		SELECT config_id, default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
		       promo_label, vat_rate, currency_code, effective_from, effective_to, created_by, created_at
		FROM platform_config
		ORDER BY effective_from DESC, config_id DESC
	`)
	return items, err
}

func (r *PlatformConfigRepository) Create(tx *sqlx.Tx, cfg *domain.PlatformConfig) error {
	return tx.QueryRow(`
		INSERT INTO platform_config (
			default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
			promo_label, vat_rate, currency_code, effective_from, effective_to, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING config_id, created_at
	`,
		cfg.DefaultCommissionRate,
		cfg.PromoCommissionRate,
		nullableTimeValue(cfg.PromoStartAt),
		nullableTimeValue(cfg.PromoEndAt),
		nullableStringPtr(cfg.PromoLabel),
		cfg.VatRate,
		cfg.CurrencyCode,
		cfg.EffectiveFrom,
		nullableTimeValue(cfg.EffectiveTo),
		nullableInt64Value(cfg.CreatedBy),
	).Scan(&cfg.ConfigID, &cfg.CreatedAt)
}

func (r *PlatformConfigRepository) CloseActive(tx *sqlx.Tx) error {
	_, err := tx.Exec(`
		UPDATE platform_config
		SET effective_to = $1
		WHERE effective_to IS NULL
	`, time.Now().UTC())
	return err
}

func (r *PlatformConfigRepository) GetByID(id int64) (*domain.PlatformConfig, error) {
	var item domain.PlatformConfig
	err := r.db.Get(&item, `
		SELECT config_id, default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
		       promo_label, vat_rate, currency_code, effective_from, effective_to, created_by, created_at
		FROM platform_config
		WHERE config_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

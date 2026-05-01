package repository

import (
	"database/sql"
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
		SELECT config_id, label, default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
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
		SELECT config_id, label, default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
		       promo_label, vat_rate, currency_code, effective_from, effective_to, created_by, created_at
		FROM platform_config
		ORDER BY effective_from DESC, config_id DESC
	`)
	return items, err
}

func (r *PlatformConfigRepository) ListAll() ([]domain.PlatformConfig, error) {
	var items []domain.PlatformConfig
	err := r.db.Select(&items, `
		SELECT config_id, label, default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
		       promo_label, vat_rate, currency_code, effective_from, effective_to, created_by, created_at
		FROM platform_config
		ORDER BY config_id ASC
	`)
	return items, err
}

func (r *PlatformConfigRepository) Create(tx *sqlx.Tx, cfg *domain.PlatformConfig) error {
	return tx.QueryRow(`
		INSERT INTO platform_config (
			label, default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
			promo_label, vat_rate, currency_code, effective_from, effective_to, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING config_id, created_at
	`,
		nullableStringPtr(cfg.Label),
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

func (r *PlatformConfigRepository) CreatePackage(tx *sqlx.Tx, cfg *domain.PlatformConfig) error {
	return tx.Get(cfg, `
		INSERT INTO platform_config (
			label, default_commission_rate, vat_rate, currency_code, effective_from, effective_to, created_by
		) VALUES ($1, $2, $3, $4, NOW(), $5, $6)
		RETURNING config_id, label, default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
		          promo_label, vat_rate, currency_code, effective_from, effective_to, created_by, created_at
	`,
		nullableStringPtr(cfg.Label),
		cfg.DefaultCommissionRate,
		cfg.VatRate,
		cfg.CurrencyCode,
		nullableTimeValue(cfg.EffectiveTo),
		nullableInt64Value(cfg.CreatedBy),
	)
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
		SELECT config_id, label, default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
		       promo_label, vat_rate, currency_code, effective_from, effective_to, created_by, created_at
		FROM platform_config
		WHERE config_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PlatformConfigRepository) GetDefault() (*domain.PlatformConfig, error) {
	var item domain.PlatformConfig
	err := r.db.Get(&item, `
		SELECT config_id, label, default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
		       promo_label, vat_rate, currency_code, effective_from, effective_to, created_by, created_at
		FROM platform_config
		ORDER BY config_id ASC
		LIMIT 1
	`)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PlatformConfigRepository) UpdateConfig(tx *sqlx.Tx, id int64, cfg *domain.PlatformConfig) error {
	return tx.Get(cfg, `
		UPDATE platform_config
		SET label = $1,
		    default_commission_rate = $2,
		    vat_rate = $3
		WHERE config_id = $4
		RETURNING config_id, label, default_commission_rate, promo_commission_rate, promo_start_at, promo_end_at,
		          promo_label, vat_rate, currency_code, effective_from, effective_to, created_by, created_at
	`, nullableStringPtr(cfg.Label), cfg.DefaultCommissionRate, cfg.VatRate, id)
}

func (r *PlatformConfigRepository) CountFactoriesUsingConfig(configID int64) (int, error) {
	var count int
	err := r.db.Get(&count, `
		SELECT COUNT(*)
		FROM factory_profiles
		WHERE config_id = $1
	`, configID)
	return count, err
}

func (r *PlatformConfigRepository) DeleteConfig(configID int64) error {
	res, err := r.db.Exec(`DELETE FROM platform_config WHERE config_id = $1`, configID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *PlatformConfigRepository) GetByFactoryID(factoryID int64) (*domain.PlatformConfig, error) {
	var cfg domain.PlatformConfig
	err := r.db.Get(&cfg, `
		SELECT
			COALESCE(pc.config_id, pc_default.config_id) AS config_id,
			COALESCE(pc.label, pc_default.label, 'มาตรฐาน') AS label,
			COALESCE(pc.default_commission_rate, pc_default.default_commission_rate) AS default_commission_rate,
			pc.promo_commission_rate,
			pc.promo_start_at,
			pc.promo_end_at,
			pc.promo_label,
			COALESCE(pc.vat_rate, pc_default.vat_rate) AS vat_rate,
			COALESCE(pc.currency_code, pc_default.currency_code) AS currency_code,
			COALESCE(pc.effective_from, pc_default.effective_from) AS effective_from,
			COALESCE(pc.effective_to, pc_default.effective_to) AS effective_to,
			COALESCE(pc.created_by, pc_default.created_by) AS created_by,
			COALESCE(pc.created_at, pc_default.created_at) AS created_at
		FROM factory_profiles fp
		LEFT JOIN platform_config pc ON pc.config_id = fp.config_id
		CROSS JOIN (
			SELECT config_id, label, default_commission_rate, vat_rate, currency_code,
			       effective_from, effective_to, created_by, created_at
			FROM platform_config
			ORDER BY config_id ASC
			LIMIT 1
		) pc_default
		WHERE fp.user_id = $1
	`, factoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			return r.GetDefault()
		}
		return nil, err
	}
	return &cfg, nil
}

func (r *PlatformConfigRepository) GetFactoryConfig(factoryID int64) (*domain.FactoryConfigResponse, error) {
	var item domain.FactoryConfigResponse
	err := r.db.Get(&item, `
		SELECT
			fp.user_id AS factory_id,
			COALESCE(pc.config_id, pc_default.config_id) AS config_id,
			COALESCE(pc.label, pc_default.label, 'มาตรฐาน') AS label,
			COALESCE(pc.default_commission_rate, pc_default.default_commission_rate) AS default_commission_rate,
			COALESCE(pc.vat_rate, pc_default.vat_rate) AS vat_rate
		FROM factory_profiles fp
		LEFT JOIN platform_config pc ON pc.config_id = fp.config_id
		CROSS JOIN (
			SELECT config_id, label, default_commission_rate, vat_rate
			FROM platform_config
			ORDER BY config_id ASC
			LIMIT 1
		) pc_default
		WHERE fp.user_id = $1
	`, factoryID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *PlatformConfigRepository) GetFactoryAssignedConfigID(factoryID int64) (*int64, error) {
	var id sql.NullInt64
	err := r.db.Get(&id, `SELECT config_id FROM factory_profiles WHERE user_id = $1`, factoryID)
	if err != nil {
		return nil, err
	}
	if !id.Valid {
		return nil, nil
	}
	return &id.Int64, nil
}

func (r *PlatformConfigRepository) AssignFactoryConfig(factoryID, configID int64) error {
	res, err := r.db.Exec(`
		UPDATE factory_profiles
		SET config_id = $1
		WHERE user_id = $2
	`, configID, factoryID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

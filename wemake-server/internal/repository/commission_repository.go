package repository

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type CommissionRepository struct {
	db *sqlx.DB
}

func NewCommissionRepository(db *sqlx.DB) *CommissionRepository {
	return &CommissionRepository{db: db}
}

func (r *CommissionRepository) ListRules(factoryID *int64, activeOnly bool) ([]domain.CommissionRule, error) {
	items := []domain.CommissionRule{}
	query := `
		SELECT cr.rule_id, cr.factory_id, fp.factory_name, cr.rate_percent, cr.effective_from, cr.effective_to, cr.note, cr.created_by, cr.created_at
		FROM commission_rules cr
		LEFT JOIN factory_profiles fp ON fp.user_id = cr.factory_id
		WHERE 1=1
	`
	args := []interface{}{}
	if factoryID != nil {
		args = append(args, *factoryID)
		query += " AND cr.factory_id = $" + "1"
	}
	if activeOnly {
		query += " AND cr.effective_to IS NULL"
	}
	query += " ORDER BY cr.created_at DESC, cr.rule_id DESC"
	if err := r.db.Select(&items, query, args...); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *CommissionRepository) CreateRule(rule *domain.CommissionRule) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if rule.FactoryID != nil {
		if _, err := tx.Exec(`
			UPDATE commission_rules
			SET effective_to = NOW()
			WHERE factory_id = $1 AND effective_to IS NULL
		`, *rule.FactoryID); err != nil {
			return err
		}
	}

	if err := tx.QueryRow(`
		INSERT INTO commission_rules (factory_id, rate_percent, effective_from, effective_to, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING rule_id, created_at
	`, nullableInt64Value(rule.FactoryID), rule.RatePercent, rule.EffectiveFrom, nullableTimeValue(rule.EffectiveTo), nullableStringPtr(rule.Note), rule.CreatedBy).Scan(&rule.RuleID, &rule.CreatedAt); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *CommissionRepository) DeactivateRule(ruleID int64) (*domain.CommissionRule, error) {
	var item domain.CommissionRule
	if err := r.db.Get(&item, `
		UPDATE commission_rules
		SET effective_to = NOW()
		WHERE rule_id = $1
		RETURNING rule_id, factory_id, rate_percent, effective_from, effective_to, note, created_by, created_at
	`, ruleID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CommissionRepository) ListExemptions(activeOnly bool) ([]domain.CommissionExemption, error) {
	items := []domain.CommissionExemption{}
	query := `
		SELECT
			e.exemption_id,
			e.factory_id,
			fp.factory_name,
			e.reason,
			e.expires_at,
			e.created_by,
			e.revoked_by,
			e.revoked_at,
			e.created_at,
			(e.revoked_at IS NULL AND (e.expires_at IS NULL OR e.expires_at > NOW())) AS is_active
		FROM factory_commission_exemptions e
		LEFT JOIN factory_profiles fp ON fp.user_id = e.factory_id
	`
	if activeOnly {
		query += ` WHERE e.revoked_at IS NULL AND (e.expires_at IS NULL OR e.expires_at > NOW())`
	}
	query += ` ORDER BY e.created_at DESC, e.exemption_id DESC`
	if err := r.db.Select(&items, query); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *CommissionRepository) CreateExemption(item *domain.CommissionExemption) error {
	if err := r.db.QueryRow(`
		INSERT INTO factory_commission_exemptions (factory_id, reason, expires_at, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING exemption_id, created_at
	`, item.FactoryID, item.Reason, nullableTimeValue(item.ExpiresAt), item.CreatedBy).Scan(&item.ExemptionID, &item.CreatedAt); err != nil {
		return err
	}
	return nil
}

func (r *CommissionRepository) RevokeExemption(exemptionID, actorID int64) (*domain.CommissionExemption, error) {
	var item domain.CommissionExemption
	if err := r.db.Get(&item, `
		UPDATE factory_commission_exemptions
		SET revoked_at = NOW(), revoked_by = $2
		WHERE exemption_id = $1
		RETURNING exemption_id, factory_id, reason, expires_at, created_by, revoked_by, revoked_at, created_at,
		          (revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW())) AS is_active
	`, exemptionID, actorID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *CommissionRepository) GetActiveRuleForFactory(factoryID int64) (*domain.CommissionRule, error) {
	var item domain.CommissionRule
	err := r.db.Get(&item, `
		SELECT rule_id, factory_id, rate_percent, effective_from, effective_to, note, created_by, created_at
		FROM commission_rules
		WHERE factory_id = $1
		  AND effective_to IS NULL
		  AND effective_from <= NOW()
		ORDER BY effective_from DESC, rule_id DESC
		LIMIT 1
	`, factoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *CommissionRepository) GetActiveExemptionForFactory(factoryID int64) (*domain.CommissionExemption, error) {
	var item domain.CommissionExemption
	err := r.db.Get(&item, `
		SELECT exemption_id, factory_id, reason, expires_at, created_by, revoked_by, revoked_at, created_at,
		       TRUE AS is_active
		FROM factory_commission_exemptions
		WHERE factory_id = $1
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
		LIMIT 1
	`, factoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *CommissionRepository) FactoryHasActiveExemption(factoryID int64) (bool, error) {
	var ok bool
	err := r.db.Get(&ok, `
		SELECT EXISTS(
			SELECT 1
			FROM factory_commission_exemptions
			WHERE factory_id = $1
			  AND revoked_at IS NULL
			  AND (expires_at IS NULL OR expires_at > NOW())
		)
	`, factoryID)
	return ok, err
}

func (r *CommissionRepository) FactoryHasActiveExemptionConflict(factoryID int64) (bool, error) {
	return r.FactoryHasActiveExemption(factoryID)
}

func (r *CommissionRepository) ActiveExemptionConflict(factoryID int64) (bool, error) {
	return r.FactoryHasActiveExemption(factoryID)
}

func (r *CommissionRepository) ActiveExemptionExists(factoryID int64) (bool, error) {
	return r.FactoryHasActiveExemption(factoryID)
}

func (r *CommissionRepository) Now() time.Time {
	return time.Now().UTC()
}

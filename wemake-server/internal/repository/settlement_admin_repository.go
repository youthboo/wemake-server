package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type SettlementAdminRepository struct {
	db *sqlx.DB
}

func NewSettlementAdminRepository(db *sqlx.DB) *SettlementAdminRepository {
	return &SettlementAdminRepository{db: db}
}

func (r *SettlementAdminRepository) ListByFactory(factoryID int64, limit, offset int) ([]domain.AdminSettlementListItem, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := r.db.Get(&total, `SELECT COUNT(*) FROM settlements WHERE factory_id = $1`, factoryID); err != nil {
		return nil, 0, err
	}

	var items []domain.AdminSettlementListItem
	if err := r.db.Select(&items, `
		SELECT
			settlement_id,
			factory_id,
			order_id,
			amount,
			status,
			created_at::text AS created_at,
			COALESCE(updated_at::text, '') AS updated_at
		FROM settlements
		WHERE factory_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, factoryID, limit, offset); err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []domain.AdminSettlementListItem{}
	}
	return items, total, nil
}

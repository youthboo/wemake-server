package repository

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type SettlementRepository struct {
	db *sqlx.DB
}

func NewSettlementRepository(db *sqlx.DB) *SettlementRepository {
	return &SettlementRepository{db: db}
}

func (r *SettlementRepository) ListByFactoryID(factoryID int64) ([]domain.Settlement, error) {
	var items []domain.Settlement
	err := r.db.Select(&items, `
		SELECT settlement_id, factory_id, order_id, amount, status, settled_at, note, created_at, updated_at
		FROM settlements
		WHERE factory_id = $1
		ORDER BY created_at DESC
	`, factoryID)
	return items, err
}

func (r *SettlementRepository) GetByID(settlementID, factoryID int64) (*domain.Settlement, error) {
	var item domain.Settlement
	err := r.db.Get(&item, `
		SELECT settlement_id, factory_id, order_id, amount, status, settled_at, note, created_at, updated_at
		FROM settlements
		WHERE settlement_id = $1 AND factory_id = $2
	`, settlementID, factoryID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *SettlementRepository) Create(s *domain.Settlement) error {
	return r.db.QueryRow(`
		INSERT INTO settlements (factory_id, order_id, amount, status, note)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING settlement_id, created_at, updated_at
	`, s.FactoryID, s.OrderID, s.Amount, s.Status, s.Note).
		Scan(&s.SettlementID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *SettlementRepository) UpdateStatus(settlementID int64, status string) error {
	res, err := r.db.Exec(`
		UPDATE settlements
		SET status = $1, updated_at = NOW(),
		    settled_at = CASE WHEN $1 = 'CP' THEN NOW() ELSE settled_at END
		WHERE settlement_id = $2
	`, status, settlementID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

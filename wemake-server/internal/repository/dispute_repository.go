package repository

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type DisputeRepository struct {
	db *sqlx.DB
}

func NewDisputeRepository(db *sqlx.DB) *DisputeRepository {
	return &DisputeRepository{db: db}
}

func (r *DisputeRepository) Create(d *domain.Dispute) error {
	return r.db.QueryRow(`
		INSERT INTO disputes (order_id, opened_by, reason, status)
		VALUES ($1, $2, $3, 'OP')
		RETURNING dispute_id, created_at, updated_at
	`, d.OrderID, d.OpenedBy, d.Reason).
		Scan(&d.DisputeID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *DisputeRepository) GetByOrderID(orderID int64) (*domain.Dispute, error) {
	var item domain.Dispute
	err := r.db.Get(&item, `
		SELECT dispute_id, order_id, opened_by, reason, status, resolution, resolved_at, created_at, updated_at
		FROM disputes
		WHERE order_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, orderID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DisputeRepository) GetByID(disputeID int64) (*domain.Dispute, error) {
	var item domain.Dispute
	err := r.db.Get(&item, `
		SELECT dispute_id, order_id, opened_by, reason, status, resolution, resolved_at, created_at, updated_at
		FROM disputes
		WHERE dispute_id = $1
	`, disputeID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *DisputeRepository) UpdateStatus(disputeID int64, status string, resolution *string) error {
	res, err := r.db.Exec(`
		UPDATE disputes
		SET status = $1,
		    resolution = COALESCE($2, resolution),
		    resolved_at = CASE WHEN $1 IN ('RS','CL') THEN NOW() ELSE resolved_at END,
		    updated_at = NOW()
		WHERE dispute_id = $3
	`, status, resolution, disputeID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

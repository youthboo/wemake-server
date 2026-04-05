package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type ProductionRepository struct {
	db *sqlx.DB
}

func NewProductionRepository(db *sqlx.DB) *ProductionRepository {
	return &ProductionRepository{db: db}
}

func (r *ProductionRepository) Create(item *domain.ProductionUpdate) error {
	if item.Status == "" {
		item.Status = "CR"
	}
	if item.UpdateDate.IsZero() {
		item.UpdateDate = item.CreatedAt
	}
	query := `
		INSERT INTO production_updates (order_id, step_id, status, description, image_url, created_at, update_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING update_id
	`
	return r.db.QueryRow(
		query,
		item.OrderID,
		item.StepID,
		item.Status,
		item.Description,
		item.ImageURL,
		item.CreatedAt,
		item.UpdateDate,
	).Scan(&item.UpdateID)
}

func (r *ProductionRepository) ListByOrderID(orderID int64) ([]domain.ProductionUpdate, error) {
	var items []domain.ProductionUpdate
	query := `
		SELECT update_id, order_id, step_id, status, description, image_url, created_at, update_date
		FROM production_updates
		WHERE order_id = $1
		ORDER BY created_at ASC
	`
	err := r.db.Select(&items, query, orderID)
	return items, err
}

func (r *ProductionRepository) Patch(updateID int64, description *string, imageURL *string) error {
	query := `
		UPDATE production_updates
		SET description = COALESCE($1, description),
		    image_url = COALESCE($2, image_url)
		WHERE update_id = $3
	`
	_, err := r.db.Exec(query, description, imageURL, updateID)
	return err
}

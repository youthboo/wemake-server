package repository

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type RFQRepository struct {
	db *sqlx.DB
}

func NewRFQRepository(db *sqlx.DB) *RFQRepository {
	return &RFQRepository{db: db}
}

func (r *RFQRepository) Create(rfq *domain.RFQ) error {
	query := `
		INSERT INTO rfqs (user_id, category_id, sub_category_id, title, quantity, unit_id, budget_per_piece, details, address_id, shipping_method_id, status, deadline_date, uploaded_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING rfq_id
	`
	return r.db.QueryRow(
		query,
		rfq.UserID,
		rfq.CategoryID,
		nullableInt64Value(rfq.SubCategoryID),
		rfq.Title,
		rfq.Quantity,
		rfq.UnitID,
		rfq.BudgetPerPiece,
		rfq.Details,
		rfq.AddressID,
		nullableInt64Value(rfq.ShippingMethodID),
		rfq.Status,
		nullableTimeValue(rfq.DeadlineDate),
		nullableTimeValue(rfq.UploadedAt),
		rfq.CreatedAt,
		rfq.UpdatedAt,
	).Scan(&rfq.RFQID)
}

func (r *RFQRepository) ListByUserID(userID int64, status string) ([]domain.RFQ, error) {
	var rfqs []domain.RFQ
	query := `
		SELECT rfq_id, user_id, category_id, sub_category_id, title, quantity, unit_id, budget_per_piece, details, address_id, shipping_method_id, status, deadline_date, uploaded_at, created_at, updated_at
		FROM rfqs
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	err := r.db.Select(&rfqs, query, args...)
	return rfqs, err
}

func (r *RFQRepository) GetByID(userID, rfqID int64) (*domain.RFQ, error) {
	var rfq domain.RFQ
	query := `
		SELECT rfq_id, user_id, category_id, sub_category_id, title, quantity, unit_id, budget_per_piece, details, address_id, shipping_method_id, status, deadline_date, uploaded_at, created_at, updated_at
		FROM rfqs
		WHERE user_id = $1 AND rfq_id = $2
	`
	if err := r.db.Get(&rfq, query, userID, rfqID); err != nil {
		return nil, err
	}
	return &rfq, nil
}

func (r *RFQRepository) Cancel(userID, rfqID int64) error {
	query := "UPDATE rfqs SET status = 'CC', updated_at = NOW() WHERE user_id = $1 AND rfq_id = $2"
	_, err := r.db.Exec(query, userID, rfqID)
	return err
}

func (r *RFQRepository) CreateImage(image *domain.RFQImage) error {
	query := "INSERT INTO rfq_images (image_id, rfq_id, image_url) VALUES ($1, $2, $3)"
	_, err := r.db.Exec(query, image.ImageID, image.RFQID, image.ImageURL)
	return err
}

func (r *RFQRepository) CountImages(rfqID int64) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM rfq_images WHERE rfq_id = $1"
	err := r.db.Get(&count, query, rfqID)
	return count, err
}

func (r *RFQRepository) ListImages(rfqID int64) ([]domain.RFQImage, error) {
	var images []domain.RFQImage
	query := "SELECT image_id, rfq_id, image_url FROM rfq_images WHERE rfq_id = $1 ORDER BY image_id"
	err := r.db.Select(&images, query, rfqID)
	return images, err
}

func (r *RFQRepository) SubCategoryBelongsToCategory(subCategoryID, categoryID int64) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM lbi_sub_categories
			WHERE sub_category_id = $1
				AND category_id = $2
				AND status = '1'
		)
	`
	err := r.db.Get(&exists, query, subCategoryID, categoryID)
	return exists, err
}

func (r *RFQRepository) ShippingMethodExists(shippingMethodID int64) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM lbi_shipping_methods
			WHERE shipping_method_id = $1
				AND status = '1'
		)
	`
	err := r.db.Get(&exists, query, shippingMethodID)
	return exists, err
}

func nullableInt64Value(value *int64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTimeValue(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

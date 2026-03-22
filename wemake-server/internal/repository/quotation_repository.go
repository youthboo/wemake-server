package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type QuotationRepository struct {
	db *sqlx.DB
}

func NewQuotationRepository(db *sqlx.DB) *QuotationRepository {
	return &QuotationRepository{db: db}
}

func (r *QuotationRepository) Create(item *domain.Quotation) error {
	query := `
		INSERT INTO quotations (rfq_id, factory_id, price_per_piece, mold_cost, lead_time_days, shipping_method_id, status, create_time, log_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING quote_id
	`
	return r.db.QueryRow(
		query,
		item.RFQID,
		item.FactoryID,
		item.PricePerPiece,
		item.MoldCost,
		item.LeadTimeDays,
		item.ShippingMethodID,
		item.Status,
		item.CreateTime,
		item.LogTimestamp,
	).Scan(&item.QuotationID)
}

func (r *QuotationRepository) ListByRFQID(rfqID int64) ([]domain.Quotation, error) {
	var items []domain.Quotation
	query := `
		SELECT quote_id, rfq_id, factory_id, price_per_piece, mold_cost, lead_time_days, shipping_method_id, status, create_time, log_timestamp
		FROM quotations
		WHERE rfq_id = $1
		ORDER BY create_time DESC
	`
	err := r.db.Select(&items, query, rfqID)
	return items, err
}

func (r *QuotationRepository) GetByID(quotationID int64) (*domain.Quotation, error) {
	var item domain.Quotation
	query := `
		SELECT quote_id, rfq_id, factory_id, price_per_piece, mold_cost, lead_time_days, shipping_method_id, status, create_time, log_timestamp
		FROM quotations
		WHERE quote_id = $1
	`
	if err := r.db.Get(&item, query, quotationID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *QuotationRepository) UpdateStatus(quotationID int64, status string) error {
	query := "UPDATE quotations SET status = $1, log_timestamp = NOW() WHERE quote_id = $2"
	_, err := r.db.Exec(query, status, quotationID)
	return err
}

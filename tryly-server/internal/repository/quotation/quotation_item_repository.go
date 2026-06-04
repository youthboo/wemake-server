package quotation

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type QuotationItemRepository struct {
	db *sqlx.DB
}

func NewQuotationItemRepository(db *sqlx.DB) *QuotationItemRepository {
	return &QuotationItemRepository{db: db}
}

func (r *QuotationItemRepository) ListByQuotation(_ int64) ([]domain.QuotationItem, error) {
	// quotation_items table removed — items live in quotation.items JSON column.
	return nil, nil
}

func (r *QuotationItemRepository) BulkInsert(tx *sqlx.Tx, qid int64, items []domain.QuotationItem) error {
	// quotation_items table has been removed — items are embedded in the quotation JSON.
	return nil
}

func (r *QuotationItemRepository) DeleteByQuotation(_ *sqlx.Tx, _ int64) error {
	return nil
}

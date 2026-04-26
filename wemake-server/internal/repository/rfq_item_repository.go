package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/yourusername/wemake/internal/domain"
)

type RFQItemRepository struct {
	db *sqlx.DB
}

func NewRFQItemRepository(db *sqlx.DB) *RFQItemRepository {
	return &RFQItemRepository{db: db}
}

func (r *RFQItemRepository) ListByRFQID(rfqID int64) ([]domain.RFQItem, error) {
	var items []domain.RFQItem
	err := r.db.Select(&items, `
		SELECT item_id, rfq_id, item_no, description, specification, qty::float8 AS qty, unit,
		       unit_price::float8 AS unit_price, discount_pct::float8 AS discount_pct,
		       line_total::float8 AS line_total, note, created_at
		FROM rfq_items
		WHERE rfq_id = $1
		ORDER BY item_no ASC, item_id ASC
	`, rfqID)
	return items, err
}

func (r *RFQItemRepository) BulkInsertTx(tx *sqlx.Tx, rfqID int64, items []domain.RFQItem) error {
	if len(items) == 0 {
		return nil
	}
	stmt, err := tx.Preparex(pq.CopyIn("rfq_items",
		"rfq_id", "item_no", "description", "specification", "qty", "unit", "unit_price", "discount_pct", "line_total", "note"))
	if err != nil {
		return err
	}
	for _, item := range items {
		if _, err := stmt.Exec(rfqID, item.ItemNo, item.Description, nullableStringPtr(item.Specification), item.Qty, nullableStringPtr(item.Unit), item.UnitPrice, item.DiscountPct, item.LineTotal, nullableStringPtr(item.Note)); err != nil {
			_ = stmt.Close()
			return err
		}
	}
	if _, err := stmt.Exec(); err != nil {
		_ = stmt.Close()
		return err
	}
	return stmt.Close()
}

func (r *RFQItemRepository) DeleteByRFQIDTx(tx *sqlx.Tx, rfqID int64) error {
	_, err := tx.Exec(`DELETE FROM rfq_items WHERE rfq_id = $1`, rfqID)
	return err
}

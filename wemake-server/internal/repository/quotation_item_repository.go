package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/yourusername/wemake/internal/domain"
)

type QuotationItemRepository struct {
	db *sqlx.DB
}

func NewQuotationItemRepository(db *sqlx.DB) *QuotationItemRepository {
	return &QuotationItemRepository{db: db}
}

func (r *QuotationItemRepository) ListByQuotation(qid int64) ([]domain.QuotationItem, error) {
	var items []domain.QuotationItem
	err := r.db.Select(&items, `
		SELECT item_id, quotation_id, item_no, description, qty::float8 AS qty, unit,
		       unit_price::float8 AS unit_price, discount_pct::float8 AS discount_pct,
		       line_total::float8 AS line_total, note, created_at
		FROM quotation_items
		WHERE quotation_id = $1
		ORDER BY item_no ASC, item_id ASC
	`, qid)
	return items, err
}

func (r *QuotationItemRepository) BulkInsert(tx *sqlx.Tx, qid int64, items []domain.QuotationItem) error {
	if len(items) == 0 {
		return nil
	}
	stmt, err := tx.Preparex(pq.CopyIn("quotation_items",
		"quotation_id", "item_no", "description", "qty", "unit", "unit_price", "discount_pct", "line_total", "note"))
	if err != nil {
		return err
	}
	for _, item := range items {
		if _, err := stmt.Exec(qid, item.ItemNo, item.Description, item.Qty, nullableStringPtr(item.Unit), item.UnitPrice, item.DiscountPct, item.LineTotal, nullableStringPtr(item.Note)); err != nil {
			stmt.Close()
			return err
		}
	}
	if _, err := stmt.Exec(); err != nil {
		stmt.Close()
		return err
	}
	return stmt.Close()
}

func (r *QuotationItemRepository) DeleteByQuotation(tx *sqlx.Tx, qid int64) error {
	_, err := tx.Exec(`DELETE FROM quotation_items WHERE quotation_id = $1`, qid)
	return err
}

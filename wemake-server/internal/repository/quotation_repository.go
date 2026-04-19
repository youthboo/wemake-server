package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type QuotationRepository struct {
	db *sqlx.DB
}

func NewQuotationRepository(db *sqlx.DB) *QuotationRepository {
	return &QuotationRepository{db: db}
}

func quotationSelectBase() string {
	return `SELECT quote_id, rfq_id, factory_id, price_per_piece, mold_cost, lead_time_days, shipping_method_id, status, create_time, log_timestamp,
		COALESCE(version, 1) AS version, COALESCE(is_locked, false) AS is_locked, last_edited_at, last_edited_by
		FROM quotations`
}

func (r *QuotationRepository) Create(item *domain.Quotation) error {
	query := `
		INSERT INTO quotations (rfq_id, factory_id, price_per_piece, mold_cost, lead_time_days, shipping_method_id, status, create_time, log_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING quote_id
	`
	if err := r.db.QueryRow(
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
	).Scan(&item.QuotationID); err != nil {
		return err
	}
	item.Version = 1
	item.IsLocked = false
	return nil
}

func (r *QuotationRepository) ListByRFQID(rfqID int64) ([]domain.Quotation, error) {
	var items []domain.Quotation
	query := quotationSelectBase() + `
		WHERE rfq_id = $1
		ORDER BY create_time DESC
	`
	err := r.db.Select(&items, query, rfqID)
	return items, err
}

func (r *QuotationRepository) ListByFactoryID(factoryID int64, status string) ([]domain.Quotation, error) {
	var items []domain.Quotation
	query := quotationSelectBase() + ` WHERE factory_id = $1`
	args := []interface{}{factoryID}
	statuses := splitCSVUpper(status)
	if len(statuses) == 1 {
		query += ` AND status = $2`
		args = append(args, statuses[0])
	} else if len(statuses) > 1 {
		placeholders := make([]string, 0, len(statuses))
		for _, st := range statuses {
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, st)
		}
		query += ` AND status IN (` + strings.Join(placeholders, ", ") + `)`
	}
	query += ` ORDER BY create_time DESC`
	err := r.db.Select(&items, query, args...)
	return items, err
}

func (r *QuotationRepository) GetByID(quotationID int64) (*domain.Quotation, error) {
	var item domain.Quotation
	query := quotationSelectBase() + `
		WHERE quote_id = $1
	`
	if err := r.db.Get(&item, query, quotationID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *QuotationRepository) UpdateStatus(quotationID int64, status string) error {
	query := `
		UPDATE quotations
		SET status = $1,
		    log_timestamp = NOW(),
		    is_locked = CASE WHEN $1 = 'AC' THEN TRUE ELSE COALESCE(is_locked, false) END
		WHERE quote_id = $2
	`
	_, err := r.db.Exec(query, status, quotationID)
	return err
}

// UpdateStatusTx updates quotation status inside a transaction (same rules as UpdateStatus).
func (r *QuotationRepository) UpdateStatusTx(tx *sqlx.Tx, quotationID int64, status string) error {
	query := `
		UPDATE quotations
		SET status = $1,
		    log_timestamp = NOW(),
		    is_locked = CASE WHEN $1 = 'AC' THEN TRUE ELSE COALESCE(is_locked, false) END
		WHERE quote_id = $2
	`
	_, err := tx.Exec(query, status, quotationID)
	return err
}

// RejectOtherPendingQuotationsTx sets status RJ for other PD quotations on the same RFQ (excluding acceptedQuoteID).
func (r *QuotationRepository) RejectOtherPendingQuotationsTx(tx *sqlx.Tx, rfqID, acceptedQuoteID int64) error {
	_, err := tx.Exec(`
		UPDATE quotations
		SET status = 'RJ',
		    log_timestamp = NOW(),
		    is_locked = COALESCE(is_locked, false)
		WHERE rfq_id = $1
		  AND quote_id <> $2
		  AND status = 'PD'
	`, rfqID, acceptedQuoteID)
	return err
}

func (r *QuotationRepository) InsertHistory(entry *domain.QuotationHistoryEntry) error {
	query := `
		INSERT INTO quotation_history (quote_id, event_type, version_after, price_per_piece, mold_cost, lead_time_days, shipping_method_id, status, reason, edited_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING history_id, created_at
	`
	return r.db.QueryRow(
		query,
		entry.QuoteID,
		entry.EventType,
		entry.VersionAfter,
		nullableFloat64(entry.PricePerPiece),
		nullableFloat64(entry.MoldCost),
		nullableInt64Ptr(entry.LeadTimeDays),
		nullableInt64Ptr(entry.ShippingMethodID),
		nullableStringPtr(entry.Status),
		nullableStringPtr(entry.Reason),
		nullableInt64Ptr(entry.EditedBy),
	).Scan(&entry.HistoryID, &entry.CreatedAt)
}

func nullableFloat64(f *float64) interface{} {
	if f == nil {
		return nil
	}
	return *f
}

func nullableStringPtr(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

func nullableInt64Ptr(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func splitCSVUpper(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(strings.ToUpper(part))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func (r *QuotationRepository) ListHistory(quoteID int64) ([]domain.QuotationHistoryEntry, error) {
	var items []domain.QuotationHistoryEntry
	query := `
		SELECT history_id, quote_id, event_type, version_after, price_per_piece, mold_cost, lead_time_days, shipping_method_id, status, reason, edited_by, created_at
		FROM quotation_history
		WHERE quote_id = $1
		ORDER BY created_at DESC
	`
	err := r.db.Select(&items, query, quoteID)
	return items, err
}

func (r *QuotationRepository) UpdateBody(quoteID int64, pricePerPiece float64, moldCost float64, leadTimeDays int64, shippingMethodID int64, editorID int64, newVersion int) error {
	query := `
		UPDATE quotations
		SET price_per_piece = $1, mold_cost = $2, lead_time_days = $3, shipping_method_id = $4,
		    version = $5, last_edited_at = NOW(), last_edited_by = $6, log_timestamp = NOW()
		WHERE quote_id = $7 AND COALESCE(is_locked, false) = false AND status = 'PD'
	`
	res, err := r.db.Exec(query, pricePerPiece, moldCost, leadTimeDays, shippingMethodID, newVersion, editorID, quoteID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *QuotationRepository) ShippingMethodValid(shippingMethodID int64) (bool, error) {
	var ok bool
	err := r.db.Get(&ok, `
		SELECT EXISTS (SELECT 1 FROM lbi_shipping_methods WHERE shipping_method_id = $1 AND status = '1')
	`, shippingMethodID)
	return ok, err
}

// SnapshotFromQuotation builds a history row from current quotation row (for CR on create).
func SnapshotFromQuotation(q *domain.Quotation, eventType string, reason *string, editedBy *int64) *domain.QuotationHistoryEntry {
	pp := q.PricePerPiece
	mc := q.MoldCost
	lt := q.LeadTimeDays
	sm := q.ShippingMethodID
	st := q.Status
	return &domain.QuotationHistoryEntry{
		QuoteID:          q.QuotationID,
		EventType:        eventType,
		VersionAfter:     q.Version,
		PricePerPiece:    &pp,
		MoldCost:         &mc,
		LeadTimeDays:     &lt,
		ShippingMethodID: &sm,
		Status:           &st,
		Reason:           reason,
		EditedBy:         editedBy,
	}
}

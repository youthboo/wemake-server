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
	return `SELECT q.quote_id, q.rfq_id, q.factory_id,
		NULLIF(TRIM(COALESCE(to_jsonb(u)->>'factory_name', CONCAT_WS(' ', to_jsonb(u)->>'first_name', to_jsonb(u)->>'last_name'))), '') AS factory_name,
		fp.image_url AS factory_logo_url,
		fp.rating::double precision AS factory_rating_avg,
		q.price_per_piece, q.mold_cost, q.lead_time_days, q.shipping_method_id,
		NULLIF(TRIM(COALESCE(to_jsonb(sm)->>'method_name', to_jsonb(sm)->>'name')), '') AS shipping_method_name,
		q.factory_highlight,
		q.status, q.create_time, q.log_timestamp,
		COALESCE(q.version, 1) AS version, COALESCE(q.is_locked, false) AS is_locked, q.last_edited_at, q.last_edited_by,
		q.subtotal, q.discount_amount, q.shipping_cost, q.shipping_method, q.packaging_cost, q.tooling_mold_cost,
		q.vat_rate, q.vat_amount, q.platform_commission_rate, q.platform_commission_amount, q.platform_config_id,
		q.grand_total, q.factory_net_receivable, q.production_start_date, q.delivery_date, q.incoterms, q.payment_terms,
		validity_days, COALESCE(q.valid_until, (q.create_time + (q.validity_days::text || ' day')::interval)::date) AS valid_until,
		q.warranty_period_months, COALESCE(q.revision_no, 1) AS revision_no, q.parent_quotation_id,
		COALESCE(q.image_urls::text, '[]') AS image_urls,
		NULL::text AS material_detail,
		NULL::text AS payment_condition,
		0::double precision AS sample_cost,
		'[]'::jsonb AS certifications
		FROM quotations q
		LEFT JOIN users u ON u.user_id = q.factory_id
		LEFT JOIN factory_profiles fp ON fp.user_id = q.factory_id
		LEFT JOIN lbi_shipping_methods sm ON sm.shipping_method_id = q.shipping_method_id`
}

func (r *QuotationRepository) Create(item *domain.Quotation) error {
	return r.createWithExecutor(r.db, item)
}

func (r *QuotationRepository) CreateTx(tx *sqlx.Tx, item *domain.Quotation) error {
	return r.createWithExecutor(tx, item)
}

type quotationExecutor interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

func (r *QuotationRepository) createWithExecutor(exec quotationExecutor, item *domain.Quotation) error {
	imageURLs := item.ImageURLs
	if imageURLs == nil {
		imageURLs = domain.StringArray{}
	}
	query := `
		INSERT INTO quotations (
			rfq_id, factory_id, price_per_piece, mold_cost, lead_time_days, shipping_method_id, factory_highlight, status, create_time, log_timestamp,
			subtotal, discount_amount, shipping_cost, shipping_method, packaging_cost, tooling_mold_cost,
			vat_rate, vat_amount, platform_commission_rate, platform_commission_amount, platform_config_id,
			grand_total, factory_net_receivable, production_start_date, delivery_date, incoterms, payment_terms,
			validity_days, valid_until, warranty_period_months, revision_no, parent_quotation_id, image_urls
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		        $11,$12,$13,$14,$15,$16,
		        $17,$18,$19,$20,$21,
		        $22,$23,$24,$25,$26,$27,
		        $28,$29,$30,$31,$32,$33)
		RETURNING quote_id
	`
	if err := exec.QueryRow(
		query,
		item.RFQID,
		item.FactoryID,
		item.PricePerPiece,
		item.MoldCost,
		item.LeadTimeDays,
		nullableInt64ForZero(item.ShippingMethodID),
		nullableStringPtr(item.FactoryHighlight),
		item.Status,
		item.CreateTime,
		item.LogTimestamp,
		item.Subtotal,
		item.DiscountAmount,
		item.ShippingCost,
		nullableStringPtr(item.ShippingMethod),
		item.PackagingCost,
		item.ToolingMoldCost,
		item.VatRate,
		item.VatAmount,
		item.PlatformCommissionRate,
		item.PlatformCommissionAmount,
		nullableInt64Value(item.PlatformConfigID),
		item.GrandTotal,
		item.FactoryNetReceivable,
		nullableTimeValue(item.ProductionStartDate),
		nullableTimeValue(item.DeliveryDate),
		nullableStringPtr(item.Incoterms),
		nullableStringPtr(item.PaymentTerms),
		item.ValidityDays,
		nullableTimeValue(item.ValidUntil),
		nullableIntValue(item.WarrantyPeriodMonths),
		item.RevisionNo,
		nullableInt64Value(item.ParentQuotationID),
		imageURLs,
	).Scan(&item.QuotationID); err != nil {
		return err
	}
	item.Version = 1
	item.IsLocked = false
	return nil
}

func (r *QuotationRepository) ListByRFQID(rfqID int64) ([]domain.Quotation, error) {
	var items []domain.Quotation
	query := `SELECT
		q.quote_id, q.rfq_id, q.factory_id,
		NULLIF(TRIM(COALESCE(fp.factory_name, CONCAT_WS(' ', to_jsonb(u)->>'first_name', to_jsonb(u)->>'last_name'))), '') AS factory_name,
		fp.image_url AS factory_logo_url,
		fp.rating::double precision AS factory_rating_avg,
		COALESCE(r.quantity, 1)::double precision AS quote_quantity,
		q.price_per_piece, q.mold_cost, q.lead_time_days, q.shipping_method_id,
		NULL::text AS shipping_method_name,
		q.factory_highlight,
		q.status, q.create_time, q.log_timestamp,
		COALESCE(q.version, 1) AS version, COALESCE(q.is_locked, false) AS is_locked, q.last_edited_at, q.last_edited_by,
		q.subtotal, q.discount_amount, q.shipping_cost, q.shipping_method, q.packaging_cost, q.tooling_mold_cost,
		q.vat_rate, q.vat_amount, q.platform_commission_rate, q.platform_commission_amount, q.platform_config_id,
		q.grand_total, q.factory_net_receivable, q.production_start_date, q.delivery_date, q.incoterms, q.payment_terms,
		q.validity_days, COALESCE(q.valid_until, (q.create_time + (q.validity_days::text || ' day')::interval)::date) AS valid_until,
		q.warranty_period_months, COALESCE(q.revision_no, 1) AS revision_no, q.parent_quotation_id,
		COALESCE(q.image_urls::text, '[]') AS image_urls,
		NULL::text AS material_detail,
		NULL::text AS payment_condition,
		0::double precision AS sample_cost,
		'[]'::jsonb AS certifications
		FROM quotations q
		INNER JOIN rfqs r ON r.rfq_id = q.rfq_id
		LEFT JOIN users u ON u.user_id = q.factory_id
		LEFT JOIN factory_profiles fp ON fp.user_id = q.factory_id
		WHERE q.rfq_id = $1
		ORDER BY create_time DESC
	`
	if err := r.db.Select(&items, query, rfqID); err != nil {
		return items, err
	}
	for i := range items {
		enrichQuotationComputed(&items[i])
	}
	return items, nil
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
	for i := range items {
		enrichQuotationComputed(&items[i])
	}
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
	enrichQuotationComputed(&item)
	return &item, nil
}

func (r *QuotationRepository) UpdateStatus(quotationID int64, status string) error {
	query := `
		UPDATE quotations
		SET status = $1,
		    log_timestamp = NOW(),
		    is_locked = CASE WHEN $2 = 'AC' THEN TRUE ELSE COALESCE(is_locked, false) END
		WHERE quote_id = $3
	`
	_, err := r.db.Exec(query, status, status, quotationID)
	return err
}

// UpdateStatusTx updates quotation status inside a transaction (same rules as UpdateStatus).
func (r *QuotationRepository) UpdateStatusTx(tx *sqlx.Tx, quotationID int64, status string) error {
	query := `
		UPDATE quotations
		SET status = $1,
		    log_timestamp = NOW(),
		    is_locked = CASE WHEN $2 = 'AC' THEN TRUE ELSE COALESCE(is_locked, false) END
		WHERE quote_id = $3
	`
	_, err := tx.Exec(query, status, status, quotationID)
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

func (r *QuotationRepository) ListRevisionChain(root *domain.Quotation) ([]domain.Quotation, error) {
	rootID := root.QuotationID
	for root.ParentQuotationID != nil {
		parent, err := r.GetByID(*root.ParentQuotationID)
		if err != nil {
			return nil, err
		}
		root = parent
		rootID = root.QuotationID
	}
	var items []domain.Quotation
	err := r.db.Select(&items, quotationSelectBase()+`
		WHERE quote_id = $1 OR parent_quotation_id = $1
		ORDER BY revision_no ASC, create_time ASC
	`, rootID)
	for i := range items {
		enrichQuotationComputed(&items[i])
	}
	return items, err
}

func (r *QuotationRepository) UpdateBody(
	quoteID int64,
	pricePerPiece, moldCost, shippingCost, packagingCost, toolingMoldCost float64,
	leadTimeDays, shippingMethodID int64,
	editorID int64,
	newVersion int,
	paymentTerms *string,
	factoryHighlight *string,
) error {
	query := `
		UPDATE quotations
		SET price_per_piece = $1,
		    mold_cost = $2,
		    tooling_mold_cost = $3,
		    shipping_cost = $4,
		    packaging_cost = $5,
		    lead_time_days = $6,
		    shipping_method_id = CASE WHEN $7 > 0 THEN $7 ELSE shipping_method_id END,
		    payment_terms = CASE WHEN $8::text IS NOT NULL THEN $8 ELSE payment_terms END,
		    factory_highlight = CASE WHEN $12::text IS NOT NULL THEN $12 ELSE factory_highlight END,
		    version = $9,
		    last_edited_at = NOW(),
		    last_edited_by = $10,
		    log_timestamp = NOW()
		WHERE quote_id = $11 AND COALESCE(is_locked, false) = false AND status = 'PD'
	`
	res, err := r.db.Exec(query, pricePerPiece, moldCost, toolingMoldCost, shippingCost, packagingCost, leadTimeDays, shippingMethodID, nullableStringPtr(paymentTerms), newVersion, editorID, quoteID, nullableStringPtr(factoryHighlight))
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

func enrichQuotationComputed(item *domain.Quotation) {
	if item == nil {
		return
	}
	item.QuoteTotal = item.GrandTotal
	if item.QuoteTotal <= 0 {
		qty := item.QuoteQuantity
		if qty <= 0 {
			qty = 1
		}
		item.QuoteTotal = roundQuotationTotal((item.PricePerPiece * qty) + item.MoldCost)
	}
	item.Factory = &domain.FactoryBrief{
		ID:        item.FactoryID,
		Name:      item.FactoryName,
		LogoURL:   item.FactoryLogoURL,
		RatingAvg: item.FactoryRatingAvg,
	}
}

func roundQuotationTotal(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func (r *QuotationRepository) UpdateTotals(quoteID int64, vatRate, vatAmount, platformCommissionRate, platformCommissionAmount, grandTotal, factoryNetReceivable float64) error {
	_, err := r.db.Exec(`
		UPDATE quotations
		SET vat_rate = $1,
		    vat_amount = $2,
		    platform_commission_rate = $3,
		    platform_commission_amount = $4,
		    grand_total = $5,
		    factory_net_receivable = $6,
		    log_timestamp = NOW()
		WHERE quote_id = $7
	`, vatRate, vatAmount, platformCommissionRate, platformCommissionAmount, grandTotal, factoryNetReceivable, quoteID)
	return err
}

func nullableInt64ForZero(v int64) interface{} {
	if v <= 0 {
		return nil
	}
	return v
}

func (r *QuotationRepository) UpdateImageURLs(quoteID int64, imageURLs domain.StringArray) error {
	if imageURLs == nil {
		imageURLs = domain.StringArray{}
	}
	_, err := r.db.Exec(`
		UPDATE quotations
		SET image_urls = $1, log_timestamp = NOW()
		WHERE quote_id = $2 AND COALESCE(is_locked, false) = false AND status = 'PD'
	`, imageURLs, quoteID)
	return err
}

func (r *QuotationRepository) ShippingMethodValid(shippingMethodID int64) (bool, error) {
	var ok bool
	err := r.db.Get(&ok, `
		SELECT EXISTS (SELECT 1 FROM lbi_shipping_methods WHERE shipping_method_id = $1 AND status = '1')
	`, shippingMethodID)
	return ok, err
}

func (r *QuotationRepository) MarkAncestorsRevised(tx *sqlx.Tx, rfqID, factoryID int64) error {
	_, err := tx.Exec(`
		UPDATE quotations
		SET status = 'RV', log_timestamp = NOW(), is_locked = TRUE
		WHERE rfq_id = $1 AND factory_id = $2 AND status IN ('PD', 'AC')
	`, rfqID, factoryID)
	return err
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

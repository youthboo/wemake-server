package quotation

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
	"github.com/yourusername/wemake/internal/dbutil"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/helper"
)

type QuotationRepository struct {
	db *sqlx.DB
}

func NewQuotationRepository(db *sqlx.DB) *QuotationRepository {
	return &QuotationRepository{db: db}
}

func quotationSelectBase() string {
	return `SELECT q.quote_id, q.rfq_id, q.factory_id,
		COALESCE(
			NULLIF(TRIM(fp.factory_name), ''),
			NULLIF(TRIM(CONCAT_WS(' ', fc.first_name, fc.last_name)), ''),
			fu.email
		) AS factory_name,
		fp.image_url AS factory_logo_url,
		fp.rating::double precision AS factory_rating_avg,
		COALESCE((SELECT COUNT(*) FROM orders o WHERE o.factory_id = q.factory_id AND TRIM(o.status) = 'CP'), 0)::bigint AS completed_orders,
		COALESCE(r.quantity, 1)::double precision AS quote_quantity,
		q.price_per_piece, q.mold_cost, q.lead_time_days, q.shipping_method_id,
		COALESCE(r.status, 'OP') AS rfq_status,
		COALESCE(r.request_kind, 'PR') AS request_kind,
		NULL::integer AS sample_qty,
		NULLIF(TRIM(COALESCE(to_jsonb(sm)->>'method_name', to_jsonb(sm)->>'name')), '') AS shipping_method_name,
		q.factory_highlight, q.factory_note,
		q.status, q.create_time, q.log_timestamp,
		COALESCE(q.version, 1) AS version, COALESCE(q.is_locked, false) AS is_locked, NULL::timestamptz AS last_edited_at, NULL::bigint AS last_edited_by,
		q.subtotal, q.discount_amount, q.shipping_cost, NULL::text AS shipping_method, q.packaging_cost, q.mold_cost AS tooling_mold_cost,
		q.vat_rate, q.vat_amount, q.platform_commission_rate, q.platform_commission_amount, q.platform_config_id,
		q.grand_total, q.factory_net_receivable, NULL::date AS production_start_date, NULL::date AS delivery_date, NULL::text AS incoterms, q.payment_terms,
		validity_days, COALESCE(q.valid_until, (q.create_time + (q.validity_days::text || ' day')::interval)::date) AS valid_until,
		NULL::integer AS warranty_period_months, 1 AS revision_no, NULL::bigint AS parent_quotation_id,
		COALESCE(q.image_urls::text, '[]') AS image_urls,
		NULL::text AS material_detail,
		NULL::text AS payment_condition,
		0::double precision AS sample_cost,
		'[]'::jsonb AS certifications,
		q.factory_qty, q.factory_unit_id,
		u.name_th AS factory_unit_name
		FROM quotations q
		LEFT JOIN lbi_units u ON u.unit_id = q.factory_unit_id
		LEFT JOIN rfqs r ON r.rfq_id = q.rfq_id
		LEFT JOIN factory_profiles fp ON fp.user_id = q.factory_id
		LEFT JOIN users fu ON fu.user_id = q.factory_id
		LEFT JOIN customers fc ON fc.user_id = q.factory_id
		LEFT JOIN lbi_shipping_methods sm ON sm.shipping_method_id = q.shipping_method_id`
}

func (r *QuotationRepository) Create(item *domain.Quotation) error {
	return r.createWithExecutor(r.db, item)
}

func (r *QuotationRepository) CreateTx(tx *sqlx.Tx, item *domain.Quotation) error {
	return r.createWithExecutor(tx, item)
}

func (r *QuotationRepository) createWithExecutor(exec dbutil.QueryRower, item *domain.Quotation) error {
	imageURLs := item.ImageURLs
	if imageURLs == nil {
		imageURLs = domain.StringArray{}
	}
	query := `
		INSERT INTO quotations (
			rfq_id, factory_id, price_per_piece, mold_cost, lead_time_days, shipping_method_id, factory_highlight, factory_note, status, create_time, log_timestamp,
			subtotal, discount_amount, shipping_cost, packaging_cost,
			vat_rate, vat_amount, platform_commission_rate, platform_commission_amount, platform_config_id,
			grand_total, factory_net_receivable, payment_terms,
			validity_days, valid_until, image_urls,
			factory_qty, factory_unit_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
		        $12,$13,$14,$15,
		        $16,$17,$18,$19,$20,
		        $21,$22,$23,
		        $24,$25,$26,
		        $27,$28)
		RETURNING quote_id
	`
	if err := exec.QueryRow(
		query,
		item.RFQID,
		item.FactoryID,
		item.PricePerPiece,
		item.MoldCost,
		item.LeadTimeDays,
		domainutil.NullablePositiveInt64(item.ShippingMethodID),
		domainutil.Nullable(item.FactoryHighlight),
		domainutil.Nullable(item.FactoryNote),
		item.Status,
		item.CreateTime,
		item.LogTimestamp,
		item.Subtotal,
		item.DiscountAmount,
		item.ShippingCost,
		item.PackagingCost,
		item.VatRate,
		item.VatAmount,
		item.PlatformCommissionRate,
		item.PlatformCommissionAmount,
		domainutil.Nullable(item.PlatformConfigID),
		item.GrandTotal,
		item.FactoryNetReceivable,
		domainutil.Nullable(item.PaymentTerms),
		item.ValidityDays,
		domainutil.Nullable(item.ValidUntil),
		imageURLs,
		item.FactoryQty,
		item.FactoryUnitID,
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
		WHERE q.rfq_id = $1
		ORDER BY q.create_time DESC
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
	query := quotationSelectBase() + ` WHERE q.factory_id = $1`
	args := []interface{}{factoryID}
	statuses := splitCSVUpper(status)
	if len(statuses) == 1 {
		query += ` AND q.status = $2`
		args = append(args, statuses[0])
	} else if len(statuses) > 1 {
		placeholders := make([]string, 0, len(statuses))
		for _, st := range statuses {
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, st)
		}
		query += ` AND q.status IN (` + strings.Join(placeholders, ", ") + `)`
	}
	query += ` ORDER BY q.create_time DESC`
	err := r.db.Select(&items, query, args...)
	for i := range items {
		enrichQuotationComputed(&items[i])
	}
	return items, err
}

func (r *QuotationRepository) GetByID(quotationID int64) (*domain.Quotation, error) {
	var item domain.Quotation
	query := quotationSelectBase() + `
		WHERE q.quote_id = $1
	`
	if err := r.db.Get(&item, query, quotationID); err != nil {
		return nil, err
	}
	enrichQuotationComputed(&item)
	return &item, nil
}

// GetActiveByRFQAndFactory returns the latest active (PD/AC) quotation for a given RFQ + factory pair.
// Used by MessageService to authoratively populate quote_data when creating a QT message.
func (r *QuotationRepository) GetActiveByRFQAndFactory(rfqID, factoryID int64) (*domain.Quotation, error) {
	var item domain.Quotation
	query := quotationSelectBase() + `
		WHERE q.rfq_id = $1 AND q.factory_id = $2 AND q.status IN ('PD', 'AC')
		ORDER BY q.log_timestamp DESC
		LIMIT 1
	`
	if err := r.db.Get(&item, query, rfqID, factoryID); err != nil {
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
		domainutil.Nullable(entry.PricePerPiece),
		domainutil.Nullable(entry.MoldCost),
		domainutil.Nullable(entry.LeadTimeDays),
		domainutil.Nullable(entry.ShippingMethodID),
		domainutil.Nullable(entry.Status),
		domainutil.Nullable(entry.Reason),
		domainutil.Nullable(entry.EditedBy),
	).Scan(&entry.HistoryID, &entry.CreatedAt)
}

func splitCSVUpper(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		item := domainutil.NormalizeStatus(part)
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
		WHERE q.quote_id = $1
		ORDER BY q.create_time ASC
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
	validityDays *int,
	validUntil *time.Time,
	factoryNote *string,
	factoryQty *int, factoryUnitID *int64,
) error {
	query := `
		UPDATE quotations
		SET price_per_piece = $1,
		    mold_cost = $2,
		    shipping_cost = $3,
		    packaging_cost = $4,
		    lead_time_days = $5,
		    shipping_method_id = CASE WHEN $6 > 0 THEN $6 ELSE shipping_method_id END,
		    payment_terms = CASE WHEN $7::text IS NOT NULL THEN $7 ELSE payment_terms END,
		    factory_highlight = CASE WHEN $10::text IS NOT NULL THEN $10 ELSE factory_highlight END,
		    validity_days = CASE WHEN $11::int IS NOT NULL THEN $11 ELSE validity_days END,
		    valid_until   = CASE WHEN $12::timestamp IS NOT NULL THEN $12 ELSE valid_until END,
		    factory_note  = CASE WHEN $13::text IS NOT NULL THEN $13 ELSE factory_note END,
		    factory_qty     = $14,
		    factory_unit_id = $15,
		    version = $8,
		    log_timestamp = NOW()
		WHERE quote_id = $9 AND COALESCE(is_locked, false) = false AND status = 'PD'
	`
	res, err := r.db.Exec(query,
		pricePerPiece, moldCost, shippingCost, packagingCost,
		leadTimeDays, shippingMethodID,
		domainutil.Nullable(paymentTerms),
		newVersion, quoteID,
		domainutil.Nullable(factoryHighlight),
		validityDays, validUntil,
		domainutil.Nullable(factoryNote),
		factoryQty, factoryUnitID,
	)
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

// UpdateFactoryNote updates only factory_note — no lock/status check.
// Any quotation owned by factoryID can be updated at any time.
func (r *QuotationRepository) UpdateFactoryNote(quoteID, factoryID int64, note *string) error {
	const query = `
		UPDATE quotations
		SET factory_note = $1, log_timestamp = NOW()
		WHERE quote_id = $2 AND factory_id = $3
	`
	res, err := r.db.Exec(query, domainutil.Nullable(note), quoteID, factoryID)
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
	if helper.IsMoneyLessOrEqual(item.QuoteTotal, helper.ZeroMoney()) {
		qty := item.QuoteQuantity
		if qty <= 0 {
			qty = 1
		}
		lineTotal := helper.MultiplyMoney(item.PricePerPiece, decimal.NewFromFloat(qty))
		item.QuoteTotal = helper.AddMoney(lineTotal, item.MoldCost)
	}
	item.Factory = &domain.FactoryBrief{
		ID:        item.FactoryID,
		Name:      item.FactoryName,
		LogoURL:   item.FactoryLogoURL,
		RatingAvg: item.FactoryRatingAvg,
	}
}

func (r *QuotationRepository) UpdateTotals(quoteID int64, subtotal, vatRate, vatAmount, platformCommissionRate, platformCommissionAmount, grandTotal, factoryNetReceivable float64) error {
	_, err := r.db.Exec(`
		UPDATE quotations
		SET subtotal = $1,
		    vat_rate = $2,
		    vat_amount = $3,
		    platform_commission_rate = $4,
		    platform_commission_amount = $5,
		    grand_total = $6,
		    factory_net_receivable = $7,
		    log_timestamp = NOW()
		WHERE quote_id = $8
	`, subtotal, vatRate, vatAmount, platformCommissionRate, platformCommissionAmount, grandTotal, factoryNetReceivable, quoteID)
	return err
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

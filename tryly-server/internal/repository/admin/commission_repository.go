package admin

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

// CommissionInvoice maps to commission_invoices table.
type CommissionInvoice struct {
	InvoiceID        int64               `db:"invoice_id"        json:"invoice_id"`
	FactoryID        int64               `db:"factory_id"        json:"factory_id"`
	FactoryName      string              `db:"factory_name"      json:"factory_name"`
	PeriodMonth      int                 `db:"period_month"      json:"period_month"`
	PeriodYear       int                 `db:"period_year"       json:"period_year"`
	TotalOrders      int                 `db:"total_orders"      json:"total_orders"`
	TotalAmount      float64             `db:"total_amount"      json:"total_amount"`
	CommissionAmount float64             `db:"commission_amount" json:"commission_amount"`
	VatAmount        float64             `db:"vat_amount"        json:"vat_amount"`
	GrandTotal       float64             `db:"grand_total"       json:"grand_total"`
	Status           string              `db:"status"            json:"status"`
	SlipURLs         domain.StringArray  `db:"slip_urls"         json:"slip_urls"`
	SlipNote         *string             `db:"slip_note"         json:"slip_note,omitempty"`
	VerifiedBy       *int64              `db:"verified_by"       json:"verified_by,omitempty"`
	VerifiedAt       *string             `db:"verified_at"       json:"verified_at,omitempty"`
	EmailSentAt      *string             `db:"email_sent_at"     json:"email_sent_at,omitempty"`
	CreatedAt        string              `db:"created_at"        json:"created_at"`
}

type CommissionInvoiceItem struct {
	ItemID           int64   `db:"item_id"           json:"item_id"`
	InvoiceID        int64   `db:"invoice_id"        json:"invoice_id"`
	OrderID          int64   `db:"order_id"          json:"order_id"`
	OrderAmount      float64 `db:"order_amount"      json:"order_amount"`
	CommissionRate   float64 `db:"commission_rate"    json:"commission_rate"`
	CommissionAmount float64 `db:"commission_amount"  json:"commission_amount"`
}

type CommissionInvoiceRepository struct {
	db *sqlx.DB
}

func NewCommissionInvoiceRepository(db *sqlx.DB) *CommissionInvoiceRepository {
	return &CommissionInvoiceRepository{db: db}
}

// EscrowCommissionRow — ค่าคอมมิชชันที่ Tryly ได้รับต่อ 1 ออเดอร์ (escrow mode).
// อ่านจาก quotation ที่ยอมรับแล้วของแต่ละ order
type EscrowCommissionRow struct {
	OrderID        int64   `db:"order_id"         json:"order_id"`
	FactoryID      int64   `db:"factory_id"       json:"factory_id"`
	FactoryName    string  `db:"factory_name"     json:"factory_name"`
	CustomerName   string  `db:"customer_name"    json:"customer_name"`
	Status         string  `db:"status"           json:"status"`
	GrandTotal     float64 `db:"grand_total"      json:"grand_total"`
	CommissionRate float64 `db:"commission_rate"  json:"commission_rate"`
	CommissionAmt  float64 `db:"commission_amount" json:"commission_amount"`
	VatAmount      float64 `db:"vat_amount"       json:"vat_amount"`
	FactoryNet     float64 `db:"factory_net"      json:"factory_net"`
	CreatedAt      string  `db:"created_at"       json:"created_at"`
}

// ListEscrowCommissions returns per-order commission Tryly earns in the escrow flow.
// Only orders whose payment slip was approved (slip_status = 'AP') count as realized.
func (r *CommissionInvoiceRepository) ListEscrowCommissions() ([]EscrowCommissionRow, error) {
	rows := []EscrowCommissionRow{}
	err := r.db.Select(&rows, `
		SELECT o.order_id,
		       o.factory_id,
		       COALESCE(fp.factory_name, 'Factory #' || o.factory_id::text) AS factory_name,
		       COALESCE(NULLIF(TRIM(CONCAT(cu.first_name, ' ', cu.last_name)), ''),
		                'ลูกค้า #' || o.customer_id::text)                    AS customer_name,
		       TRIM(o.status)                            AS status,
		       COALESCE(q.grand_total, o.total_amount)   AS grand_total,
		       COALESCE(q.platform_commission_rate, 0)   AS commission_rate,
		       COALESCE(q.platform_commission_amount, 0) AS commission_amount,
		       COALESCE(q.vat_amount, 0)                 AS vat_amount,
		       COALESCE(q.factory_net_receivable, 0)     AS factory_net,
		       o.created_at::text                        AS created_at
		FROM orders o
		LEFT JOIN quotations q       ON q.quote_id  = o.quote_id
		LEFT JOIN factory_profiles fp ON fp.user_id  = o.factory_id
		LEFT JOIN customers cu        ON cu.user_id  = o.customer_id
		WHERE TRIM(COALESCE(o.slip_status, '')) = 'AP'
		  -- exclude cancelled/refunded orders: their commission was voided (CM → RJ)
		  -- even though slip_status may still read 'AP' from before the resolution
		  AND TRIM(o.status) NOT IN ('CC', 'CN', 'RF')
		ORDER BY o.created_at DESC, o.order_id DESC
	`)
	return rows, err
}

// GenerateForMonth creates invoices for all factories that have approved orders in the given month.
// Factories that already have an invoice for the period are skipped (incremental).
func (r *CommissionInvoiceRepository) GenerateForMonth(month, year int) (int, error) {

	// Aggregate per factory: orders where slip was approved in the target month
	rows, err := r.db.Queryx(`
		SELECT
			o.factory_id,
			COUNT(o.order_id)::int                            AS total_orders,
			COALESCE(SUM(q.grand_total), 0)                   AS total_amount,
			COALESCE(SUM(q.platform_commission_amount), 0)    AS commission_amount,
			COALESCE(SUM(q.vat_amount), 0)                    AS vat_amount
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		WHERE TRIM(o.slip_status) = 'AP'
		  AND EXTRACT(MONTH FROM o.updated_at) = $1
		  AND EXTRACT(YEAR FROM o.updated_at)  = $2
		GROUP BY o.factory_id
		HAVING COUNT(o.order_id) > 0
	`, month, year)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var created int
	for rows.Next() {
		var factoryID int64
		var totalOrders int
		var totalAmount, commissionAmount, vatAmount float64
		if err := rows.Scan(&factoryID, &totalOrders, &totalAmount, &commissionAmount, &vatAmount); err != nil {
			continue
		}
		grandTotal := commissionAmount + vatAmount

		var invoiceID int64
		err := r.db.Get(&invoiceID, `
			INSERT INTO commission_invoices
			    (factory_id, period_month, period_year, total_orders, total_amount,
			     commission_amount, vat_amount, grand_total, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'DR')
			ON CONFLICT (factory_id, period_month, period_year) DO NOTHING
			RETURNING invoice_id
		`, factoryID, month, year, totalOrders, totalAmount, commissionAmount, vatAmount, grandTotal)
		if err != nil {
			continue
		}

		// Insert line items
		r.db.Exec(`
			INSERT INTO commission_invoice_items (invoice_id, order_id, order_amount, commission_rate, commission_amount)
			SELECT $1, o.order_id, q.grand_total, q.platform_commission_rate, q.platform_commission_amount
			FROM orders o
			INNER JOIN quotations q ON q.quote_id = o.quote_id
			WHERE o.factory_id = $2
			  AND TRIM(o.slip_status) = 'AP'
			  AND EXTRACT(MONTH FROM o.updated_at) = $3
			  AND EXTRACT(YEAR FROM o.updated_at)  = $4
		`, invoiceID, factoryID, month, year)

		created++
	}

	return created, nil
}

// ListAll returns all invoices for admin, with factory name.
func (r *CommissionInvoiceRepository) ListAll(month, year int) ([]CommissionInvoice, error) {
	var items []CommissionInvoice
	err := r.db.Select(&items, `
		SELECT ci.invoice_id, ci.factory_id,
		       COALESCE(fp.factory_name, '') AS factory_name,
		       ci.period_month, ci.period_year,
		       ci.total_orders, ci.total_amount,
		       ci.commission_amount, ci.vat_amount, ci.grand_total,
		       TRIM(ci.status) AS status,
		       ci.slip_urls, ci.slip_note,
		       ci.verified_by, ci.verified_at::text AS verified_at,
		       ci.email_sent_at::text AS email_sent_at,
		       ci.created_at::text AS created_at
		FROM commission_invoices ci
		LEFT JOIN factory_profiles fp ON fp.user_id = ci.factory_id
		WHERE ($1 = 0 OR ci.period_month = $1)
		  AND ($2 = 0 OR ci.period_year = $2)
		ORDER BY ci.period_year DESC, ci.period_month DESC, ci.factory_id
	`, month, year)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []CommissionInvoice{}
	}
	return items, nil
}

// ListByFactory returns invoices for a specific factory.
func (r *CommissionInvoiceRepository) ListByFactory(factoryID int64) ([]CommissionInvoice, error) {
	var items []CommissionInvoice
	err := r.db.Select(&items, `
		SELECT ci.invoice_id, ci.factory_id,
		       COALESCE(fp.factory_name, '') AS factory_name,
		       ci.period_month, ci.period_year,
		       ci.total_orders, ci.total_amount,
		       ci.commission_amount, ci.vat_amount, ci.grand_total,
		       TRIM(ci.status) AS status,
		       ci.slip_urls, ci.slip_note,
		       ci.verified_by, ci.verified_at::text AS verified_at,
		       ci.email_sent_at::text AS email_sent_at,
		       ci.created_at::text AS created_at
		FROM commission_invoices ci
		LEFT JOIN factory_profiles fp ON fp.user_id = ci.factory_id
		WHERE ci.factory_id = $1
		ORDER BY ci.period_year DESC, ci.period_month DESC
	`, factoryID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []CommissionInvoice{}
	}
	return items, nil
}

// CurrentPeriodOrder is one approved order within the current billing period.
type CurrentPeriodOrder struct {
	OrderID          int64   `db:"order_id"           json:"order_id"`
	RFQTitle         string  `db:"rfq_title"          json:"rfq_title"`
	OrderAmount      float64 `db:"order_amount"       json:"order_amount"`
	CommissionRate   float64 `db:"commission_rate"    json:"commission_rate"`
	CommissionAmount float64 `db:"commission_amount"  json:"commission_amount"`
	CommissionVat    float64 `db:"commission_vat"     json:"commission_vat"`
	LineTotal        float64 `db:"line_total"         json:"line_total"`
	ApprovedAt       string  `db:"approved_at"        json:"approved_at"`
}

// CurrentPeriodSummary aggregates the unbilled month's commission for one factory.
type CurrentPeriodSummary struct {
	Month            int                  `json:"month"`
	Year             int                  `json:"year"`
	TotalOrders      int                  `json:"total_orders"`
	TotalAmount      float64              `json:"total_amount"`
	CommissionAmount float64              `json:"commission_amount"`
	VatAmount        float64              `json:"vat_amount"`
	GrandTotal       float64              `json:"grand_total"`
	Orders           []CurrentPeriodOrder `json:"orders"`
}

// GetCurrentPeriod returns unbilled orders for a factory in the given month/year.
func (r *CommissionInvoiceRepository) GetCurrentPeriod(factoryID int64, month, year int) (*CurrentPeriodSummary, error) {
	var orders []CurrentPeriodOrder
	err := r.db.Select(&orders, `
		SELECT
			o.order_id,
			COALESCE(r.title, '')                                    AS rfq_title,
			q.grand_total                                            AS order_amount,
			q.platform_commission_rate                               AS commission_rate,
			q.platform_commission_amount                             AS commission_amount,
			ROUND((q.platform_commission_amount * 0.07)::numeric, 2) AS commission_vat,
			ROUND((q.platform_commission_amount * 1.07)::numeric, 2) AS line_total,
			o.updated_at::text                                       AS approved_at
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		LEFT JOIN rfqs r ON r.rfq_id = q.rfq_id
		WHERE o.factory_id = $1
		  AND TRIM(o.slip_status) = 'AP'
		  AND EXTRACT(MONTH FROM o.updated_at) = $2
		  AND EXTRACT(YEAR FROM o.updated_at)  = $3
		ORDER BY o.updated_at DESC
	`, factoryID, month, year)
	if err != nil {
		return nil, err
	}
	if orders == nil {
		orders = []CurrentPeriodOrder{}
	}

	var summary CurrentPeriodSummary
	summary.Month = month
	summary.Year = year
	summary.Orders = orders
	for _, o := range orders {
		summary.TotalOrders++
		summary.TotalAmount += o.OrderAmount
		summary.CommissionAmount += o.CommissionAmount
		summary.VatAmount += o.CommissionVat
	}
	summary.GrandTotal = summary.CommissionAmount + summary.VatAmount
	return &summary, nil
}

// GetByID returns a single invoice.
func (r *CommissionInvoiceRepository) GetByID(invoiceID int64) (*CommissionInvoice, error) {
	var item CommissionInvoice
	err := r.db.Get(&item, `
		SELECT ci.invoice_id, ci.factory_id,
		       COALESCE(fp.factory_name, '') AS factory_name,
		       ci.period_month, ci.period_year,
		       ci.total_orders, ci.total_amount,
		       ci.commission_amount, ci.vat_amount, ci.grand_total,
		       TRIM(ci.status) AS status,
		       ci.slip_urls, ci.slip_note,
		       ci.verified_by, ci.verified_at::text AS verified_at,
		       ci.email_sent_at::text AS email_sent_at,
		       ci.created_at::text AS created_at
		FROM commission_invoices ci
		LEFT JOIN factory_profiles fp ON fp.user_id = ci.factory_id
		WHERE ci.invoice_id = $1
	`, invoiceID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetItems returns line items for an invoice.
func (r *CommissionInvoiceRepository) GetItems(invoiceID int64) ([]CommissionInvoiceItem, error) {
	var items []CommissionInvoiceItem
	err := r.db.Select(&items, `
		SELECT item_id, invoice_id, order_id, order_amount, commission_rate, commission_amount
		FROM commission_invoice_items
		WHERE invoice_id = $1
		ORDER BY order_id
	`, invoiceID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []CommissionInvoiceItem{}
	}
	return items, nil
}

// AttachFactorySlip — factory uploads or edits comm payment slip(s) (while ST/RJ/PA).
// Replaces the full slip_urls set with slipURLs each time (so "edit" = re-upload wins).
func (r *CommissionInvoiceRepository) AttachFactorySlip(invoiceID int64, slipURLs domain.StringArray, slipNote string) error {
	res, err := r.db.Exec(`
		UPDATE commission_invoices
		SET slip_urls = $2, slip_note = NULLIF($3, ''), status = 'PA', updated_at = NOW()
		WHERE invoice_id = $1 AND TRIM(status) IN ('ST', 'RJ', 'PA')
	`, invoiceID, slipURLs, slipNote)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// AdminVerifySlip — SA approves/rejects factory's comm slip.
func (r *CommissionInvoiceRepository) AdminVerifySlip(invoiceID, adminID int64, approve bool) error {
	var newStatus string
	if approve {
		newStatus = "VR"
	} else {
		newStatus = "ST" // back to sent, factory must re-attach
	}
	res, err := r.db.Exec(`
		UPDATE commission_invoices
		SET status = $2, verified_by = $3, verified_at = NOW(), updated_at = NOW()
		WHERE invoice_id = $1 AND TRIM(status) = 'PA'
	`, invoiceID, newStatus, adminID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// MarkSent transitions DR → ST and sets email_sent_at.
func (r *CommissionInvoiceRepository) MarkSent(invoiceID int64) error {
	_, err := r.db.Exec(`
		UPDATE commission_invoices
		SET status = 'ST', email_sent_at = NOW(), updated_at = NOW()
		WHERE invoice_id = $1 AND TRIM(status) = 'DR'
	`, invoiceID)
	return err
}

// UpdateEmailSentAt refreshes email_sent_at without changing status (used by resend).
func (r *CommissionInvoiceRepository) UpdateEmailSentAt(invoiceID int64) error {
	_, err := r.db.Exec(`
		UPDATE commission_invoices SET email_sent_at = NOW(), updated_at = NOW()
		WHERE invoice_id = $1
	`, invoiceID)
	return err
}

// GetSummary returns aggregated commission summary for admin dashboard.
type CommissionSummary struct {
	TotalInvoices      int     `db:"total_invoices"      json:"total_invoices"`
	TotalCommission    float64 `db:"total_commission"    json:"total_commission"`
	TotalVat           float64 `db:"total_vat"           json:"total_vat"`
	TotalGrand         float64 `db:"total_grand"         json:"total_grand"`
	DraftCount         int     `db:"draft_count"         json:"draft_count"`
	SentCount          int     `db:"sent_count"          json:"sent_count"`
	PaidCount          int     `db:"paid_count"          json:"paid_count"`
	VerifiedCount      int     `db:"verified_count"      json:"verified_count"`
}

func (r *CommissionInvoiceRepository) GetSummary(month, year int) (*CommissionSummary, error) {
	var item CommissionSummary
	err := r.db.Get(&item, `
		SELECT
			COUNT(*)::int                                                            AS total_invoices,
			COALESCE(SUM(commission_amount), 0)                                      AS total_commission,
			COALESCE(SUM(vat_amount), 0)                                             AS total_vat,
			COALESCE(SUM(grand_total), 0)                                            AS total_grand,
			COUNT(*) FILTER (WHERE TRIM(status) = 'DR')::int                         AS draft_count,
			COUNT(*) FILTER (WHERE TRIM(status) = 'ST')::int                         AS sent_count,
			COUNT(*) FILTER (WHERE TRIM(status) = 'PA')::int                         AS paid_count,
			COUNT(*) FILTER (WHERE TRIM(status) = 'VR')::int                         AS verified_count
		FROM commission_invoices
		WHERE ($1 = 0 OR period_month = $1)
		  AND ($2 = 0 OR period_year = $2)
	`, month, year)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

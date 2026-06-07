package order

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/helper"
)

type OrderRepository struct {
	db *sqlx.DB
}

type QuotationOrderSource struct {
	QuotationID   int64      `db:"quote_id"`
	RFQID         int64      `db:"rfq_id"`
	UserID        int64      `db:"user_id"`
	FactoryID     int64      `db:"factory_id"`
	PricePerPiece float64    `db:"price_per_piece"`
	Quantity      int64      `db:"quantity"`    // RFQ qty
	RFQUnitID     *int64     `db:"rfq_unit_id"` // RFQ unit
	FactoryQty    *int       `db:"factory_qty"`
	FactoryUnitID *int64     `db:"factory_unit_id"`
	MoldCost      float64    `db:"mold_cost"`
	LeadTimeDays  int64      `db:"lead_time_days"`
	DeliveryDate  *time.Time `db:"delivery_date"`
	Status        string     `db:"status"`
	GrandTotal    float64    `db:"grand_total"`
}

// EffectiveQty returns factory_qty if set, else RFQ quantity.
func (q *QuotationOrderSource) EffectiveQty() *int {
	if q.FactoryQty != nil && *q.FactoryQty > 0 {
		return q.FactoryQty
	}
	v := int(q.Quantity)
	return &v
}

// EffectiveUnitID returns factory_unit_id if set, else RFQ unit_id.
func (q *QuotationOrderSource) EffectiveUnitID() *int64 {
	if q.FactoryUnitID != nil {
		return q.FactoryUnitID
	}
	return q.RFQUnitID
}

func (q *QuotationOrderSource) ToOrderDomain() *domain.Order {
	return &domain.Order{
		QuotationID: q.QuotationID,
		UserID:      q.UserID,
		FactoryID:   q.FactoryID,
		TotalAmount: helper.MoneyDecimal(q.GrandTotal),
		Quantity:    q.EffectiveQty(),
		UnitID:      q.EffectiveUnitID(),
	}
}

type OrderDetailRow struct {
	OrderID            int64      `db:"order_id"`
	QuotationID        int64      `db:"quote_id"`
	UserID             int64      `db:"user_id"`
	FactoryID          int64      `db:"factory_id"`
	TotalAmount        float64    `db:"total_amount"`
	DepositAmount      float64    `db:"deposit_amount"`
	Status             string     `db:"status"`
	EstimatedDelivery  *time.Time `db:"estimated_delivery"`
	TrackingNo         *string    `db:"tracking_no"`
	Courier            *string    `db:"courier"`
	ShippedAt          *time.Time `db:"shipped_at"`
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
	FactoryName        string     `db:"factory_name"`
	DepositScheduleDue *time.Time `db:"deposit_schedule_due"`
	PaymentType        *string    `db:"payment_type"`
	PricePerPiece      float64    `db:"price_per_piece"`
	MoldCost           float64    `db:"mold_cost"`
	LeadTimeDays       int64      `db:"lead_time_days"`
	RFQID              int64      `db:"rfq_id"`
	RFQTitle           string     `db:"rfq_title"`
	RFQDetails         *string    `db:"rfq_details"`
	RFQQuantity        int64      `db:"rfq_quantity"`
	RFQBudget          float64    `db:"rfq_budget"`
	RFQCreatedAt       time.Time  `db:"rfq_created_at"`
	RFQCategoryID      int64      `db:"rfq_category_id"`
	RFQCategoryName    *string    `db:"rfq_category_name"`
	RFQRequestKind     string     `db:"rfq_request_kind"`
	// RFQ enrichment
	RFQSubCategoryID      *int64          `db:"rfq_sub_category_id"`
	RFQSubCategoryName    *string         `db:"rfq_sub_category_name"`
	RFQShippingMethodName *string         `db:"rfq_shipping_method_name"`
	RFQMaterialGrade      *string         `db:"rfq_material_grade"`
	RFQCertifications     pq.StringArray  `db:"rfq_certifications"`
	RFQTargetLeadTimeDays *int            `db:"rfq_target_lead_time_days"`
	RFQTargetPrice        *float64        `db:"rfq_target_price"`
	// Delivery address
	RFQAddrDetail     *string `db:"rfq_addr_detail"`
	RFQAddrSubDistrict *string `db:"rfq_addr_sub_district"`
	RFQAddrDistrict   *string `db:"rfq_addr_district"`
	RFQAddrProvince   *string `db:"rfq_addr_province"`
	RFQAddrZipCode    *string `db:"rfq_addr_zip_code"`
	// Customer / factory contact
	CustomerName    string `db:"customer_name"`
	CustomerPhone   string `db:"customer_phone"`
	FactoryPhone    string `db:"factory_phone"`
	FactoryProvince string `db:"factory_province"`
	// Quotation enrichment
	QuoteGrandTotal       float64         `db:"quote_grand_total"`
	QuoteSubtotal         float64         `db:"quote_subtotal"`
	QuoteDiscountAmount   float64         `db:"quote_discount_amount"`
	QuoteShippingCost     float64         `db:"quote_shipping_cost"`
	QuotePackagingCost    float64         `db:"quote_packaging_cost"`
	QuoteVatRate          float64         `db:"quote_vat_rate"`
	QuoteVatAmount        float64         `db:"quote_vat_amount"`
	QuoteValidityDays     int             `db:"quote_validity_days"`
	QuoteValidUntil       *string         `db:"quote_valid_until"`
	QuotePaymentTerms     *string         `db:"quote_payment_terms"`
	QuoteImageURLs        domain.StringArray `db:"quote_image_urls"`
	QuoteFactoryHighlight *string         `db:"quote_factory_highlight"`
	QuoteFactoryNote      *string         `db:"quote_factory_note"`
}

func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) GetOrderSourceByQuotationID(quotationID, userID int64) (*QuotationOrderSource, error) {
	var src QuotationOrderSource
	query := `
		SELECT q.quote_id, q.rfq_id, rfq.user_id, q.factory_id, q.price_per_piece,
		       rfq.quantity, rfq.unit_id AS rfq_unit_id,
		       q.factory_qty, q.factory_unit_id,
		       q.mold_cost, q.lead_time_days, NULL::date AS delivery_date, q.status, COALESCE(q.grand_total, 0) AS grand_total
		FROM quotations q
		INNER JOIN rfqs rfq ON rfq.rfq_id = q.rfq_id
		WHERE q.quote_id = $1 AND rfq.user_id = $2
	`
	if err := r.db.Get(&src, query, quotationID, userID); err != nil {
		return nil, err
	}
	return &src, nil
}

func (r *OrderRepository) Create(order *domain.Order) error {
	query := `
		INSERT INTO orders (quote_id, customer_id, factory_id, total_amount, deposit_amount, status, estimated_delivery, quantity, unit_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING order_id
	`
	return r.db.QueryRow(
		query,
		order.QuotationID,
		order.UserID,
		order.FactoryID,
		order.TotalAmount,
		order.DepositAmount,
		order.Status,
		domainutil.Nullable(order.EstimatedDelivery),
		order.Quantity,
		order.UnitID,
		order.CreatedAt,
		order.UpdatedAt,
	).Scan(&order.OrderID)
}

// CreateTx inserts an order inside an existing transaction.
func (r *OrderRepository) CreateTx(tx *sqlx.Tx, order *domain.Order) error {
	query := `
		INSERT INTO orders (quote_id, customer_id, factory_id, total_amount, deposit_amount, status, estimated_delivery, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING order_id
	`
	return tx.QueryRow(
		query,
		order.QuotationID,
		order.UserID,
		order.FactoryID,
		order.TotalAmount,
		order.DepositAmount,
		order.Status,
		domainutil.Nullable(order.EstimatedDelivery),
		order.CreatedAt,
		order.UpdatedAt,
	).Scan(&order.OrderID)
}

// OrderExistsForQuoteTx returns true if an order already exists for the quotation.
func (r *OrderRepository) OrderExistsForQuoteTx(tx *sqlx.Tx, quoteID int64) (bool, error) {
	var n int
	err := tx.Get(&n, `SELECT COUNT(*) FROM orders WHERE quote_id = $1`, quoteID)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// GetOrderSourceByQuotationIDTx loads quotation + RFQ ownership inside a transaction.
func (r *OrderRepository) GetOrderSourceByQuotationIDTx(tx *sqlx.Tx, quotationID, userID int64) (*QuotationOrderSource, error) {
	var src QuotationOrderSource
	query := `
		SELECT q.quote_id, q.rfq_id, rfq.user_id, q.factory_id, q.price_per_piece,
		       rfq.quantity, rfq.unit_id AS rfq_unit_id,
		       q.factory_qty, q.factory_unit_id,
		       q.mold_cost, q.lead_time_days, NULL::date AS delivery_date, q.status, COALESCE(q.grand_total, 0) AS grand_total
		FROM quotations q
		INNER JOIN rfqs rfq ON rfq.rfq_id = q.rfq_id
		WHERE q.quote_id = $1 AND rfq.user_id = $2
	`
	if err := tx.Get(&src, query, quotationID, userID); err != nil {
		return nil, err
	}
	return &src, nil
}

type orderRow struct {
	OrderID           int64      `db:"order_id"`
	QuotationID       int64      `db:"quote_id"`
	UserID            int64      `db:"user_id"`
	FactoryID         int64      `db:"factory_id"`
	TotalAmount       float64    `db:"total_amount"`
	DepositAmount     float64    `db:"deposit_amount"`
	Status            string     `db:"status"`
	EstimatedDelivery *time.Time `db:"estimated_delivery"`
	TrackingNo        *string    `db:"tracking_no"`
	Courier           *string    `db:"courier"`
	ShippedAt         *time.Time `db:"shipped_at"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

func (row *orderRow) toDomain() *domain.Order {
	return &domain.Order{
		OrderID:           row.OrderID,
		QuotationID:       row.QuotationID,
		UserID:            row.UserID,
		FactoryID:         row.FactoryID,
		TotalAmount:       helper.MoneyDecimal(row.TotalAmount),
		DepositAmount:     helper.MoneyDecimal(row.DepositAmount),
		Status:            row.Status,
		EstimatedDelivery: row.EstimatedDelivery,
		TrackingNo:        row.TrackingNo,
		Courier:           row.Courier,
		ShippedAt:         row.ShippedAt,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

func (r *OrderRepository) ListByUserID(userID int64, status string) ([]domain.Order, error) {
	var rows []orderRow

	query := sq.Select(
		"order_id",
		"quote_id",
		"customer_id AS user_id",
		"factory_id",
		"total_amount",
		"deposit_amount",
		"status",
		"estimated_delivery",
		"tracking_no",
		"courier",
		"NULL::timestamp AS shipped_at",
		"created_at",
		"updated_at",
	).
		From("orders").
		Where(sq.Eq{"customer_id": userID})

	statuses := splitOrderStatuses(status)
	if len(statuses) > 0 {
		if len(statuses) == 1 {
			query = query.Where(sq.Eq{"status": statuses[0]})
		} else {
			statusInterfaces := make([]interface{}, len(statuses))
			for i, s := range statuses {
				statusInterfaces[i] = s
			}
			query = query.Where(sq.Eq{"status": statusInterfaces})
		}
	}

	query = query.OrderBy("created_at DESC")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	err = r.db.Select(&rows, sql, args...)
	if err != nil {
		return nil, err
	}
	orders := make([]domain.Order, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, *row.toDomain())
	}
	return orders, nil
}

type orderListRow struct {
	OrderID             int64      `db:"order_id"`
	QuotationID         int64      `db:"quote_id"`
	UserID              int64      `db:"user_id"`
	FactoryID           int64      `db:"factory_id"`
	Status              string     `db:"status"`
	TotalAmount         float64    `db:"total_amount"`
	DepositAmount       float64    `db:"deposit_amount"`
	EstimatedDelivery   *time.Time `db:"estimated_delivery"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
	RFQID               int64      `db:"rfq_id"`
	RFQTitle            string     `db:"rfq_title"`
	RFQQuantity         int64      `db:"rfq_quantity"`
	UnitName            string     `db:"unit_name"`
	CustomerDisplayName string     `db:"customer_display_name"`
	RequestKind         string     `db:"request_kind"`
	FactoryHighlight    *string    `db:"factory_highlight"`
	CurrentStepID       *int64     `db:"current_step_id"`
	CurrentStepNameTH   *string    `db:"current_step_name_th"`
	CurrentUpdateStatus *string    `db:"current_update_status"`
	CompletedCount      int64      `db:"completed_count"`
	TotalCount          int64      `db:"total_count"`
	LastUpdatedAt       *time.Time `db:"last_updated_at"`
	HasRejected         bool       `db:"has_rejected"`
}

func (r orderListRow) toDomain() domain.OrderListItem {
	return domain.OrderListItem{
		OrderID:           r.OrderID,
		QuotationID:       r.QuotationID,
		UserID:            r.UserID,
		FactoryID:         r.FactoryID,
		Status:            r.Status,
		TotalAmount:       helper.MoneyDecimal(r.TotalAmount),
		DepositAmount:     helper.MoneyDecimal(r.DepositAmount),
		EstimatedDelivery: r.EstimatedDelivery,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
		RFQ: domain.OrderListRFQSummary{
			RFQID:    r.RFQID,
			Title:    r.RFQTitle,
			Quantity: r.RFQQuantity,
			UnitName: r.UnitName,
		},
		Quotation: domain.OrderListQuotationSummary{
			QuoteID:          r.QuotationID,
			FactoryHighlight: r.FactoryHighlight,
		},
		Customer: domain.OrderListCustomerSummary{
			UserID:      r.UserID,
			DisplayName: r.CustomerDisplayName,
		},
		ProductionSummary: domain.OrderProductionSummary{
			CurrentStepID:       r.CurrentStepID,
			CurrentStepNameTH:   r.CurrentStepNameTH,
			CurrentUpdateStatus: r.CurrentUpdateStatus,
			CompletedCount:      r.CompletedCount,
			TotalCount:          r.TotalCount,
			LastUpdatedAt:       r.LastUpdatedAt,
			HasRejected:         r.HasRejected,
		},
		RFQID:               r.RFQID,
		RFQTitle:            r.RFQTitle,
		RFQQuantity:         r.RFQQuantity,
		UnitName:            r.UnitName,
		CustomerDisplayName: r.CustomerDisplayName,
		RequestKind:         r.RequestKind,
		OrderType:           orderTypeFromRequestKind(r.RequestKind),
	}
}

const orderListEnrichedSelect = `
	SELECT
		o.order_id,
		o.quote_id,
		o.customer_id AS user_id,
		o.factory_id,
		o.total_amount,
		o.deposit_amount,
		o.status,
		o.estimated_delivery,
		o.created_at,
		o.updated_at,
		r.rfq_id,
		COALESCE(r.request_kind, 'PR') AS request_kind,
		q.factory_highlight,
		COALESCE(r.title, '') AS rfq_title,
		r.quantity AS rfq_quantity,
		'' AS unit_name,
		COALESCE(NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), ''), 'ลูกค้า #' || o.customer_id::text) AS customer_display_name,
		COALESCE(cur.step_id, CASE WHEN o.status IN ('PD', 'PR') THEN 0 END) AS current_step_id,
		cur.step_name_th AS current_step_name_th,
		cur.status AS current_update_status,
		COALESCE(prod.completed_count, 0) AS completed_count,
		COALESCE(total_steps.total_count, 0) AS total_count,
		prod.last_updated_at,
		COALESCE(prod.has_rejected, FALSE) AS has_rejected
	FROM orders o
	INNER JOIN quotations q ON q.quote_id = o.quote_id
	INNER JOIN rfqs r ON r.rfq_id = q.rfq_id
	LEFT JOIN customers c ON c.user_id = o.customer_id
	LEFT JOIN LATERAL (
		SELECT
			pu.step_id,
			lp.step_name_th,
			pu.status,
			COALESCE(pu.last_updated_at, pu.created_at) AS updated_at,
			COALESCE(lp.sort_order, lp.step_id)::bigint AS sort_order
		FROM production_updates pu
		INNER JOIN lbi_production lp ON lp.step_id = pu.step_id
		WHERE pu.order_id = o.order_id
		  AND pu.status = 'CD'
		ORDER BY
			COALESCE(lp.sort_order, lp.step_id) DESC,
			COALESCE(pu.last_updated_at, pu.created_at) DESC
		LIMIT 1
	) cur ON TRUE
	LEFT JOIN LATERAL (
		SELECT
			COUNT(*) FILTER (WHERE pu.status = 'CD')::bigint AS completed_count,
			MAX(COALESCE(pu.last_updated_at, pu.created_at)) AS last_updated_at,
			BOOL_OR(pu.status = 'RJ') AS has_rejected
		FROM production_updates pu
		WHERE pu.order_id = o.order_id
	) prod ON TRUE
	CROSS JOIN LATERAL (
		SELECT COUNT(*)::bigint AS total_count
		FROM lbi_production
	) total_steps
`

func (r *OrderRepository) ListEnrichedByUserID(userID int64, status string, rfqID *int64, requestKinds []string) ([]domain.OrderListItem, error) {
	return r.listEnriched("o.customer_id = $1", userID, status, rfqID, requestKinds)
}

func (r *OrderRepository) ListEnrichedByFactoryID(factoryID int64, status string, rfqID *int64, requestKinds []string) ([]domain.OrderListItem, error) {
	return r.listEnriched("o.factory_id = $1", factoryID, status, rfqID, requestKinds)
}

func (r *OrderRepository) listEnriched(ownerClause string, ownerID int64, status string, rfqID *int64, requestKinds []string) ([]domain.OrderListItem, error) {
	query := orderListEnrichedSelect + " WHERE " + ownerClause
	args := []interface{}{ownerID}
	statuses := splitOrderStatuses(status)
	if len(statuses) == 1 {
		query += fmt.Sprintf(" AND o.status = $%d", len(args)+1)
		args = append(args, statuses[0])
	} else if len(statuses) > 1 {
		placeholders := make([]string, 0, len(statuses))
		for _, st := range statuses {
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, st)
		}
		query += " AND o.status IN (" + strings.Join(placeholders, ", ") + ")"
	}
	if rfqID != nil && *rfqID > 0 {
		query += fmt.Sprintf(" AND r.rfq_id = $%d", len(args)+1)
		args = append(args, *rfqID)
	}
	if len(requestKinds) > 0 {
		query += fmt.Sprintf(" AND COALESCE(r.request_kind, 'PR') = ANY($%d)", len(args)+1)
		args = append(args, pq.Array(requestKinds))
	}
	query += " ORDER BY o.created_at DESC"
	var rows []orderListRow
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, err
	}
	items := make([]domain.OrderListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.toDomain())
	}
	return items, nil
}

func orderTypeFromRequestKind(kind string) string {
	switch domainutil.NormalizeStatus(kind) {
	case "PS", "MS":
		return "sample"
	default:
		return "production"
	}
}

func (r *OrderRepository) GetByID(orderID, userID int64) (*domain.Order, error) {
	var row orderRow
	query := `
		SELECT order_id, quote_id, customer_id AS user_id, factory_id, total_amount, deposit_amount, status,
		       estimated_delivery, tracking_no, courier, NULL::timestamp AS shipped_at, created_at, updated_at
		FROM orders
		WHERE order_id = $1 AND customer_id = $2
	`
	if err := r.db.Get(&row, query, orderID, userID); err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *OrderRepository) UpdateStatus(orderID int64, status string) error {
	query := "UPDATE orders SET status = $1, updated_at = NOW() WHERE order_id = $2"
	_, err := r.db.Exec(query, status, orderID)
	return err
}

func (r *OrderRepository) UpdateStatusTx(tx *sqlx.Tx, orderID int64, status string) error {
	_, err := tx.Exec(`UPDATE orders SET status = $1, updated_at = NOW() WHERE order_id = $2`, status, orderID)
	return err
}

func (r *OrderRepository) ListByFactoryID(factoryID int64, status string) ([]domain.Order, error) {
	var rows []orderRow

	query := sq.Select(
		"order_id",
		"quote_id",
		"customer_id AS user_id",
		"factory_id",
		"total_amount",
		"deposit_amount",
		"status",
		"estimated_delivery",
		"tracking_no",
		"courier",
		"NULL::timestamp AS shipped_at",
		"created_at",
		"updated_at",
	).
		From("orders").
		Where(sq.Eq{"factory_id": factoryID})

	statuses := splitOrderStatuses(status)
	if len(statuses) > 0 {
		if len(statuses) == 1 {
			query = query.Where(sq.Eq{"status": statuses[0]})
		} else {
			statusInterfaces := make([]interface{}, len(statuses))
			for i, s := range statuses {
				statusInterfaces[i] = s
			}
			query = query.Where(sq.Eq{"status": statusInterfaces})
		}
	}

	query = query.OrderBy("created_at DESC")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	err = r.db.Select(&rows, sql, args...)
	if err != nil {
		return nil, err
	}
	orders := make([]domain.Order, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, *row.toDomain())
	}
	return orders, nil
}

func (r *OrderRepository) GetByParticipant(orderID, userID int64, role string) (*domain.Order, error) {
	var row orderRow
	query := `
		SELECT order_id, quote_id, customer_id AS user_id, factory_id, total_amount, deposit_amount, status,
		       estimated_delivery, tracking_no, courier, NULL::timestamp AS shipped_at, created_at, updated_at
		FROM orders
		WHERE order_id = $1
	`
	if err := r.db.Get(&row, query, orderID); err != nil {
		return nil, err
	}
	if role == "FT" {
		if row.FactoryID != userID {
			return nil, sql.ErrNoRows
		}
	} else {
		if row.UserID != userID {
			return nil, sql.ErrNoRows
		}
	}
	return row.toDomain(), nil
}

func (r *OrderRepository) GetDetailByParticipant(orderID, userID int64, role string) (*OrderDetailRow, error) {
	var item OrderDetailRow
	query := `
		SELECT
			o.order_id,
			o.quote_id,
			o.customer_id AS user_id,
			o.factory_id,
			o.total_amount,
			o.deposit_amount,
			o.status,
			o.payment_type,
			o.estimated_delivery,
			o.tracking_no,
			o.courier,
			NULL::timestamp AS shipped_at,
			o.created_at,
			o.updated_at,
			COALESCE(fp.factory_name, '') AS factory_name,
			q.price_per_piece,
			q.mold_cost,
			q.lead_time_days,
			r.rfq_id,
			COALESCE(r.title, '') AS rfq_title,
			r.details AS rfq_details,
			r.quantity AS rfq_quantity,
			COALESCE(r.target_price, 0) AS rfq_budget,
			r.created_at AS rfq_created_at,
			r.category_id AS rfq_category_id,
			cat.name AS rfq_category_name,
			COALESCE(r.request_kind, '') AS rfq_request_kind,
			(
				SELECT ps.due_date::timestamp
				FROM payment_schedules ps
				WHERE ps.order_id = o.order_id
				ORDER BY ps.installment_no ASC, ps.schedule_id ASC
				LIMIT 1
			) AS deposit_schedule_due,
			-- RFQ enrichment
			r.sub_category_id AS rfq_sub_category_id,
			subcat.name AS rfq_sub_category_name,
			rfq_sm.method_name AS rfq_shipping_method_name,
			r.material_grade AS rfq_material_grade,
			COALESCE(r.certifications_required, ARRAY[]::text[]) AS rfq_certifications,
			r.target_lead_time_days AS rfq_target_lead_time_days,
			r.target_price AS rfq_target_price,
			-- Delivery address
			addr.address_detail AS rfq_addr_detail,
			sd.name_th AS rfq_addr_sub_district,
			d.name_th AS rfq_addr_district,
			p.name_th AS rfq_addr_province,
			COALESCE(addr.zip_code, sd.zip_code) AS rfq_addr_zip_code,
			-- Customer / factory contact
			COALESCE(NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), ''), '') AS customer_name,
			COALESCE(uc.phone, '') AS customer_phone,
			COALESCE(uf.phone, '') AS factory_phone,
			COALESCE(pf.name_th, '') AS factory_province,
			-- Quotation enrichment
			COALESCE(q.grand_total, 0) AS quote_grand_total,
			COALESCE(q.subtotal, 0) AS quote_subtotal,
			COALESCE(q.discount_amount, 0) AS quote_discount_amount,
			COALESCE(q.shipping_cost, 0) AS quote_shipping_cost,
			COALESCE(q.packaging_cost, 0) AS quote_packaging_cost,
			COALESCE(q.vat_rate, 0) AS quote_vat_rate,
			COALESCE(q.vat_amount, 0) AS quote_vat_amount,
			COALESCE(q.validity_days, 0) AS quote_validity_days,
			TO_CHAR(COALESCE(q.valid_until, (q.create_time + (q.validity_days::text || ' day')::interval)::date), 'YYYY-MM-DD') AS quote_valid_until,
			q.payment_terms AS quote_payment_terms,
			COALESCE(q.image_urls::text, '[]') AS quote_image_urls,
			q.factory_highlight AS quote_factory_highlight,
			q.factory_note AS quote_factory_note
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		INNER JOIN rfqs r ON r.rfq_id = q.rfq_id
		LEFT JOIN lbi_categories cat ON cat.category_id = r.category_id
		LEFT JOIN lbi_categories subcat ON subcat.category_id = r.sub_category_id
		LEFT JOIN lbi_shipping_methods rfq_sm ON rfq_sm.shipping_method_id = r.shipping_method_id
		LEFT JOIN addresses addr ON addr.address_id = r.delivery_address_id
		LEFT JOIN lbi_sub_districts sd ON sd.row_id = addr.sub_district_id
		LEFT JOIN lbi_districts d ON d.row_id = addr.district_id
		LEFT JOIN lbi_provinces p ON p.row_id = addr.province_id
		LEFT JOIN factory_profiles fp ON fp.user_id = o.factory_id
		LEFT JOIN customers c ON c.user_id = o.customer_id
		LEFT JOIN users uc ON uc.user_id = o.customer_id
		LEFT JOIN users uf ON uf.user_id = o.factory_id
		LEFT JOIN lbi_provinces pf ON pf.row_id = fp.province_id
		WHERE o.order_id = $1
	`
	if err := r.db.Get(&item, query, orderID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *OrderRepository) GetRfqImages(rfqID int64) ([]domain.RfqImage, error) {
	type rfqImageRow struct {
		ReferenceImages pq.StringArray `db:"reference_images"`
	}
	var row rfqImageRow
	err := r.db.Get(&row, `
		SELECT
			COALESCE(reference_images, ARRAY[]::TEXT[]) AS reference_images
		FROM rfqs
		WHERE rfq_id = $1
	`, rfqID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	items := make([]domain.RfqImage, 0)
	counter := 1

	for _, rawURL := range row.ReferenceImages {
		url := strings.TrimSpace(rawURL)
		if url == "" {
			continue
		}
		if _, dup := seen[url]; dup {
			continue
		}
		seen[url] = struct{}{}
		items = append(items, domain.RfqImage{
			ImageID:  fmt.Sprintf("rfq-%d-image-%d", rfqID, counter),
			ImageURL: url,
		})
		counter++
	}

	return items, nil
}
func splitOrderStatuses(raw string) []string {
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

// GetByIDForUpdateTx locks an order row for atomic status transitions.
// MarkCompletedTx sets order status to CP and stores completed timestamp.
// UpsertCompletedStepTx writes production step_id=6 as completed.
// ListAutoCloseCandidates returns shipped orders older than cutoff without rejected production steps.

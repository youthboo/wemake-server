package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type OrderRepository struct {
	db *sqlx.DB
}

type QuotationOrderSource struct {
	QuotationID   int64   `db:"quote_id"`
	RFQID         int64   `db:"rfq_id"`
	UserID        int64   `db:"user_id"`
	FactoryID     int64   `db:"factory_id"`
	PricePerPiece float64 `db:"price_per_piece"`
	Quantity      int64   `db:"quantity"`
	MoldCost      float64 `db:"mold_cost"`
	LeadTimeDays  int64   `db:"lead_time_days"`
	Status        string  `db:"status"`
}

type OrderDetailRow struct {
	domain.Order
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
	RFQDeadline        *time.Time `db:"rfq_deadline"`
	RFQCreatedAt       time.Time  `db:"rfq_created_at"`
	RFQCategoryID      int64      `db:"rfq_category_id"`
	RFQCategoryName    *string    `db:"rfq_category_name"`
	RFQUnitID          int64      `db:"rfq_unit_id"`
	RFQUnitName        *string    `db:"rfq_unit_name"`
}

func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) GetOrderSourceByQuotationID(quotationID, userID int64) (*QuotationOrderSource, error) {
	var src QuotationOrderSource
	query := `
		SELECT q.quote_id, q.rfq_id, rfq.user_id, q.factory_id, q.price_per_piece, rfq.quantity, q.mold_cost, q.lead_time_days, q.status
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
		INSERT INTO orders (quote_id, user_id, factory_id, total_amount, deposit_amount, status, estimated_delivery, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
		nullableTimeValue(order.EstimatedDelivery),
		order.CreatedAt,
		order.UpdatedAt,
	).Scan(&order.OrderID)
}

// CreateTx inserts an order inside an existing transaction.
func (r *OrderRepository) CreateTx(tx *sqlx.Tx, order *domain.Order) error {
	query := `
		INSERT INTO orders (quote_id, user_id, factory_id, total_amount, deposit_amount, status, estimated_delivery, created_at, updated_at)
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
		nullableTimeValue(order.EstimatedDelivery),
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
		SELECT q.quote_id, q.rfq_id, rfq.user_id, q.factory_id, q.price_per_piece, rfq.quantity, q.mold_cost, q.lead_time_days, q.status
		FROM quotations q
		INNER JOIN rfqs rfq ON rfq.rfq_id = q.rfq_id
		WHERE q.quote_id = $1 AND rfq.user_id = $2
	`
	if err := tx.Get(&src, query, quotationID, userID); err != nil {
		return nil, err
	}
	return &src, nil
}

func (r *OrderRepository) ListByUserID(userID int64, status string) ([]domain.Order, error) {
	var orders []domain.Order
	query := `
		SELECT order_id, quote_id, user_id, factory_id, total_amount, deposit_amount, status, estimated_delivery, tracking_no, courier, shipped_at, created_at, updated_at
		FROM orders
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	statuses := splitOrderStatuses(status)
	if len(statuses) == 1 {
		query += " AND status = $2"
		args = append(args, statuses[0])
	} else if len(statuses) > 1 {
		placeholders := make([]string, 0, len(statuses))
		for _, st := range statuses {
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, st)
		}
		query += " AND status IN (" + strings.Join(placeholders, ", ") + ")"
	}
	query += " ORDER BY created_at DESC"
	err := r.db.Select(&orders, query, args...)
	return orders, err
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
		TotalAmount:       r.TotalAmount,
		DepositAmount:     r.DepositAmount,
		EstimatedDelivery: r.EstimatedDelivery,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
		RFQ: domain.OrderListRFQSummary{
			RFQID:    r.RFQID,
			Title:    r.RFQTitle,
			Quantity: r.RFQQuantity,
			UnitName: r.UnitName,
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
	}
}

const orderListEnrichedSelect = `
	SELECT
		o.order_id,
		o.quote_id,
		o.user_id,
		o.factory_id,
		o.total_amount,
		o.deposit_amount,
		o.status,
		o.estimated_delivery,
		o.created_at,
		o.updated_at,
		r.rfq_id,
		COALESCE(r.title, '') AS rfq_title,
		r.quantity AS rfq_quantity,
		COALESCE(lu.unit_name_th, '') AS unit_name,
		COALESCE(NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), ''), 'ลูกค้า #' || o.user_id::text) AS customer_display_name,
		cur.step_id AS current_step_id,
		cur.step_name_th AS current_step_name_th,
		cur.status AS current_update_status,
		COALESCE(prod.completed_count, 0) AS completed_count,
		COALESCE(total_steps.total_count, 0) AS total_count,
		prod.last_updated_at,
		COALESCE(prod.has_rejected, FALSE) AS has_rejected
	FROM orders o
	INNER JOIN quotations q ON q.quote_id = o.quote_id
	INNER JOIN rfqs r ON r.rfq_id = q.rfq_id
	LEFT JOIN lbi_units lu ON lu.unit_id = r.unit_id
	LEFT JOIN customers c ON c.user_id = o.user_id
	LEFT JOIN LATERAL (
		SELECT
			pu.step_id,
			lp.step_name_th,
			pu.status,
			COALESCE(pu.last_updated_at, pu.update_date, pu.created_at) AS updated_at,
			COALESCE(lp.sort_order, lp.sequence) AS sort_order
		FROM production_updates pu
		INNER JOIN lbi_production lp ON lp.step_id = pu.step_id
		WHERE pu.order_id = o.order_id
		  AND pu.status IN ('IP', 'CD', 'RJ')
		ORDER BY
			CASE pu.status WHEN 'IP' THEN 3 WHEN 'RJ' THEN 2 WHEN 'CD' THEN 1 ELSE 0 END DESC,
			COALESCE(lp.sort_order, lp.sequence) DESC,
			COALESCE(pu.last_updated_at, pu.update_date, pu.created_at) DESC
		LIMIT 1
	) cur ON TRUE
	LEFT JOIN LATERAL (
		SELECT
			COUNT(*) FILTER (WHERE pu.status = 'CD')::bigint AS completed_count,
			MAX(COALESCE(pu.last_updated_at, pu.update_date, pu.created_at)) AS last_updated_at,
			BOOL_OR(pu.status = 'RJ') AS has_rejected
		FROM production_updates pu
		WHERE pu.order_id = o.order_id
	) prod ON TRUE
	CROSS JOIN LATERAL (
		SELECT COUNT(*)::bigint AS total_count
		FROM lbi_production
		WHERE COALESCE(is_active, FALSE) = TRUE
	) total_steps
`

func (r *OrderRepository) ListEnrichedByUserID(userID int64, status string) ([]domain.OrderListItem, error) {
	return r.listEnriched("o.user_id = $1", userID, status)
}

func (r *OrderRepository) ListEnrichedByFactoryID(factoryID int64, status string) ([]domain.OrderListItem, error) {
	return r.listEnriched("o.factory_id = $1", factoryID, status)
}

func (r *OrderRepository) listEnriched(ownerClause string, ownerID int64, status string) ([]domain.OrderListItem, error) {
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

func (r *OrderRepository) GetByID(orderID, userID int64) (*domain.Order, error) {
	var order domain.Order
	query := `
		SELECT order_id, quote_id, user_id, factory_id, total_amount, deposit_amount, status, estimated_delivery, tracking_no, courier, shipped_at, created_at, updated_at
		FROM orders
		WHERE order_id = $1 AND user_id = $2
	`
	if err := r.db.Get(&order, query, orderID, userID); err != nil {
		return nil, err
	}
	return &order, nil
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
	var orders []domain.Order
	query := `
		SELECT order_id, quote_id, user_id, factory_id, total_amount, deposit_amount, status, estimated_delivery, tracking_no, courier, shipped_at, created_at, updated_at
		FROM orders
		WHERE factory_id = $1
	`
	args := []interface{}{factoryID}
	statuses := splitOrderStatuses(status)
	if len(statuses) == 1 {
		query += " AND status = $2"
		args = append(args, statuses[0])
	} else if len(statuses) > 1 {
		placeholders := make([]string, 0, len(statuses))
		for _, st := range statuses {
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
			args = append(args, st)
		}
		query += " AND status IN (" + strings.Join(placeholders, ", ") + ")"
	}
	query += " ORDER BY created_at DESC"
	err := r.db.Select(&orders, query, args...)
	return orders, err
}

func (r *OrderRepository) GetByParticipant(orderID, userID int64, role string) (*domain.Order, error) {
	var order domain.Order
	query := `
		SELECT order_id, quote_id, user_id, factory_id, total_amount, deposit_amount, status, estimated_delivery, tracking_no, courier, shipped_at, created_at, updated_at
		FROM orders
		WHERE order_id = $1
	`
	if err := r.db.Get(&order, query, orderID); err != nil {
		return nil, err
	}
	if role == "FT" {
		if order.FactoryID != userID {
			return nil, sql.ErrNoRows
		}
	} else {
		if order.UserID != userID {
			return nil, sql.ErrNoRows
		}
	}
	return &order, nil
}

func (r *OrderRepository) GetDetailByParticipant(orderID, userID int64, role string) (*OrderDetailRow, error) {
	var item OrderDetailRow
	query := `
		SELECT
			o.order_id,
			o.quote_id,
			o.user_id,
			o.factory_id,
			o.total_amount,
			o.deposit_amount,
			o.status,
			o.payment_type,
			o.estimated_delivery,
			o.tracking_no,
			o.courier,
			o.shipped_at,
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
			r.budget_per_piece AS rfq_budget,
			r.deadline_date AS rfq_deadline,
			r.created_at AS rfq_created_at,
			r.category_id AS rfq_category_id,
			cat.name AS rfq_category_name,
			r.unit_id AS rfq_unit_id,
			un.unit_name_th AS rfq_unit_name,
			(
				SELECT ps.due_date::timestamp
				FROM payment_schedules ps
				WHERE ps.order_id = o.order_id
				ORDER BY ps.installment_no ASC, ps.schedule_id ASC
				LIMIT 1
			) AS deposit_schedule_due
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		INNER JOIN rfqs r ON r.rfq_id = q.rfq_id
		LEFT JOIN categories cat ON cat.category_id = r.category_id
		LEFT JOIN lbi_units un ON un.unit_id = r.unit_id
		LEFT JOIN factory_profiles fp ON fp.user_id = o.factory_id
		WHERE o.order_id = $1
	`
	if err := r.db.Get(&item, query, orderID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *OrderRepository) GetRfqImages(rfqID int64) ([]domain.RfqImage, error) {
	var raw string
	err := r.db.Get(&raw, `
		SELECT COALESCE(image_urls, '[]'::jsonb)::text
		FROM rfqs
		WHERE rfq_id = $1
	`, rfqID)
	if err != nil {
		return nil, err
	}

	var urls []string
	if err := json.Unmarshal([]byte(raw), &urls); err != nil {
		return nil, err
	}
	items := make([]domain.RfqImage, 0, len(urls))
	for i, url := range urls {
		items = append(items, domain.RfqImage{
			ImageID:  fmt.Sprintf("rfq-%d-image-%d", rfqID, i+1),
			ImageURL: url,
		})
	}
	return items, nil
}

func (r *OrderRepository) InsertActivity(orderID int64, actorUserID *int64, eventCode string, payload map[string]interface{}) error {
	var payloadArg interface{}
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		payloadArg = b
	}
	_, err := r.db.Exec(
		`INSERT INTO order_activity_log (order_id, actor_user_id, event_code, payload) VALUES ($1, $2, $3, $4)`,
		orderID, actorUserID, eventCode, payloadArg,
	)
	return err
}

// InsertActivityTx writes an order activity row inside an existing transaction.
func (r *OrderRepository) InsertActivityTx(tx *sqlx.Tx, orderID int64, actorUserID *int64, eventCode string, payload map[string]interface{}) error {
	var payloadArg interface{}
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		payloadArg = b
	}
	_, err := tx.Exec(
		`INSERT INTO order_activity_log (order_id, actor_user_id, event_code, payload) VALUES ($1, $2, $3, $4)`,
		orderID, actorUserID, eventCode, payloadArg,
	)
	return err
}

func (r *OrderRepository) ListActivity(orderID int64) ([]domain.OrderActivityEntry, error) {
	var rows []domain.OrderActivityEntry
	err := r.db.Select(&rows, `
		SELECT activity_id, order_id, actor_user_id, event_code, payload, created_at
		FROM order_activity_log
		WHERE order_id = $1
		ORDER BY created_at ASC
	`, orderID)
	return rows, err
}

func splitOrderStatuses(raw string) []string {
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

func (r *OrderRepository) MarkShipped(orderID, factoryID int64, trackingNo, courier string) error {
	res, err := r.db.Exec(`
		UPDATE orders
		SET status = 'SH',
		    tracking_no = $1,
		    courier = $2,
		    shipped_at = NOW(),
		    updated_at = NOW()
		WHERE order_id = $3 AND factory_id = $4
	`, trackingNo, courier, orderID, factoryID)
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

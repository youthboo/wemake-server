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
			un.name AS rfq_unit_name,
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
		LEFT JOIN units un ON un.unit_id = r.unit_id
		LEFT JOIN factory_profiles fp ON fp.user_id = o.factory_id
		WHERE o.order_id = $1
	`
	if err := r.db.Get(&item, query, orderID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *OrderRepository) GetRfqImages(rfqID int64) ([]domain.RfqImage, error) {
	var items []domain.RfqImage
	err := r.db.Select(&items, `
		SELECT image_id, image_url
		FROM rfq_images
		WHERE rfq_id = $1
		ORDER BY image_id
	`, rfqID)
	return items, err
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

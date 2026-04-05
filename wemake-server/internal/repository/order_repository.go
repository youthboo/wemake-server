package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type OrderRepository struct {
	db *sqlx.DB
}

type QuotationOrderSource struct {
	QuotationID   int64   `db:"quote_id"`
	UserID        int64   `db:"user_id"`
	FactoryID     int64   `db:"factory_id"`
	PricePerPiece float64 `db:"price_per_piece"`
	Quantity      int64   `db:"quantity"`
	MoldCost      float64 `db:"mold_cost"`
	LeadTimeDays  int64   `db:"lead_time_days"`
	Status        string  `db:"status"`
}

func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) GetOrderSourceByQuotationID(quotationID, userID int64) (*QuotationOrderSource, error) {
	var src QuotationOrderSource
	query := `
		SELECT q.quote_id, rfq.user_id, q.factory_id, q.price_per_piece, rfq.quantity, q.mold_cost, q.lead_time_days, q.status
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

func (r *OrderRepository) ListByUserID(userID int64, status string) ([]domain.Order, error) {
	var orders []domain.Order
	query := `
		SELECT order_id, quote_id, user_id, factory_id, total_amount, deposit_amount, status, estimated_delivery, created_at, updated_at
		FROM orders
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	err := r.db.Select(&orders, query, args...)
	return orders, err
}

func (r *OrderRepository) GetByID(orderID, userID int64) (*domain.Order, error) {
	var order domain.Order
	query := `
		SELECT order_id, quote_id, user_id, factory_id, total_amount, deposit_amount, status, estimated_delivery, created_at, updated_at
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

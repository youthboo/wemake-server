package frontend

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	sessionRFQLimit    = 10
	sessionOrderLimit  = 10
	sessionThreadLimit = 20
)

type SessionRepository struct {
	db *sqlx.DB
}

func NewSessionRepository(db *sqlx.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

type SessionUserRow struct {
	ID             int64          `db:"id"`
	Role           string         `db:"role"`
	FirstName      sql.NullString `db:"first_name"`
	LastName       sql.NullString `db:"last_name"`
	FactoryName    sql.NullString `db:"factory_name"`
	Email          string         `db:"email"`
	Phone          sql.NullString `db:"phone"`
	MemberSince    string         `db:"member_since"`
	Balance        float64        `db:"balance"`
	PendingBalance float64        `db:"pending_balance"`
}

type SessionRFQRow struct {
	RFQID        int64           `db:"rfq_id"`
	Title        string          `db:"title"`
	Status       string          `db:"status"`
	Quantity     sql.NullInt64   `db:"quantity"`
	TargetPrice  sql.NullFloat64 `db:"target_price"`
	CategoryName string          `db:"category_name"`
	CreatedAt    string          `db:"created_at"`
	OfferCount   int64           `db:"offer_count"`
}

type SessionOfferRow struct {
	QuoteID         int64           `db:"quote_id"`
	RFQID           int64           `db:"rfq_id"`
	FactoryID       int64           `db:"factory_id"`
	FactoryName     string          `db:"factory_name"`
	FactoryImageURL sql.NullString  `db:"factory_image_url"`
	FactoryVerified bool            `db:"factory_verified"`
	PricePerPiece   sql.NullFloat64 `db:"price_per_piece"`
	GrandTotal      sql.NullFloat64 `db:"grand_total"`
	LeadTimeDays    sql.NullInt64   `db:"lead_time_days"`
	Status          string          `db:"status"`
}

type SessionOrderRow struct {
	OrderID           int64           `db:"order_id"`
	Title             string          `db:"title"`
	FactoryID         int64           `db:"factory_id"`
	FactoryName       string          `db:"factory_name"`
	FactoryImageURL   sql.NullString  `db:"factory_image_url"`
	Status            string          `db:"status"`
	TotalAmount       sql.NullFloat64 `db:"total_amount"`
	DepositAmount     sql.NullFloat64 `db:"deposit_amount"`
	EstimatedDelivery sql.NullString  `db:"estimated_delivery"`
	CreatedAt         string          `db:"created_at"`
}

type SessionThreadRow struct {
	ConvID              int64          `db:"conv_id"`
	CounterpartID       int64          `db:"counterpart_id"`
	CounterpartName     string         `db:"counterpart_name"`
	CounterpartImageURL sql.NullString `db:"counterpart_image_url"`
	LastMessage         sql.NullString `db:"last_message"`
	LastMessageAt       string         `db:"last_message_at"`
	Unread              int64          `db:"unread"`
}

func (r *SessionRepository) GetUser(userID int64) (*SessionUserRow, error) {
	var row SessionUserRow
	err := r.db.Get(&row, `
		SELECT
			u.user_id AS id,
			u.role,
			c.first_name,
			c.last_name,
			fp.factory_name,
			u.email,
			u.phone,
			TO_CHAR(u.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS member_since,
			COALESCE(w.good_fund, 0)::float8    AS balance,
			COALESCE(w.pending_fund, 0)::float8 AS pending_balance
		FROM users u
		LEFT JOIN customers c ON c.user_id = u.user_id
		LEFT JOIN factory_profiles fp ON fp.user_id = u.user_id
		LEFT JOIN wallets w ON w.user_id = u.user_id
		WHERE u.user_id = $1 AND u.is_active = TRUE
		LIMIT 1
	`, userID)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *SessionRepository) ListRFQsByUserID(userID int64) ([]SessionRFQRow, error) {
	var rows []SessionRFQRow
	err := r.db.Select(&rows, `
		SELECT
			r.rfq_id,
			r.title,
			r.status,
			r.quantity,
			r.target_price,
			c.name AS category_name,
			TO_CHAR(r.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
			COUNT(q.quote_id) AS offer_count
		FROM rfqs r
		JOIN lbi_categories c ON c.category_id = r.category_id
		LEFT JOIN quotations q ON q.rfq_id = r.rfq_id
		WHERE r.user_id = $1 AND r.status != 'CC'
		GROUP BY r.rfq_id, r.title, r.status, r.quantity, r.target_price, c.name, r.created_at
		ORDER BY r.created_at DESC
		LIMIT $2
	`, userID, sessionRFQLimit)
	return rows, err
}

func (r *SessionRepository) ListOffersForRFQs(userID int64, rfqIDs []int64) ([]SessionOfferRow, error) {
	if len(rfqIDs) == 0 {
		return []SessionOfferRow{}, nil
	}
	var rows []SessionOfferRow
	err := r.db.Select(&rows, `
		SELECT
			q.quote_id,
			q.rfq_id,
			q.factory_id,
			fp.factory_name,
			fp.image_url                AS factory_image_url,
			(fp.approval_status = 'AP') AS factory_verified,
			q.price_per_piece,
			((q.price_per_piece * r.quantity) + COALESCE(q.mold_cost, 0)) AS grand_total,
			q.lead_time_days,
			q.status
		FROM quotations q
		JOIN rfqs r ON r.rfq_id = q.rfq_id
		JOIN factory_profiles fp ON fp.user_id = q.factory_id
		WHERE r.user_id = $1 AND r.status != 'CC' AND r.rfq_id = ANY($2)
		ORDER BY q.rfq_id, q.create_time DESC
	`, userID, pq.Array(rfqIDs))
	return rows, err
}

func (r *SessionRepository) ListOrdersByUserID(userID int64) ([]SessionOrderRow, error) {
	var rows []SessionOrderRow
	err := r.db.Select(&rows, `
		SELECT
			o.order_id,
			rfq.title,
			o.factory_id,
			fp.factory_name,
			fp.image_url AS factory_image_url,
			o.status,
			o.total_amount,
			o.deposit_amount,
			TO_CHAR(
				COALESCE(
					o.estimated_delivery::timestamp,
					(o.created_at + (q.lead_time_days || ' days')::interval)
				), 'YYYY-MM-DD'
			) AS estimated_delivery,
			TO_CHAR(o.created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at
		FROM orders o
		JOIN quotations q ON q.quote_id = o.quote_id
		JOIN rfqs rfq ON rfq.rfq_id = q.rfq_id
		JOIN factory_profiles fp ON fp.user_id = o.factory_id
		WHERE o.customer_id = $1 AND o.status NOT IN ('CP', 'CN', 'CL')
		ORDER BY o.created_at DESC
		LIMIT $2
	`, userID, sessionOrderLimit)
	return rows, err
}

func (r *SessionRepository) ListThreadsByUserID(userID int64) ([]SessionThreadRow, error) {
	var rows []SessionThreadRow
	err := r.db.Select(&rows, `
		SELECT
			c.conv_id,
			c.factory_id                  AS counterpart_id,
			fp.factory_name               AS counterpart_name,
			fp.image_url                  AS counterpart_image_url,
			c.last_message,
			TO_CHAR(c.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS last_message_at,
			COALESCE(c.unread_customer, 0) AS unread
		FROM conversations c
		JOIN factory_profiles fp ON fp.user_id = c.factory_id
		WHERE c.customer_id = $1
		ORDER BY c.updated_at DESC
		LIMIT $2
	`, userID, sessionThreadLimit)
	return rows, err
}

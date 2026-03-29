package repository

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type FrontendRepository struct {
	db *sqlx.DB
}

type FrontendCurrentUserRow struct {
	ID             int64           `db:"id"`
	Role           string          `db:"role"`
	FirstName      sql.NullString  `db:"first_name"`
	LastName       sql.NullString  `db:"last_name"`
	FactoryName    sql.NullString  `db:"factory_name"`
	Email          string          `db:"email"`
	Phone          sql.NullString  `db:"phone"`
	WalletBalance  sql.NullFloat64 `db:"wallet_balance"`
	PendingBalance sql.NullFloat64 `db:"pending_balance"`
	MemberSince    string          `db:"member_since"`
}

type FrontendCategoryRow struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}

type FrontendFactoryRow struct {
	ID              int64           `db:"id"`
	Name            string          `db:"name"`
	Location        sql.NullString  `db:"location"`
	Specialization  sql.NullString  `db:"specialization"`
	Verified        bool            `db:"verified"`
	CompletedOrders int64           `db:"completed_orders"`
	AverageLeadDays sql.NullFloat64 `db:"average_lead_days"`
	Description     sql.NullString  `db:"description"`
}

type FrontendFactoryDetailRow struct {
	ID              int64           `db:"id"`
	Name            string          `db:"name"`
	Location        sql.NullString  `db:"location"`
	Specialization  sql.NullString  `db:"specialization"`
	Verified        bool            `db:"verified"`
	CompletedOrders int64           `db:"completed_orders"`
	AverageLeadDays sql.NullFloat64 `db:"average_lead_days"`
	Description     sql.NullString  `db:"description"`
	AddressDetail   sql.NullString  `db:"address_detail"`
	ProvinceName    sql.NullString  `db:"province_name"`
	Email           string          `db:"email"`
	Phone           sql.NullString  `db:"phone"`
}

type FrontendRFQRow struct {
	ID          int64   `db:"id"`
	ProjectName string  `db:"project_name"`
	Category    string  `db:"category"`
	Status      string  `db:"status"`
	OfferCount  int64   `db:"offer_count"`
	Budget      float64 `db:"budget"`
	Quantity    int64   `db:"quantity"`
	CreatedAt   string  `db:"created_at"`
	Description string  `db:"description"`
}

type FrontendQuotationRow struct {
	ID              int64   `db:"id"`
	FactoryID       int64   `db:"factory_id"`
	FactoryName     string  `db:"factory_name"`
	Verified        bool    `db:"verified"`
	CompletedOrders int64   `db:"completed_orders"`
	LeadTime        int64   `db:"lead_time"`
	Status          string  `db:"status"`
	TotalPrice      float64 `db:"total_price"`
}

type FrontendImageRow struct {
	ImageURL string `db:"image_url"`
}

type FrontendOrderRow struct {
	ID                int64   `db:"id"`
	ProjectName       string  `db:"project_name"`
	RFQID             int64   `db:"rfq_id"`
	FactoryID         int64   `db:"factory_id"`
	FactoryName       string  `db:"factory_name"`
	TotalAmount       float64 `db:"total_amount"`
	DepositPaid       float64 `db:"deposit_paid"`
	Status            string  `db:"status"`
	EstimatedDelivery string  `db:"estimated_delivery"`
	CreatedAt         string  `db:"created_at"`
}

type FrontendOrderTimelineRow struct {
	ID          int64          `db:"id"`
	Title       sql.NullString `db:"title"`
	Date        string         `db:"date"`
	Description sql.NullString `db:"description"`
	Photo       sql.NullString `db:"photo"`
}

type FrontendMessageThreadRow struct {
	ReferenceType string `db:"reference_type"`
	ReferenceID   string `db:"reference_id"`
	LastMessage   string `db:"last_message"`
	LastMessageAt string `db:"last_message_at"`
	CounterpartID int64  `db:"counterpart_id"`
}

type FrontendMessageRow struct {
	MessageID     string         `db:"message_id"`
	ReferenceType string         `db:"reference_type"`
	ReferenceID   string         `db:"reference_id"`
	SenderID      int64          `db:"sender_id"`
	ReceiverID    int64          `db:"receiver_id"`
	Content       string         `db:"content"`
	AttachmentURL sql.NullString `db:"attachment_url"`
	CreatedAt     string         `db:"created_at"`
}

type FrontendUserLabelRow struct {
	Name string `db:"name"`
}

type FrontendReferenceLabelRow struct {
	ProjectName string `db:"project_name"`
	HasQuote    bool   `db:"has_quote"`
}

func NewFrontendRepository(db *sqlx.DB) *FrontendRepository {
	return &FrontendRepository{db: db}
}

func (r *FrontendRepository) GetCurrentUser(userID int64) (*FrontendCurrentUserRow, error) {
	var item FrontendCurrentUserRow
	query := `
		SELECT
			u.user_id AS id,
			u.role,
			c.first_name,
			c.last_name,
			fp.factory_name,
			u.email,
			u.phone,
			COALESCE(w.good_fund, 0) AS wallet_balance,
			COALESCE(w.pending_fund, 0) AS pending_balance,
			TO_CHAR(u.created_at, 'YYYY') AS member_since
		FROM users u
		LEFT JOIN customers c ON c.user_id = u.user_id
		LEFT JOIN factory_profiles fp ON fp.user_id = u.user_id
		LEFT JOIN wallets w ON w.user_id = u.user_id
		WHERE u.user_id = $1
	`
	if err := r.db.Get(&item, query, userID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *FrontendRepository) ListCategories() ([]FrontendCategoryRow, error) {
	var items []FrontendCategoryRow
	query := `SELECT category_id AS id, name FROM categories ORDER BY name`
	err := r.db.Select(&items, query)
	return items, err
}

func (r *FrontendRepository) ListFactories() ([]FrontendFactoryRow, error) {
	var items []FrontendFactoryRow
	query := `
		SELECT
			u.user_id AS id,
			fp.factory_name AS name,
			p.name_th AS location,
			ft.type_name AS specialization,
			COALESCE(fp.tax_id, '') <> '' AS verified,
			COALESCE(completed.completed_orders, 0) AS completed_orders,
			lead.average_lead_days,
			'' AS description
		FROM users u
		INNER JOIN factory_profiles fp ON fp.user_id = u.user_id
		LEFT JOIN lbi_factory_types ft ON ft.factory_type_id = fp.factory_type_id
		LEFT JOIN addresses a ON a.user_id = u.user_id AND a.is_default = TRUE
		LEFT JOIN lbi_provinces p ON p.row_id = a.province_id
		LEFT JOIN (
			SELECT factory_id, COUNT(*) AS completed_orders
			FROM orders
			WHERE status = 'CP'
			GROUP BY factory_id
		) completed ON completed.factory_id = u.user_id
		LEFT JOIN (
			SELECT factory_id, AVG(lead_time_days)::float AS average_lead_days
			FROM quotations
			GROUP BY factory_id
		) lead ON lead.factory_id = u.user_id
		WHERE u.role = 'FT' AND u.is_active = TRUE
		ORDER BY completed_orders DESC, fp.factory_name ASC
	`
	err := r.db.Select(&items, query)
	return items, err
}

func (r *FrontendRepository) GetFactoryDetail(factoryID int64) (*FrontendFactoryDetailRow, error) {
	var item FrontendFactoryDetailRow
	query := `
		SELECT
			u.user_id AS id,
			fp.factory_name AS name,
			p.name_th AS location,
			ft.type_name AS specialization,
			COALESCE(fp.tax_id, '') <> '' AS verified,
			COALESCE(completed.completed_orders, 0) AS completed_orders,
			lead.average_lead_days,
			'' AS description,
			a.address_detail,
			p.name_th AS province_name,
			u.email,
			u.phone
		FROM users u
		INNER JOIN factory_profiles fp ON fp.user_id = u.user_id
		LEFT JOIN lbi_factory_types ft ON ft.factory_type_id = fp.factory_type_id
		LEFT JOIN addresses a ON a.user_id = u.user_id AND a.is_default = TRUE
		LEFT JOIN lbi_provinces p ON p.row_id = a.province_id
		LEFT JOIN (
			SELECT factory_id, COUNT(*) AS completed_orders
			FROM orders
			WHERE status = 'CP'
			GROUP BY factory_id
		) completed ON completed.factory_id = u.user_id
		LEFT JOIN (
			SELECT factory_id, AVG(lead_time_days)::float AS average_lead_days
			FROM quotations
			GROUP BY factory_id
		) lead ON lead.factory_id = u.user_id
		WHERE u.user_id = $1 AND u.role = 'FT'
	`
	if err := r.db.Get(&item, query, factoryID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *FrontendRepository) ListRFQsByUserID(userID int64) ([]FrontendRFQRow, error) {
	var items []FrontendRFQRow
	query := `
		SELECT
			r.rfq_id AS id,
			r.title AS project_name,
			c.name AS category,
			r.status,
			COUNT(q.quote_id) AS offer_count,
			(r.budget_per_piece * r.quantity) AS budget,
			r.quantity,
			TO_CHAR(r.created_at, 'YYYY-MM-DD') AS created_at,
			COALESCE(r.details, '') AS description
		FROM rfqs r
		INNER JOIN categories c ON c.category_id = r.category_id
		LEFT JOIN quotations q ON q.rfq_id = r.rfq_id
		WHERE r.user_id = $1
		GROUP BY r.rfq_id, r.title, c.name, r.status, r.budget_per_piece, r.quantity, r.created_at, r.details
		ORDER BY r.created_at DESC
	`
	err := r.db.Select(&items, query, userID)
	return items, err
}

func (r *FrontendRepository) GetRFQByUserID(userID, rfqID int64) (*FrontendRFQRow, error) {
	var item FrontendRFQRow
	query := `
		SELECT
			r.rfq_id AS id,
			r.title AS project_name,
			c.name AS category,
			r.status,
			COUNT(q.quote_id) AS offer_count,
			(r.budget_per_piece * r.quantity) AS budget,
			r.quantity,
			TO_CHAR(r.created_at, 'YYYY-MM-DD') AS created_at,
			COALESCE(r.details, '') AS description
		FROM rfqs r
		INNER JOIN categories c ON c.category_id = r.category_id
		LEFT JOIN quotations q ON q.rfq_id = r.rfq_id
		WHERE r.user_id = $1 AND r.rfq_id = $2
		GROUP BY r.rfq_id, r.title, c.name, r.status, r.budget_per_piece, r.quantity, r.created_at, r.details
	`
	if err := r.db.Get(&item, query, userID, rfqID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *FrontendRepository) ListQuotationsByRFQID(rfqID int64) ([]FrontendQuotationRow, error) {
	var items []FrontendQuotationRow
	query := `
		SELECT
			q.quote_id AS id,
			q.factory_id,
			fp.factory_name,
			COALESCE(fp.tax_id, '') <> '' AS verified,
			COALESCE(completed.completed_orders, 0) AS completed_orders,
			q.lead_time_days AS lead_time,
			q.status,
			((q.price_per_piece * rfq.quantity) + q.mold_cost) AS total_price
		FROM quotations q
		INNER JOIN rfqs rfq ON rfq.rfq_id = q.rfq_id
		INNER JOIN factory_profiles fp ON fp.user_id = q.factory_id
		LEFT JOIN (
			SELECT factory_id, COUNT(*) AS completed_orders
			FROM orders
			WHERE status = 'CP'
			GROUP BY factory_id
		) completed ON completed.factory_id = q.factory_id
		WHERE q.rfq_id = $1
		ORDER BY total_price ASC, q.lead_time_days ASC
	`
	err := r.db.Select(&items, query, rfqID)
	return items, err
}

func (r *FrontendRepository) ListRFQImages(rfqID int64) ([]FrontendImageRow, error) {
	var items []FrontendImageRow
	query := `SELECT image_url FROM rfq_images WHERE rfq_id = $1 ORDER BY image_id`
	err := r.db.Select(&items, query, rfqID)
	return items, err
}

func (r *FrontendRepository) ListOrdersByUserID(userID int64) ([]FrontendOrderRow, error) {
	var items []FrontendOrderRow
	query := `
		SELECT
			o.order_id AS id,
			rfq.title AS project_name,
			rfq.rfq_id,
			o.factory_id,
			fp.factory_name,
			o.total_amount,
			o.deposit_amount AS deposit_paid,
			o.status,
			TO_CHAR((o.created_at + (q.lead_time_days || ' days')::interval), 'YYYY-MM-DD') AS estimated_delivery,
			TO_CHAR(o.created_at, 'YYYY-MM-DD') AS created_at
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		INNER JOIN rfqs rfq ON rfq.rfq_id = q.rfq_id
		INNER JOIN factory_profiles fp ON fp.user_id = o.factory_id
		WHERE o.user_id = $1
		ORDER BY o.created_at DESC
	`
	err := r.db.Select(&items, query, userID)
	return items, err
}

func (r *FrontendRepository) GetOrderByUserID(userID, orderID int64) (*FrontendOrderRow, error) {
	var item FrontendOrderRow
	query := `
		SELECT
			o.order_id AS id,
			rfq.title AS project_name,
			rfq.rfq_id,
			o.factory_id,
			fp.factory_name,
			o.total_amount,
			o.deposit_amount AS deposit_paid,
			o.status,
			TO_CHAR((o.created_at + (q.lead_time_days || ' days')::interval), 'YYYY-MM-DD') AS estimated_delivery,
			TO_CHAR(o.created_at, 'YYYY-MM-DD') AS created_at
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		INNER JOIN rfqs rfq ON rfq.rfq_id = q.rfq_id
		INNER JOIN factory_profiles fp ON fp.user_id = o.factory_id
		WHERE o.user_id = $1 AND o.order_id = $2
	`
	if err := r.db.Get(&item, query, userID, orderID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *FrontendRepository) ListOrderTimeline(orderID int64) ([]FrontendOrderTimelineRow, error) {
	var items []FrontendOrderTimelineRow
	query := `
		SELECT
			pu.update_id AS id,
			lp.step_name AS title,
			TO_CHAR(pu.created_at, 'YYYY-MM-DD') AS date,
			pu.description,
			pu.image_url AS photo
		FROM production_updates pu
		LEFT JOIN lbi_production lp ON lp.step_id = pu.step_id
		WHERE pu.order_id = $1
		ORDER BY pu.created_at ASC
	`
	err := r.db.Select(&items, query, orderID)
	return items, err
}

func (r *FrontendRepository) ListMessageThreads(userID int64) ([]FrontendMessageThreadRow, error) {
	var items []FrontendMessageThreadRow
	query := `
		SELECT
			m.reference_type,
			m.reference_id,
			m.content AS last_message,
			TO_CHAR(m.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') AS last_message_at,
			CASE
				WHEN m.sender_id = $1 THEN m.receiver_id
				ELSE m.sender_id
			END AS counterpart_id
		FROM messages m
		INNER JOIN (
			SELECT reference_type, reference_id, MAX(created_at) AS max_created_at
			FROM messages
			WHERE sender_id = $1 OR receiver_id = $1
			GROUP BY reference_type, reference_id
		) latest
			ON latest.reference_type = m.reference_type
			AND latest.reference_id = m.reference_id
			AND latest.max_created_at = m.created_at
		ORDER BY m.created_at DESC
	`
	err := r.db.Select(&items, query, userID)
	return items, err
}

func (r *FrontendRepository) GetUserLabel(userID int64) (*FrontendUserLabelRow, error) {
	var item FrontendUserLabelRow
	query := `
		SELECT COALESCE(fp.factory_name, CONCAT_WS(' ', c.first_name, c.last_name), u.email) AS name
		FROM users u
		LEFT JOIN factory_profiles fp ON fp.user_id = u.user_id
		LEFT JOIN customers c ON c.user_id = u.user_id
		WHERE u.user_id = $1
	`
	if err := r.db.Get(&item, query, userID); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *FrontendRepository) GetReferenceLabel(referenceType, referenceID string) (*FrontendReferenceLabelRow, error) {
	var item FrontendReferenceLabelRow
	switch referenceType {
	case "RFQ":
		query := `
			SELECT
				r.title AS project_name,
				EXISTS(SELECT 1 FROM quotations q WHERE q.rfq_id = r.rfq_id) AS has_quote
			FROM rfqs r
			WHERE r.rfq_id::text = $1
		`
		if err := r.db.Get(&item, query, referenceID); err != nil {
			return nil, err
		}
	case "ORDER":
		query := `
			SELECT
				r.title AS project_name,
				TRUE AS has_quote
			FROM orders o
			INNER JOIN quotations q ON q.quote_id = o.quote_id
			INNER JOIN rfqs r ON r.rfq_id = q.rfq_id
			WHERE o.order_id::text = $1
		`
		if err := r.db.Get(&item, query, referenceID); err != nil {
			return nil, err
		}
	default:
		return &FrontendReferenceLabelRow{}, nil
	}
	return &item, nil
}

func (r *FrontendRepository) ListMessagesByReference(referenceType, referenceID string, userID int64) ([]FrontendMessageRow, error) {
	var items []FrontendMessageRow
	query := `
		SELECT
			message_id,
			reference_type,
			reference_id,
			sender_id,
			receiver_id,
			content,
			attachment_url,
			TO_CHAR(created_at, 'HH24:MI') AS created_at
		FROM messages
		WHERE reference_type = $1
		  AND reference_id = $2
		  AND (sender_id = $3 OR receiver_id = $3)
		ORDER BY created_at ASC
	`
	err := r.db.Select(&items, query, referenceType, referenceID, userID)
	return items, err
}

func (r *FrontendRepository) GetProducts(limit int, categoryID string) ([]domain.Product, error) {
	var items []domain.Product
	var err error

	if categoryID != "" {
		query := `SELECT id, title, price, image_url, discount, factory_id, category_id FROM products WHERE category_id = $1 ORDER BY id ASC LIMIT $2`
		err = r.db.Select(&items, query, categoryID, limit)
	} else {
		query := `SELECT id, title, price, image_url, discount, factory_id, category_id FROM products ORDER BY id ASC LIMIT $1`
		err = r.db.Select(&items, query, limit)
	}

	if err == sql.ErrNoRows {
		return []domain.Product{}, nil
	}
	return items, err
}

func (r *FrontendRepository) GetPromotions(limit int) ([]domain.Promotion, error) {
	var items []domain.Promotion
	query := `SELECT id, title, description, price, image_url, tag, factory_id FROM promotions ORDER BY id ASC LIMIT $1`
	err := r.db.Select(&items, query, limit)
	if err == sql.ErrNoRows {
		return []domain.Promotion{}, nil
	}
	return items, err
}

func (r *FrontendRepository) GetPromoCodes() ([]domain.PromoCode, error) {
	var items []domain.PromoCode
	query := `SELECT id, title, subtitle, code, valid_until FROM promo_codes ORDER BY id ASC`
	err := r.db.Select(&items, query)
	if err == sql.ErrNoRows {
		return []domain.PromoCode{}, nil
	}
	return items, err
}

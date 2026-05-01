package repository

import (
	"database/sql"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type CustomerAdminRepository struct {
	db *sqlx.DB
}

func NewCustomerAdminRepository(db *sqlx.DB) *CustomerAdminRepository {
	return &CustomerAdminRepository{db: db}
}

func (r *CustomerAdminRepository) ListCustomers(search string, isActive *bool, limit, offset int) ([]domain.AdminCustomerListItem, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	search = strings.TrimSpace(search)

	countQuery := `
		SELECT COUNT(DISTINCT u.user_id)
		FROM users u
		LEFT JOIN customers c ON c.user_id = u.user_id
		WHERE u.role = 'CT'
		  AND ($1 = '' OR u.email ILIKE '%' || $1 || '%'
		               OR c.first_name ILIKE '%' || $1 || '%'
		               OR c.last_name  ILIKE '%' || $1 || '%')
		  AND ($2::boolean IS NULL OR u.is_active = $2)
	`
	var total int
	if err := r.db.Get(&total, countQuery, search, isActive); err != nil {
		return nil, 0, err
	}

	dataQuery := `
		SELECT
			u.user_id,
			u.email,
			COALESCE(c.first_name, '')                                    AS first_name,
			COALESCE(c.last_name, '')                                     AS last_name,
			COALESCE(u.phone, '')                                         AS phone,
			u.is_active,
			COUNT(DISTINCT o.order_id)::int                               AS total_orders,
			COALESCE(SUM(o.grand_total) FILTER (WHERE o.status <> 'CA'), 0) AS total_spend,
			COALESCE(w.good_fund, 0) + COALESCE(w.pending_fund, 0)       AS wallet_balance,
			u.created_at::text                                            AS created_at
		FROM users u
		LEFT JOIN customers c ON c.user_id = u.user_id
		LEFT JOIN orders o    ON o.user_id = u.user_id
		LEFT JOIN wallets w   ON w.user_id = u.user_id
		WHERE u.role = 'CT'
		  AND ($1 = '' OR u.email ILIKE '%' || $1 || '%'
		               OR c.first_name ILIKE '%' || $1 || '%'
		               OR c.last_name  ILIKE '%' || $1 || '%')
		  AND ($2::boolean IS NULL OR u.is_active = $2)
		GROUP BY u.user_id, u.email, c.first_name, c.last_name, u.phone,
		         u.is_active, w.good_fund, w.pending_fund, u.created_at
		ORDER BY total_spend DESC, u.created_at DESC
		LIMIT $3 OFFSET $4
	`
	var items []domain.AdminCustomerListItem
	if err := r.db.Select(&items, dataQuery, search, isActive, limit, offset); err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []domain.AdminCustomerListItem{}
	}
	return items, total, nil
}

func (r *CustomerAdminRepository) GetCustomerDetail(userID int64) (*domain.AdminCustomerDetail, error) {
	var item domain.AdminCustomerDetail
	err := r.db.Get(&item, `
		SELECT
			u.user_id,
			u.email,
			COALESCE(c.first_name, '')                                    AS first_name,
			COALESCE(c.last_name, '')                                     AS last_name,
			COALESCE(u.phone, '')                                         AS phone,
			COALESCE(NULLIF(TRIM(CONCAT_WS(', ',
				NULLIF(c.address_line1, ''),
				NULLIF(c.sub_district, ''),
				NULLIF(c.district, ''),
				NULLIF(c.province, ''),
				NULLIF(c.postal_code, '')
			)), ''), '')                                                   AS address,
			u.is_active,
			COUNT(DISTINCT o.order_id)::int                               AS total_orders,
			COALESCE(SUM(o.grand_total) FILTER (WHERE o.status <> 'CA'), 0) AS total_spend,
			u.created_at::text                                            AS created_at,
			w.wallet_id,
			COALESCE(w.good_fund, 0)                                      AS good_fund,
			COALESCE(w.pending_fund, 0)                                   AS pending_fund
		FROM users u
		LEFT JOIN customers c ON c.user_id = u.user_id
		LEFT JOIN orders o    ON o.user_id = u.user_id
		LEFT JOIN wallets w   ON w.user_id = u.user_id
		WHERE u.user_id = $1 AND u.role = 'CT'
		GROUP BY u.user_id, u.email, c.first_name, c.last_name, u.phone,
		         c.address_line1, c.sub_district, c.district, c.province, c.postal_code,
		         u.is_active, u.created_at, w.wallet_id, w.good_fund, w.pending_fund
	`, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &item, nil
}

func (r *CustomerAdminRepository) GetCustomerWallet(userID int64) (*domain.AdminCustomerWallet, error) {
	result := &domain.AdminCustomerWallet{
		UserID:       userID,
		Transactions: []domain.AdminWalletTxItem{},
	}

	var row struct {
		WalletID    int64   `db:"wallet_id"`
		GoodFund    float64 `db:"good_fund"`
		PendingFund float64 `db:"pending_fund"`
	}
	err := r.db.Get(&row, `
		SELECT wallet_id, good_fund, pending_fund
		FROM wallets
		WHERE user_id = $1
	`, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// no wallet yet — return zero balances without error
			return result, nil
		}
		return nil, err
	}

	result.WalletID = &row.WalletID
	result.GoodFund = row.GoodFund
	result.PendingFund = row.PendingFund
	result.Total = row.GoodFund + row.PendingFund

	var txs []domain.AdminWalletTxItem
	if err := r.db.Select(&txs, `
		SELECT
			tx_id,
			wallet_id,
			order_id,
			type,
			amount,
			status,
			created_at::text AS created_at
		FROM transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT 200
	`, row.WalletID); err != nil {
		return nil, err
	}
	if txs != nil {
		result.Transactions = txs
	}
	return result, nil
}

func (r *CustomerAdminRepository) ListCustomerOrders(userID int64, limit, offset int) ([]domain.AdminCustomerOrderItem, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	if err := r.db.Get(&total, `SELECT COUNT(*) FROM orders WHERE user_id = $1`, userID); err != nil {
		return nil, 0, err
	}

	var items []domain.AdminCustomerOrderItem
	if err := r.db.Select(&items, `
		SELECT
			o.order_id,
			o.rfq_id,
			o.factory_id,
			COALESCE(fp.factory_name, '')    AS factory_name,
			COALESCE(o.grand_total, 0)       AS grand_total,
			o.status,
			o.created_at::text               AS created_at
		FROM orders o
		LEFT JOIN factory_profiles fp ON fp.user_id = o.factory_id
		WHERE o.user_id = $1
		ORDER BY o.created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset); err != nil {
		return nil, 0, err
	}
	if items == nil {
		items = []domain.AdminCustomerOrderItem{}
	}
	return items, total, nil
}

func (r *CustomerAdminRepository) ListTopCustomers(limit int) ([]domain.AdminTopCustomer, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	var items []domain.AdminTopCustomer
	if err := r.db.Select(&items, `
		SELECT
			u.user_id,
			COALESCE(c.first_name, '') AS first_name,
			COALESCE(c.last_name, '')  AS last_name,
			u.email,
			COUNT(o.order_id)::int     AS total_orders,
			COALESCE(SUM(o.grand_total) FILTER (WHERE o.status <> 'CA'), 0) AS total_spend
		FROM users u
		LEFT JOIN customers c ON c.user_id = u.user_id
		LEFT JOIN orders o    ON o.user_id = u.user_id
		WHERE u.role = 'CT'
		GROUP BY u.user_id, c.first_name, c.last_name, u.email
		ORDER BY total_spend DESC, total_orders DESC
		LIMIT $1
	`, limit); err != nil {
		return nil, err
	}
	if items == nil {
		items = []domain.AdminTopCustomer{}
	}
	return items, nil
}

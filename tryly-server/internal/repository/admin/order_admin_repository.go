package admin

import (
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
)

type AdminOrderRepository struct {
	db *sqlx.DB
}

func NewAdminOrderRepository(db *sqlx.DB) *AdminOrderRepository {
	return &AdminOrderRepository{db: db}
}

func (r *AdminOrderRepository) ListAdmin(filter domain.AdminOrderFilter) ([]domain.AdminOrderListItem, int, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize

	conditions := sq.And{}
	if filter.Status != "" {
		conditions = append(conditions, sq.Eq{"o.status": domainutil.NormalizeStatus(filter.Status)})
	}
	if filter.FactoryID != nil {
		conditions = append(conditions, sq.Eq{"o.factory_id": *filter.FactoryID})
	}
	if filter.UserID != nil {
		conditions = append(conditions, sq.Eq{"o.customer_id": *filter.UserID})
	}
	if filter.DateFrom != nil {
		conditions = append(conditions, sq.GtOrEq{"o.created_at": *filter.DateFrom})
	}
	if filter.DateTo != nil {
		conditions = append(conditions, sq.Lt{"o.created_at": filter.DateTo.Add(24 * time.Hour)})
	}
	if filter.Search != "" {
		searchTerm := "%" + domainutil.NormalizeLower(filter.Search) + "%"
		conditions = append(conditions, sq.Like{"LOWER(r.title)": searchTerm})
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	countQuery := psql.Select("COUNT(*)").
		From("orders o").
		InnerJoin("quotations q ON q.quote_id = o.quote_id").
		InnerJoin("rfqs r ON r.rfq_id = q.rfq_id").
		Where(conditions)

	var total int
	countSQL, countArgs, err := countQuery.ToSql()
	if err != nil {
		return nil, 0, err
	}
	if err := r.db.Get(&total, countSQL, countArgs...); err != nil {
		return nil, 0, err
	}

	query := psql.Select(
		"o.order_id",
		"o.quote_id",
		"r.rfq_id",
		"COALESCE(r.title, '') AS rfq_title",
		"o.factory_id",
		"COALESCE(fp.factory_name, 'Factory #' || o.factory_id::text) AS factory_name",
		"o.customer_id AS user_id",
		"COALESCE(NULLIF(TRIM(CONCAT(cu.first_name, ' ', cu.last_name)), ''), 'ลูกค้า #' || o.customer_id::text) AS customer_name",
		"o.status",
		"o.total_amount",
		"COALESCE(q.platform_commission_amount, 0) AS platform_commission_amount",
		"COALESCE(q.vat_amount, 0) AS vat_amount",
		"COALESCE(q.factory_net_receivable, 0) AS factory_net_receivable",
		"o.payment_type",
		"o.estimated_delivery",
		"o.created_at",
	).
		From("orders o").
		InnerJoin("quotations q ON q.quote_id = o.quote_id").
		InnerJoin("rfqs r ON r.rfq_id = q.rfq_id").
		LeftJoin("factory_profiles fp ON fp.user_id = o.factory_id").
		LeftJoin("customers cu ON cu.user_id = o.customer_id").
		Where(conditions).
		OrderBy("o.created_at DESC", "o.order_id DESC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset))

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, 0, err
	}

	items := []domain.AdminOrderListItem{}
	if err := r.db.Select(&items, sql, args...); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *AdminOrderRepository) GetAdminFinance(orderID int64) (*domain.AdminOrderFinance, error) {
	var item domain.AdminOrderFinance
	if err := r.db.Get(&item, `
		SELECT
			COALESCE(q.platform_commission_rate, 0)::float8 AS platform_commission_rate,
			COALESCE(q.platform_commission_amount, 0)::float8 AS platform_commission_amount,
			COALESCE(q.vat_rate, 0)::float8 AS vat_rate,
			COALESCE(q.vat_amount, 0)::float8 AS vat_amount,
			COALESCE(q.factory_net_receivable, 0)::float8 AS factory_net_receivable,
			COALESCE(q.grand_total, 0)::float8 AS grand_total
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		WHERE o.order_id = $1
	`, orderID); err != nil {
		return nil, err
	}
	return &item, nil
}

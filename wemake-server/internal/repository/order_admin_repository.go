package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
)

func (r *OrderRepository) ListAdmin(filter domain.AdminOrderFilter) ([]domain.AdminOrderListItem, int, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := []string{"1=1"}
	args := []interface{}{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Status != "" {
		where = append(where, "o.status = "+arg(strings.TrimSpace(strings.ToUpper(filter.Status))))
	}
	if filter.FactoryID != nil {
		where = append(where, "o.factory_id = "+arg(*filter.FactoryID))
	}
	if filter.UserID != nil {
		where = append(where, "o.user_id = "+arg(*filter.UserID))
	}
	if filter.DateFrom != nil {
		where = append(where, "o.created_at >= "+arg(*filter.DateFrom))
	}
	if filter.DateTo != nil {
		where = append(where, "o.created_at < "+arg(filter.DateTo.Add(24*time.Hour)))
	}
	if filter.Search != "" {
		where = append(where, "LOWER(r.title) LIKE "+arg("%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%"))
	}
	condition := strings.Join(where, " AND ")
	var total int
	if err := r.db.Get(&total, `
		SELECT COUNT(*)
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		INNER JOIN rfqs r ON r.rfq_id = q.rfq_id
		WHERE `+condition, args...); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	items := []domain.AdminOrderListItem{}
	query := `
		SELECT
			o.order_id,
			o.quote_id,
			r.rfq_id,
			COALESCE(r.title, '') AS rfq_title,
			o.factory_id,
			COALESCE(fp.factory_name, 'Factory #' || o.factory_id::text) AS factory_name,
			o.user_id,
			COALESCE(NULLIF(TRIM(CONCAT(cu.first_name, ' ', cu.last_name)), ''), 'ลูกค้า #' || o.user_id::text) AS customer_name,
			o.status,
			o.total_amount,
			COALESCE(q.platform_commission_amount, 0)::float8 AS platform_commission_amount,
			COALESCE(q.vat_amount, 0)::float8 AS vat_amount,
			COALESCE(q.factory_net_receivable, 0)::float8 AS factory_net_receivable,
			o.payment_type,
			o.estimated_delivery,
			o.created_at
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		INNER JOIN rfqs r ON r.rfq_id = q.rfq_id
		LEFT JOIN factory_profiles fp ON fp.user_id = o.factory_id
		LEFT JOIN customers cu ON cu.user_id = o.user_id
		WHERE ` + condition + `
		ORDER BY o.created_at DESC, o.order_id DESC
		LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	if err := r.db.Select(&items, query, args...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *OrderRepository) GetAdminFinance(orderID int64) (*domain.AdminOrderFinance, error) {
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

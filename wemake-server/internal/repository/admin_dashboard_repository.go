package repository

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type AdminDashboardRepository struct {
	db *sqlx.DB
}

func NewAdminDashboardRepository(db *sqlx.DB) *AdminDashboardRepository {
	return &AdminDashboardRepository{db: db}
}

func (r *AdminDashboardRepository) GetSummary(from, to time.Time, period string) (*domain.AdminDashboardSummary, error) {
	out := &domain.AdminDashboardSummary{
		Period:   period,
		DateFrom: from.Format("2006-01-02"),
		DateTo:   to.Format("2006-01-02"),
	}
	if err := r.db.Get(&out.Revenue, `
		SELECT
			COALESCE(SUM(o.total_amount), 0)::float8 AS gross_order_value,
			COALESCE(SUM(q.vat_amount), 0)::float8 AS total_vat_collected,
			COALESCE(SUM(q.platform_commission_amount), 0)::float8 AS platform_commission,
			COALESCE(SUM(q.factory_net_receivable), 0)::float8 AS factory_net_payable
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		WHERE o.status <> 'CC'
		  AND o.created_at >= $1
		  AND o.created_at < $2
	`, from, to); err != nil {
		return nil, err
	}
	if err := r.db.Get(&out.Orders, `
		SELECT
			COUNT(*)::bigint AS total,
			COUNT(*) FILTER (WHERE status = 'CP')::bigint AS completed,
			COUNT(*) FILTER (WHERE status IN ('PP','PR','WF','QC','SH','DL','AC'))::bigint AS active,
			COUNT(*) FILTER (WHERE status = 'CC')::bigint AS cancelled,
			COUNT(DISTINCT d.dispute_id)::bigint AS disputed
		FROM orders o
		LEFT JOIN disputes d ON d.order_id = o.order_id AND d.status = 'OP'
		WHERE o.created_at >= $1
		  AND o.created_at < $2
	`, from, to); err != nil {
		return nil, err
	}
	if err := r.db.Get(&out.RFQs, `
		SELECT
			COUNT(*)::bigint AS total,
			COUNT(*) FILTER (WHERE status = 'OP')::bigint AS open,
			COUNT(*) FILTER (WHERE status IN ('CL','CC'))::bigint AS closed
		FROM rfqs
		WHERE created_at >= $1
		  AND created_at < $2
	`, from, to); err != nil {
		return nil, err
	}
	if err := r.db.Get(&out.Factories, `
		SELECT
			COUNT(*)::bigint AS total_registered,
			COUNT(*) FILTER (WHERE approval_status = 'PE')::bigint AS pending_approval,
			COUNT(*) FILTER (WHERE approval_status = 'AP')::bigint AS approved,
			COUNT(*) FILTER (WHERE approval_status = 'RJ')::bigint AS rejected,
			COUNT(*) FILTER (WHERE approval_status = 'SU')::bigint AS suspended
		FROM factory_profiles
	`); err != nil {
		return nil, err
	}
	if err := r.db.Get(&out.Customers, `
		SELECT COUNT(*)::bigint AS total FROM users WHERE role = 'CT'
	`); err != nil {
		return nil, err
	}
	if err := r.db.Get(&out.Settlements, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE status = 'PE'), 0)::float8 AS pending_amount,
			COALESCE(SUM(amount) FILTER (WHERE status = 'CP'), 0)::float8 AS completed_amount
		FROM settlements
		WHERE created_at >= $1
		  AND created_at < $2
	`, from, to); err != nil {
		return nil, err
	}
	if err := r.db.Get(&out.Withdrawals, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'PE')::bigint AS pending_count,
			COALESCE(SUM(amount) FILTER (WHERE status = 'PE'), 0)::float8 AS pending_amount
		FROM withdrawal_requests
		WHERE created_at >= $1
		  AND created_at < $2
	`, from, to); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *AdminDashboardRepository) GetRevenueChart(from, to time.Time, granularity string) ([]domain.RevenueChartPoint, error) {
	bucket := "day"
	switch granularity {
	case "week":
		bucket = "week"
	case "month":
		bucket = "month"
	}
	var items []domain.RevenueChartPoint
	err := r.db.Select(&items, fmt.Sprintf(`
		SELECT
			TO_CHAR(date_trunc('%s', o.created_at), CASE WHEN '%s' = 'month' THEN 'YYYY-MM' ELSE 'YYYY-MM-DD' END) AS bucket,
			COALESCE(SUM(o.total_amount), 0)::float8 AS gross_order_value,
			COALESCE(SUM(q.platform_commission_amount), 0)::float8 AS platform_commission,
			COALESCE(SUM(q.vat_amount), 0)::float8 AS vat_collected,
			COUNT(*)::bigint AS order_count
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		WHERE o.status <> 'CC'
		  AND o.created_at >= $1
		  AND o.created_at < $2
		GROUP BY 1
		ORDER BY 1
	`, bucket, bucket), from, to)
	return items, err
}

func (r *AdminDashboardRepository) GetTopFactories(from, to time.Time, limit int) ([]domain.TopFactoryRow, error) {
	if limit <= 0 {
		limit = 10
	}
	items := []domain.TopFactoryRow{}
	err := r.db.Select(&items, `
		SELECT
			o.factory_id,
			COALESCE(fp.factory_name, 'Factory #' || o.factory_id::text) AS factory_name,
			COUNT(*)::bigint AS total_orders,
			COUNT(*) FILTER (WHERE o.status = 'CP')::bigint AS completed_orders,
			COALESCE(SUM(o.total_amount), 0)::float8 AS gross_revenue,
			COALESCE(SUM(q.platform_commission_amount), 0)::float8 AS platform_commission,
			ROUND(AVG(fr.rating::numeric), 2)::float8 AS avg_rating
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		LEFT JOIN factory_profiles fp ON fp.user_id = o.factory_id
		LEFT JOIN factory_reviews fr ON fr.factory_id = o.factory_id
		WHERE o.status <> 'CC'
		  AND o.created_at >= $1
		  AND o.created_at < $2
		GROUP BY o.factory_id, fp.factory_name
		ORDER BY gross_revenue DESC, total_orders DESC
		LIMIT $3
	`, from, to, limit)
	return items, err
}

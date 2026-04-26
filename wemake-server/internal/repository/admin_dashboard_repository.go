package repository

import (
	"fmt"
	"log"
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
	log.Printf("[AdminDashboard] GetSummary called: period=%s, from=%s, to=%s", period, from.Format("2006-01-02"), to.Format("2006-01-02"))
	out := &domain.AdminDashboardSummary{
		Period:   period,
		DateFrom: from.Format("2006-01-02"),
		DateTo:   to.Format("2006-01-02"),
	}
	hasQuoteVAT, err := r.hasColumn("quotations", "vat_amount")
	if err != nil {
		log.Printf("[AdminDashboard] Error checking vat_amount column: %v", err)
		return nil, err
	}
	hasQuoteCommission, err := r.hasColumn("quotations", "platform_commission_amount")
	if err != nil {
		return nil, err
	}
	hasQuoteNetReceivable, err := r.hasColumn("quotations", "factory_net_receivable")
	if err != nil {
		return nil, err
	}
	revenueQuery := `
		SELECT
			COALESCE(SUM(o.total_amount), 0)::float8 AS gross_order_value,
			%s AS total_vat_collected,
			%s AS platform_commission,
			%s AS factory_net_payable
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		WHERE o.status <> 'CC'
		  AND o.created_at >= $1
		  AND o.created_at < $2
	`
	vatExpr := "0::float8"
	if hasQuoteVAT {
		vatExpr = "COALESCE(SUM(q.vat_amount), 0)::float8"
	}
	commissionExpr := "0::float8"
	if hasQuoteCommission {
		commissionExpr = "COALESCE(SUM(q.platform_commission_amount), 0)::float8"
	}
	netExpr := "0::float8"
	if hasQuoteNetReceivable {
		netExpr = "COALESCE(SUM(q.factory_net_receivable), 0)::float8"
	}
	if err := r.db.Get(&out.Revenue, fmt.Sprintf(revenueQuery, vatExpr, commissionExpr, netExpr), from, to); err != nil {
		log.Printf("[AdminDashboard] Error fetching revenue: %v", err)
		return nil, err
	}
	log.Printf("[AdminDashboard] Revenue fetched successfully: %+v", out.Revenue)
	hasDisputesTable, err := r.hasTable("disputes")
	if err != nil {
		return nil, err
	}
	disputedExpr := "0::bigint"
	disputeJoin := ""
	if hasDisputesTable {
		disputedExpr = "COUNT(DISTINCT d.dispute_id)::bigint"
		disputeJoin = "LEFT JOIN disputes d ON d.order_id = o.order_id AND d.status = 'OP'"
	}
	if err := r.db.Get(&out.Orders, fmt.Sprintf(`
		SELECT
			COUNT(*)::bigint AS total,
			COUNT(*) FILTER (WHERE o.status = 'CP')::bigint AS completed,
			COUNT(*) FILTER (WHERE o.status IN ('PP','PR','WF','QC','SH','DL','AC'))::bigint AS active,
			COUNT(*) FILTER (WHERE o.status = 'CC')::bigint AS cancelled,
			%s AS disputed
		FROM orders o
		%s
		WHERE o.created_at >= $1
		  AND o.created_at < $2
	`, disputedExpr, disputeJoin), from, to); err != nil {
		log.Printf("[AdminDashboard] Error fetching orders: %v", err)
		return nil, err
	}
	log.Printf("[AdminDashboard] Orders fetched successfully: %+v", out.Orders)
	if err := r.db.Get(&out.RFQs, `
		SELECT
			COUNT(*)::bigint AS total,
			COUNT(*) FILTER (WHERE status = 'OP')::bigint AS open,
			COUNT(*) FILTER (WHERE status IN ('CL','CC'))::bigint AS closed
		FROM rfqs
		WHERE created_at >= $1
		  AND created_at < $2
	`, from, to); err != nil {
		log.Printf("[AdminDashboard] Error fetching RFQs: %v", err)
		return nil, err
	}
	log.Printf("[AdminDashboard] RFQs fetched successfully: %+v", out.RFQs)
	hasApprovalStatus, err := r.hasColumn("factory_profiles", "approval_status")
	if err != nil {
		log.Printf("[AdminDashboard] Error checking approval_status column: %v", err)
		return nil, err
	}
	if hasApprovalStatus {
		if err := r.db.Get(&out.Factories, `
			SELECT
				COUNT(*)::bigint AS total_registered,
				COUNT(*) FILTER (WHERE approval_status = 'PE')::bigint AS pending_approval,
				COUNT(*) FILTER (WHERE approval_status = 'AP')::bigint AS approved,
				COUNT(*) FILTER (WHERE approval_status = 'RJ')::bigint AS rejected,
				COUNT(*) FILTER (WHERE approval_status = 'SU')::bigint AS suspended
			FROM factory_profiles
		`); err != nil {
			log.Printf("[AdminDashboard] Error fetching factories (with approval_status): %v", err)
			return nil, err
		}
	} else {
		if err := r.db.Get(&out.Factories, `
			SELECT
				COUNT(*)::bigint AS total_registered,
				COUNT(*) FILTER (WHERE COALESCE(is_verified, FALSE) = FALSE)::bigint AS pending_approval,
				COUNT(*) FILTER (WHERE COALESCE(is_verified, FALSE) = TRUE)::bigint AS approved,
				0::bigint AS rejected,
				0::bigint AS suspended
			FROM factory_profiles
		`); err != nil {
			log.Printf("[AdminDashboard] Error fetching factories (without approval_status): %v", err)
			return nil, err
		}
	}
	log.Printf("[AdminDashboard] Factories fetched successfully: %+v", out.Factories)
	if err := r.db.Get(&out.Customers, `
		SELECT COUNT(*)::bigint AS total FROM users WHERE role = 'CT'
	`); err != nil {
		log.Printf("[AdminDashboard] Error fetching customers: %v", err)
		return nil, err
	}
	log.Printf("[AdminDashboard] Customers fetched successfully: %+v", out.Customers)
	hasSettlementsTable, err := r.hasTable("settlements")
	if err != nil {
		log.Printf("[AdminDashboard] Error checking settlements table: %v", err)
		return nil, err
	}
	if hasSettlementsTable {
		if err := r.db.Get(&out.Settlements, `
			SELECT
				COALESCE(SUM(amount) FILTER (WHERE status = 'PE'), 0)::float8 AS pending_amount,
				COALESCE(SUM(amount) FILTER (WHERE status = 'CP'), 0)::float8 AS completed_amount
			FROM settlements
			WHERE created_at >= $1
			  AND created_at < $2
		`, from, to); err != nil {
			log.Printf("[AdminDashboard] Error fetching settlements: %v", err)
			return nil, err
		}
		log.Printf("[AdminDashboard] Settlements fetched successfully: %+v", out.Settlements)
	}
	hasWithdrawalsTable, err := r.hasTable("withdrawal_requests")
	if err != nil {
		log.Printf("[AdminDashboard] Error checking withdrawal_requests table: %v", err)
		return nil, err
	}
	if hasWithdrawalsTable {
		if err := r.db.Get(&out.Withdrawals, `
			SELECT
				COUNT(*) FILTER (WHERE status = 'PE')::bigint AS pending_count,
				COALESCE(SUM(amount) FILTER (WHERE status = 'PE'), 0)::float8 AS pending_amount
			FROM withdrawal_requests
			WHERE created_at >= $1
			  AND created_at < $2
		`, from, to); err != nil {
			log.Printf("[AdminDashboard] Error fetching withdrawals: %v", err)
			return nil, err
		}
		log.Printf("[AdminDashboard] Withdrawals fetched successfully: %+v", out.Withdrawals)
	}
	log.Printf("[AdminDashboard] GetSummary completed successfully")
	return out, nil
}

func (r *AdminDashboardRepository) hasColumn(tableName, columnName string) (bool, error) {
	var exists bool
	err := r.db.Get(&exists, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = $1
			  AND column_name = $2
		)
	`, tableName, columnName)
	return exists, err
}

func (r *AdminDashboardRepository) hasTable(tableName string) (bool, error) {
	var exists bool
	err := r.db.Get(&exists, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = $1
		)
	`, tableName)
	return exists, err
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
	hasQuoteCommission, err := r.hasColumn("quotations", "platform_commission_amount")
	if err != nil {
		return nil, err
	}
	hasQuoteVAT, err := r.hasColumn("quotations", "vat_amount")
	if err != nil {
		return nil, err
	}
	commissionExpr := "0::float8"
	if hasQuoteCommission {
		commissionExpr = "COALESCE(SUM(q.platform_commission_amount), 0)::float8"
	}
	vatExpr := "0::float8"
	if hasQuoteVAT {
		vatExpr = "COALESCE(SUM(q.vat_amount), 0)::float8"
	}
	err = r.db.Select(&items, fmt.Sprintf(`
		SELECT
			TO_CHAR(date_trunc('%s', o.created_at), CASE WHEN '%s' = 'month' THEN 'YYYY-MM' ELSE 'YYYY-MM-DD' END) AS bucket,
			COALESCE(SUM(o.total_amount), 0)::float8 AS gross_order_value,
			%s AS platform_commission,
			%s AS vat_collected,
			COUNT(*)::bigint AS order_count
		FROM orders o
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		WHERE o.status <> 'CC'
		  AND o.created_at >= $1
		  AND o.created_at < $2
		GROUP BY 1
		ORDER BY 1
	`, bucket, bucket, commissionExpr, vatExpr), from, to)
	return items, err
}

func (r *AdminDashboardRepository) GetTopFactories(from, to time.Time, limit int) ([]domain.TopFactoryRow, error) {
	if limit <= 0 {
		limit = 10
	}
	items := []domain.TopFactoryRow{}
	hasQuoteCommission, err := r.hasColumn("quotations", "platform_commission_amount")
	if err != nil {
		return nil, err
	}
	commissionExpr := "0::float8"
	if hasQuoteCommission {
		commissionExpr = "COALESCE(SUM(q.platform_commission_amount), 0)::float8"
	}
	err = r.db.Select(&items, fmt.Sprintf(`
		SELECT
			o.factory_id,
			COALESCE(fp.factory_name, 'Factory #' || o.factory_id::text) AS factory_name,
			COUNT(*)::bigint AS total_orders,
			COUNT(*) FILTER (WHERE o.status = 'CP')::bigint AS completed_orders,
			COALESCE(SUM(o.total_amount), 0)::float8 AS gross_revenue,
			%s AS platform_commission,
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
	`, commissionExpr), from, to, limit)
	return items, err
}

package repository

import (
	"fmt"

	"github.com/yourusername/wemake/internal/domain"
)

func (r *DisputeRepository) ListAdmin(status string, orderID *int64, page, pageSize int) ([]domain.AdminDisputeListItem, int, error) {
	page, pageSize = normalizePage(page, pageSize)
	where := "1=1"
	args := []interface{}{}
	if status != "" {
		args = append(args, status)
		where += fmt.Sprintf(" AND d.status = $%d", len(args))
	}
	if orderID != nil {
		args = append(args, *orderID)
		where += fmt.Sprintf(" AND d.order_id = $%d", len(args))
	}
	var total int
	if err := r.db.Get(&total, `SELECT COUNT(*) FROM disputes d WHERE `+where, args...); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	items := []domain.AdminDisputeListItem{}
	if err := r.db.Select(&items, `
		SELECT
			d.dispute_id,
			d.order_id,
			COALESCE(r.title, '') AS rfq_title,
			COALESCE(fp.factory_name, 'Factory #' || o.factory_id::text) AS factory_name,
			COALESCE(NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), ''), 'ลูกค้า #' || o.user_id::text) AS customer_name,
			d.opened_by,
			d.reason,
			d.status,
			d.resolution,
			d.created_at,
			d.resolved_at
		FROM disputes d
		INNER JOIN orders o ON o.order_id = d.order_id
		INNER JOIN quotations q ON q.quote_id = o.quote_id
		INNER JOIN rfqs r ON r.rfq_id = q.rfq_id
		LEFT JOIN factory_profiles fp ON fp.user_id = o.factory_id
		LEFT JOIN customers c ON c.user_id = o.user_id
		WHERE `+where+`
		ORDER BY created_at DESC, dispute_id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)), args...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

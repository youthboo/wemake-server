package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/yourusername/wemake/internal/domain"
)

func (r *RFQRepository) ListAdmin(filter domain.AdminRFQFilter) ([]domain.AdminRFQListItem, int, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := []string{"1=1"}
	args := []interface{}{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.Status != "" {
		where = append(where, "r.status = "+arg(strings.TrimSpace(strings.ToUpper(filter.Status))))
	}
	if filter.UserID != nil {
		where = append(where, "r.user_id = "+arg(*filter.UserID))
	}
	if filter.CategoryID != nil {
		where = append(where, "r.category_id = "+arg(*filter.CategoryID))
	}
	if filter.DateFrom != nil {
		where = append(where, "r.created_at >= "+arg(*filter.DateFrom))
	}
	if filter.DateTo != nil {
		where = append(where, "r.created_at < "+arg(filter.DateTo.Add(24*time.Hour)))
	}
	if filter.Search != "" {
		where = append(where, "LOWER(r.title) LIKE "+arg("%"+strings.ToLower(strings.TrimSpace(filter.Search))+"%"))
	}
	condition := strings.Join(where, " AND ")
	var total int
	if err := r.db.Get(&total, `SELECT COUNT(*) FROM rfqs r WHERE `+condition, args...); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	items := []domain.AdminRFQListItem{}
	query := `
		SELECT
			r.rfq_id,
			r.title,
			r.user_id,
			COALESCE(NULLIF(TRIM(CONCAT(cu.first_name, ' ', cu.last_name)), ''), 'ลูกค้า #' || r.user_id::text) AS customer_name,
			u.email AS customer_email,
			c.name AS category_name,
			sc.name AS sub_category_name,
			r.quantity,
			r.status,
			COUNT(q.quote_id)::bigint AS quotation_count,
			r.target_unit_price,
			r.created_at
		FROM rfqs r
		INNER JOIN users u ON u.user_id = r.user_id
		LEFT JOIN customers cu ON cu.user_id = r.user_id
		LEFT JOIN categories c ON c.category_id = r.category_id
		LEFT JOIN lbi_sub_categories sc ON sc.sub_category_id = r.sub_category_id
		LEFT JOIN quotations q ON q.rfq_id = r.rfq_id
		WHERE ` + condition + `
		GROUP BY r.rfq_id, u.email, cu.first_name, cu.last_name, c.name, sc.name
		ORDER BY r.created_at DESC, r.rfq_id DESC
		LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	if err := r.db.Select(&items, query, args...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *RFQRepository) UpdateStatusAdmin(rfqID int64, status string) error {
	_, err := r.db.Exec(`UPDATE rfqs SET status = $1, updated_at = NOW() WHERE rfq_id = $2`, status, rfqID)
	return err
}

func (r *RFQRepository) GetAdminDetail(rfqID int64) (*domain.AdminRFQDetail, error) {
	rfq, err := r.GetByIDAny(rfqID)
	if err != nil {
		return nil, err
	}
	out := &domain.AdminRFQDetail{RFQ: rfq}
	type row struct {
		CustomerName   string         `db:"customer_name"`
		CustomerEmail  string         `db:"customer_email"`
		CustomerPhone  sql.NullString `db:"customer_phone"`
		QuotationCount int64          `db:"quotation_count"`
	}
	var meta row
	if err := r.db.Get(&meta, `
		SELECT
			COALESCE(NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), ''), 'ลูกค้า #' || u.user_id::text) AS customer_name,
			u.email AS customer_email,
			NULLIF(u.phone, '') AS customer_phone,
			(SELECT COUNT(*) FROM quotations q WHERE q.rfq_id = r.rfq_id)::bigint AS quotation_count
		FROM rfqs r
		INNER JOIN users u ON u.user_id = r.user_id
		LEFT JOIN customers c ON c.user_id = r.user_id
		WHERE r.rfq_id = $1
	`, rfqID); err != nil {
		return nil, err
	}
	out.CustomerName = meta.CustomerName
	out.CustomerEmail = meta.CustomerEmail
	if meta.CustomerPhone.Valid {
		out.CustomerPhone = &meta.CustomerPhone.String
	}
	out.QuotationCount = meta.QuotationCount
	return out, nil
}

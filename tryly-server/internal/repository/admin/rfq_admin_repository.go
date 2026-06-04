package admin

import (
	"database/sql"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	rfqrepo "github.com/yourusername/wemake/internal/repository/rfq"
)

type AdminRFQRepository struct {
	db   *sqlx.DB
	rfqs *rfqrepo.RFQRepository
}

func NewAdminRFQRepository(db *sqlx.DB, rfqs *rfqrepo.RFQRepository) *AdminRFQRepository {
	return &AdminRFQRepository{db: db, rfqs: rfqs}
}

func (r *AdminRFQRepository) ListAdmin(filter domain.AdminRFQFilter) ([]domain.AdminRFQListItem, int, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize

	conditions := sq.And{}
	if filter.Status != "" {
		conditions = append(conditions, sq.Eq{"r.status": domainutil.NormalizeStatus(filter.Status)})
	}
	if filter.UserID != nil {
		conditions = append(conditions, sq.Eq{"r.user_id": *filter.UserID})
	}
	if filter.CategoryID != nil {
		conditions = append(conditions, sq.Eq{"r.category_id": *filter.CategoryID})
	}
	if filter.DateFrom != nil {
		conditions = append(conditions, sq.GtOrEq{"r.created_at": *filter.DateFrom})
	}
	if filter.DateTo != nil {
		conditions = append(conditions, sq.Lt{"r.created_at": filter.DateTo.Add(24 * time.Hour)})
	}
	if filter.Search != "" {
		searchTerm := "%" + domainutil.NormalizeLower(filter.Search) + "%"
		conditions = append(conditions, sq.Like{"LOWER(r.title)": searchTerm})
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	countQuery := psql.Select("COUNT(*)").
		From("rfqs r").
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
		"r.rfq_id",
		"r.title",
		"r.user_id",
		"COALESCE(NULLIF(TRIM(CONCAT(cu.first_name, ' ', cu.last_name)), ''), 'ลูกค้า #' || r.user_id::text) AS customer_name",
		"u.email AS customer_email",
		"c.name AS category_name",
		"sc.name AS sub_category_name",
		"r.quantity",
		"r.status",
		"COUNT(q.quote_id)::bigint AS quotation_count",
		"r.target_price",
		"r.created_at",
	).
		From("rfqs r").
		InnerJoin("users u ON u.user_id = r.user_id").
		LeftJoin("customers cu ON cu.user_id = r.user_id").
		LeftJoin("lbi_categories c ON c.category_id = r.category_id").
		LeftJoin("lbi_sub_categories sc ON sc.sub_category_id = r.sub_category_id").
		LeftJoin("quotations q ON q.rfq_id = r.rfq_id").
		Where(conditions).
		GroupBy("r.rfq_id, u.email, cu.first_name, cu.last_name, c.name, sc.name").
		OrderBy("r.created_at DESC", "r.rfq_id DESC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset))

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, 0, err
	}

	items := []domain.AdminRFQListItem{}
	if err := r.db.Select(&items, sql, args...); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *AdminRFQRepository) UpdateStatusAdmin(rfqID int64, status string) error {
	_, err := r.db.Exec(`UPDATE rfqs SET status = $1, updated_at = NOW() WHERE rfq_id = $2`, status, rfqID)
	return err
}

func (r *AdminRFQRepository) GetAdminDetail(rfqID int64) (*domain.AdminRFQDetail, error) {
	rfq, err := r.rfqs.GetByIDAny(rfqID)
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

	query := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).Select(
		"COALESCE(NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), ''), 'ลูกค้า #' || u.user_id::text) AS customer_name",
		"u.email AS customer_email",
		"NULLIF(u.phone, '') AS customer_phone",
		"(SELECT COUNT(*) FROM quotations q WHERE q.rfq_id = r.rfq_id)::bigint AS quotation_count",
	).
		From("rfqs r").
		InnerJoin("users u ON u.user_id = r.user_id").
		LeftJoin("customers c ON c.user_id = r.user_id").
		Where(sq.Eq{"r.rfq_id": rfqID})

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}

	var meta row
	if err := r.db.Get(&meta, sql, args...); err != nil {
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

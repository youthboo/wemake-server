package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/yourusername/wemake/internal/domain"
)

func (r *FactoryRepository) ListAdmin(filter domain.AdminFactoryFilter) ([]domain.AdminFactoryListItem, int, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := []string{"u.role = 'FT'"}
	args := []interface{}{}
	arg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if filter.ApprovalStatus != "" {
		where = append(where, "fp.approval_status = "+arg(strings.TrimSpace(strings.ToUpper(filter.ApprovalStatus))))
	}
	if filter.IsVerified != nil {
		where = append(where, "COALESCE(fp.is_verified, FALSE) = "+arg(*filter.IsVerified))
	}
	if filter.Search != "" {
		like := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		where = append(where, "(LOWER(fp.factory_name) LIKE "+arg(like)+" OR LOWER(u.email) LIKE "+arg(like)+")")
	}
	condition := strings.Join(where, " AND ")
	var total int
	if err := r.db.Get(&total, `
		SELECT COUNT(*)
		FROM factory_profiles fp
		INNER JOIN users u ON u.user_id = fp.user_id
		WHERE `+condition, args...); err != nil {
		return nil, 0, err
	}
	args = append(args, pageSize, (page-1)*pageSize)
	items := []domain.AdminFactoryListItem{}
	query := `
		SELECT
			fp.user_id AS factory_id,
			fp.factory_name,
			u.email,
			NULLIF(u.phone, '') AS phone,
			NULLIF(fp.tax_id, '') AS tax_id,
			ft.type_name AS factory_type_name,
			p.name_th AS province_name,
			COALESCE(fp.approval_status, 'PE') AS approval_status,
			COALESCE(fp.is_verified, FALSE) AS is_verified,
			fp.submitted_at,
			fp.verified_at,
			fp.verified_by,
			fp.rejection_reason,
			u.created_at
		FROM factory_profiles fp
		INNER JOIN users u ON u.user_id = fp.user_id
		LEFT JOIN lbi_factory_types ft ON ft.factory_type_id = fp.factory_type_id
		LEFT JOIN lbi_provinces p ON p.row_id = fp.province_id
		WHERE ` + condition + `
		ORDER BY fp.submitted_at DESC NULLS LAST, u.created_at DESC, fp.user_id DESC
		LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	if err := r.db.Select(&items, query, args...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *FactoryRepository) GetAdminDetail(factoryID int64) (*domain.AdminFactoryDetail, error) {
	var item domain.AdminFactoryDetail
	if err := r.db.Get(&item, `
		SELECT
			fp.user_id AS factory_id,
			fp.factory_name,
			u.email,
			NULLIF(u.phone, '') AS phone,
			NULLIF(fp.tax_id, '') AS tax_id,
			fp.factory_type_id,
			ft.type_name AS factory_type_name,
			fp.specialization,
			fp.province_id,
			p.name_th AS province_name,
			fp.image_url,
			fp.description,
			COALESCE(fp.approval_status, 'PE') AS approval_status,
			COALESCE(fp.is_verified, FALSE) AS is_verified,
			fp.submitted_at,
			fp.verified_at,
			fp.verified_by,
			fp.rejection_reason,
			u.created_at
		FROM factory_profiles fp
		INNER JOIN users u ON u.user_id = fp.user_id AND u.role = 'FT'
		LEFT JOIN lbi_factory_types ft ON ft.factory_type_id = fp.factory_type_id
		LEFT JOIN lbi_provinces p ON p.row_id = fp.province_id
		WHERE fp.user_id = $1
	`, factoryID); err != nil {
		return nil, err
	}
	item.Categories, _ = r.selectFactoryCategories(factoryID)
	item.SubCategories, _ = r.selectFactorySubCategories(factoryID)
	item.Certificates, _ = r.selectFactoryCertificates(factoryID)
	_ = r.db.Get(&item.Stats, `
		SELECT
			(SELECT COUNT(*) FROM orders WHERE factory_id = $1)::bigint AS total_orders,
			(SELECT COUNT(*) FROM quotations WHERE factory_id = $1)::bigint AS total_quotations,
			(SELECT COUNT(*) FROM factory_showcases WHERE factory_id = $1)::bigint AS total_showcases
	`, factoryID)
	return &item, nil
}

func (r *FactoryRepository) UpdateApprovalStatus(factoryID int64, status string, verifiedBy *int64, reason *string, noteSetsVerified bool) error {
	query := `
		UPDATE factory_profiles
		SET approval_status = $1,
		    is_verified = CASE WHEN $1 = 'AP' THEN TRUE ELSE FALSE END,
		    verified_at = CASE WHEN $1 = 'AP' THEN NOW() ELSE NULL END,
		    verified_by = CASE WHEN $1 = 'AP' THEN $2 ELSE verified_by END,
		    rejection_reason = CASE WHEN $1 IN ('RJ','SU') THEN $3 ELSE NULL END
		WHERE user_id = $4
	`
	res, err := r.db.Exec(query, status, nullableInt64Value(verifiedBy), nullableStringPtr(reason), factoryID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *FactoryRepository) GetApprovalStatus(factoryID int64) (string, error) {
	var status string
	err := r.db.Get(&status, `
		SELECT COALESCE(approval_status, 'PE')
		FROM factory_profiles
		WHERE user_id = $1
	`, factoryID)
	return status, err
}

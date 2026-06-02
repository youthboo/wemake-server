package admin

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	factoryrepo "github.com/yourusername/wemake/internal/repository/factory"
)

type AdminFactoryRepository struct {
	db        *sqlx.DB
	factories *factoryrepo.FactoryRepository
}

func NewAdminFactoryRepository(db *sqlx.DB, factories *factoryrepo.FactoryRepository) *AdminFactoryRepository {
	return &AdminFactoryRepository{db: db, factories: factories}
}

func (r *AdminFactoryRepository) ListAdmin(filter domain.AdminFactoryFilter) ([]domain.AdminFactoryListItem, int, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize

	conditions := sq.And{sq.Eq{"u.role": "FT"}}
	if filter.ApprovalStatus != "" {
		conditions = append(conditions, sq.Eq{"fp.approval_status": domainutil.NormalizeStatus(filter.ApprovalStatus)})
	}
	if filter.IsVerified != nil {
		if *filter.IsVerified {
			conditions = append(conditions, sq.Eq{"fp.approval_status": "AP"})
		} else {
			conditions = append(conditions, sq.NotEq{"fp.approval_status": "AP"})
		}
	}
	if filter.Search != "" {
		searchTerm := "%" + domainutil.NormalizeLower(filter.Search) + "%"
		conditions = append(conditions, sq.Or{
			sq.Like{"LOWER(fp.factory_name)": searchTerm},
			sq.Like{"LOWER(u.email)": searchTerm},
		})
	}

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	countQuery := psql.Select("COUNT(*)").
		From("factory_profiles fp").
		InnerJoin("users u ON u.user_id = fp.user_id").
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
		"fp.user_id AS factory_id",
		"fp.factory_name",
		"u.email",
		"NULLIF(u.phone, '') AS phone",
		"NULLIF(fp.tax_id, '') AS tax_id",
		"ft.type_name AS factory_type_name",
		"p.name_th AS province_name",
		"COALESCE(fp.approval_status, 'PE') AS approval_status",
		"( fp.approval_status = 'AP') AS is_verified",
		"fp.submitted_at",
		"fp.verified_at",
		"fp.verified_by",
		"fp.rejection_reason",
		"u.created_at",
	).
		From("factory_profiles fp").
		InnerJoin("users u ON u.user_id = fp.user_id").
		LeftJoin("lbi_factory_types ft ON ft.factory_type_id = fp.factory_type_id").
		LeftJoin("lbi_provinces p ON p.row_id = fp.province_id").
		Where(conditions).
		OrderBy("fp.submitted_at DESC NULLS LAST", "u.created_at DESC", "fp.user_id DESC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset))

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, 0, err
	}

	items := []domain.AdminFactoryListItem{}
	if err := r.db.Select(&items, sql, args...); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *AdminFactoryRepository) GetAdminDetail(factoryID int64) (*domain.AdminFactoryDetail, error) {
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
			ft.type_name AS specialization,
			fp.province_id,
			p.name_th AS province_name,
			fp.image_url,
			fp.description,
			COALESCE(fp.approval_status, 'PE') AS approval_status,
			( fp.approval_status = 'AP') AS is_verified,
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
	item.Categories, _ = r.factories.ListFactoryCategories(factoryID)
	item.SubCategories, _ = r.factories.ListFactorySubCategories(factoryID)
	item.Certificates, _ = r.selectFactoryCertificates(factoryID)
	_ = r.db.Get(&item.Stats, `
		SELECT
			(SELECT COUNT(*) FROM orders WHERE factory_id = $1)::bigint AS total_orders,
			(SELECT COUNT(*) FROM quotations WHERE factory_id = $1)::bigint AS total_quotations,
			(SELECT COUNT(*) FROM factory_showcases WHERE factory_id = $1)::bigint AS total_showcases
	`, factoryID)
	return &item, nil
}

func (r *AdminFactoryRepository) UpdateApprovalStatus(factoryID int64, status string, verifiedBy *int64, reason *string, noteSetsVerified bool) error {
	query := `
		UPDATE factory_profiles
		SET approval_status = $1,
		    verified_at = CASE WHEN $1 = 'AP' THEN NOW() ELSE NULL END,
		    verified_by = CASE WHEN $1 = 'AP' THEN $2 ELSE verified_by END,
		    rejection_reason = CASE WHEN $1 IN ('RJ','SU') THEN $3 ELSE NULL END
		WHERE user_id = $4
	`
	res, err := r.db.Exec(query, status, domainutil.Nullable(verifiedBy), domainutil.Nullable(reason), factoryID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// PatchCertificateStatus updates verify_status of a single certificate row.
// Allowed statuses: 'AP' (approve), 'RJ' (reject), 'PE' (reset to pending).
func (r *AdminFactoryRepository) PatchCertificateStatus(factoryID, mapID int64, status string) error {
	res, err := r.db.Exec(`
		UPDATE map_factory_certificates
		SET verify_status = $1
		WHERE map_id = $2 AND factory_id = $3
	`, status, mapID, factoryID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *AdminFactoryRepository) GetApprovalStatus(factoryID int64) (string, error) {
	var status string
	err := r.db.Get(&status, `
		SELECT COALESCE(approval_status, 'PE')
		FROM factory_profiles
		WHERE user_id = $1
	`, factoryID)
	return status, err
}

func (r *AdminFactoryRepository) selectFactoryCertificates(factoryID int64) ([]domain.FactoryProfileCertificate, error) {
	var items []domain.FactoryProfileCertificate
	q := `
		SELECT
			mfc.map_id,
			lc.cert_id,
			lc.cert_name,
			COALESCE(mfc.verify_status, 'PE') AS verify_status,
			NULLIF(mfc.document_url, '')       AS document_url,
			mfc.cert_number,
			mfc.expire_date::text              AS expire_date
		FROM map_factory_certificates mfc
		INNER JOIN lbi_certificates lc ON mfc.cert_id = lc.cert_id
		WHERE mfc.factory_id = $1
		ORDER BY lc.cert_id
	`
	if err := r.db.Select(&items, q, factoryID); err != nil {
		return nil, err
	}
	return items, nil
}

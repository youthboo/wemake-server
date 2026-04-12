package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/yourusername/wemake/internal/domain"
)

type FactoryRepository struct {
	db *sqlx.DB
}

func NewFactoryRepository(db *sqlx.DB) *FactoryRepository {
	return &FactoryRepository{db: db}
}

// ErrDuplicateFactoryCategory is returned when map_factory_categories unique pair exists.
var ErrDuplicateFactoryCategory = errors.New("factory already has this category")

// ErrInvalidFactoryCategory is returned when category_id is not a valid FK.
var ErrInvalidFactoryCategory = errors.New("invalid category_id")

func (r *FactoryRepository) ListPublicVerified() ([]domain.FactoryListItem, error) {
	var items []domain.FactoryListItem
	query := `
		SELECT
			fp.user_id AS factory_id,
			fp.factory_name,
			fp.factory_type_id,
			ft.type_name AS factory_type_name,
			fp.specialization,
			fp.rating::float8 AS rating,
			COALESCE(fp.review_count, 0)::bigint AS review_count,
			fp.min_order,
			fp.lead_time_desc,
			COALESCE(fp.is_verified, FALSE) AS is_verified,
			COALESCE(fp.completed_orders, 0)::bigint AS completed_orders,
			fp.image_url,
			fp.description,
			fp.price_range,
			fp.province_id,
			p.name_th AS province_name
		FROM factory_profiles fp
		INNER JOIN users u ON u.user_id = fp.user_id AND u.role = 'FT' AND u.is_active = TRUE
		LEFT JOIN lbi_factory_types ft ON ft.factory_type_id = fp.factory_type_id
		LEFT JOIN lbi_provinces p ON p.row_id = fp.province_id
		WHERE COALESCE(fp.is_verified, FALSE) = TRUE
		ORDER BY fp.rating DESC NULLS LAST, fp.factory_name ASC
	`
	err := r.db.Select(&items, query)
	return items, err
}

type factoryDetailHeadRow struct {
	FactoryID       int64           `db:"factory_id"`
	FactoryName     string          `db:"factory_name"`
	FactoryTypeID   int64           `db:"factory_type_id"`
	FactoryTypeName sql.NullString  `db:"factory_type_name"`
	TaxID           sql.NullString  `db:"tax_id"`
	Specialization  sql.NullString  `db:"specialization"`
	MinOrder        sql.NullInt64   `db:"min_order"`
	LeadTimeDesc    sql.NullString  `db:"lead_time_desc"`
	IsVerified      bool            `db:"is_verified"`
	Rating          sql.NullFloat64 `db:"rating"`
	ReviewCount     int64           `db:"review_count"`
	CompletedOrders int64           `db:"completed_orders"`
	ImageURL        sql.NullString  `db:"image_url"`
	Description     sql.NullString  `db:"description"`
	PriceRange      sql.NullString  `db:"price_range"`
	ProvinceID      sql.NullInt64   `db:"province_id"`
	ProvinceName    sql.NullString  `db:"province_name"`
}

func (r *FactoryRepository) factoryExistsActive(factoryID int64) (bool, error) {
	var ok bool
	q := `
		SELECT EXISTS(
			SELECT 1 FROM factory_profiles fp
			INNER JOIN users u ON u.user_id = fp.user_id
			WHERE fp.user_id = $1 AND u.role = 'FT' AND u.is_active = TRUE
		)
	`
	if err := r.db.Get(&ok, q, factoryID); err != nil {
		return false, err
	}
	return ok, nil
}

func (r *FactoryRepository) getFactoryDetailHead(factoryID int64) (factoryDetailHeadRow, error) {
	var head factoryDetailHeadRow
	headQuery := `
		SELECT
			fp.user_id AS factory_id,
			fp.factory_name,
			fp.factory_type_id,
			ft.type_name AS factory_type_name,
			fp.tax_id,
			fp.specialization,
			fp.min_order,
			fp.lead_time_desc,
			COALESCE(fp.is_verified, FALSE) AS is_verified,
			fp.rating::float8 AS rating,
			COALESCE(fp.review_count, 0)::bigint AS review_count,
			COALESCE(fp.completed_orders, 0)::bigint AS completed_orders,
			fp.image_url,
			fp.description,
			fp.price_range,
			fp.province_id,
			p.name_th AS province_name
		FROM factory_profiles fp
		INNER JOIN users u ON u.user_id = fp.user_id AND u.role = 'FT' AND u.is_active = TRUE
		LEFT JOIN lbi_factory_types ft ON ft.factory_type_id = fp.factory_type_id
		LEFT JOIN lbi_provinces p ON p.row_id = fp.province_id
		WHERE fp.user_id = $1
	`
	if err := r.db.Get(&head, headQuery, factoryID); err != nil {
		return factoryDetailHeadRow{}, err
	}
	return head, nil
}

func factoryDetailFromHead(head factoryDetailHeadRow) *domain.FactoryPublicDetail {
	out := &domain.FactoryPublicDetail{
		FactoryID:       head.FactoryID,
		FactoryName:     head.FactoryName,
		FactoryTypeID:   head.FactoryTypeID,
		IsVerified:      head.IsVerified,
		ReviewCount:     head.ReviewCount,
		CompletedOrders: head.CompletedOrders,
		Categories:      []domain.FactoryProfileCategory{},
		SubCategories:   []domain.FactoryProfileSubCategory{},
		Certificates:    []domain.FactoryProfileCertificate{},
		Reviews:         []domain.FactoryProfileReview{},
	}
	if head.FactoryTypeName.Valid {
		out.FactoryTypeName = &head.FactoryTypeName.String
	}
	if head.TaxID.Valid {
		out.TaxID = &head.TaxID.String
	}
	if head.Specialization.Valid {
		out.Specialization = &head.Specialization.String
	}
	if head.MinOrder.Valid {
		v := int(head.MinOrder.Int64)
		out.MinOrder = &v
	}
	if head.LeadTimeDesc.Valid {
		out.LeadTimeDesc = &head.LeadTimeDesc.String
	}
	if head.Rating.Valid {
		v := head.Rating.Float64
		out.Rating = &v
	}
	if head.ImageURL.Valid {
		out.ImageURL = &head.ImageURL.String
	}
	if head.Description.Valid {
		out.Description = &head.Description.String
	}
	if head.PriceRange.Valid {
		out.PriceRange = &head.PriceRange.String
	}
	if head.ProvinceID.Valid {
		out.ProvinceID = &head.ProvinceID.Int64
	}
	if head.ProvinceName.Valid {
		out.ProvinceName = &head.ProvinceName.String
	}
	return out
}

func (r *FactoryRepository) selectFactoryCategories(factoryID int64) ([]domain.FactoryProfileCategory, error) {
	var items []domain.FactoryProfileCategory
	q := `
		SELECT c.category_id, c.name
		FROM map_factory_categories mfc
		INNER JOIN categories c ON mfc.category_id = c.category_id
		WHERE mfc.factory_id = $1
		ORDER BY c.category_id
	`
	if err := r.db.Select(&items, q, factoryID); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FactoryRepository) selectFactorySubCategories(factoryID int64) ([]domain.FactoryProfileSubCategory, error) {
	var items []domain.FactoryProfileSubCategory
	q := `
		SELECT sc.sub_category_id, sc.name, sc.category_id
		FROM map_factory_sub_categories mfs
		INNER JOIN lbi_sub_categories sc ON mfs.sub_category_id = sc.sub_category_id
		WHERE mfs.factory_id = $1
		ORDER BY sc.category_id, sc.sort_order, sc.sub_category_id
	`
	if err := r.db.Select(&items, q, factoryID); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *FactoryRepository) selectFactoryCertificates(factoryID int64) ([]domain.FactoryProfileCertificate, error) {
	var items []domain.FactoryProfileCertificate
	q := `
		SELECT lc.cert_id, lc.cert_name, mfc.verify_status
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

type factoryReviewScanRow struct {
	ReviewID  int64          `db:"review_id"`
	UserID    int64          `db:"user_id"`
	Rating    int            `db:"rating"`
	Comment   sql.NullString `db:"comment"`
	CreatedAt time.Time      `db:"created_at"`
	FirstName sql.NullString `db:"first_name"`
	LastName  sql.NullString `db:"last_name"`
}

func (r *FactoryRepository) selectFactoryReviews(factoryID int64, limit int) ([]domain.FactoryProfileReview, error) {
	var revRows []factoryReviewScanRow
	q := `
		SELECT fr.review_id, fr.user_id, fr.rating, fr.comment, fr.created_at,
		       c.first_name, c.last_name
		FROM factory_reviews fr
		LEFT JOIN customers c ON c.user_id = fr.user_id
		WHERE fr.factory_id = $1
		ORDER BY fr.created_at DESC
		LIMIT $2
	`
	if err := r.db.Select(&revRows, q, factoryID, limit); err != nil {
		return nil, err
	}
	out := make([]domain.FactoryProfileReview, 0, len(revRows))
	for _, rw := range revRows {
		rwCopy := domain.FactoryProfileReview{
			ReviewID:  rw.ReviewID,
			UserID:    rw.UserID,
			Rating:    rw.Rating,
			CreatedAt: rw.CreatedAt,
		}
		if rw.Comment.Valid {
			s := rw.Comment.String
			rwCopy.Comment = &s
		}
		if rw.FirstName.Valid {
			s := rw.FirstName.String
			rwCopy.FirstName = &s
		}
		if rw.LastName.Valid {
			s := rw.LastName.String
			rwCopy.LastName = &s
		}
		out = append(out, rwCopy)
	}
	return out, nil
}

func (r *FactoryRepository) GetPublicDetail(factoryID int64) (*domain.FactoryPublicDetail, error) {
	head, err := r.getFactoryDetailHead(factoryID)
	if err != nil {
		return nil, err
	}
	out := factoryDetailFromHead(head)

	cats, err := r.selectFactoryCategories(factoryID)
	if err != nil {
		return nil, err
	}
	out.Categories = cats

	subs, err := r.selectFactorySubCategories(factoryID)
	if err != nil {
		return nil, err
	}
	out.SubCategories = subs

	certs, err := r.selectFactoryCertificates(factoryID)
	if err != nil {
		return nil, err
	}
	out.Certificates = certs

	reviews, err := r.selectFactoryReviews(factoryID, 10)
	if err != nil {
		return nil, err
	}
	out.Reviews = reviews

	return out, nil
}

// ListFactoryCategories returns categories linked to the factory (map_factory_categories).
func (r *FactoryRepository) ListFactoryCategories(factoryID int64) ([]domain.FactoryProfileCategory, error) {
	return r.selectFactoryCategories(factoryID)
}

// AddFactoryCategory inserts (factory_id, category_id). Caller must authorize factory owner.
func (r *FactoryRepository) AddFactoryCategory(factoryID, categoryID int64) error {
	var dup bool
	if err := r.db.Get(&dup, `
		SELECT EXISTS(
			SELECT 1 FROM map_factory_categories WHERE factory_id = $1 AND category_id = $2
		)`, factoryID, categoryID); err != nil {
		return err
	}
	if dup {
		return ErrDuplicateFactoryCategory
	}
	_, err := r.db.Exec(
		`INSERT INTO map_factory_categories (factory_id, category_id) VALUES ($1, $2)`,
		factoryID, categoryID,
	)
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			return ErrDuplicateFactoryCategory
		case "23503":
			return ErrInvalidFactoryCategory
		}
	}
	return err
}

// RemoveFactoryCategory deletes one mapping. Returns sql.ErrNoRows when no row removed.
func (r *FactoryRepository) RemoveFactoryCategory(factoryID, categoryID int64) error {
	res, err := r.db.Exec(
		`DELETE FROM map_factory_categories WHERE factory_id = $1 AND category_id = $2`,
		factoryID, categoryID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *FactoryRepository) FactoryExistsActive(factoryID int64) (bool, error) {
	return r.factoryExistsActive(factoryID)
}

var (
	ErrDuplicateFactorySubCategory = errors.New("factory already has this sub-category")
	ErrInvalidFactorySubCategory   = errors.New("invalid sub_category_id")
)

func (r *FactoryRepository) ListFactorySubCategories(factoryID int64) ([]domain.FactoryProfileSubCategory, error) {
	return r.selectFactorySubCategories(factoryID)
}

func (r *FactoryRepository) AddFactorySubCategory(factoryID, subCategoryID int64) error {
	var dup bool
	if err := r.db.Get(&dup, `
		SELECT EXISTS(
			SELECT 1 FROM map_factory_sub_categories WHERE factory_id = $1 AND sub_category_id = $2
		)`, factoryID, subCategoryID); err != nil {
		return err
	}
	if dup {
		return ErrDuplicateFactorySubCategory
	}
	_, err := r.db.Exec(
		`INSERT INTO map_factory_sub_categories (factory_id, sub_category_id) VALUES ($1, $2)`,
		factoryID, subCategoryID,
	)
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			return ErrDuplicateFactorySubCategory
		case "23503":
			return ErrInvalidFactorySubCategory
		}
	}
	return err
}

func (r *FactoryRepository) RemoveFactorySubCategory(factoryID, subCategoryID int64) error {
	res, err := r.db.Exec(
		`DELETE FROM map_factory_sub_categories WHERE factory_id = $1 AND sub_category_id = $2`,
		factoryID, subCategoryID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

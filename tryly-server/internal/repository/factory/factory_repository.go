package factory

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/yourusername/wemake/internal/domain"
)

func isSchemaDriftError(err error) bool {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return false
	}
	// 42P01: undefined_table, 42703: undefined_column
	return pqErr.Code == "42P01" || pqErr.Code == "42703"
}

type FactoryRepository struct {
	db *sqlx.DB
}

func NewFactoryRepository(db *sqlx.DB) *FactoryRepository {
	return &FactoryRepository{db: db}
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

// ErrFactoryProfileExists is returned when the user already has a factory_profiles record.
var ErrFactoryProfileExists = errors.New("factory profile already exists")

// ErrDuplicateFactoryCategory is returned when map_factory_categories unique pair exists.
var ErrDuplicateFactoryCategory = errors.New("factory already has this category")

// ErrInvalidFactoryCategory is returned when category_id is not a valid FK.
var ErrInvalidFactoryCategory = errors.New("invalid category_id")


// ListPublicVerified returns active factories for the public explore listing.
// Pass an optional scope ("PD" or "MT") to restrict to factories that have at
// least one linked category with that scope in map_factory_categories.
func (r *FactoryRepository) ListPublicVerified(scope ...string) ([]domain.FactoryListItem, error) {
	var items []domain.FactoryListItem
	filterScope := ""
	if len(scope) > 0 {
		filterScope = strings.ToUpper(strings.TrimSpace(scope[0]))
	}

	query := `
		SELECT
			fp.user_id AS factory_id,
			fp.factory_name,
			fp.description AS specialization,
			COALESCE(rev.avg_rating, fp.rating, 0)::float8 AS rating,
			COALESCE(rev.review_cnt, fp.review_count, 0)::bigint AS review_count,
			fp.lead_time_desc,
			(fp.approval_status = 'AP') AS is_verified,
			COALESCE(fp.completed_orders, 0)::bigint AS completed_orders,
			fp.image_url,
			fp.background_image_url,
			fp.description,
			NULL::text AS price_range,
			fp.province_id,
			p.name_th AS province_name,
			COALESCE(
				(
					SELECT jsonb_agg(c.name ORDER BY c.name)::text
					FROM map_factory_categories mfc
					JOIN lbi_categories c ON c.category_id = mfc.category_id
					WHERE mfc.factory_id = fp.user_id
				),
				'[]'
			) AS tags,
			COALESCE(cats.category_scopes, '[]') AS category_scopes
		FROM factory_profiles fp
		INNER JOIN users u ON u.user_id = fp.user_id AND u.role = 'FT' AND u.is_active = TRUE
		LEFT JOIN lbi_provinces p ON p.row_id = fp.province_id
		LEFT JOIN (
			SELECT factory_id::bigint AS factory_id,
			       ROUND(AVG(rating::numeric), 2)::float8 AS avg_rating,
			       COUNT(*)::bigint AS review_cnt
			FROM factory_reviews
			GROUP BY factory_id
		) rev ON rev.factory_id = fp.user_id
		LEFT JOIN (
			SELECT mfc.factory_id,
				array_to_json(array_agg(DISTINCT COALESCE(c.scope, 'PD')::text))::text AS category_scopes
			FROM map_factory_categories mfc
			JOIN lbi_categories c ON c.category_id = mfc.category_id
			GROUP BY mfc.factory_id
		) cats ON cats.factory_id = fp.user_id
		WHERE u.is_active = TRUE`

	var args []interface{}
	if filterScope != "" {
		query += ` AND EXISTS (
			SELECT 1 FROM map_factory_categories mfc2
			JOIN lbi_categories c2 ON c2.category_id = mfc2.category_id
			WHERE mfc2.factory_id = fp.user_id AND COALESCE(c2.scope, 'PD') = $1
		)`
		args = append(args, filterScope)
	}
	query += "\n\t\tORDER BY COALESCE(rev.avg_rating, fp.rating) DESC NULLS LAST, fp.factory_name ASC"

	err := r.db.Select(&items, query, args...)
	return items, err
}

type factoryDetailHeadRow struct {
	FactoryID          int64           `db:"factory_id"`
	FactoryName        string          `db:"factory_name"`
	TaxID              sql.NullString  `db:"tax_id"`
	Specialization     sql.NullString  `db:"specialization"`
	LeadTimeDesc       sql.NullString  `db:"lead_time_desc"`
	IsVerified         bool            `db:"is_verified"`
	Rating             sql.NullFloat64 `db:"rating"`
	ReviewCount        int64           `db:"review_count"`
	CompletedOrders    int64           `db:"completed_orders"`
	ImageURL           sql.NullString  `db:"image_url"`
	BackgroundImageURL sql.NullString  `db:"background_image_url"`
	Description        sql.NullString  `db:"description"`
	PriceRange         sql.NullString  `db:"price_range"`
	ProvinceID         sql.NullInt64   `db:"province_id"`
	ProvinceName       sql.NullString  `db:"province_name"`
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
			fp.tax_id,
			fp.description AS specialization,
			fp.lead_time_desc,
			(fp.approval_status = 'AP') AS is_verified,
			COALESCE(rev.avg_rating, fp.rating, 0)::float8 AS rating,
			COALESCE(rev.review_cnt, fp.review_count, 0)::bigint AS review_count,
			COALESCE(fp.completed_orders, 0)::bigint AS completed_orders,
			fp.image_url,
			fp.background_image_url,
			fp.description,
			NULL::text AS price_range,
			fp.province_id,
			p.name_th AS province_name
		FROM factory_profiles fp
		INNER JOIN users u ON u.user_id = fp.user_id AND u.role = 'FT' AND u.is_active = TRUE
		LEFT JOIN lbi_provinces p ON p.row_id = fp.province_id
		LEFT JOIN (
			SELECT factory_id::bigint AS factory_id,
			       ROUND(AVG(rating::numeric), 2)::float8 AS avg_rating,
			       COUNT(*)::bigint AS review_cnt
			FROM factory_reviews
			GROUP BY factory_id
		) rev ON rev.factory_id = fp.user_id
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
		IsVerified:      head.IsVerified,
		ReviewCount:     head.ReviewCount,
		CompletedOrders: head.CompletedOrders,
		Categories:      []domain.FactoryProfileCategory{},
		SubCategories:   []domain.FactoryProfileSubCategory{},
		Certificates:    []domain.FactoryProfileCertificate{},
		Reviews:         []domain.FactoryProfileReview{},
	}
	if head.TaxID.Valid {
		out.TaxID = &head.TaxID.String
	}
	if head.Specialization.Valid {
		out.Specialization = &head.Specialization.String
	}
	if head.LeadTimeDesc.Valid {
		out.LeadTimeDesc = &head.LeadTimeDesc.String
		out.LeadTimeDese = &head.LeadTimeDesc.String
	}
	if head.Rating.Valid {
		v := head.Rating.Float64
		out.Rating = &v
	}
	if head.ImageURL.Valid {
		out.ImageURL = &head.ImageURL.String
	}
	if head.BackgroundImageURL.Valid {
		out.BackgroundImageURL = &head.BackgroundImageURL.String
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
		INNER JOIN lbi_categories c ON mfc.category_id = c.category_id
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
	// lbi_sub_categories.category_id references categories; join categories for
	// parent category_name. sub_category_name aliases Name for the FE.
	q := `
		SELECT sc.sub_category_id,
		       sc.category_id,
		       COALESCE(cat.name, '') AS category_name,
		       sc.name,
		       sc.name AS sub_category_name
		FROM map_factory_sub_categories mfs
		INNER JOIN lbi_sub_categories sc ON mfs.sub_category_id = sc.sub_category_id
		LEFT JOIN lbi_categories cat ON cat.category_id = sc.category_id
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
		SELECT mfc.map_id,
		       lc.cert_id,
		       lc.cert_name,
		       mfc.verify_status,
		       mfc.document_url,
		       mfc.cert_number,
		       mfc.expire_date::text AS expire_date
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
	ReviewID  int64              `db:"review_id"`
	UserID    int64              `db:"user_id"`
	Rating    int                `db:"rating"`
	Comment   sql.NullString     `db:"comment"`
	ImageURLs domain.StringArray `db:"image_urls"`
	CreatedAt time.Time          `db:"created_at"`
	FirstName sql.NullString     `db:"first_name"`
	LastName  sql.NullString     `db:"last_name"`
}

func (r *FactoryRepository) selectFactoryReviews(factoryID int64, limit int) ([]domain.FactoryProfileReview, error) {
	var revRows []factoryReviewScanRow
	q := `
		SELECT fr.review_id, fr.user_id, fr.rating, fr.comment, fr.image_urls, fr.created_at,
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
			ImageURLs: rw.ImageURLs,
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

func (r *FactoryRepository) CreateProfile(userID int64, factoryName string, taxID string, provinceID *int64, categoryIDs []int64, subCategoryIDs []int64, certID int64, documentURL string, certNumber string, certExpireDate string) error {
	var pid sql.NullInt64
	if provinceID != nil && *provinceID > 0 {
		pid = sql.NullInt64{Int64: *provinceID, Valid: true}
	}
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.Exec(`
		INSERT INTO factory_profiles (user_id, factory_name, tax_id, province_id, review_count, completed_orders, approval_status, submitted_at)
		VALUES ($1, $2, $3, $4, 0, 0, 'PE', NOW())
	`, userID, factoryName, taxID, pid); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrFactoryProfileExists
		}
		return err
	}
	for _, catID := range categoryIDs {
		if _, err = tx.Exec(`INSERT INTO map_factory_categories (factory_id, category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, catID); err != nil {
			return err
		}
	}
	for _, subID := range subCategoryIDs {
		if _, err = tx.Exec(`INSERT INTO map_factory_sub_categories (factory_id, sub_category_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, subID); err != nil {
			return err
		}
	}
	// Insert certificate record if cert data is provided
	if certID > 0 && documentURL != "" {
		var expireDate interface{}
		if certExpireDate != "" {
			if t, parseErr := time.Parse("2006-01-02", certExpireDate); parseErr == nil {
				expireDate = t
			}
		}
		var certNum interface{}
		if certNumber != "" {
			certNum = certNumber
		}
		if _, err = tx.Exec(`
			INSERT INTO map_factory_certificates (factory_id, cert_id, document_url, expire_date, cert_number, verify_status)
			VALUES ($1, $2, $3, $4, $5, 'PE')
		`, userID, certID, documentURL, expireDate, certNum); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *FactoryRepository) PatchProfile(factoryID int64, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	allowed := map[string]bool{
		"factory_name":         true,
		"tax_id":               true,
		"description":          true,
		"lead_time_desc":       true,
		"image_url":            true,
		"background_image_url": true,
		"province_id":          true,
	}

	setClauses := make([]string, 0, len(fields))
	args := make([]interface{}, 0, len(fields)+1)
	i := 1
	for key, value := range fields {
		if !allowed[key] {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, i))
		args = append(args, value)
		i++
	}
	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, factoryID)
	query := fmt.Sprintf(
		"UPDATE factory_profiles SET %s WHERE user_id = $%d",
		strings.Join(setClauses, ", "),
		i,
	)
	res, err := r.db.Exec(query, args...)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
		}
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

// ListFactoryCategories returns categories linked to the factory (map_factory_categories).
// AddFactoryCategory inserts (factory_id, category_id). Caller must authorize factory owner.
// RemoveFactoryCategory deletes one mapping. Returns sql.ErrNoRows when no row removed.
var (
	ErrDuplicateFactorySubCategory = errors.New("factory already has this sub-category")
	ErrInvalidFactorySubCategory   = errors.New("invalid sub_category_id")
)

func (r *FactoryRepository) GetDashboard(factoryID int64) (*domain.FactoryDashboard, error) {
	var out domain.FactoryDashboard
	out.FactoryID = factoryID
	out.RecentMatchingRFQs = []domain.FactoryDashboardRFQItem{}
	out.RecentOrders = []domain.FactoryDashboardOrderItem{}
	out.RecentQuotations = []domain.FactoryDashboardQuotationItem{}
	out.RecentShowcases = []domain.FactoryDashboardShowcaseItem{}

	if err := r.db.Get(&out.Counts.PendingRFQs, `
		SELECT COUNT(DISTINCT r.rfq_id)
		FROM rfqs r
		INNER JOIN map_factory_categories mfc
			ON mfc.factory_id = $1
		   AND mfc.category_id = r.category_id
		LEFT JOIN lbi_sub_categories sc ON sc.sub_category_id = r.sub_category_id
		WHERE r.status = 'OP'
		  AND (
			r.sub_category_id IS NULL
			OR COALESCE(sc.sort_order, 0) = 99
			OR EXISTS (
				SELECT 1
				FROM map_factory_sub_categories mfs
				WHERE mfs.factory_id = $1
				  AND mfs.sub_category_id = r.sub_category_id
			)
		  )
	`, factoryID); err != nil {
		// Keep dashboard available even when this section query fails (schema/data drift).
		out.Counts.PendingRFQs = 0
	}

	if err := r.db.Get(&out.Counts.ActiveOrders, `
		SELECT COUNT(*)
		FROM orders
		WHERE factory_id = $1
		  AND status IN ('PR', 'QC', 'SH')
	`, factoryID); err != nil {
		// Keep dashboard available even when this section query fails (schema/data drift).
		out.Counts.ActiveOrders = 0
	}

	if err := r.db.Get(&out.Counts.PendingProductionUpdates, `
		SELECT COUNT(*)
		FROM production_updates pu
		INNER JOIN orders o ON o.order_id = pu.order_id
		WHERE o.factory_id = $1
		  AND pu.status = 'CR'
	`, factoryID); err != nil {
		// Keep dashboard available even when this section query fails (schema/data drift).
		out.Counts.PendingProductionUpdates = 0
	}

	if err := r.db.Get(&out.Counts.UnreadMessages, `
		SELECT COALESCE(SUM(unread_factory), 0)
		FROM conversations
		WHERE factory_id = $1
	`, factoryID); err != nil {
		// Keep dashboard available even when this section query fails (schema/data drift).
		out.Counts.UnreadMessages = 0
	}

	if err := r.db.Get(&out.Counts.UnreadNotifications, `
		SELECT COUNT(*)
		FROM notifications
		WHERE user_id = $1
		  AND is_read = FALSE
	`, factoryID); err != nil {
		// Keep dashboard available even when this section query fails (schema/data drift).
		out.Counts.UnreadNotifications = 0
	}

	if err := r.db.Get(&out.Wallet, `
		SELECT good_fund, pending_fund
		FROM wallets
		WHERE user_id = $1
	`, factoryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			out.Wallet = domain.FactoryDashboardWallet{}
		} else {
			// Keep dashboard available even when wallet row/schema is not ready.
			out.Wallet = domain.FactoryDashboardWallet{}
		}
	}

	if err := r.db.Select(&out.RecentMatchingRFQs, `
		SELECT DISTINCT r.rfq_id, r.title, r.category_id, r.sub_category_id, r.status, r.created_at
		FROM rfqs r
		INNER JOIN map_factory_categories mfc
			ON mfc.factory_id = $1
		   AND mfc.category_id = r.category_id
		LEFT JOIN lbi_sub_categories sc ON sc.sub_category_id = r.sub_category_id
		WHERE r.status = 'OP'
		  AND (
			r.sub_category_id IS NULL
			OR COALESCE(sc.sort_order, 0) = 99
			OR EXISTS (
				SELECT 1
				FROM map_factory_sub_categories mfs
				WHERE mfs.factory_id = $1
				  AND mfs.sub_category_id = r.sub_category_id
			)
		  )
		ORDER BY r.created_at DESC
		LIMIT 5
	`, factoryID); err != nil {
		// Keep dashboard available even when this section query fails (schema/data drift).
		out.RecentMatchingRFQs = []domain.FactoryDashboardRFQItem{}
	}

	if err := r.db.Select(&out.RecentOrders, `
		SELECT order_id, quote_id, user_id, status, total_amount, estimated_delivery, created_at
		FROM orders
		WHERE factory_id = $1
		ORDER BY updated_at DESC, created_at DESC
		LIMIT 5
	`, factoryID); err != nil {
		// Backward compatibility for schemas that do not have orders.updated_at.
		if isSchemaDriftError(err) {
			if err2 := r.db.Select(&out.RecentOrders, `
				SELECT order_id, quote_id, user_id, status, total_amount, estimated_delivery, created_at
				FROM orders
				WHERE factory_id = $1
				ORDER BY created_at DESC
				LIMIT 5
			`, factoryID); err2 != nil {
				// Keep dashboard available even when fallback query also fails.
				out.RecentOrders = []domain.FactoryDashboardOrderItem{}
			}
		} else {
			// Keep dashboard available even when this section query fails (schema/data drift).
			out.RecentOrders = []domain.FactoryDashboardOrderItem{}
		}
	}

	if err := r.db.Select(&out.RecentQuotations, `
		SELECT quote_id, rfq_id, status, price_per_piece, lead_time_days, log_timestamp
		FROM quotations
		WHERE factory_id = $1
		ORDER BY log_timestamp DESC, create_time DESC
		LIMIT 5
	`, factoryID); err != nil {
		// Keep dashboard available even when this section query fails (schema/data drift).
		out.RecentQuotations = []domain.FactoryDashboardQuotationItem{}
	}

	if err := r.db.Select(&out.RecentShowcases, `
		SELECT showcase_id, content_type, title, category_id, sub_category_id, created_at
		FROM factory_showcases
		WHERE factory_id = $1
		ORDER BY created_at DESC
		LIMIT 5
	`, factoryID); err != nil {
		// Keep dashboard available even when this section query fails (schema/data drift).
		out.RecentShowcases = []domain.FactoryDashboardShowcaseItem{}
	}

	return &out, nil
}

// GetAnalytics returns aggregate stats for GET /factories/me/analytics.
func (r *FactoryRepository) GetAnalytics(factoryID int64) (*domain.FactoryAnalytics, error) {
	out := &domain.FactoryAnalytics{FactoryID: factoryID}

	if err := r.db.QueryRow(`
		SELECT
			COUNT(*) AS total_orders,
			COUNT(*) FILTER (WHERE status = 'CP') AS completed_orders,
			COUNT(*) FILTER (WHERE status IN ('PR','QC','SH','PP','WF','DL')) AS active_orders,
			COUNT(*) FILTER (WHERE status = 'CC') AS cancelled_orders,
			COALESCE(SUM(total_amount) FILTER (WHERE status = 'CP'), 0) AS total_revenue
		FROM orders
		WHERE factory_id = $1
	`, factoryID).Scan(
		&out.TotalOrders, &out.CompletedOrders, &out.ActiveOrders,
		&out.CancelledOrders, &out.TotalRevenue,
	); err != nil {
		return nil, err
	}

	if err := r.db.QueryRow(`
		SELECT
			COUNT(*) AS total_quotations,
			COUNT(*) FILTER (WHERE status = 'AC') AS accepted_quotes,
			COUNT(*) FILTER (WHERE status = 'PD') AS pending_quotes
		FROM quotations
		WHERE factory_id = $1
	`, factoryID).Scan(&out.TotalQuotations, &out.AcceptedQuotes, &out.PendingQuotes); err != nil {
		return nil, err
	}

	if err := r.db.QueryRow(`
		SELECT
			COUNT(*) AS total_showcases,
			0 AS total_views,
			COALESCE(SUM(likes_count), 0) AS total_likes
		FROM factory_showcases
		WHERE factory_id = $1
	`, factoryID).Scan(&out.TotalShowcases, &out.TotalViews, &out.TotalLikes); err != nil {
		return nil, err
	}

	if err := r.db.QueryRow(`
		SELECT
			COALESCE(AVG(rating), 0) AS average_rating,
			COUNT(*) AS total_reviews
		FROM factory_reviews
		WHERE factory_id = $1
	`, factoryID).Scan(&out.AverageRating, &out.TotalReviews); err != nil {
		return nil, err
	}

	return out, nil
}

// GetPortal returns all data needed for the factory dashboard page in one call,
// replacing the six separate API requests the FE previously made.
func (r *FactoryRepository) GetPortal(factoryID int64) (*domain.FactoryPortal, error) {
	// 1. Analytics summary — reuse existing method.
	analytics, err := r.GetAnalytics(factoryID)
	if err != nil {
		return nil, err
	}

	// 2. Dashboard counts, wallet, and recent-5 slices — reuse existing method.
	dashboard, err := r.GetDashboard(factoryID)
	if err != nil {
		return nil, err
	}

	// 3. Full order list (lightweight) for chart series.
	var orders []domain.PortalOrderItem
	if err := r.db.Select(&orders, `
		SELECT order_id, factory_id, status, total_amount::float8 AS total_amount, created_at
		FROM orders
		WHERE factory_id = $1
		ORDER BY created_at DESC
	`, factoryID); err != nil {
		orders = []domain.PortalOrderItem{}
	}

	// 4. Full quotation list (lightweight) for chart series.
	var quotations []domain.PortalQuotationItem
	if err := r.db.Select(&quotations, `
		SELECT quote_id, factory_id, status, create_time AS created_at
		FROM quotations
		WHERE factory_id = $1
		ORDER BY create_time DESC
	`, factoryID); err != nil {
		quotations = []domain.PortalQuotationItem{}
	}

	// 5. Full matching-RFQ list (all open RFQs matching factory categories).
	var matchingRFQs []domain.FactoryDashboardRFQItem
	if err := r.db.Select(&matchingRFQs, `
		SELECT DISTINCT r.rfq_id, r.title, r.category_id, r.sub_category_id, r.status, r.created_at
		FROM rfqs r
		INNER JOIN map_factory_categories mfc
			ON mfc.factory_id = $1
		   AND mfc.category_id = r.category_id
		LEFT JOIN lbi_sub_categories sc ON sc.sub_category_id = r.sub_category_id
		WHERE r.status = 'OP'
		  AND (
			r.sub_category_id IS NULL
			OR COALESCE(sc.sort_order, 0) = 99
			OR EXISTS (
				SELECT 1 FROM map_factory_sub_categories mfs
				WHERE mfs.factory_id = $1 AND mfs.sub_category_id = r.sub_category_id
			)
		  )
		ORDER BY r.created_at DESC
	`, factoryID); err != nil {
		matchingRFQs = []domain.FactoryDashboardRFQItem{}
	}

	// 6. RFQ history — every RFQ this factory has ever submitted a quotation for,
	// with the original RFQ created_at. Used for the per-period chart series so the
	// "RFQ received" bars reflect real timestamps instead of falling back to estimates.
	var rfqHistory []domain.RFQHistoryItem
	if err := r.db.Select(&rfqHistory, `
		SELECT DISTINCT r.rfq_id, r.created_at
		FROM rfqs r
		JOIN quotations q ON q.rfq_id = r.rfq_id
		WHERE q.factory_id = $1
		ORDER BY r.created_at DESC
	`, factoryID); err != nil {
		rfqHistory = []domain.RFQHistoryItem{}
	}

	return &domain.FactoryPortal{
		Analytics:        analytics,
		Counts:           dashboard.Counts,
		Wallet:           dashboard.Wallet,
		MatchingRFQs:     matchingRFQs,
		RFQHistory:       rfqHistory,
		Orders:           orders,
		Quotations:       quotations,
		RecentRFQs:       dashboard.RecentMatchingRFQs,
		RecentOrders:     dashboard.RecentOrders,
		RecentQuotations: dashboard.RecentQuotations,
		RecentShowcases:  dashboard.RecentShowcases,
	}, nil
}

// GetFactoryName returns the factory_name for the given factory_id.
// Returns empty string if not found (caller should use fallback).
func (r *FactoryRepository) GetFactoryName(factoryID int64) string {
	var name string
	_ = r.db.Get(&name, `SELECT COALESCE(factory_name, '') FROM factory_profiles WHERE user_id = $1`, factoryID)
	return strings.TrimSpace(name)
}

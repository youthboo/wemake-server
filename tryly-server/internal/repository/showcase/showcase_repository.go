package showcase

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/helper"
)

type ShowcaseRepository struct {
	db *sqlx.DB
}

func NewShowcaseRepository(db *sqlx.DB) *ShowcaseRepository {
	return &ShowcaseRepository{db: db}
}

const showcaseExploreBaseSQL = `
	SELECT
		fs.showcase_id,
		fs.factory_id,
		fs.content_type,
		fs.title,
		NULL::text AS excerpt,
		NULLIF(fs.linked_showcases->>0, '') AS image_url,
		fs.category_id,
		fs.sub_category_id,
		fs.moq, fs.unit_id, fs.lead_time_days,
		fs.base_price,
		fs.promo_price,
		fs.start_date,
		fs.end_date,
		fs.linked_showcases,
		'[]'::jsonb AS tags,
		fs.likes_count,
		0::bigint AS view_count,
		fs.status,
		fs.created_at,
		fs.updated_at,
		fs.published_at,
		fp.factory_name,
		fp.image_url AS factory_image_url,
		fp.rating::float8 AS factory_rating,
		(fp.approval_status = 'AP') AS factory_verified,
		p.name_th AS province_name,
		c.name AS category_name,
		sc.name AS sub_category_name
	FROM factory_showcases fs
	INNER JOIN factory_profiles fp ON fs.factory_id = fp.user_id
	LEFT JOIN lbi_provinces p ON fp.province_id = p.row_id
	LEFT JOIN lbi_categories c ON fs.category_id = c.category_id
	LEFT JOIN lbi_sub_categories sc ON fs.sub_category_id = sc.sub_category_id
`

func (r *ShowcaseRepository) ListExplore(contentType string) ([]domain.ShowcaseExploreItem, error) {
	var items []domain.ShowcaseExploreItem
	var query string
	var args []interface{}
	if contentType != "" {
		query = showcaseExploreBaseSQL + ` WHERE fs.status = 'AC' AND fp.approval_status = 'AP' AND fs.content_type = $1 ORDER BY fs.created_at DESC`
		args = append(args, contentType)
	} else {
		query = showcaseExploreBaseSQL + ` WHERE fs.status = 'AC' AND fp.approval_status = 'AP' ORDER BY fs.created_at DESC`
	}
	err := r.db.Select(&items, query, args...)
	return items, err
}

func (r *ShowcaseRepository) ListExploreByFactory(factoryID int64, contentType string) ([]domain.ShowcaseExploreItem, error) {
	var items []domain.ShowcaseExploreItem
	clauses := []string{"fs.factory_id = $1", "fs.status = 'AC'"}
	args := []interface{}{factoryID}
	argPos := 2
	if contentType != "" {
		clauses = append(clauses, fmt.Sprintf("fs.content_type = $%d", argPos))
		args = append(args, contentType)
		argPos++
	}
	query := showcaseExploreBaseSQL + ` WHERE ` + strings.Join(clauses, " AND ") + ` ORDER BY fs.created_at DESC`
	err := r.db.Select(&items, query, args...)
	return items, err
}

func (r *ShowcaseRepository) ListStructured(filter domain.ShowcaseListFilter) ([]domain.ShowcaseExploreItem, error) {
	var items []domain.ShowcaseExploreItem
	clauses := []string{}
	args := []interface{}{}
	argPos := 1

	if filter.FactoryID != nil {
		clauses = append(clauses, fmt.Sprintf("fs.factory_id = $%d", argPos))
		args = append(args, *filter.FactoryID)
		argPos++
	}
	if filter.Type != "" {
		clauses = append(clauses, fmt.Sprintf("fs.content_type = $%d", argPos))
		args = append(args, filter.Type)
		argPos++
	}
	if filter.Status != "" {
		clauses = append(clauses, fmt.Sprintf("fs.status = $%d", argPos))
		args = append(args, filter.Status)
		argPos++
		if filter.Status != "AC" && (filter.FactoryID == nil || filter.ViewerID == 0 || *filter.FactoryID != filter.ViewerID) {
			clauses = append(clauses, "1 = 0")
		}
	} else if filter.FactoryID == nil || filter.ViewerID == 0 || *filter.FactoryID != filter.ViewerID {
		clauses = append(clauses, "fs.status = 'AC'")
	}
	if filter.CategoryID != nil {
		clauses = append(clauses, fmt.Sprintf("fs.category_id = $%d", argPos))
		args = append(args, *filter.CategoryID)
		argPos++
	}
	if filter.SubCategoryID != nil {
		clauses = append(clauses, fmt.Sprintf("fs.sub_category_id = $%d", argPos))
		args = append(args, *filter.SubCategoryID)
		argPos++
	}

	query := showcaseExploreBaseSQL
	if len(clauses) > 0 {
		query += ` WHERE ` + strings.Join(clauses, " AND ")
	}
	query += ` ORDER BY fs.updated_at DESC, fs.created_at DESC`
	err := r.db.Select(&items, query, args...)
	return items, err
}

// GetShowcasesByFactory returns showcases for a factory page.
// callerID=0 means public → only AC status; callerID==factoryID means owner → all statuses.
func (r *ShowcaseRepository) GetShowcasesByFactory(factoryID int64, contentType string, callerID int64) ([]domain.ShowcaseByFactoryItem, error) {
	var items []domain.ShowcaseByFactoryItem

	clauses := []string{"fs.factory_id = $1"}
	args := []interface{}{factoryID}
	argPos := 2

	if callerID != factoryID {
		clauses = append(clauses, "fs.status = 'AC'")
	}
	if contentType != "" {
		clauses = append(clauses, fmt.Sprintf("fs.content_type = $%d", argPos))
		args = append(args, contentType)
		argPos++
	}

	basePriceExpr := "NULL::numeric AS base_price"
	if hasBasePrice, _ := r.hasFactoryShowcaseColumn("base_price"); hasBasePrice {
		basePriceExpr = "fs.base_price"
	}
	leadTimeExpr := "NULL::int AS lead_time_days"
	if hasLeadTimeDays, _ := r.hasFactoryShowcaseColumn("lead_time_days"); hasLeadTimeDays {
		leadTimeExpr = "fs.lead_time_days"
	}
	contentTypeExpr := "NULL::text AS content_type"
	if hasContentType, _ := r.hasFactoryShowcaseColumn("content_type"); hasContentType {
		contentTypeExpr = "fs.content_type"
	} else if hasLegacyType, _ := r.hasFactoryShowcaseColumn("type"); hasLegacyType {
		contentTypeExpr = `fs."type" AS content_type`
	}
	subCategoryJoin := "LEFT JOIN lbi_sub_categories sc ON fs.sub_category_id = sc.sub_category_id"
	if ok, _ := r.hasTable("lbi_sub_categories"); !ok {
		if hasLegacySubTable, _ := r.hasTable("sub_categories"); hasLegacySubTable {
			subCategoryJoin = "LEFT JOIN sub_categories sc ON fs.sub_category_id = sc.sub_category_id"
		} else {
			subCategoryJoin = ""
		}
	}
	subCategoryNameExpr := "sc.name AS sub_category_name"
	if subCategoryJoin == "" {
		subCategoryNameExpr = "NULL::text AS sub_category_name"
	}

	query := `
		SELECT
			fs.showcase_id, ` + contentTypeExpr + `, fs.title,
			NULL::text AS excerpt, NULLIF(fs.linked_showcases->>0, '') AS image_url,
			fs.category_id, fs.sub_category_id,
			fs.moq, fs.unit_id, ` + basePriceExpr + `, ` + leadTimeExpr + `,
			fs.likes_count, fs.status, fs.created_at,
			c.name  AS category_name,
			` + subCategoryNameExpr + `
		FROM factory_showcases fs
		LEFT JOIN lbi_categories c         ON fs.category_id     = c.category_id
		` + subCategoryJoin + `
		WHERE ` + strings.Join(clauses, " AND ") + `
		ORDER BY fs.created_at DESC`

	err := r.db.Select(&items, query, args...)
	return items, err
}

func (r *ShowcaseRepository) hasFactoryShowcaseColumn(columnName string) (bool, error) {
	var exists bool
	err := r.db.Get(&exists, `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'factory_showcases'
			  AND column_name = $1
		)
	`, columnName)
	return exists, err
}

func (r *ShowcaseRepository) hasTable(tableName string) (bool, error) {
	var exists bool
	err := r.db.Get(&exists, `SELECT to_regclass('public.' || $1) IS NOT NULL`, tableName)
	return exists, err
}

// sectionRow is the flat row returned by the sections+items JOIN query.
type sectionRow struct {
	SectionID    int64   `db:"section_id"`
	SectionType  string  `db:"section_type"`
	SectionTitle string  `db:"section_title"`
	SortOrder    int     `db:"sort_order"`
	ItemID       *int64  `db:"item_id"`
	ItemTitle    *string `db:"item_title"`
	Description  *string `db:"item_description"`
	IconName     *string `db:"icon_name"`
	ItemSort     *int    `db:"item_sort_order"`
}

// aggregateSections converts flat sectionRows into nested ShowcaseSections.
func aggregateSections(rows []sectionRow) []domain.ShowcaseSection {
	sectionMap := map[int64]*domain.ShowcaseSection{}
	var order []int64
	for _, row := range rows {
		if _, ok := sectionMap[row.SectionID]; !ok {
			sec := &domain.ShowcaseSection{
				SectionID:    row.SectionID,
				SectionType:  row.SectionType,
				SectionTitle: row.SectionTitle,
				SortOrder:    row.SortOrder,
				Items:        []domain.ShowcaseSectionItem{},
			}
			sectionMap[row.SectionID] = sec
			order = append(order, row.SectionID)
		}
		if row.ItemID != nil {
			sectionMap[row.SectionID].Items = append(sectionMap[row.SectionID].Items, domain.ShowcaseSectionItem{
				ItemID:      *row.ItemID,
				Title:       row.ItemTitle,
				Description: helper.DerefString(row.Description),
				IconName:    row.IconName,
				SortOrder:   domainutil.IntValue(row.ItemSort),
			})
		}
	}
	sections := make([]domain.ShowcaseSection, 0, len(order))
	for _, id := range order {
		sections = append(sections, *sectionMap[id])
	}
	return sections
}

// GetDetail returns the full showcase detail including images and sections.
// callerID=0 means unauthenticated public request.
func (r *ShowcaseRepository) GetDetail(showcaseID int64) (*domain.ShowcaseDetail, error) {
	var s domain.ShowcaseDetail
	err := r.db.Get(&s, `
		SELECT
			fs.showcase_id, fs.factory_id, fs.content_type,
			fs.title, NULL::text AS excerpt, NULLIF(fs.linked_showcases->>0, '') AS image_url,
			fs.category_id, fs.sub_category_id,
			fs.moq, fs.unit_id, fs.lead_time_days,
			fs.base_price, fs.promo_price, fs.start_date, fs.end_date,
			fs.content, fs.linked_showcases, '[]'::jsonb AS tags,
			fs.likes_count, 0::bigint AS view_count, fs.status, fs.created_at,
			fs.updated_at, fs.published_at,
			fp.factory_name,
			fp.description       AS factory_specialization,
			fp.image_url         AS factory_image_url,
			fp.rating::float8    AS factory_rating,
			(fp.approval_status = 'AP') AS factory_verified,
			fp.review_count      AS factory_review_count,
			p.name_th            AS province_name,
			c.name               AS category_name,
			sc.name              AS sub_category_name
		FROM factory_showcases fs
		INNER JOIN factory_profiles fp       ON fs.factory_id      = fp.user_id
		LEFT JOIN lbi_categories c              ON fs.category_id     = c.category_id
		LEFT JOIN  lbi_sub_categories sc     ON fs.sub_category_id = sc.sub_category_id
		LEFT JOIN  lbi_provinces p           ON fp.province_id     = p.row_id
		WHERE fs.showcase_id = $1
	`, showcaseID)
	if err != nil {
		return nil, err
	}

	s.Images = domain.JSONStringArray{}
	s.Sections = []domain.ShowcaseSection{}
	s.Specs = []domain.ShowcaseSpec{}

	if len(s.LinkedShowcases) > 0 {
		for _, ref := range s.LinkedShowcases {
			trimmed := strings.TrimSpace(ref)
			lower := domainutil.NormalizeLower(ref)
			if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
				s.Images = append(s.Images, trimmed)
			}
		}
	}

	return &s, nil
}

func (r *ShowcaseRepository) GetByID(showcaseID, factoryID int64) (*domain.FactoryShowcase, error) {
	var s domain.FactoryShowcase
	err := r.db.Get(&s, `
		SELECT
			showcase_id,
			factory_id,
			content_type,
			title,
			NULL::text AS excerpt,
			NULLIF(linked_showcases->>0, '') AS image_url,
			category_id,
			sub_category_id,
			moq,
			unit_id,
			lead_time_days,
			base_price,
			promo_price,
			start_date,
			end_date,
			content,
			linked_showcases,
			'[]'::jsonb AS tags,
			COALESCE(likes_count, 0) AS likes_count,
			0::bigint AS view_count,
			status,
			created_at,
			updated_at,
			published_at
		FROM factory_showcases
		WHERE showcase_id = $1 AND factory_id = $2
	`, showcaseID, factoryID)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ShowcaseRepository) GetAnalytics(showcaseID, factoryID int64) (*domain.ShowcaseAnalytics, error) {
	var item domain.ShowcaseAnalytics
	err := r.db.Get(&item, `
		SELECT
			showcase_id,
			factory_id,
			title,
			content_type,
			likes_count,
			0::bigint AS view_count,
			0::float8 AS engagement_score
		FROM factory_showcases
		WHERE showcase_id = $1 AND factory_id = $2
	`, showcaseID, factoryID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

const showcasePaginatedBaseSQL = `
	SELECT
		fs.showcase_id,
		fs.factory_id,
		fs.content_type,
		fs.title,
		NULL::text AS excerpt,
		NULL::text AS image_url,
		fs.category_id,
		fs.sub_category_id,
		fs.moq, fs.unit_id,
		fs.base_price,
		fs.promo_price,
		fs.lead_time_days,
		fs.start_date,
		fs.end_date,
		COALESCE(fs.linked_showcases, '[]'::jsonb) AS linked_showcases,
		'[]'::jsonb AS tags,
		fs.likes_count,
		0::bigint AS view_count,
		fs.status,
		fs.published_at AS created_at,
		fs.published_at AS updated_at,
		fs.published_at,
		fp.factory_name,
		fp.image_url AS factory_image_url,
		fp.rating::float8 AS factory_rating,
		(fp.approval_status = 'AP') AS factory_verified,
		p.name_th AS province_name,
		c.name AS category_name,
		sc.name AS sub_category_name
	FROM factory_showcases fs
	INNER JOIN factory_profiles fp ON fs.factory_id = fp.user_id
	LEFT JOIN lbi_provinces p ON fp.province_id = p.row_id
	LEFT JOIN lbi_categories c ON fs.category_id = c.category_id
	LEFT JOIN lbi_sub_categories sc ON fs.sub_category_id = sc.sub_category_id
`

func (r *ShowcaseRepository) ListPaginated(filter domain.ShowcasePaginatedFilter) ([]domain.ShowcaseExploreItem, int64, error) {
	clauses := []string{"fs.status = 'AC'", "fp.approval_status = 'AP'"}
	args := []interface{}{}
	argPos := 1

	if len(filter.Types) > 0 {
		clauses = append(clauses, fmt.Sprintf("fs.content_type = ANY($%d)", argPos))
		args = append(args, pq.Array(filter.Types))
		argPos++
		clauses = append(clauses, "(fs.content_type != 'PM' OR fs.end_date IS NULL OR fs.end_date >= CURRENT_DATE)")
	}
	if filter.Keyword != "" {
		clauses = append(clauses, fmt.Sprintf("fs.title ILIKE $%d", argPos))
		args = append(args, "%"+filter.Keyword+"%")
		argPos++
	}
	if filter.CategoryID != nil {
		clauses = append(clauses, fmt.Sprintf("fs.category_id = $%d", argPos))
		args = append(args, *filter.CategoryID)
		argPos++
	}
	if filter.SubCategoryID != nil {
		clauses = append(clauses, fmt.Sprintf("fs.sub_category_id = $%d", argPos))
		args = append(args, *filter.SubCategoryID)
		argPos++
	}
	if filter.HubID != nil {
		clauses = append(clauses, fmt.Sprintf(
			"fs.category_id IN (SELECT category_id FROM lbi_categories WHERE hub_id = $%d)", argPos))
		args = append(args, *filter.HubID)
		argPos++
	}

	where := " WHERE " + strings.Join(clauses, " AND ")

	var total int64
	if err := r.db.Get(&total, "SELECT COUNT(*) FROM factory_showcases fs INNER JOIN factory_profiles fp ON fs.factory_id = fp.user_id"+where, args...); err != nil {
		return nil, 0, err
	}

	orderBy := "fs.published_at DESC NULLS LAST, fs.created_at DESC"
	if filter.Sort == "popular" {
		orderBy = "fs.likes_count DESC, " + orderBy
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	dataArgs := append(args, limit, offset)
	query := showcasePaginatedBaseSQL + where +
		fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", orderBy, argPos, argPos+1)

	var items []domain.ShowcaseExploreItem
	if err := r.db.Select(&items, query, dataArgs...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetHomeShowcases returns showcases grouped by content_type, limited per type (for home page).
func (r *ShowcaseRepository) GetHomeShowcases(types []string, limitPerType int) (map[string][]domain.ShowcaseExploreItem, error) {
	result := make(map[string][]domain.ShowcaseExploreItem, len(types))
	for _, t := range types {
		result[t] = []domain.ShowcaseExploreItem{}
	}
	if len(types) == 0 || limitPerType <= 0 {
		return result, nil
	}
	var rows []domain.ShowcaseExploreItem
	err := r.db.Select(&rows, `
		SELECT
			fs.showcase_id, fs.factory_id, fs.content_type, fs.title,
			NULL::text AS excerpt,
			NULLIF(fs.linked_showcases->>0, '') AS image_url,
			fs.category_id, fs.sub_category_id, fs.moq, fs.unit_id, fs.lead_time_days,
			fs.base_price, fs.promo_price,
			fs.start_date, fs.end_date,
			COALESCE(fs.linked_showcases, '[]'::jsonb) AS linked_showcases,
			'[]'::jsonb AS tags,
			fs.likes_count, 0::bigint AS view_count,
			fs.status, fs.created_at, fs.updated_at, fs.published_at,
			fp.factory_name,
			fp.image_url AS factory_image_url,
			fp.rating::float8 AS factory_rating,
			(fp.approval_status = 'AP') AS factory_verified,
			p.name_th AS province_name,
			cat.name AS category_name,
			sub.name AS sub_category_name
		FROM (
			SELECT *, ROW_NUMBER() OVER (PARTITION BY content_type ORDER BY published_at DESC NULLS LAST) AS rn
			FROM factory_showcases
			WHERE status = 'AC' AND content_type = ANY($1)
		) fs
		INNER JOIN factory_profiles fp ON fp.user_id = fs.factory_id AND fp.approval_status = 'AP'
		LEFT JOIN lbi_provinces p ON p.row_id = fp.province_id
		LEFT JOIN lbi_categories cat ON cat.category_id = fs.category_id
		LEFT JOIN lbi_sub_categories sub ON sub.sub_category_id = fs.sub_category_id
		WHERE fs.rn <= $2
		ORDER BY fs.content_type, fs.published_at DESC NULLS LAST
	`, pq.Array(types), limitPerType)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.ContentType] = append(result[row.ContentType], row)
	}
	return result, nil
}

// ListHomePromoSlides returns banner slides from PM showcases (fallback until promo_slides table exists).
func (r *ShowcaseRepository) ListHomePromoSlides(limit int) ([]domain.HomePromoSlide, error) {
	if limit <= 0 {
		limit = 5
	}
	var items []domain.HomePromoSlide
	err := r.db.Select(&items, `
		SELECT
			showcase_id AS slide_id,
			title,
			NULLIF(linked_showcases->>0, '') AS image_url,
			'/showcases/' || showcase_id AS link_to
		FROM factory_showcases
		WHERE status = 'AC'
		  AND content_type = 'PM'
		  AND (end_date IS NULL OR end_date >= CURRENT_DATE)
		ORDER BY likes_count DESC, published_at DESC NULLS LAST
		LIMIT $1
	`, limit)
	return items, err
}

func (r *ShowcaseRepository) ListPromoSlides() ([]domain.PromoSlide, error) {
	var items []domain.PromoSlide
	query := `
		SELECT
			slide_id,
			title,
			subtitle,
			code,
			image_url,
			status
		FROM promo_slides
		WHERE status = '1'
		ORDER BY slide_id DESC
	`
	err := r.db.Select(&items, query)
	return items, err
}

func (r *ShowcaseRepository) CategoryExists(categoryID int64) (bool, error) {
	var ok bool
	err := r.db.Get(&ok, `SELECT EXISTS(SELECT 1 FROM lbi_categories WHERE category_id = $1)`, categoryID)
	return ok, err
}

func (r *ShowcaseRepository) SubCategoryBelongsToCategory(subCategoryID, categoryID int64) (bool, error) {
	var ok bool
	err := r.db.Get(&ok, `
		SELECT EXISTS(
			SELECT 1 FROM lbi_sub_categories WHERE sub_category_id = $1 AND category_id = $2
		)
	`, subCategoryID, categoryID)
	return ok, err
}

type LinkedShowcaseCheckRow struct {
	ShowcaseID int64  `db:"showcase_id"`
	FactoryID  int64  `db:"factory_id"`
	Type       string `db:"content_type"`
	Status     string `db:"status"`
}

func (r *ShowcaseRepository) CheckLinkedShowcases(ids []int64) ([]LinkedShowcaseCheckRow, error) {
	if len(ids) == 0 {
		return []LinkedShowcaseCheckRow{}, nil
	}
	query, args, err := sqlx.In(`
		SELECT showcase_id, factory_id, content_type, status
		FROM factory_showcases
		WHERE showcase_id IN (?)
	`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var rows []LinkedShowcaseCheckRow
	err = r.db.Select(&rows, query, args...)
	return rows, err
}

func (r *ShowcaseRepository) ListLinkedShowcaseCards(ids []int64) ([]domain.LinkedShowcaseCard, error) {
	if len(ids) == 0 {
		return []domain.LinkedShowcaseCard{}, nil
	}
	query, args, err := sqlx.In(`
		SELECT showcase_id, title, COALESCE(NULLIF(linked_showcases->>0, ''), '') AS image_url, base_price
		FROM factory_showcases
		WHERE showcase_id IN (?)
	`, ids)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var rows []domain.LinkedShowcaseCard
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, err
	}
	byID := make(map[int64]domain.LinkedShowcaseCard, len(rows))
	for _, row := range rows {
		byID[row.ShowcaseID] = row
	}
	out := make([]domain.LinkedShowcaseCard, 0, len(ids))
	for _, id := range ids {
		if row, ok := byID[id]; ok {
			out = append(out, row)
		}
	}
	return out, nil
}

// GetFactoryReviewsForShowcase returns review summary + latest items for the factory that owns a showcase.
func (r *ShowcaseRepository) GetFactoryReviewsForShowcase(factoryID int64) (*domain.ShowcaseEmbeddedReviews, error) {
	type summaryRow struct {
		Average float64 `db:"average"`
		Total   int64   `db:"total"`
		Star5   int64   `db:"star5"`
		Star4   int64   `db:"star4"`
		Star3   int64   `db:"star3"`
		Star2   int64   `db:"star2"`
		Star1   int64   `db:"star1"`
	}
	var sr summaryRow
	if err := r.db.Get(&sr, `
		SELECT
			COALESCE(ROUND(AVG(rating::numeric),2),0)::float8 AS average,
			COUNT(*)::bigint AS total,
			COUNT(*) FILTER (WHERE ROUND(rating::numeric) = 5)::bigint AS star5,
			COUNT(*) FILTER (WHERE ROUND(rating::numeric) = 4)::bigint AS star4,
			COUNT(*) FILTER (WHERE ROUND(rating::numeric) = 3)::bigint AS star3,
			COUNT(*) FILTER (WHERE ROUND(rating::numeric) = 2)::bigint AS star2,
			COUNT(*) FILTER (WHERE ROUND(rating::numeric) = 1)::bigint AS star1
		FROM factory_reviews WHERE factory_id = $1 AND deleted_at IS NULL
	`, factoryID); err != nil {
		return nil, err
	}

	type itemRow struct {
		ReviewID       int64              `db:"review_id"`
		ReviewerName   sql.NullString     `db:"reviewer_name"`
		Rating         float64            `db:"rating"`
		Comment        string             `db:"comment"`
		CreatedAt      string             `db:"created_at"`
		ImageURLs      domain.StringArray `db:"image_urls"`
		FactoryReply   sql.NullString     `db:"factory_reply"`
		FactoryReplyAt sql.NullString     `db:"factory_reply_at"`
	}
	var items []itemRow
	if err := r.db.Select(&items, `
		SELECT fr.review_id, fr.rating, fr.comment, fr.image_urls,
		       TO_CHAR(fr.created_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS created_at,
		       NULLIF(TRIM(CONCAT(c.first_name,' ',c.last_name)),'') AS reviewer_name,
		       fr.factory_reply,
		       TO_CHAR(fr.factory_reply_at,'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS factory_reply_at
		FROM factory_reviews fr
		LEFT JOIN customers c ON c.user_id = fr.user_id
		WHERE fr.factory_id = $1 AND fr.deleted_at IS NULL
		ORDER BY fr.created_at DESC LIMIT 5
	`, factoryID); err != nil {
		return nil, err
	}

	result := &domain.ShowcaseEmbeddedReviews{
		Summary: domain.ShowcaseReviewSummary{
			Average:   sr.Average,
			Total:     sr.Total,
			Breakdown: map[string]int64{"5": sr.Star5, "4": sr.Star4, "3": sr.Star3, "2": sr.Star2, "1": sr.Star1},
		},
		Items: make([]domain.ShowcaseReviewItem, 0, len(items)),
	}
	for _, it := range items {
		name := "ลูกค้า"
		if it.ReviewerName.Valid && it.ReviewerName.String != "" {
			name = it.ReviewerName.String
		}
		imgURLs := it.ImageURLs
		if imgURLs == nil {
			imgURLs = domain.StringArray{}
		}
		ri := domain.ShowcaseReviewItem{
			ReviewID:     strconv.FormatInt(it.ReviewID, 10),
			ReviewerName: name,
			Rating:       it.Rating,
			Comment:      it.Comment,
			ImageURLs:    imgURLs,
			CreatedAt:    it.CreatedAt,
		}
		if it.FactoryReply.Valid {
			ri.FactoryReply = it.FactoryReply.String
		}
		if it.FactoryReplyAt.Valid {
			ri.FactoryReplyAt = it.FactoryReplyAt.String
		}
		result.Items = append(result.Items, ri)
	}
	return result, nil
}

// CreateImage adds a gallery image to a showcase (max 10 per showcase, ownership verified).
// DeleteImage removes a gallery image (ownership verified via showcase).
// GetSections returns all sections + items for a showcase (ownership verified).
// BulkReplaceSections replaces all sections + items for a showcase in a single transaction.
// GetSpecs returns all specs for a showcase (ownership verified, PD only).
// BulkReplaceSpecs replaces all specs for a showcase in a single transaction.
// PatchImage updates sort_order and/or caption of a gallery image (ownership verified).
// DeleteSection removes a single section (and its items via CASCADE) with ownership check.

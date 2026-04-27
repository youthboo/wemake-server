package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
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
		fs.excerpt,
		fs.image_url,
		fs.category_id,
		fs.sub_category_id,
		fs.moq,
		fs.base_price,
		fs.promo_price,
		fs.start_date,
		fs.end_date,
		fs.linked_showcases,
		COALESCE(fs.tags, '[]'::jsonb) AS tags,
		fs.likes_count,
		fs.view_count,
		fs.status,
		fs.created_at,
		fs.updated_at,
		fs.published_at,
		fp.factory_name,
		fp.image_url AS factory_image_url,
		fp.rating::float8 AS factory_rating,
		COALESCE(fp.is_verified, FALSE) AS factory_verified,
		c.name AS category_name,
		sc.name AS sub_category_name
	FROM factory_showcases fs
	INNER JOIN factory_profiles fp ON fs.factory_id = fp.user_id
	LEFT JOIN categories c ON fs.category_id = c.category_id
	LEFT JOIN lbi_sub_categories sc ON fs.sub_category_id = sc.sub_category_id
`

func (r *ShowcaseRepository) ListExplore(contentType string) ([]domain.ShowcaseExploreItem, error) {
	var items []domain.ShowcaseExploreItem
	var query string
	var args []interface{}
	if contentType != "" {
		query = showcaseExploreBaseSQL + ` WHERE fs.status = 'AC' AND fs.content_type = $1 ORDER BY fs.created_at DESC`
		args = append(args, contentType)
	} else {
		query = showcaseExploreBaseSQL + ` WHERE fs.status = 'AC' ORDER BY fs.created_at DESC`
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
			fs.excerpt, fs.image_url,
			fs.category_id, fs.sub_category_id,
			fs.moq, ` + basePriceExpr + `, ` + leadTimeExpr + `,
			fs.likes_count, fs.status, fs.created_at,
			c.name  AS category_name,
			` + subCategoryNameExpr + `
		FROM factory_showcases fs
		LEFT JOIN categories c         ON fs.category_id     = c.category_id
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
				Description: derefString(row.Description),
				IconName:    row.IconName,
				SortOrder:   derefInt(row.ItemSort),
			})
		}
	}
	sections := make([]domain.ShowcaseSection, 0, len(order))
	for _, id := range order {
		sections = append(sections, *sectionMap[id])
	}
	return sections
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// GetDetail returns the full showcase detail including images and sections.
// callerID=0 means unauthenticated public request.
func (r *ShowcaseRepository) GetDetail(showcaseID int64) (*domain.ShowcaseDetail, error) {
	var s domain.ShowcaseDetail
	err := r.db.Get(&s, `
		SELECT
			fs.showcase_id, fs.factory_id, fs.content_type,
			fs.title, fs.excerpt, fs.image_url,
			fs.category_id, fs.sub_category_id,
			fs.moq,
			fs.base_price, fs.promo_price, fs.start_date, fs.end_date,
			fs.content, fs.linked_showcases, COALESCE(fs.tags, '[]'::jsonb) AS tags,
			fs.likes_count, fs.view_count, fs.status, fs.created_at,
			fs.updated_at, fs.published_at,
			fp.factory_name,
			fp.image_url         AS factory_image_url,
			fp.rating::float8    AS factory_rating,
			COALESCE(fp.is_verified, FALSE) AS factory_verified,
			fp.specialization    AS factory_specialization,
			fp.review_count      AS factory_review_count,
			p.name_th            AS province_name,
			c.name               AS category_name,
			sc.name              AS sub_category_name
		FROM factory_showcases fs
		INNER JOIN factory_profiles fp       ON fs.factory_id      = fp.user_id
		LEFT JOIN  categories c              ON fs.category_id     = c.category_id
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
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(ref)), "http://") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(ref)), "https://") {
				s.Images = append(s.Images, strings.TrimSpace(ref))
			}
		}
	}

	return &s, nil
}

func (r *ShowcaseRepository) Create(showcase *domain.FactoryShowcase) error {
	query := `
		INSERT INTO factory_showcases
			(factory_id, content_type, title, excerpt, image_url,
			 category_id, sub_category_id, moq,
			 base_price, promo_price, start_date, end_date,
			 content, linked_showcases, tags, status,
			 published_at, updated_at)
		VALUES
			(:factory_id, :content_type, :title, :excerpt, :image_url,
			 :category_id, :sub_category_id, :moq,
			 :base_price, :promo_price, :start_date, :end_date,
			 :content, :linked_showcases, :tags,
			 COALESCE(NULLIF(:status, ''), 'DR'),
			 CASE WHEN COALESCE(NULLIF(:status, ''), 'DR') = 'AC' THEN NOW() ELSE NULL END,
			 NOW())
		RETURNING showcase_id, created_at, updated_at, published_at, likes_count, status
	`
	rows, err := r.db.NamedQuery(query, showcase)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&showcase.ShowcaseID, &showcase.CreatedAt, &showcase.UpdatedAt, &showcase.PublishedAt, &showcase.LikesCount, &showcase.Status)
	}
	rows.Close()
	return err
}

func (r *ShowcaseRepository) GetByID(showcaseID, factoryID int64) (*domain.FactoryShowcase, error) {
	var s domain.FactoryShowcase
	err := r.db.Get(&s, `SELECT * FROM factory_showcases WHERE showcase_id = $1 AND factory_id = $2`, showcaseID, factoryID)
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
			view_count,
			CASE
				WHEN view_count > 0 THEN ROUND((likes_count::numeric / view_count::numeric) * 100, 2)::float8
				ELSE 0
			END AS engagement_score
		FROM factory_showcases
		WHERE showcase_id = $1 AND factory_id = $2
	`, showcaseID, factoryID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ShowcaseRepository) ListPromoSlides() ([]domain.PromoSlide, error) {
	var items []domain.PromoSlide
	query := `SELECT * FROM promo_slides WHERE status = '1' ORDER BY slide_id DESC`
	err := r.db.Select(&items, query)
	return items, err
}

func (r *ShowcaseRepository) Update(s *domain.FactoryShowcase) error {
	query := `
		UPDATE factory_showcases
		SET content_type    = :content_type,
		    title           = :title,
		    excerpt         = :excerpt,
		    image_url       = :image_url,
		    category_id     = :category_id,
		    sub_category_id = :sub_category_id,
		    moq             = :moq,
		    base_price      = :base_price,
		    promo_price     = :promo_price,
		    start_date      = :start_date,
		    end_date        = :end_date,
		    content         = :content,
		    linked_showcases = :linked_showcases,
		    tags            = :tags,
		    status          = CASE WHEN :status = '' THEN status ELSE :status END,
		    published_at    = CASE
		                        WHEN :status = 'AC' AND published_at IS NULL THEN NOW()
		                        ELSE published_at
		                      END,
		    updated_at      = NOW()
		WHERE showcase_id = :showcase_id AND factory_id = :factory_id
	`
	res, err := r.db.NamedExec(query, s)
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

func (r *ShowcaseRepository) UpdateStatus(showcaseID, factoryID int64, status string) error {
	res, err := r.db.Exec(`
		UPDATE factory_showcases
		SET status = $1,
		    published_at = CASE WHEN $1 = 'AC' AND published_at IS NULL THEN NOW() ELSE published_at END,
		    updated_at = NOW()
		WHERE showcase_id = $2 AND factory_id = $3
	`, status, showcaseID, factoryID)
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

func (r *ShowcaseRepository) CategoryExists(categoryID int64) (bool, error) {
	var ok bool
	err := r.db.Get(&ok, `SELECT EXISTS(SELECT 1 FROM categories WHERE category_id = $1)`, categoryID)
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
		SELECT showcase_id, title, COALESCE(image_url, '') AS image_url, base_price
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

func (r *ShowcaseRepository) IncrementViewCount(showcaseID int64) error {
	_, err := r.db.Exec(`UPDATE factory_showcases SET view_count = view_count + 1 WHERE showcase_id = $1`, showcaseID)
	return err
}

func (r *ShowcaseRepository) Delete(showcaseID, factoryID int64) error {
	res, err := r.db.Exec(`DELETE FROM factory_showcases WHERE showcase_id = $1 AND factory_id = $2`, showcaseID, factoryID)
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

// CreateImage adds a gallery image to a showcase (max 10 per showcase, ownership verified).
func (r *ShowcaseRepository) CreateImage(img *domain.ShowcaseImage, factoryID int64) error {
	// Verify ownership
	var ownerID int64
	if err := r.db.Get(&ownerID, `SELECT factory_id FROM factory_showcases WHERE showcase_id = $1`, img.ShowcaseID); err != nil {
		return sql.ErrNoRows
	}
	if ownerID != factoryID {
		return domain.ErrForbidden
	}

	// Enforce max 10 images per showcase
	var count int
	if err := r.db.Get(&count, `SELECT COUNT(*) FROM showcase_images WHERE showcase_id = $1`, img.ShowcaseID); err != nil {
		return err
	}
	if count >= 10 {
		return domain.ErrImageLimitExceeded
	}

	return r.db.Get(img, `
		INSERT INTO showcase_images (showcase_id, image_url, sort_order, caption)
		VALUES ($1, $2, $3, $4)
		RETURNING image_id, showcase_id, image_url, sort_order, caption
	`, img.ShowcaseID, img.ImageURL, img.SortOrder, img.Caption)
}

// DeleteImage removes a gallery image (ownership verified via showcase).
func (r *ShowcaseRepository) DeleteImage(showcaseID, imageID, factoryID int64) error {
	res, err := r.db.Exec(`
		DELETE FROM showcase_images
		WHERE image_id = $1
		  AND showcase_id = $2
		  AND EXISTS (SELECT 1 FROM factory_showcases WHERE showcase_id = $2 AND factory_id = $3)
	`, imageID, showcaseID, factoryID)
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

// GetSections returns all sections + items for a showcase (ownership verified).
func (r *ShowcaseRepository) GetSections(showcaseID, factoryID int64) ([]domain.ShowcaseSection, error) {
	// Verify ownership
	var ownerID int64
	if err := r.db.Get(&ownerID, `SELECT factory_id FROM factory_showcases WHERE showcase_id = $1`, showcaseID); err != nil {
		return nil, sql.ErrNoRows
	}
	if ownerID != factoryID {
		return nil, domain.ErrForbidden
	}

	var rows []sectionRow
	if err := r.db.Select(&rows, `
		SELECT
			s.section_id, s.section_type, s.section_title, s.sort_order,
			i.item_id,
			i.title       AS item_title,
			i.description AS item_description,
			i.icon_name,
			i.sort_order  AS item_sort_order
		FROM showcase_sections s
		LEFT JOIN showcase_section_items i ON s.section_id = i.section_id
		WHERE s.showcase_id = $1
		ORDER BY s.sort_order, s.section_id, i.sort_order, i.item_id
	`, showcaseID); err != nil {
		return nil, err
	}
	return aggregateSections(rows), nil
}

// BulkReplaceSections replaces all sections + items for a showcase in a single transaction.
func (r *ShowcaseRepository) BulkReplaceSections(showcaseID, factoryID int64, inputs []domain.ShowcaseSectionInput) error {
	// Verify ownership
	var ownerID int64
	if err := r.db.Get(&ownerID, `SELECT factory_id FROM factory_showcases WHERE showcase_id = $1`, showcaseID); err != nil {
		return sql.ErrNoRows
	}
	if ownerID != factoryID {
		return domain.ErrForbidden
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM showcase_sections WHERE showcase_id = $1`, showcaseID); err != nil {
		return err
	}

	for _, sec := range inputs {
		var sectionID int64
		if err := tx.QueryRow(
			`INSERT INTO showcase_sections (showcase_id, section_type, section_title, sort_order)
			 VALUES ($1, $2, $3, $4) RETURNING section_id`,
			showcaseID, sec.SectionType, sec.SectionTitle, sec.SortOrder,
		).Scan(&sectionID); err != nil {
			return err
		}
		for _, item := range sec.Items {
			if _, err := tx.Exec(
				`INSERT INTO showcase_section_items (section_id, title, description, icon_name, sort_order)
				 VALUES ($1, $2, $3, $4, $5)`,
				sectionID, item.Title, item.Description, item.IconName, item.SortOrder,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// GetSpecs returns all specs for a showcase (ownership verified, PD only).
func (r *ShowcaseRepository) GetSpecs(showcaseID, factoryID int64) ([]domain.ShowcaseSpec, error) {
	var ownerID int64
	if err := r.db.Get(&ownerID, `SELECT factory_id FROM factory_showcases WHERE showcase_id = $1`, showcaseID); err != nil {
		return nil, sql.ErrNoRows
	}
	if ownerID != factoryID {
		return nil, domain.ErrForbidden
	}
	var specs []domain.ShowcaseSpec
	if err := r.db.Select(&specs, `
		SELECT spec_id, showcase_id, spec_key, spec_value, sort_order
		FROM showcase_specs
		WHERE showcase_id = $1
		ORDER BY sort_order, spec_id
	`, showcaseID); err != nil {
		return nil, err
	}
	if specs == nil {
		specs = []domain.ShowcaseSpec{}
	}
	return specs, nil
}

// BulkReplaceSpecs replaces all specs for a showcase in a single transaction.
func (r *ShowcaseRepository) BulkReplaceSpecs(showcaseID, factoryID int64, inputs []domain.ShowcaseSpecInput) error {
	var ownerID int64
	if err := r.db.Get(&ownerID, `SELECT factory_id FROM factory_showcases WHERE showcase_id = $1`, showcaseID); err != nil {
		return sql.ErrNoRows
	}
	if ownerID != factoryID {
		return domain.ErrForbidden
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM showcase_specs WHERE showcase_id = $1`, showcaseID); err != nil {
		return err
	}
	for _, spec := range inputs {
		if _, err := tx.Exec(
			`INSERT INTO showcase_specs (showcase_id, spec_key, spec_value, sort_order)
			 VALUES ($1, $2, $3, $4)`,
			showcaseID, spec.SpecKey, spec.SpecValue, spec.SortOrder,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// PatchImage updates sort_order and/or caption of a gallery image (ownership verified).
func (r *ShowcaseRepository) PatchImage(showcaseID, imageID, factoryID int64, sortOrder *int, caption *string) (*domain.ShowcaseImage, error) {
	res, err := r.db.Exec(`
		UPDATE showcase_images
		SET sort_order = COALESCE($1, sort_order),
		    caption    = COALESCE($2, caption)
		WHERE image_id = $3
		  AND showcase_id = $4
		  AND EXISTS (SELECT 1 FROM factory_showcases WHERE showcase_id = $4 AND factory_id = $5)
	`, sortOrder, caption, imageID, showcaseID, factoryID)
	if err != nil {
		return nil, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, sql.ErrNoRows
	}
	var img domain.ShowcaseImage
	err = r.db.Get(&img, `
		SELECT image_id, showcase_id, image_url, sort_order, caption
		FROM showcase_images WHERE image_id = $1
	`, imageID)
	return &img, err
}

// DeleteSection removes a single section (and its items via CASCADE) with ownership check.
func (r *ShowcaseRepository) DeleteSection(showcaseID, sectionID, factoryID int64) error {
	res, err := r.db.Exec(`
		DELETE FROM showcase_sections
		WHERE section_id = $1
		  AND showcase_id = $2
		  AND EXISTS (SELECT 1 FROM factory_showcases WHERE showcase_id = $2 AND factory_id = $3)
	`, sectionID, showcaseID, factoryID)
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

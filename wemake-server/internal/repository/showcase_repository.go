package repository

import (
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
		fs.min_order,
		fs.lead_time_days,
		fs.likes_count,
		fs.created_at,
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
		query = showcaseExploreBaseSQL + ` WHERE fs.content_type = $1 ORDER BY fs.created_at DESC`
		args = append(args, contentType)
	} else {
		query = showcaseExploreBaseSQL + ` ORDER BY fs.created_at DESC`
	}
	err := r.db.Select(&items, query, args...)
	return items, err
}

func (r *ShowcaseRepository) ListExploreByFactory(factoryID int64, contentType string) ([]domain.ShowcaseExploreItem, error) {
	var items []domain.ShowcaseExploreItem
	clauses := []string{"fs.factory_id = $1"}
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

func (r *ShowcaseRepository) Create(showcase *domain.FactoryShowcase) error {
	query := `
		INSERT INTO factory_showcases (factory_id, content_type, title, excerpt, image_url, category_id, sub_category_id, min_order, lead_time_days)
		VALUES (:factory_id, :content_type, :title, :excerpt, :image_url, :category_id, :sub_category_id, :min_order, :lead_time_days)
		RETURNING showcase_id, created_at, likes_count
	`
	rows, err := r.db.NamedQuery(query, showcase)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&showcase.ShowcaseID, &showcase.CreatedAt, &showcase.LikesCount)
	}
	rows.Close()
	return err
}

func (r *ShowcaseRepository) ListPromoSlides() ([]domain.PromoSlide, error) {
	var items []domain.PromoSlide
	query := `SELECT * FROM promo_slides WHERE status = '1' ORDER BY slide_id DESC`
	err := r.db.Select(&items, query)
	return items, err
}

package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type ShowcaseRepository struct {
	db *sqlx.DB
}

func NewShowcaseRepository(db *sqlx.DB) *ShowcaseRepository {
	return &ShowcaseRepository{db: db}
}

func (r *ShowcaseRepository) ListAll(contentType string) ([]domain.FactoryShowcase, error) {
	var items []domain.FactoryShowcase
	var err error
	if contentType != "" {
		query := `SELECT * FROM factory_showcases WHERE content_type = $1 ORDER BY created_at DESC`
		err = r.db.Select(&items, query, contentType)
	} else {
		query := `SELECT * FROM factory_showcases ORDER BY created_at DESC`
		err = r.db.Select(&items, query)
	}
	return items, err
}

func (r *ShowcaseRepository) Create(showcase *domain.FactoryShowcase) error {
	query := `
		INSERT INTO factory_showcases (factory_id, content_type, title, excerpt, image_url, category_id, min_order, lead_time_days)
		VALUES (:factory_id, :content_type, :title, :excerpt, :image_url, :category_id, :min_order, :lead_time_days)
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

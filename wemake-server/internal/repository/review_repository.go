package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type ReviewRepository struct {
	db *sqlx.DB
}

func NewReviewRepository(db *sqlx.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) ListByFactoryID(factoryID int64) ([]domain.FactoryReview, error) {
	var items []domain.FactoryReview
	query := `SELECT * FROM factory_reviews WHERE factory_id = $1 ORDER BY created_at DESC`
	err := r.db.Select(&items, query, factoryID)
	return items, err
}

func (r *ReviewRepository) Create(review *domain.FactoryReview) error {
	query := `
		INSERT INTO factory_reviews (factory_id, user_id, rating, comment)
		VALUES (:factory_id, :user_id, :rating, :comment)
		RETURNING review_id, created_at
	`
	rows, err := r.db.NamedQuery(query, review)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&review.ReviewID, &review.CreatedAt)
	}
	rows.Close()
	return err
}

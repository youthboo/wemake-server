package repository

import (
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/yourusername/wemake/internal/domain"
)

type ReviewRepository struct {
	db *sqlx.DB
}

var ErrReviewAlreadyExists = errors.New("review already exists for this order")

func NewReviewRepository(db *sqlx.DB) *ReviewRepository {
	return &ReviewRepository{db: db}
}

func (r *ReviewRepository) DB() *sqlx.DB {
	return r.db
}

func (r *ReviewRepository) ListByFactoryID(factoryID int64) ([]domain.FactoryReview, error) {
	var items []domain.FactoryReview
	query := `
		SELECT fr.review_id, fr.factory_id, fr.user_id, fr.order_id, fr.rating, fr.comment,
		       fr.created_at, fr.updated_at, fr.factory_reply, fr.factory_reply_at, fr.factory_reply_by,
		       NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), '') AS reviewer_name
		FROM factory_reviews fr
		LEFT JOIN customers c ON c.user_id = fr.user_id
		WHERE fr.factory_id = $1
		  AND fr.deleted_at IS NULL
		ORDER BY fr.created_at DESC
	`
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

func (r *ReviewRepository) GetByOrderAndUser(orderID, userID int64) (*domain.FactoryReview, error) {
	var item domain.FactoryReview
	err := r.db.Get(&item, `
		SELECT fr.review_id, fr.factory_id, fr.user_id, fr.order_id, fr.rating, fr.comment,
		       fr.created_at, fr.updated_at, fr.factory_reply, fr.factory_reply_at, fr.factory_reply_by,
		       NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), '') AS reviewer_name
		FROM factory_reviews fr
		LEFT JOIN customers c ON c.user_id = fr.user_id
		WHERE fr.order_id = $1 AND fr.user_id = $2 AND fr.deleted_at IS NULL
		LIMIT 1
	`, orderID, userID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ReviewRepository) CreateForOrderTx(tx *sqlx.Tx, review *domain.FactoryReview) error {
	err := tx.QueryRow(`
		INSERT INTO factory_reviews (factory_id, user_id, order_id, rating, comment, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING review_id, created_at, updated_at
	`, review.FactoryID, review.UserID, review.OrderID, review.Rating, strings.TrimSpace(review.Comment)).
		Scan(&review.ReviewID, &review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrReviewAlreadyExists
		}
		return err
	}
	return nil
}

func (r *ReviewRepository) SyncFactoryAggregateTx(tx *sqlx.Tx, factoryID int64) error {
	_, err := tx.Exec(`
		UPDATE factory_profiles fp
		SET rating = agg.avg_rating,
		    review_count = agg.review_count
		FROM (
			SELECT
				$1::bigint AS factory_id,
				CASE WHEN COUNT(*) = 0 THEN NULL ELSE ROUND(AVG(rating::numeric), 2) END AS avg_rating,
				COUNT(*)::int AS review_count
			FROM factory_reviews
			WHERE factory_id = $1 AND deleted_at IS NULL
		) agg
		WHERE fp.user_id = agg.factory_id
	`, factoryID)
	return err
}

func (r *ReviewRepository) GetSummaryByFactoryID(factoryID int64) (*domain.FactoryReviewSummary, error) {
	type summaryRow struct {
		AverageRating float64 `db:"average_rating"`
		ReviewCount   int64   `db:"review_count"`
		Star5         int64   `db:"star5"`
		Star4         int64   `db:"star4"`
		Star3         int64   `db:"star3"`
		Star2         int64   `db:"star2"`
		Star1         int64   `db:"star1"`
	}
	var row summaryRow
	if err := r.db.Get(&row, `
		SELECT
			COALESCE(ROUND(AVG(rating::numeric), 2), 0)::float8 AS average_rating,
			COUNT(*)::bigint AS review_count,
			COUNT(*) FILTER (WHERE rating = 5)::bigint AS star5,
			COUNT(*) FILTER (WHERE rating = 4)::bigint AS star4,
			COUNT(*) FILTER (WHERE rating = 3)::bigint AS star3,
			COUNT(*) FILTER (WHERE rating = 2)::bigint AS star2,
			COUNT(*) FILTER (WHERE rating = 1)::bigint AS star1
		FROM factory_reviews
		WHERE factory_id = $1 AND deleted_at IS NULL
	`, factoryID); err != nil {
		return nil, err
	}
	return &domain.FactoryReviewSummary{
		FactoryID:     factoryID,
		AverageRating: row.AverageRating,
		ReviewCount:   row.ReviewCount,
		RatingBreakdown: map[string]int64{
			"5": row.Star5,
			"4": row.Star4,
			"3": row.Star3,
			"2": row.Star2,
			"1": row.Star1,
		},
	}, nil
}

func (r *ReviewRepository) UpdateByUser(reviewID, userID int64, rating int, comment string) (*domain.FactoryReview, error) {
	var item domain.FactoryReview
	err := r.db.Get(&item, `
		UPDATE factory_reviews
		SET rating = $1, comment = $2, updated_at = NOW()
		WHERE review_id = $3 AND user_id = $4
		  AND created_at > NOW() - INTERVAL '7 days'
		  AND deleted_at IS NULL
		RETURNING review_id, factory_id, user_id, order_id, rating, comment, created_at, updated_at, factory_reply, factory_reply_at, factory_reply_by
	`, rating, strings.TrimSpace(comment), reviewID, userID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ReviewRepository) SoftDeleteByUser(reviewID, userID int64) (*domain.FactoryReview, error) {
	var item domain.FactoryReview
	err := r.db.Get(&item, `
		UPDATE factory_reviews
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE review_id = $1 AND user_id = $2
		  AND created_at > NOW() - INTERVAL '7 days'
		  AND deleted_at IS NULL
		RETURNING review_id, factory_id, user_id, order_id, rating, comment, created_at, updated_at, factory_reply, factory_reply_at, factory_reply_by
	`, reviewID, userID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

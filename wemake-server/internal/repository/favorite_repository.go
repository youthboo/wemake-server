package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type FavoriteRepository struct {
	db *sqlx.DB
}

func NewFavoriteRepository(db *sqlx.DB) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

func (r *FavoriteRepository) ListByUserID(userID int64) ([]domain.Favorite, error) {
	var items []domain.Favorite
	query := `SELECT * FROM favorites WHERE user_id = $1 ORDER BY created_at DESC`
	err := r.db.Select(&items, query, userID)
	return items, err
}

func (r *FavoriteRepository) Add(fav *domain.Favorite) error {
	query := `
		INSERT INTO favorites (user_id, showcase_id)
		VALUES (:user_id, :showcase_id)
		RETURNING fav_id, created_at
	`
	rows, err := r.db.NamedQuery(query, fav)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&fav.FavID, &fav.CreatedAt)
	}
	rows.Close()
	return err
}

func (r *FavoriteRepository) Remove(userID, showcaseID int64) error {
	query := `DELETE FROM favorites WHERE user_id = $1 AND showcase_id = $2`
	_, err := r.db.Exec(query, userID, showcaseID)
	return err
}

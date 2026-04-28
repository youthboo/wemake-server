package repository

import (
	"errors"

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
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO favorites (user_id, showcase_id)
		VALUES (:user_id, :showcase_id)
		RETURNING fav_id, created_at
	`
	rows, err := tx.NamedQuery(query, fav)
	if err != nil {
		return err
	}
	if !rows.Next() {
		rows.Close()
		return errors.New("favorite insert: no row returned")
	}
	if err = rows.Scan(&fav.FavID, &fav.CreatedAt); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	if _, err = tx.Exec(`
		UPDATE factory_showcases
		SET likes_count = likes_count + 1
		WHERE showcase_id = $1
	`, fav.ShowcaseID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *FavoriteRepository) Remove(userID, showcaseID int64) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`DELETE FROM favorites WHERE user_id = $1 AND showcase_id = $2`, userID, showcaseID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		if _, err = tx.Exec(`
			UPDATE factory_showcases
			SET likes_count = GREATEST(likes_count - 1, 0)
			WHERE showcase_id = $1
		`, showcaseID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

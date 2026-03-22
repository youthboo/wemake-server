package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type CatalogRepository struct {
	db *sqlx.DB
}

func NewCatalogRepository(db *sqlx.DB) *CatalogRepository {
	return &CatalogRepository{db: db}
}

func (r *CatalogRepository) GetCategories() ([]domain.Category, error) {
	var categories []domain.Category
	query := "SELECT category_id, name FROM categories ORDER BY category_id ASC"
	err := r.db.Select(&categories, query)
	return categories, err
}

func (r *CatalogRepository) GetUnits() ([]domain.Unit, error) {
	var units []domain.Unit
	query := "SELECT unit_id, name FROM units ORDER BY unit_id ASC"
	err := r.db.Select(&units, query)
	return units, err
}

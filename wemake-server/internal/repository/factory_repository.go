package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type FactoryRepository struct {
	db *sqlx.DB
}

func NewFactoryRepository(db *sqlx.DB) *FactoryRepository {
	return &FactoryRepository{db: db}
}

func (r *FactoryRepository) CreateFactory(factory *domain.Factory) error {
	query := `
		INSERT INTO factories (id, name, email, phone, address, description, created_at, updated_at)
		VALUES (:id, :name, :email, :phone, :address, :description, :created_at, :updated_at)
	`
	_, err := r.db.NamedExec(query, factory)
	return err
}

func (r *FactoryRepository) GetFactoryByID(id string) (*domain.Factory, error) {
	var factory domain.Factory
	query := "SELECT * FROM factories WHERE id = $1"
	err := r.db.Get(&factory, query, id)
	return &factory, err
}

func (r *FactoryRepository) GetAllFactories() ([]domain.Factory, error) {
	var factories []domain.Factory
	query := "SELECT * FROM factories ORDER BY created_at DESC"
	err := r.db.Select(&factories, query)
	return factories, err
}

func (r *FactoryRepository) UpdateFactory(factory *domain.Factory) error {
	query := `
		UPDATE factories 
		SET name = :name, email = :email, phone = :phone, address = :address, 
		    description = :description, updated_at = :updated_at
		WHERE id = :id
	`
	_, err := r.db.NamedExec(query, factory)
	return err
}

func (r *FactoryRepository) DeleteFactory(id string) error {
	query := "DELETE FROM factories WHERE id = $1"
	_, err := r.db.Exec(query, id)
	return err
}

package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type EntrepreneurRepository struct {
	db *sqlx.DB
}

func NewEntrepreneurRepository(db *sqlx.DB) *EntrepreneurRepository {
	return &EntrepreneurRepository{db: db}
}

func (r *EntrepreneurRepository) CreateEntrepreneur(entrepreneur *domain.Entrepreneur) error {
	query := `
		INSERT INTO entrepreneurs (id, name, email, phone, company, created_at, updated_at)
		VALUES (:id, :name, :email, :phone, :company, :created_at, :updated_at)
	`
	_, err := r.db.NamedExec(query, entrepreneur)
	return err
}

func (r *EntrepreneurRepository) GetEntrepreneurByID(id string) (*domain.Entrepreneur, error) {
	var entrepreneur domain.Entrepreneur
	query := "SELECT * FROM entrepreneurs WHERE id = $1"
	err := r.db.Get(&entrepreneur, query, id)
	return &entrepreneur, err
}

func (r *EntrepreneurRepository) GetAllEntrepreneurs() ([]domain.Entrepreneur, error) {
	var entrepreneurs []domain.Entrepreneur
	query := "SELECT * FROM entrepreneurs ORDER BY created_at DESC"
	err := r.db.Select(&entrepreneurs, query)
	return entrepreneurs, err
}

func (r *EntrepreneurRepository) UpdateEntrepreneur(entrepreneur *domain.Entrepreneur) error {
	query := `
		UPDATE entrepreneurs 
		SET name = :name, email = :email, phone = :phone, company = :company, updated_at = :updated_at
		WHERE id = :id
	`
	_, err := r.db.NamedExec(query, entrepreneur)
	return err
}

func (r *EntrepreneurRepository) DeleteEntrepreneur(id string) error {
	query := "DELETE FROM entrepreneurs WHERE id = $1"
	_, err := r.db.Exec(query, id)
	return err
}

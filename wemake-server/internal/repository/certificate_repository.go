package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type CertificateRepository struct {
	db *sqlx.DB
}

func NewCertificateRepository(db *sqlx.DB) *CertificateRepository {
	return &CertificateRepository{db: db}
}

func (r *CertificateRepository) ListByFactoryID(factoryID int64) ([]domain.FactoryCertificate, error) {
	var items []domain.FactoryCertificate
	query := `SELECT * FROM map_factory_certificates WHERE factory_id = $1 ORDER BY uploaded_at DESC`
	err := r.db.Select(&items, query, factoryID)
	return items, err
}

func (r *CertificateRepository) Create(cert *domain.FactoryCertificate) error {
	query := `
		INSERT INTO map_factory_certificates (factory_id, cert_id, document_url, expire_date, cert_number)
		VALUES (:factory_id, :cert_id, :document_url, :expire_date, :cert_number)
		RETURNING map_id, verify_status, uploaded_at
	`
	rows, err := r.db.NamedQuery(query, cert)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&cert.MapID, &cert.VerifyStatus, &cert.UploadedAt)
	}
	rows.Close()
	return err
}

package repository

import (
	"database/sql"

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

func (r *CertificateRepository) DeleteByMapID(factoryID, mapID int64) error {
	res, err := r.db.Exec(`DELETE FROM map_factory_certificates WHERE map_id = $1 AND factory_id = $2`, mapID, factoryID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *CertificateRepository) DeleteByCertID(factoryID, certID int64) error {
	res, err := r.db.Exec(`DELETE FROM map_factory_certificates WHERE cert_id = $1 AND factory_id = $2`, certID, factoryID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
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

func (r *CertificateRepository) PatchByCertID(factoryID, certID int64, documentURL *string, expireDate *string, certNumber *string) error {
	res, err := r.db.Exec(`
		UPDATE map_factory_certificates
		SET document_url = COALESCE($1, document_url),
		    expire_date = COALESCE($2, expire_date),
		    cert_number = COALESCE($3, cert_number),
		    uploaded_at = NOW(),
		    verify_status = CASE
		        WHEN COALESCE($1, document_url) IS DISTINCT FROM document_url
		          OR COALESCE($2, expire_date::text) IS DISTINCT FROM expire_date::text
		          OR COALESCE($3, cert_number) IS DISTINCT FROM cert_number
		        THEN 'PD'
		        ELSE verify_status
		    END
		WHERE factory_id = $4 AND cert_id = $5
	`, nullableString(documentURL), nullableString(expireDate), nullableString(certNumber), factoryID, certID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func nullableString(v *string) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

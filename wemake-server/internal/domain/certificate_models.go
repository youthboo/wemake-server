package domain

import "time"

type FactoryCertificate struct {
	MapID        int64     `db:"map_id" json:"map_id"`
	FactoryID    int64     `db:"factory_id" json:"factory_id"`
	CertID       int64     `db:"cert_id" json:"cert_id"`
	DocumentURL  string    `db:"document_url" json:"document_url"`
	ExpireDate   *string   `db:"expire_date" json:"expire_date,omitempty"`
	CertNumber   *string   `db:"cert_number" json:"cert_number,omitempty"`
	VerifyStatus string    `db:"verify_status" json:"verify_status"`
	UploadedAt   time.Time `db:"uploaded_at" json:"uploaded_at"`
}

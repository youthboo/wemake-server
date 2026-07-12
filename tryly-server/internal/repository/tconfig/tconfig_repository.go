package tconfig

import (
	"github.com/jmoiron/sqlx"
)

type Item struct {
	Key   string `db:"key"   json:"key"`
	Value string `db:"value" json:"value"`
}

type TConfigRepository struct {
	db *sqlx.DB
}

func NewTConfigRepository(db *sqlx.DB) *TConfigRepository {
	return &TConfigRepository{db: db}
}

func (r *TConfigRepository) GetAll() ([]Item, error) {
	var items []Item
	err := r.db.Select(&items, `SELECT key, value FROM tconfig ORDER BY key`)
	return items, err
}

// GetValue returns the value for a key, or "" when the key is missing.
func (r *TConfigRepository) GetValue(key string) string {
	var v string
	if err := r.db.Get(&v, `SELECT value FROM tconfig WHERE key = $1`, key); err != nil {
		return ""
	}
	return v
}

// IsEscrowMode reports whether the platform runs the escrow payment flow.
// config_payment = "1" (or missing key) → direct-pay flow; anything else → escrow.
func (r *TConfigRepository) IsEscrowMode() bool {
	v := r.GetValue("config_payment")
	return v != "" && v != "1"
}

func (r *TConfigRepository) Set(key, value string) error {
	_, err := r.db.Exec(`
		INSERT INTO tconfig (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, key, value)
	return err
}

func (r *TConfigRepository) SetBulk(items map[string]string) error {
	for key, value := range items {
		if err := r.Set(key, value); err != nil {
			return err
		}
	}
	return nil
}

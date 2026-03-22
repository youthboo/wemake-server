package repository

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type AddressRepository struct {
	db *sqlx.DB
}

func NewAddressRepository(db *sqlx.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) ListByUserID(userID int64) ([]domain.Address, error) {
	var addresses []domain.Address
	query := `
		SELECT address_id, user_id, address_type, address_detail, sub_district_id, district_id, province_id, zip_code, is_default
		FROM addresses
		WHERE user_id = $1
		ORDER BY is_default DESC, address_id DESC
	`
	err := r.db.Select(&addresses, query, userID)
	return addresses, err
}

func (r *AddressRepository) Create(address *domain.Address) error {
	query := `
		INSERT INTO addresses (user_id, address_type, address_detail, sub_district_id, district_id, province_id, zip_code, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING address_id
	`
	return r.db.QueryRow(
		query,
		address.UserID,
		address.AddressType,
		address.AddressDetail,
		address.SubDistrictID,
		address.DistrictID,
		address.ProvinceID,
		address.ZipCode,
		address.IsDefault,
	).Scan(&address.AddressID)
}

func (r *AddressRepository) Patch(userID, addressID int64, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	allowed := map[string]bool{
		"address_type":    true,
		"address_detail":  true,
		"sub_district_id": true,
		"district_id":     true,
		"province_id":     true,
		"zip_code":        true,
		"is_default":      true,
	}

	setClauses := make([]string, 0, len(fields))
	args := make([]interface{}, 0, len(fields)+2)
	i := 1
	for key, value := range fields {
		if !allowed[key] {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, i))
		args = append(args, value)
		i++
	}
	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, userID, addressID)
	query := fmt.Sprintf(
		"UPDATE addresses SET %s WHERE user_id = $%d AND address_id = $%d",
		strings.Join(setClauses, ", "),
		i,
		i+1,
	)
	_, err := r.db.Exec(query, args...)
	return err
}

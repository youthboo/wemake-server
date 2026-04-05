package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type MasterRepository struct {
	db *sqlx.DB
}

func NewMasterRepository(db *sqlx.DB) *MasterRepository {
	return &MasterRepository{db: db}
}

func (r *MasterRepository) GetProvinces() ([]domain.LBIProvince, error) {
	var items []domain.LBIProvince
	query := "SELECT row_id, name_th, name_en, status FROM lbi_provinces WHERE status = '1' ORDER BY row_id"
	err := r.db.Select(&items, query)
	return items, err
}

func (r *MasterRepository) GetDistricts(provinceID *int64) ([]domain.LBIDistrict, error) {
	var items []domain.LBIDistrict
	query := "SELECT row_id, province_id, name_th, name_en, status FROM lbi_districts WHERE status = '1'"
	args := []interface{}{}
	if provinceID != nil {
		query += " AND province_id = $1"
		args = append(args, *provinceID)
	}
	query += " ORDER BY row_id"
	err := r.db.Select(&items, query, args...)
	return items, err
}

func (r *MasterRepository) GetSubDistricts(districtID *int64) ([]domain.LBISubDistrict, error) {
	var items []domain.LBISubDistrict
	query := "SELECT row_id, district_id, name_th, name_en, zip_code, status FROM lbi_sub_districts WHERE status = '1'"
	args := []interface{}{}
	if districtID != nil {
		query += " AND district_id = $1"
		args = append(args, *districtID)
	}
	query += " ORDER BY row_id"
	err := r.db.Select(&items, query, args...)
	return items, err
}

func (r *MasterRepository) GetFactoryTypes() ([]domain.LBIFactoryType, error) {
	var items []domain.LBIFactoryType
	query := "SELECT factory_type_id, type_name, status FROM lbi_factory_types WHERE status = '1' ORDER BY factory_type_id"
	err := r.db.Select(&items, query)
	return items, err
}

func (r *MasterRepository) GetProductCategories(parentID *int64) ([]domain.LBIProductCategory, error) {
	if parentID != nil {
		return []domain.LBIProductCategory{}, nil
	}
	var items []domain.LBIProductCategory
	query := `
		SELECT c.category_id,
		       NULL::bigint AS parent_category_id,
		       c.name AS category_name,
		       '1' AS status
		FROM categories c
		ORDER BY c.category_id
	`
	err := r.db.Select(&items, query)
	return items, err
}

func (r *MasterRepository) GetProductionSteps(factoryTypeID *int64) ([]domain.LBIProduction, error) {
	var items []domain.LBIProduction
	query := "SELECT step_id, factory_type_id, step_name, sequence, status FROM lbi_production WHERE status = '1'"
	args := []interface{}{}
	if factoryTypeID != nil {
		query += " AND factory_type_id = $1"
		args = append(args, *factoryTypeID)
	}
	query += " ORDER BY sequence, step_id"
	err := r.db.Select(&items, query, args...)
	return items, err
}

func (r *MasterRepository) GetUnits() ([]domain.LBIUnit, error) {
	var items []domain.LBIUnit
	query := "SELECT unit_id, unit_name_th, unit_name_en, status FROM lbi_units WHERE status = '1' ORDER BY unit_id"
	err := r.db.Select(&items, query)
	return items, err
}

func (r *MasterRepository) GetShippingMethods() ([]domain.LBIShippingMethod, error) {
	var items []domain.LBIShippingMethod
	query := "SELECT shipping_method_id, method_name, status FROM lbi_shipping_methods WHERE status = '1' ORDER BY shipping_method_id"
	err := r.db.Select(&items, query)
	return items, err
}

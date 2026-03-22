package domain

import "time"

type Transaction struct {
	TxID      string    `db:"tx_id" json:"tx_id"`
	WalletID  int64     `db:"wallet_id" json:"wallet_id"`
	OrderID   *int64    `db:"order_id" json:"order_id,omitempty"`
	Type      string    `db:"type" json:"type"`
	Amount    float64   `db:"amount" json:"amount"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type LBIProvince struct {
	RowID  int64  `db:"row_id" json:"row_id"`
	NameTH string `db:"name_th" json:"name_th"`
	NameEN string `db:"name_en" json:"name_en"`
	Status string `db:"status" json:"status"`
}

type LBIDistrict struct {
	RowID      int64  `db:"row_id" json:"row_id"`
	ProvinceID int64  `db:"province_id" json:"province_id"`
	NameTH     string `db:"name_th" json:"name_th"`
	NameEN     string `db:"name_en" json:"name_en"`
	Status     string `db:"status" json:"status"`
}

type LBISubDistrict struct {
	RowID      int64  `db:"row_id" json:"row_id"`
	DistrictID int64  `db:"district_id" json:"district_id"`
	NameTH     string `db:"name_th" json:"name_th"`
	NameEN     string `db:"name_en" json:"name_en"`
	ZipCode    string `db:"zip_code" json:"zip_code"`
	Status     string `db:"status" json:"status"`
}

type LBIFactoryType struct {
	FactoryTypeID int64  `db:"factory_type_id" json:"factory_type_id"`
	TypeName      string `db:"type_name" json:"type_name"`
	Status        string `db:"status" json:"status"`
}

type LBIProductCategory struct {
	CategoryID       int64  `db:"category_id" json:"category_id"`
	ParentCategoryID *int64 `db:"parent_category_id" json:"parent_category_id,omitempty"`
	CategoryName     string `db:"category_name" json:"category_name"`
	Status           string `db:"status" json:"status"`
}

type LBIProduction struct {
	StepID        int64  `db:"step_id" json:"step_id"`
	FactoryTypeID int64  `db:"factory_type_id" json:"factory_type_id"`
	StepName      string `db:"step_name" json:"step_name"`
	Sequence      int64  `db:"sequence" json:"sequence"`
	Status        string `db:"status" json:"status"`
}

type LBIUnit struct {
	UnitID     int64  `db:"unit_id" json:"unit_id"`
	UnitNameTH string `db:"unit_name_th" json:"unit_name_th"`
	UnitNameEN string `db:"unit_name_en" json:"unit_name_en"`
	Status     string `db:"status" json:"status"`
}

type LBIShippingMethod struct {
	ShippingMethodID int64  `db:"shipping_method_id" json:"shipping_method_id"`
	MethodName       string `db:"method_name" json:"method_name"`
	Status           string `db:"status" json:"status"`
}

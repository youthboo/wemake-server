package domain

import "time"

type Transaction struct {
	TxID       string    `db:"tx_id" json:"tx_id"`
	WalletID   int64     `db:"wallet_id" json:"wallet_id"`
	OrderID    *int64    `db:"order_id" json:"order_id,omitempty"`
	Type       string    `db:"type" json:"type"`
	Amount     float64   `db:"amount" json:"amount"`
	Status     string    `db:"status" json:"status"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
	UploadedAt time.Time `db:"uploaded_at" json:"uploaded_at"`
}

// LBIProvince — row_id is the official DOPA province code (1–77).
// GeographyID: 1=ภาคเหนือ 2=ภาคกลาง 3=ภาคตะวันออกเฉียงเหนือ
//
//	4=ภาคตะวันออก 5=ภาคตะวันตก 6=ภาคใต้
type LBIProvince struct {
	RowID       int32  `db:"row_id"       json:"row_id"`
	NameTH      string `db:"name_th"      json:"name_th"`
	NameEN      string `db:"name_en"      json:"name_en"`
	Status      string `db:"status"       json:"status"`
	GeographyID *int16 `db:"geography_id" json:"geography_id,omitempty"`
}

// LBIDistrict — row_id is the official 4-digit district code (e.g. 1001).
// province_id FK column stays BIGINT in DB → scan as int64.
type LBIDistrict struct {
	RowID      int32  `db:"row_id"      json:"row_id"`
	ProvinceID int64  `db:"province_id" json:"province_id"`
	NameTH     string `db:"name_th"     json:"name_th"`
	NameEN     string `db:"name_en"     json:"name_en"`
	Status     string `db:"status"      json:"status"`
}

// LBISubDistrict — row_id is the official 6-digit sub-district code (e.g. 100101).
// district_id FK column stays BIGINT in DB → scan as int64.
type LBISubDistrict struct {
	RowID      int32  `db:"row_id"      json:"row_id"`
	DistrictID int64  `db:"district_id" json:"district_id"`
	NameTH     string `db:"name_th"     json:"name_th"`
	NameEN     string `db:"name_en"     json:"name_en"`
	ZipCode    string `db:"zip_code"    json:"zip_code"`
	Status     string `db:"status"      json:"status"`
}

type LBIFactoryType struct {
	FactoryTypeID int64  `db:"factory_type_id" json:"factory_type_id"`
	TypeName      string `db:"type_name" json:"type_name"`
	Status        string `db:"status" json:"status"`
}

type LBIProductCategory struct {
	CategoryID       int64  `db:"category_id" json:"category_id"`
	ParentCategoryID *int64 `db:"parent_category_id" json:"parent_category_id,omitempty"`
	Name             string `db:"name" json:"name"`
	Status           string `db:"status" json:"status,omitempty"`
}

// LBIMasterCertificate is a row from lbi_certificates for GET /master/certificates.
type LBIMasterCertificate struct {
	CertID      int64   `db:"cert_id" json:"cert_id"`
	CertName    string  `db:"cert_name" json:"cert_name"`
	Description *string `db:"description" json:"description,omitempty"`
}

type LBIProduction struct {
	StepID      int64  `db:"step_id"      json:"step_id"`
	StepName    string `db:"step_name"    json:"step_name"`
	StepNameTH  string `db:"step_name_th" json:"step_name_th"`
	Description string `db:"description"  json:"description"`
	SortOrder   int64  `db:"sort_order"   json:"sort_order"`
}

type LBIUnit struct {
	UnitID    int64  `db:"unit_id"    json:"unit_id"`
	Code      string `db:"code"       json:"code"`
	NameTH    string `db:"name_th"    json:"name_th"`
	NameEN    string `db:"name_en"    json:"name_en"`
	GroupTH   string `db:"group_th"   json:"group_th"`
	GroupEN   string `db:"group_en"   json:"group_en"`
	SortOrder int    `db:"sort_order" json:"sort_order"`
}

type LBIShippingMethod struct {
	ShippingMethodID int64  `db:"shipping_method_id" json:"shipping_method_id"`
	MethodName       string `db:"method_name" json:"method_name"`
	Status           string `db:"status" json:"status"`
}

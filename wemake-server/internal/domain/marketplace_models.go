package domain

import (
	"time"

	"github.com/lib/pq"
)

type Category struct {
	CategoryID int64  `db:"category_id" json:"category_id"`
	Name       string `db:"name" json:"name"`
}

type SubCategory struct {
	SubCategoryID int64  `db:"sub_category_id" json:"sub_category_id"`
	CategoryID    int64  `db:"category_id" json:"category_id,omitempty"`
	Name          string `db:"name" json:"name"`
	SortOrder     int64  `db:"sort_order" json:"sort_order"`
	Status        string `db:"status" json:"status,omitempty"`
}

type Unit struct {
	UnitID     int64  `db:"unit_id" json:"unit_id"`
	Name       string `db:"name" json:"name"`
	UnitNameEn string `db:"unit_name_en" json:"unit_name_en"`
}

type Address struct {
	AddressID     int64  `db:"address_id" json:"address_id"`
	UserID        int64  `db:"user_id" json:"user_id"`
	AddressType   string `db:"address_type" json:"address_type"`
	AddressDetail string `db:"address_detail" json:"address_detail"`
	SubDistrictID int64  `db:"sub_district_id" json:"sub_district_id"`
	DistrictID    int64  `db:"district_id" json:"district_id"`
	ProvinceID    int64  `db:"province_id" json:"province_id"`
	ZipCode       string `db:"zip_code" json:"zip_code"`
	IsDefault     bool   `db:"is_default" json:"is_default"`
}

type Wallet struct {
	WalletID    int64   `db:"wallet_id" json:"wallet_id"`
	UserID      int64   `db:"user_id" json:"user_id"`
	GoodFund    float64 `db:"good_fund" json:"good_fund"`
	PendingFund float64 `db:"pending_fund" json:"pending_fund"`
}

type RFQ struct {
	RFQID            int64           `db:"rfq_id" json:"rfq_id"`
	UserID           int64           `db:"user_id" json:"user_id"`
	CategoryID       int64           `db:"category_id" json:"category_id"`
	SubCategoryID    *int64          `db:"sub_category_id" json:"sub_category_id,omitempty"`
	Title            string          `db:"title" json:"title"`
	Quantity         int64           `db:"quantity" json:"quantity"`
	UnitID           int64           `db:"unit_id" json:"unit_id"`
	BudgetPerPiece   float64         `db:"budget_per_piece" json:"budget_per_piece"`
	Details          string          `db:"details" json:"details"`
	AddressID        int64           `db:"address_id" json:"address_id"`
	ShippingMethodID *int64          `db:"shipping_method_id" json:"shipping_method_id,omitempty"`
	Status           string          `db:"status" json:"status"`
	DeadlineDate     *time.Time      `db:"deadline_date" json:"deadline_date,omitempty"`
	UploadedAt       *time.Time      `db:"uploaded_at" json:"uploaded_at,omitempty"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at" json:"updated_at"`
	ImageURLs        JSONStringArray `db:"image_urls" json:"image_urls"`

	MaterialGrade *string  `db:"material_grade" json:"material_grade,omitempty"`
	Tolerance     *string  `db:"tolerance" json:"tolerance,omitempty"`
	ColorFinish   *string  `db:"color_finish" json:"color_finish,omitempty"`
	DimensionSpec *JSONB   `db:"dimension_spec" json:"dimension_spec,omitempty"`
	WeightTargetG *float64 `db:"weight_target_g" json:"weight_target_g,omitempty"`
	PackagingSpec *string  `db:"packaging_spec" json:"packaging_spec,omitempty"`

	TargetUnitPrice      *float64   `db:"target_unit_price" json:"target_unit_price,omitempty"`
	TargetLeadTimeDays   *int       `db:"target_lead_time_days" json:"target_lead_time_days,omitempty"`
	RequiredDeliveryDate *time.Time `db:"required_delivery_date" json:"required_delivery_date,omitempty"`
	Incoterms            *string    `db:"incoterms" json:"incoterms,omitempty"`
	PaymentTerms         *string    `db:"payment_terms" json:"payment_terms,omitempty"`
	DeliveryAddressID    *int64     `db:"delivery_address_id" json:"delivery_address_id,omitempty"`

	CertificationsRequired pq.StringArray `db:"certifications_required" json:"certifications_required,omitempty"`
	SampleRequired         bool           `db:"sample_required" json:"sample_required"`
	SampleQty              *int           `db:"sample_qty" json:"sample_qty,omitempty"`
	InspectionType         *string        `db:"inspection_type" json:"inspection_type,omitempty"`

	TechDrawingURL  *string        `db:"tech_drawing_url" json:"tech_drawing_url,omitempty"`
	ReferenceImages pq.StringArray `db:"reference_images" json:"reference_images,omitempty"`
	SpecSheetURL    *string        `db:"spec_sheet_url" json:"spec_sheet_url,omitempty"`
}

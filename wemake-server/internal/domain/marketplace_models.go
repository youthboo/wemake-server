package domain

import (
	"math"
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
	RFQID              int64      `db:"rfq_id" json:"rfq_id"`
	UserID             int64      `db:"user_id" json:"user_id"`
	CategoryID         int64      `db:"category_id" json:"category_id"`
	SubCategoryID      *int64     `db:"sub_category_id" json:"sub_category_id,omitempty"`
	Title              string     `db:"title" json:"title"`
	Quantity           int64      `db:"quantity" json:"quantity"`
	Details            string     `db:"details" json:"details"`
	AddressID          int64      `db:"address_id" json:"address_id"`
	ShippingMethodID   *int64     `db:"shipping_method_id" json:"shipping_method_id,omitempty"`
	Status             string     `db:"status" json:"status"`
	UploadedAt         *time.Time `db:"uploaded_at" json:"uploaded_at,omitempty"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at" json:"updated_at"`
	MaterialGrade      *string    `db:"material_grade" json:"material_grade,omitempty"`
	PackagingSpec      *string    `db:"packaging_spec" json:"packaging_spec"`
	CategoryName       *string    `db:"category_name" json:"category_name"`
	SubCategoryName    *string    `db:"sub_category_name" json:"sub_category_name"`
	ShippingMethodName *string    `db:"shipping_method_name" json:"shipping_method_name"`
	AddressSummary     *string    `db:"address_summary" json:"address_summary"`

	TargetUnitPrice      *float64   `db:"target_unit_price" json:"target_unit_price,omitempty"`
	TargetLeadTimeDays   *int       `db:"target_lead_time_days" json:"target_lead_time_days,omitempty"`
	RequiredDeliveryDate *time.Time `db:"required_delivery_date" json:"required_delivery_date,omitempty"`
	DeliveryAddressID    *int64     `db:"delivery_address_id" json:"delivery_address_id,omitempty"`

	CertificationsRequired pq.StringArray `db:"certifications_required" json:"certifications_required,omitempty"`
	SampleRequired         bool           `db:"sample_required" json:"sample_required"`
	SampleQty              *int           `db:"sample_qty" json:"sample_qty,omitempty"`
	InspectionType         *string        `db:"inspection_type" json:"inspection_type,omitempty"`

	ReferenceImages   pq.StringArray `db:"reference_images" json:"reference_images,omitempty"`
	ImageURLs         pq.StringArray `db:"-" json:"image_urls"`
	Address           *Address       `db:"-" json:"address,omitempty"`
	RFQType           string         `db:"rfq_type" json:"rfq_type"`
	InitiatedBy       string         `db:"initiated_by" json:"initiated_by"`
	FactoryUserID     *int64         `db:"factory_user_id" json:"factory_user_id,omitempty"`
	SourceShowcaseID  *int64         `db:"source_showcase_id" json:"source_showcase_id,omitempty"`
	SourceConvID      *int64         `db:"source_conv_id" json:"source_conv_id,omitempty"`
	ConversationID    *int64         `db:"conversation_id" json:"conversation_id,omitempty"`
	BOQCurrency       *string        `db:"boq_currency" json:"boq_currency,omitempty"`
	BOQSubtotal       *float64       `db:"boq_subtotal" json:"boq_subtotal,omitempty"`
	BOQDiscountAmount *float64       `db:"boq_discount_amount" json:"boq_discount_amount,omitempty"`
	BOQVatPercent     *float64       `db:"boq_vat_percent" json:"boq_vat_percent,omitempty"`
	BOQVatAmount      *float64       `db:"boq_vat_amount" json:"boq_vat_amount,omitempty"`
	BOQGrandTotal     *float64       `db:"boq_grand_total" json:"boq_grand_total,omitempty"`
	BOQMOQ            *int           `db:"boq_moq" json:"boq_moq,omitempty"`
	BOQLeadTimeDays   *int           `db:"boq_lead_time_days" json:"boq_lead_time_days,omitempty"`
	BOQPaymentTerms   *string        `db:"boq_payment_terms" json:"boq_payment_terms,omitempty"`
	BOQValidityDays   *int           `db:"boq_validity_days" json:"boq_validity_days,omitempty"`
	BOQNote           *string        `db:"boq_note" json:"boq_note,omitempty"`
	BOQSentAt         *time.Time     `db:"boq_sent_at" json:"boq_sent_at,omitempty"`
	BOQRespondedAt    *time.Time     `db:"boq_responded_at" json:"boq_responded_at,omitempty"`
	BOQResponse       *string        `db:"boq_response" json:"boq_response,omitempty"`
	BOQDeclineReason  *string        `db:"boq_decline_reason" json:"boq_decline_reason,omitempty"`
	Items             []RFQItem      `db:"-" json:"items,omitempty"`

	// Budget UX: target_unit_price is treated as total budget (not per-piece).
	BudgetTotal    *float64 `db:"-" json:"budget_total"`
	BudgetPerPiece *float64 `db:"-" json:"budget_per_piece"`
	EstimatedTotal *float64 `db:"-" json:"estimated_total"`
}

// EnrichRFQBudgetFields sets budget_total, budget_per_piece, estimated_total from
// target_unit_price (total budget) and quantity. Idempotent.
func EnrichRFQBudgetFields(rfq *RFQ) {
	if rfq == nil {
		return
	}
	rfq.ImageURLs = rfq.ReferenceImages
	rfq.BudgetTotal = nil
	rfq.BudgetPerPiece = nil
	rfq.EstimatedTotal = nil
	if rfq.TargetUnitPrice == nil {
		return
	}
	total := *rfq.TargetUnitPrice
	bt := total
	et := total
	rfq.BudgetTotal = &bt
	rfq.EstimatedTotal = &et
	if rfq.Quantity <= 0 {
		return
	}
	per := math.Round((total/float64(rfq.Quantity))*100) / 100
	rfq.BudgetPerPiece = &per
}

package domain

import (
	"time"

	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

const (
	RequestKindProduction     = "PR"
	RequestKindProductSample  = "PS"
	RequestKindMaterialSample = "MS"
	RequestKindRawMaterial    = "MR"
)

type Category struct {
	CategoryID int64  `db:"category_id" json:"category_id"`
	Name       string `db:"name" json:"name"`
	Scope      string `db:"scope" json:"scope,omitempty"`
	HubID      int64  `db:"hub_id" json:"hub_id,omitempty"`
}

type Hub struct {
	HubID int64  `db:"hub_id" json:"hub_id"`
	Name  string `db:"name" json:"name"`
	Scope string `db:"scope" json:"scope"`
	Img   string `db:"img" json:"img"`
}

type CategoryForHub struct {
	CategoryID   int64    `db:"category_id" json:"category_id"`
	Name         string   `db:"name" json:"name"`
	FactoryCount int64    `db:"factory_count" json:"factory_count"`
	SubPreview   []string `json:"sub_preview"`
	Img          string   `db:"img" json:"img"`
}

type HubWithCategories struct {
	HubID      int64            `json:"hub_id"`
	Name       string           `json:"name"`
	Scope      string           `json:"scope"`
	Img        string           `json:"img"`
	Categories []CategoryForHub `json:"categories"`
}

// ExploreFactory is a lightweight factory card returned in GET /api/v1/explore.
type ExploreFactory struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Image    string  `json:"image"`
	Location string  `json:"location"`
	Rating   float64 `json:"rating"`
	Reviews  int64   `json:"reviews"`
	Verified bool    `json:"verified"`
}

type ExploreResponse struct {
	Categories []Category                       `json:"categories"`
	Showcases  map[string][]ShowcaseExploreItem `json:"showcases"`
	Factories  []ExploreFactory                 `json:"factories"`
}

type CategoryWithSubs struct {
	CategoryID    int64         `json:"category_id"`
	Name          string        `json:"name"`
	Scope         string        `json:"scope,omitempty"`
	HubID         int64         `json:"hub_id,omitempty"`
	SubCategories []SubCategory `json:"sub_categories"`
}

type SubCategory struct {
	SubCategoryID int64  `db:"sub_category_id" json:"sub_category_id"`
	CategoryID    int64  `db:"category_id" json:"category_id,omitempty"`
	Name          string `db:"name" json:"name"`
	SortOrder     int64  `db:"sort_order" json:"sort_order"`
	Status        string `db:"status" json:"status,omitempty"`
}

type Unit struct {
	UnitID int64  `db:"unit_id"  json:"unit_id"`
	Code   string `db:"code"     json:"code"`
	NameTH string `db:"name_th"  json:"name_th"`
	NameEN string `db:"name_en"  json:"name_en"`
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
	// Display names — joined from lbi_* master tables in ListByUserID so the
	// FE can render the full address string without a separate lookup.
	SubDistrictName string `db:"sub_district_name" json:"sub_district_name"`
	DistrictName    string `db:"district_name" json:"district_name"`
	ProvinceName    string `db:"province_name" json:"province_name"`
}

type Wallet struct {
	WalletID    int64           `db:"wallet_id" json:"wallet_id"`
	UserID      int64           `db:"user_id" json:"user_id"`
	GoodFund    decimal.Decimal `db:"good_fund" json:"good_fund"`
	PendingFund decimal.Decimal `db:"pending_fund" json:"pending_fund"`
}

type RFQ struct {
	RFQID              int64      `db:"rfq_id" json:"rfq_id"`
	UserID             int64      `db:"user_id" json:"user_id"`
	CategoryID         int64      `db:"category_id" json:"category_id"`
	SubCategoryID      *int64     `db:"sub_category_id" json:"sub_category_id,omitempty"`
	Title              string     `db:"title" json:"title"`
	Quantity           int64      `db:"quantity" json:"quantity"`
	UnitID             *int64     `db:"unit_id" json:"unit_id,omitempty"`
	UnitCode           *string    `db:"unit_code" json:"unit_code,omitempty"`
	UnitName           *string    `db:"unit_name" json:"unit_name,omitempty"`
	Details            string     `db:"details" json:"details"`
	AddressID          int64      `db:"address_id" json:"address_id"`
	ShippingMethodID   *int64     `db:"shipping_method_id" json:"shipping_method_id,omitempty"`
	Status             string     `db:"status" json:"status"`
	RequestKind        string     `db:"request_kind" json:"request_kind"`
	IsDismissed        bool       `db:"is_dismissed" json:"is_dismissed"`
	DismissedAt        *time.Time `db:"dismissed_at" json:"dismissed_at,omitempty"`
	CanDismiss         bool       `db:"can_dismiss" json:"can_dismiss"`
	UploadedAt         *time.Time `db:"uploaded_at" json:"uploaded_at,omitempty"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at" json:"updated_at"`
	ExpiredDate        *time.Time `db:"expired_date" json:"expired_date,omitempty"`
	MaterialGrade      *string    `db:"material_grade" json:"material_grade,omitempty"`
	PackagingSpec      *string    `db:"packaging_spec" json:"packaging_spec"`
	CategoryName       *string    `db:"category_name" json:"category_name"`
	SubCategoryName    *string    `db:"sub_category_name" json:"sub_category_name"`
	ShippingMethodName *string    `db:"shipping_method_name" json:"shipping_method_name"`
	AddressSummary     *string    `db:"address_summary" json:"address_summary"`

	TargetPrice          *decimal.Decimal `db:"target_price" json:"target_price,omitempty"`
	TargetLeadTimeDays   *int       `db:"target_lead_time_days" json:"target_lead_time_days,omitempty"`
	RequiredDeliveryDate *time.Time `db:"required_delivery_date" json:"required_delivery_date,omitempty"`
	DeliveryAddressID    *int64     `db:"delivery_address_id" json:"delivery_address_id,omitempty"`

	CertificationsRequired pq.StringArray `db:"certifications_required" json:"certifications_required,omitempty"`

	ReferenceImages   pq.StringArray `db:"reference_images" json:"reference_images,omitempty"`
	Address           *Address       `db:"-" json:"address,omitempty"`
	RFQType           string         `db:"rfq_type" json:"rfq_type"`
	InitiatedBy       string         `db:"initiated_by" json:"initiated_by"`
	FactoryUserID     *int64         `db:"factory_user_id" json:"factory_user_id,omitempty"`
	SourceShowcaseID  *int64         `db:"source_showcase_id" json:"source_showcase_id,omitempty"`
	SourceConvID      *int64         `db:"source_conv_id" json:"source_conv_id,omitempty"`
	ConversationID    *int64         `db:"conversation_id" json:"conversation_id,omitempty"`
	BOQCurrency       *string        `db:"boq_currency" json:"boq_currency,omitempty"`
	BOQSubtotal       *decimal.Decimal `db:"boq_subtotal" json:"boq_subtotal,omitempty"`
	BOQDiscountAmount *decimal.Decimal `db:"boq_discount_amount" json:"boq_discount_amount,omitempty"`
	BOQVatPercent     *decimal.Decimal `db:"boq_vat_percent" json:"boq_vat_percent,omitempty"`
	BOQVatAmount      *decimal.Decimal `db:"boq_vat_amount" json:"boq_vat_amount,omitempty"`
	BOQGrandTotal     *decimal.Decimal `db:"boq_grand_total" json:"boq_grand_total,omitempty"`
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

	// Targeting: 'all' broadcasts to every matching factory; 'specific' restricts to rfq_target_factories.
	Targeting       string             `db:"targeting"         json:"targeting"`
	TargetFactories []RFQTargetFactory `db:"-"                 json:"target_factories,omitempty"`
	// TargetFactoryIDs is a write-only field populated by the handler on create/update.
	TargetFactoryIDs []int64           `db:"-"                 json:"-"`
	// IsTargeted is true when this RFQ was sent specifically to the viewing factory (set by ListMatchingForFactory).
	IsTargeted      bool               `db:"is_targeted"       json:"is_targeted,omitempty"`

	// Budget UX: target_price is total budget (งบประมาณรวม), not per-piece price.
	BudgetTotal    *decimal.Decimal `db:"-" json:"budget_total"`
	BudgetPerPiece *decimal.Decimal `db:"-" json:"budget_per_piece"`
	EstimatedTotal *decimal.Decimal `db:"-" json:"estimated_total"`

	// Factory-board quotation overlay — populated by ListMatchingForFactory only.
	// nil  = factory has NOT submitted a quotation for this RFQ.
	// "PD" = submitted, waiting for buyer decision.
	// "RJ" = buyer rejected the quotation.
	// "AC" = buyer accepted (rows with AC are excluded from the board entirely).
	MyQuoteStatus *string  `db:"my_quote_status" json:"my_quote_status,omitempty"`
	MyQuoteID     *int64   `db:"my_quote_id"     json:"my_quote_id,omitempty"`
	MyQuotedPrice *decimal.Decimal `db:"my_quoted_price" json:"my_quoted_price,omitempty"`
}

// RFQTargetFactory is one row from rfq_target_factories joined with the factory name.
type RFQTargetFactory struct {
	FactoryID   int64  `db:"factory_id"   json:"factory_id"`
	FactoryName string `db:"factory_name" json:"factory_name"`
}

type FactoryRFQDismissal struct {
	FactoryID   int64     `db:"factory_id" json:"factory_id"`
	RFQID       int64     `db:"rfq_id" json:"rfq_id"`
	DismissedAt time.Time `db:"dismissed_at" json:"dismissed_at"`
}

// EnrichRFQBudgetFields sets budget_total, budget_per_piece, estimated_total from
// target_price (งบประมาณรวม / total budget) and quantity. Idempotent.
func EnrichRFQBudgetFields(rfq *RFQ) {
	if rfq == nil {
		return
	}
	rfq.BudgetTotal = nil
	rfq.BudgetPerPiece = nil
	rfq.EstimatedTotal = nil
	if rfq.TargetPrice == nil {
		return
	}
	total := *rfq.TargetPrice
	bt := total
	et := total
	rfq.BudgetTotal = &bt
	rfq.EstimatedTotal = &et
	if rfq.Quantity <= 0 {
		return
	}
	per := total.Div(decimal.NewFromInt(rfq.Quantity)).Round(2)
	rfq.BudgetPerPiece = &per
}

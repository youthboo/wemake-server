package domain

import (
	"encoding/json"
	"time"

	"github.com/shopspring/decimal"
)

type Quotation struct {
	QuotationID        int64      `db:"quote_id" json:"quote_id"`
	RFQID              int64      `db:"rfq_id" json:"rfq_id"`
	FactoryID          int64      `db:"factory_id" json:"factory_id"`
	FactoryName        *string    `db:"factory_name" json:"factory_name"`
	FactoryLogoURL     *string    `db:"factory_logo_url" json:"-"`
	FactoryRatingAvg   *float64   `db:"factory_rating_avg" json:"-"`
	QuoteQuantity      float64         `db:"quote_quantity" json:"-"`
	PricePerPiece      decimal.Decimal `db:"price_per_piece" json:"price_per_piece"`
	MoldCost           decimal.Decimal `db:"mold_cost" json:"mold_cost"`
	LeadTimeDays       int64      `db:"lead_time_days" json:"lead_time_days"`
	ShippingMethodID   int64      `db:"shipping_method_id" json:"shipping_method_id"`
	ShippingMethodName *string    `db:"shipping_method_name" json:"shipping_method_name"`
	FactoryHighlight   *string    `db:"factory_highlight" json:"factory_highlight,omitempty"`
	FactoryNote        *string    `db:"factory_note" json:"factory_note,omitempty"`
	Status             string     `db:"status" json:"status"`
	CreateTime         time.Time  `db:"create_time" json:"create_time"`
	LogTimestamp       time.Time  `db:"log_timestamp" json:"log_timestamp"`
	Version            int        `db:"version" json:"version"`
	IsLocked           bool       `db:"is_locked" json:"is_locked"`
	LastEditedAt       *time.Time `db:"last_edited_at" json:"last_edited_at,omitempty"`
	LastEditedBy       *int64     `db:"last_edited_by" json:"last_edited_by,omitempty"`

	Subtotal                 decimal.Decimal `db:"subtotal" json:"subtotal"`
	DiscountAmount           decimal.Decimal `db:"discount_amount" json:"discount_amount"`
	ShippingCost             decimal.Decimal `db:"shipping_cost" json:"shipping_cost"`
	ShippingMethod           *string         `db:"shipping_method" json:"shipping_method,omitempty"`
	PackagingCost            decimal.Decimal `db:"packaging_cost" json:"packaging_cost"`
	ToolingMoldCost          decimal.Decimal `db:"tooling_mold_cost" json:"tooling_mold_cost"`
	VatRate                  decimal.Decimal `db:"vat_rate" json:"vat_rate"`
	VatAmount                decimal.Decimal `db:"vat_amount" json:"vat_amount"`
	PlatformCommissionRate   decimal.Decimal `db:"platform_commission_rate" json:"platform_commission_rate"`
	PlatformCommissionAmount decimal.Decimal `db:"platform_commission_amount" json:"platform_commission_amount"`
	PlatformConfigID         *int64          `db:"platform_config_id" json:"platform_config_id,omitempty"`
	GrandTotal               decimal.Decimal `db:"grand_total" json:"grand_total"`
	FactoryNetReceivable     decimal.Decimal `db:"factory_net_receivable" json:"factory_net_receivable"`

	ProductionStartDate  *time.Time `db:"production_start_date" json:"production_start_date,omitempty"`
	DeliveryDate         *time.Time `db:"delivery_date" json:"delivery_date,omitempty"`
	Incoterms            *string    `db:"incoterms" json:"incoterms,omitempty"`
	PaymentTerms         *string    `db:"payment_terms" json:"payment_terms,omitempty"`
	ValidityDays         int        `db:"validity_days" json:"validity_days"`
	ValidUntil           *time.Time `db:"valid_until" json:"valid_until,omitempty"`
	WarrantyPeriodMonths *int       `db:"warranty_period_months" json:"warranty_period_months,omitempty"`

	RevisionNo        int             `db:"revision_no" json:"revision_no"`
	ParentQuotationID *int64          `db:"parent_quotation_id" json:"parent_quotation_id,omitempty"`
	ImageURLs         StringArray     `db:"image_urls" json:"image_urls"`
	MaterialDetail    *string         `db:"material_detail" json:"material_detail"`
	PaymentCondition  *string         `db:"payment_condition" json:"payment_condition"`
	SampleCost        decimal.Decimal `db:"sample_cost" json:"sample_cost"`
	Certifications    StringArray     `db:"certifications" json:"certifications"`
	Items             []QuotationItem `db:"-" json:"items,omitempty"`
	QuoteTotal        decimal.Decimal `db:"-" json:"quote_total"`
	Factory           *FactoryBrief   `db:"-" json:"factory,omitempty"`
	RFQStatus         string          `db:"rfq_status" json:"rfq_status,omitempty"`
	RequestKind       string          `db:"request_kind" json:"request_kind,omitempty"`
	SampleQty         *int            `db:"sample_qty" json:"sample_qty,omitempty"`
	// Factory counter-proposal (nil = accept RFQ qty)
	FactoryQty        *int            `db:"factory_qty" json:"factory_qty,omitempty"`
	FactoryUnitID     *int64          `db:"factory_unit_id" json:"factory_unit_id,omitempty"`
	FactoryUnitName   *string         `db:"factory_unit_name" json:"factory_unit_name,omitempty"`
}

type FactoryBrief struct {
	ID        int64    `json:"id"`
	Name      *string  `json:"name,omitempty"`
	LogoURL   *string  `json:"logo_url,omitempty"`
	RatingAvg *float64 `json:"rating_avg,omitempty"`
}

type QuotationHistoryEntry struct {
	HistoryID        int64     `db:"history_id" json:"history_id"`
	QuoteID          int64     `db:"quote_id" json:"quote_id"`
	EventType        string    `db:"event_type" json:"event_type"`
	VersionAfter     int       `db:"version_after" json:"version_after"`
	PricePerPiece    *decimal.Decimal `db:"price_per_piece" json:"price_per_piece,omitempty"`
	MoldCost         *decimal.Decimal `db:"mold_cost" json:"mold_cost,omitempty"`
	LeadTimeDays     *int64    `db:"lead_time_days" json:"lead_time_days,omitempty"`
	ShippingMethodID *int64    `db:"shipping_method_id" json:"shipping_method_id,omitempty"`
	Status           *string   `db:"status" json:"status,omitempty"`
	Reason           *string   `db:"reason" json:"reason,omitempty"`
	EditedBy         *int64    `db:"edited_by" json:"edited_by,omitempty"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

type OrderActivityEntry struct {
	ActivityID  int64           `db:"activity_id" json:"activity_id"`
	OrderID     int64           `db:"order_id" json:"order_id"`
	ActorUserID *int64          `db:"actor_user_id" json:"actor_user_id,omitempty"`
	EventCode   string          `db:"event_code" json:"event_code"`
	Payload     json.RawMessage `db:"payload" json:"payload,omitempty"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
}

type Order struct {
	OrderID           int64      `db:"order_id" json:"order_id"`
	QuotationID       int64      `db:"quote_id" json:"quote_id"`
	UserID            int64      `db:"user_id" json:"user_id"`
	FactoryID         int64      `db:"factory_id" json:"factory_id"`
	TotalAmount       decimal.Decimal `db:"total_amount" json:"total_amount"`
	DepositAmount     decimal.Decimal `db:"deposit_amount" json:"deposit_amount"`
	Status            string     `db:"status" json:"status"`
	EstimatedDelivery *time.Time `db:"estimated_delivery" json:"estimated_delivery,omitempty"`
	TrackingNo        *string    `db:"tracking_no" json:"tracking_no,omitempty"`
	Courier           *string    `db:"courier" json:"courier,omitempty"`
	ShippedAt         *time.Time `db:"shipped_at" json:"shipped_at,omitempty"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updated_at"`
	// Effective qty/unit agreed upon (factory_qty if set, else rfq.quantity)
	Quantity *int   `db:"quantity" json:"quantity,omitempty"`
	UnitID   *int64 `db:"unit_id" json:"unit_id,omitempty"`
}

type OrderListRFQSummary struct {
	RFQID    int64  `json:"rfq_id"`
	Title    string `json:"title"`
	Quantity int64  `json:"quantity"`
	UnitName string `json:"unit_name"`
}

type OrderListQuotationSummary struct {
	QuoteID          int64   `json:"quote_id"`
	FactoryHighlight *string `json:"factory_highlight,omitempty"`
}

type OrderListCustomerSummary struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
}

type OrderProductionSummary struct {
	CurrentStepID       *int64     `json:"current_step_id"`
	CurrentStepNameTH   *string    `json:"current_step_name_th"`
	CurrentUpdateStatus *string    `json:"current_update_status"`
	CompletedCount      int64      `json:"completed_count"`
	TotalCount          int64      `json:"total_count"`
	LastUpdatedAt       *time.Time `json:"last_updated_at,omitempty"`
	HasRejected         bool       `json:"has_rejected"`
}

type OrderListItem struct {
	OrderID             int64                     `json:"order_id"`
	QuotationID         int64                     `json:"quote_id"`
	UserID              int64                     `json:"user_id"`
	FactoryID           int64                     `json:"factory_id"`
	Status              string                    `json:"status"`
	TotalAmount         decimal.Decimal           `json:"total_amount"`
	DepositAmount       decimal.Decimal           `json:"deposit_amount"`
	EstimatedDelivery   *time.Time                `json:"estimated_delivery,omitempty"`
	CreatedAt           time.Time                 `json:"created_at"`
	UpdatedAt           time.Time                 `json:"updated_at"`
	RFQ                 OrderListRFQSummary       `json:"rfq"`
	Quotation           OrderListQuotationSummary `json:"quotation"`
	Customer            OrderListCustomerSummary  `json:"customer"`
	ProductionSummary   OrderProductionSummary    `json:"production_summary"`
	RFQID               int64                     `json:"rfq_id"`
	RFQTitle            string                    `json:"rfq_title"`
	RFQQuantity         int64                     `json:"rfq_quantity"`
	UnitName            string                    `json:"unit_name"`
	CustomerDisplayName string                    `json:"customer_display_name"`
	RequestKind         string                    `json:"request_kind"`
	OrderType           string                    `json:"order_type"`
}

type OrderFactorySummary struct {
	FactoryID int64  `json:"factory_id"`
	Name      string `json:"name"`
	Phone     string `json:"phone,omitempty"`
	Address   string `json:"address,omitempty"`
}

type OrderNextAction struct {
	Actor      string          `json:"actor"`
	Type       string          `json:"type"`
	Amount     decimal.Decimal `json:"amount"`
	Currency   string          `json:"currency"`
	DueDate    *time.Time `json:"due_date,omitempty"`
	CTAURL     string     `json:"cta_url,omitempty"`
	CTALabelTH string     `json:"cta_label_th,omitempty"`
}

type RfqImage struct {
	ImageID  string `db:"image_id" json:"image_id"`
	ImageURL string `db:"image_url" json:"image_url"`
}

type RfqNested struct {
	RfqID               int64           `json:"rfq_id"`
	Title               string          `json:"title"`
	Details             string          `json:"details"`
	Quantity            int64           `json:"quantity"`
	UnitName            string          `json:"unit_name"`
	BudgetPerPiece      decimal.Decimal `json:"budget_per_piece"`
	CategoryID          int64           `json:"category_id"`
	CategoryName        string          `json:"category_name"`
	RequestKind         string          `json:"request_kind,omitempty"`
	SubCategoryID       *int64          `json:"sub_category_id,omitempty"`
	SubCategoryName     *string         `json:"sub_category_name,omitempty"`
	ShippingMethodName  *string         `json:"shipping_method_name,omitempty"`
	MaterialGrade       *string         `json:"material_grade,omitempty"`
	Certifications      StringArray     `json:"certifications_required"`
	TargetLeadTimeDays  *int            `json:"target_lead_time_days,omitempty"`
	TargetPrice         *float64        `json:"target_price,omitempty"`
	// Delivery address (nested object for summarizeRfqAddress)
	Address    *RfqAddressNested `json:"address,omitempty"`
	DeadlineDate *time.Time      `json:"deadline_date,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	Images       []RfqImage      `json:"images"`
}

// RfqAddressNested holds the delivery address for an RFQ.
type RfqAddressNested struct {
	AddressDetail   string `json:"address_detail"`
	SubDistrictName string `json:"sub_district_name"`
	DistrictName    string `json:"district_name"`
	ProvinceName    string `json:"province_name"`
	ZipCode         string `json:"zip_code"`
}

type QuoteNested struct {
	QuoteID          int64           `json:"quote_id"`
	PricePerPiece    decimal.Decimal `json:"price_per_piece"`
	MoldCost         decimal.Decimal `json:"mold_cost"`
	ToolingMoldCost  decimal.Decimal `json:"tooling_mold_cost"`
	LeadTimeDays     int64           `json:"lead_time_days"`
	GrandTotal       decimal.Decimal `json:"grand_total"`
	Subtotal         decimal.Decimal `json:"subtotal"`
	DiscountAmount   decimal.Decimal `json:"discount_amount"`
	ShippingCost     decimal.Decimal `json:"shipping_cost"`
	PackagingCost    decimal.Decimal `json:"packaging_cost"`
	VatRate          decimal.Decimal `json:"vat_rate"`
	VatAmount        decimal.Decimal `json:"vat_amount"`
	ValidityDays     int             `json:"validity_days"`
	ValidUntil       *string         `json:"valid_until,omitempty"`
	PaymentTerms     *string         `json:"payment_terms,omitempty"`
	ImageURLs        StringArray     `json:"image_urls"`
	FactoryHighlight *string         `json:"factory_highlight,omitempty"`
	FactoryNote      *string         `json:"factory_note,omitempty"`
}

// OrderDetailResponse extends the legacy order payload with FE-ready action state.
type OrderDetailResponse struct {
	OrderID           int64                      `json:"order_id"`
	QuotationID       int64                      `json:"quote_id"`
	UserID            int64                      `json:"user_id"`
	FactoryID         int64                      `json:"factory_id"`
	TotalAmount       decimal.Decimal            `json:"total_amount"`
	DepositAmount     decimal.Decimal            `json:"deposit_amount"`
	Status            string                     `json:"status"`
	StatusLabelTH     string                     `json:"status_label_th"`
	PaymentType       *string                    `json:"payment_type,omitempty"`
	Currency          string                     `json:"currency"`
	Factory           OrderFactorySummary        `json:"factory"`
	CustomerUserID    int64                      `json:"customer_user_id"`
	CustomerName      string                     `json:"customer_name,omitempty"`
	CustomerPhone     string                     `json:"customer_phone,omitempty"`
	EstimatedDelivery *time.Time                 `json:"estimated_delivery,omitempty"`
	ShippingDays      int                        `json:"shipping_days"`
	LeadTimeDays      *int                       `json:"lead_time_days,omitempty"`
	TrackingNo        *string                    `json:"tracking_no,omitempty"`
	Courier           *string                    `json:"courier,omitempty"`
	ShippedAt         *time.Time                 `json:"shipped_at,omitempty"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
	NextAction        *OrderNextAction           `json:"next_action"`
	PaymentSchedule   []OrderPaymentScheduleItem `json:"payment_schedule"`
	RFQ               RfqNested                  `json:"rfq"`
	Quotation         QuoteNested                `json:"quotation"`
}

type ProductionUpdate struct {
	UpdateID        int64       `db:"update_id" json:"update_id"`
	OrderID         int64       `db:"order_id" json:"order_id"`
	StepID          int64       `db:"step_id" json:"step_id"`
	StepCode        string      `db:"step_code" json:"step_code"`
	StepNameTH      string      `db:"step_name_th" json:"step_name_th"`
	StepNameEN      string      `db:"step_name_en" json:"step_name_en"`
	SortOrder       int64       `db:"sort_order" json:"sort_order"`
	Status          string      `db:"status" json:"status"`
	Description     string      `db:"description" json:"description"`
	ImageURLs       StringArray `db:"image_urls" json:"image_urls"`
	CompletedAt     *time.Time  `db:"completed_at" json:"completed_at,omitempty"`
	RejectedReason  *string     `db:"rejected_reason" json:"rejected_reason,omitempty"`
	UpdatedByUserID *int64      `db:"updated_by_user_id" json:"updated_by_user_id,omitempty"`
	LastUpdatedAt   *time.Time  `db:"last_updated_at" json:"last_updated_at,omitempty"`
	CreatedAt       time.Time   `db:"created_at" json:"created_at"`
}

type Message struct {
	MessageID     string    `db:"message_id" json:"message_id"`
	ReferenceType string    `db:"reference_type" json:"reference_type"`
	ReferenceID   int64     `db:"reference_id" json:"reference_id"`
	RFQTitle        *string   `db:"rfq_title" json:"rfq_title,omitempty"`
	ReferenceTitle  *string   `db:"reference_title" json:"reference_title,omitempty"`
	SenderID      int64     `db:"sender_id" json:"sender_id"`
	ReceiverID    int64     `db:"receiver_id" json:"receiver_id"`
	Content       string    `db:"content" json:"content"`
	AttachmentURL string    `db:"attachment_url" json:"attachment_url,omitempty"`
	ConvID        *int64    `db:"conv_id" json:"conv_id,omitempty"`
	MessageType   string    `db:"message_type" json:"message_type"`
	QuoteData     *string   `db:"quote_data" json:"quote_data,omitempty"`
	BOQRfqID      *int64    `db:"boq_rfq_id" json:"boq_rfq_id,omitempty"`
	IsRead        bool      `db:"is_read" json:"is_read"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type MessageThread struct {
	ReferenceType string    `db:"reference_type" json:"reference_type"`
	ReferenceID   int64     `db:"reference_id" json:"reference_id"`
	LastMessage   string    `db:"last_message" json:"last_message"`
	LastMessageAt time.Time `db:"last_message_at" json:"last_message_at"`
}

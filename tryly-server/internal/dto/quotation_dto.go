package dto

import "github.com/yourusername/wemake/internal/domain"

// Quotation Request DTOs
type CreateQuotationRequest struct {
	FactoryID        int64              `json:"factory_id" validate:"gt=0"`
	PricePerPiece    float64            `json:"price_per_piece" validate:"gte=0"`
	MoldCost         float64            `json:"mold_cost"`
	ToolingMoldCost  float64            `json:"tooling_mold_cost"`
	ShippingCost     float64            `json:"shipping_cost"`
	PackagingCost    float64            `json:"packaging_cost"`
	LeadTimeDays     int64              `json:"lead_time_days" validate:"gt=0"`
	ValidityDays     int                `json:"validity_days"`
	ShippingMethodID int64              `json:"shipping_method_id"`
	PaymentTerms     *string            `json:"payment_terms"`
	ImageURLs        domain.StringArray `json:"image_urls"`
	FactoryHighlight *string            `json:"factory_highlight"`
	FactoryNote      *string            `json:"factory_note"`
	// Factory counter-proposal qty/unit (optional — nil = accept RFQ qty)
	FactoryQty    *int   `json:"factory_qty"`
	FactoryUnitID *int64 `json:"factory_unit_id"`
}

type PreviewQuotationRequest struct {
	Items           []domain.QuotationItem `json:"items"`
	DiscountAmount  float64                `json:"discount_amount"`
	ShippingCost    float64                `json:"shipping_cost"`
	PackagingCost   float64                `json:"packaging_cost"`
	ToolingMoldCost float64                `json:"tooling_mold_cost"`
}

type CreateDetailedQuotationRequest struct {
	RFQID                int64                  `json:"rfq_id"`
	Items                []domain.QuotationItem `json:"items"`
	DiscountAmount       float64                `json:"discount_amount"`
	ShippingCost         float64                `json:"shipping_cost"`
	ShippingMethod       *string                `json:"shipping_method"`
	ShippingMethodID     *int64                 `json:"shipping_method_id"`
	PackagingCost        float64                `json:"packaging_cost"`
	ToolingMoldCost      float64                `json:"tooling_mold_cost"`
	LeadTimeDays         *int64                 `json:"lead_time_days"`
	ProductionStartDate  *string                `json:"production_start_date"`
	DeliveryDate         *string                `json:"delivery_date"`
	Incoterms            *string                `json:"incoterms"`
	PaymentTerms         *string                `json:"payment_terms"`
	ValidityDays         int                    `json:"validity_days"`
	WarrantyPeriodMonths *int                   `json:"warranty_period_months"`
	FactoryHighlight     *string                `json:"factory_highlight"`
	FactoryNote          *string                `json:"factory_note"`
	FactoryQty           *int                   `json:"factory_qty"`
	FactoryUnitID        *int64                 `json:"factory_unit_id"`
}

type CreateRevisionQuotationRequest struct {
	Items                []domain.QuotationItem `json:"items"`
	DiscountAmount       float64                `json:"discount_amount"`
	ShippingCost         float64                `json:"shipping_cost"`
	ShippingMethod       *string                `json:"shipping_method"`
	PackagingCost        float64                `json:"packaging_cost"`
	ToolingMoldCost      float64                `json:"tooling_mold_cost"`
	LeadTimeDays         *int64                 `json:"lead_time_days"`
	ProductionStartDate  *string                `json:"production_start_date"`
	DeliveryDate         *string                `json:"delivery_date"`
	Incoterms            *string                `json:"incoterms"`
	PaymentTerms         *string                `json:"payment_terms"`
	ValidityDays         int                    `json:"validity_days"`
	WarrantyPeriodMonths *int                   `json:"warranty_period_months"`
}

type PatchQuotationRequest struct {
	Status *string `json:"status"`
	Notes  *string `json:"notes"`
}

type PatchQuotationStatusRequest struct {
	Status string `json:"status" validate:"notblank"`
}

type PatchQuotationBodyRequest struct {
	PricePerPiece    float64            `json:"price_per_piece"`
	MoldCost         float64            `json:"mold_cost"`
	ToolingMoldCost  float64            `json:"tooling_mold_cost"`
	ShippingCost     float64            `json:"shipping_cost"`
	PackagingCost    float64            `json:"packaging_cost"`
	LeadTimeDays     int64              `json:"lead_time_days"`
	ValidityDays     int                `json:"validity_days"`
	ShippingMethodID int64              `json:"shipping_method_id"`
	PaymentTerms     *string            `json:"payment_terms"`
	FactoryHighlight *string            `json:"factory_highlight"`
	FactoryNote      *string            `json:"factory_note"`
	Reason           string             `json:"reason"`
	ImageURLs        domain.StringArray `json:"image_urls"`
	FactoryQty       *int               `json:"factory_qty"`
	FactoryUnitID    *int64             `json:"factory_unit_id"`
}

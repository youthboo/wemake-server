package domain

import "time"

type PlatformConfig struct {
	ConfigID              int64      `db:"config_id" json:"config_id"`
	Label                 *string    `db:"label" json:"label,omitempty"`
	DefaultCommissionRate float64    `db:"default_commission_rate" json:"default_commission_rate"`
	PromoCommissionRate   *float64   `db:"promo_commission_rate" json:"promo_commission_rate,omitempty"`
	PromoStartAt          *time.Time `db:"promo_start_at" json:"promo_start_at,omitempty"`
	PromoEndAt            *time.Time `db:"promo_end_at" json:"promo_end_at,omitempty"`
	PromoLabel            *string    `db:"promo_label" json:"promo_label,omitempty"`
	VatRate               float64    `db:"vat_rate" json:"vat_rate"`
	CurrencyCode          string     `db:"currency_code" json:"currency_code"`
	EffectiveFrom         time.Time  `db:"effective_from" json:"effective_from"`
	EffectiveTo           *time.Time `db:"effective_to" json:"effective_to,omitempty"`
	CreatedBy             *int64     `db:"created_by" json:"created_by,omitempty"`
	CreatedAt             time.Time  `db:"created_at" json:"created_at"`
}

type UpdatePlatformConfigRequest struct {
	Label                 string  `json:"label"`
	DefaultCommissionRate float64 `json:"default_commission_rate"`
	VatRate               float64 `json:"vat_rate"`
}

type CreatePlatformConfigRequest struct {
	Label                 string   `json:"label"`
	DefaultCommissionRate float64  `json:"default_commission_rate"`
	VatRate               *float64 `json:"vat_rate"`
	CurrencyCode          string   `json:"currency_code"`
	EffectiveTo           *string  `json:"effective_to,omitempty"`
}

type AssignFactoryConfigRequest struct {
	ConfigID int64  `json:"config_id"`
	Note     string `json:"note"`
}

type FactoryConfigResponse struct {
	FactoryID             int64   `db:"factory_id" json:"factory_id"`
	ConfigID              int64   `db:"config_id" json:"config_id"`
	Label                 string  `db:"label" json:"label"`
	DefaultCommissionRate float64 `db:"default_commission_rate" json:"default_commission_rate"`
	VatRate               float64 `db:"vat_rate" json:"vat_rate"`
}

type QuotationItem struct {
	ItemID      int64     `db:"item_id" json:"item_id"`
	QuotationID int64     `db:"quotation_id" json:"quotation_id"`
	ItemNo      int       `db:"item_no" json:"item_no"`
	Description string    `db:"description" json:"description"`
	Qty         float64   `db:"qty" json:"qty"`
	Unit        *string   `db:"unit" json:"unit,omitempty"`
	UnitPrice   float64   `db:"unit_price" json:"unit_price"`
	DiscountPct float64   `db:"discount_pct" json:"discount_pct"`
	LineTotal   float64   `db:"line_total" json:"line_total"`
	Note        *string   `db:"note" json:"note,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

package domain

import "time"

type RFQItem struct {
	ItemID        int64      `db:"item_id" json:"item_id"`
	RFQID         int64      `db:"rfq_id" json:"rfq_id"`
	ItemNo        int        `db:"item_no" json:"item_no"`
	Description   string     `db:"description" json:"description"`
	Specification *string    `db:"specification" json:"specification,omitempty"`
	Qty           float64    `db:"qty" json:"qty"`
	Unit          *string    `db:"unit" json:"unit,omitempty"`
	UnitPrice     float64    `db:"unit_price" json:"unit_price"`
	DiscountPct   float64    `db:"discount_pct" json:"discount_pct"`
	LineTotal     float64    `db:"line_total" json:"line_total"`
	Note          *string    `db:"note" json:"note,omitempty"`
	CreatedAt     *time.Time `db:"created_at" json:"created_at,omitempty"`
}

type BOQFactorySummary struct {
	FactoryID   int64   `json:"factory_id"`
	FactoryName string  `json:"factory_name"`
	ImageURL    *string `json:"image_url,omitempty"`
}

type BOQBuyerSummary struct {
	UserID      int64  `json:"user_id"`
	DisplayName string `json:"display_name"`
}

type BOQDetail struct {
	RFQID             int64             `json:"rfq_id"`
	RFQType           string            `json:"rfq_type"`
	InitiatedBy       string            `json:"initiated_by"`
	Status            string            `json:"status"`
	BOQResponse       *string           `json:"boq_response,omitempty"`
	BOQDeclineReason  *string           `json:"boq_decline_reason,omitempty"`
	BOQSentAt         *time.Time        `json:"boq_sent_at,omitempty"`
	BOQRespondedAt    *time.Time        `json:"boq_responded_at,omitempty"`
	BOQValidityDays   int               `json:"boq_validity_days"`
	BOQExpiresAt      *time.Time        `json:"boq_expires_at,omitempty"`
	IsExpired         bool              `json:"is_expired"`
	BOQGrandTotal     float64           `json:"boq_grand_total"`
	BOQCurrency       string            `json:"boq_currency"`
	BOQSubtotal       float64           `json:"boq_subtotal"`
	BOQDiscountAmount float64           `json:"boq_discount_amount"`
	BOQVatPercent     float64           `json:"boq_vat_percent"`
	BOQVatAmount      float64           `json:"boq_vat_amount"`
	BOQMOQ            *int              `json:"boq_moq,omitempty"`
	BOQLeadTimeDays   *int              `json:"boq_lead_time_days,omitempty"`
	BOQPaymentTerms   *string           `json:"boq_payment_terms,omitempty"`
	BOQNote           *string           `json:"boq_note,omitempty"`
	SourceConvID      *int64            `json:"source_conv_id,omitempty"`
	SourceShowcaseID  *int64            `json:"source_showcase_id,omitempty"`
	Factory           BOQFactorySummary `json:"factory"`
	Buyer             BOQBuyerSummary   `json:"buyer"`
	Items             []RFQItem         `json:"items"`
}

type BOQSummary struct {
	RFQID            int64      `json:"rfq_id"`
	Status           string     `json:"status"`
	BOQResponse      *string    `json:"boq_response,omitempty"`
	BOQDeclineReason *string    `json:"boq_decline_reason,omitempty"`
	BOQSentAt        *time.Time `json:"boq_sent_at,omitempty"`
	BOQRespondedAt   *time.Time `json:"boq_responded_at,omitempty"`
	BOQValidityDays  int        `json:"boq_validity_days"`
	BOQExpiresAt     *time.Time `json:"boq_expires_at,omitempty"`
	IsExpired        bool       `json:"is_expired"`
	BOQGrandTotal    float64    `json:"boq_grand_total"`
	BOQCurrency      string     `json:"boq_currency"`
	SourceConvID     *int64     `json:"source_conv_id,omitempty"`
	SourceShowcaseID *int64     `json:"source_showcase_id,omitempty"`
	BuyerDisplayName string     `json:"buyer_display_name"`
	FactoryName      string     `json:"factory_name"`
}

type BOQQuoteData struct {
	BOQRFQID       int64      `json:"boq_rfq_id"`
	FactoryName    string     `json:"factory_name"`
	Items          []RFQItem  `json:"items"`
	Currency       string     `json:"currency"`
	Subtotal       float64    `json:"subtotal"`
	DiscountAmount float64    `json:"discount_amount"`
	VatPercent     float64    `json:"vat_percent"`
	VatAmount      float64    `json:"vat_amount"`
	GrandTotal     float64    `json:"grand_total"`
	MOQ            *int       `json:"moq,omitempty"`
	LeadTimeDays   *int       `json:"lead_time_days,omitempty"`
	PaymentTerms   *string    `json:"payment_terms,omitempty"`
	ValidityDays   int        `json:"validity_days"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	Note           *string    `json:"note,omitempty"`
	Status         string     `json:"status"`
	Price          float64    `json:"price"`
	LeadTime       *int       `json:"leadTime,omitempty"`
	ValidUntil     *string    `json:"validUntil,omitempty"`
}

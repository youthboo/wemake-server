package dto

import "encoding/json"

// Showcase Request DTOs
type ShowcaseWriteRequest struct {
	Type            *string         `json:"type"`
	ContentType     *string         `json:"content_type"`
	Status          *string         `json:"status"`
	Title           *string         `json:"title"`
	CategoryID      *int64          `json:"category_id"`
	SubCategoryID   *int64          `json:"sub_category_id"`
	MOQ             *int            `json:"moq"`
	UnitID          *int            `json:"unit_id"`
	LeadTimeDays    *int            `json:"lead_time_days"`
	BasePrice       *float64        `json:"base_price"`
	PromoPrice      *float64        `json:"promo_price"`
	StartDate       *string         `json:"start_date"`
	EndDate         *string         `json:"end_date"`
	Content         *string         `json:"content"`
	LinkedShowcases json.RawMessage `json:"linked_showcases"`
	Excerpt         *string         `json:"excerpt"`
	ImageURL        *string         `json:"image_url"`
	Tags            *[]string       `json:"tags"`
}

type CreateShowcaseRequest struct {
	Title            string   `json:"title" validate:"notblank"`
	Description      string   `json:"description"`
	CategoryID       int64    `json:"category_id" validate:"gt=0"`
	SubCategoryID    *int64   `json:"sub_category_id"`
	Price            float64  `json:"price" validate:"gte=0"`
	Unit             string   `json:"unit"`
	MinOrderQuantity int64    `json:"min_order_quantity"`
	MaxOrderQuantity *int64   `json:"max_order_quantity"`
	LeadTimeDays     *int     `json:"lead_time_days"`
	ImageURLs        []string `json:"image_urls"`
	Specifications   map[string]interface{} `json:"specifications"`
	Features         []string `json:"features"`
	CertificationIDs []int64  `json:"certification_ids"`
}

type PatchShowcaseRequest struct {
	Title            *string  `json:"title"`
	Description      *string  `json:"description"`
	CategoryID       *int64   `json:"category_id"`
	SubCategoryID    *int64   `json:"sub_category_id"`
	Price            *float64 `json:"price"`
	Unit             *string  `json:"unit"`
	MinOrderQuantity *int64   `json:"min_order_quantity"`
	MaxOrderQuantity *int64   `json:"max_order_quantity"`
	LeadTimeDays     *int     `json:"lead_time_days"`
	Specifications   map[string]interface{} `json:"specifications"`
	Features         []string `json:"features"`
	CertificationIDs []int64  `json:"certification_ids"`
}

type PatchShowcaseStatusRequest struct {
	Status string `json:"status" validate:"notblank"`
}

type CreateShowcaseImageRequest struct {
	ImageURL    string `json:"image_url" validate:"notblank"`
	SortOrder   *int   `json:"sort_order"`
	IsMainImage *bool  `json:"is_main_image"`
}

type CreateShowcaseInquiryRequest struct {
	Quantity int64  `json:"quantity" validate:"gt=0"`
	Message  string `json:"message"`
}

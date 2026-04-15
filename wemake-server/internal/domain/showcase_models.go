package domain

import "time"

type FactoryShowcase struct {
	ShowcaseID    int64     `db:"showcase_id" json:"showcase_id"`
	FactoryID     int64     `db:"factory_id" json:"factory_id"`
	ContentType   string    `db:"content_type" json:"content_type"`
	Title         string    `db:"title" json:"title"`
	Excerpt       *string   `db:"excerpt" json:"excerpt,omitempty"`
	Description   *string   `db:"description" json:"description,omitempty"`
	ImageURL      *string   `db:"image_url" json:"image_url,omitempty"`
	CategoryID    *int64    `db:"category_id" json:"category_id,omitempty"`
	SubCategoryID *int64    `db:"sub_category_id" json:"sub_category_id,omitempty"`
	MinOrder      *int      `db:"min_order" json:"min_order,omitempty"`
	LeadTimeDays  *int      `db:"lead_time_days" json:"lead_time_days,omitempty"`
	PriceRange    *string   `db:"price_range" json:"price_range,omitempty"`
	LikesCount    int       `db:"likes_count" json:"likes_count"`
	ViewCount     int64     `db:"view_count" json:"view_count"`
	Status        string    `db:"status" json:"status"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// ShowcaseExploreItem is the list payload for GET /showcases (Explore / FE normShowcase).
type ShowcaseExploreItem struct {
	ShowcaseID      int64     `db:"showcase_id" json:"showcase_id"`
	FactoryID       int64     `db:"factory_id" json:"factory_id"`
	ContentType     string    `db:"content_type" json:"content_type"`
	Title           string    `db:"title" json:"title"`
	Excerpt         *string   `db:"excerpt" json:"excerpt,omitempty"`
	ImageURL        *string   `db:"image_url" json:"image_url,omitempty"`
	CategoryID      *int64    `db:"category_id" json:"category_id,omitempty"`
	SubCategoryID   *int64    `db:"sub_category_id" json:"sub_category_id,omitempty"`
	MinOrder        *int      `db:"min_order" json:"min_order,omitempty"`
	LeadTimeDays    *int      `db:"lead_time_days" json:"lead_time_days,omitempty"`
	PriceRange      *string   `db:"price_range" json:"price_range,omitempty"`
	LikesCount      int       `db:"likes_count" json:"likes_count"`
	ViewCount       int64     `db:"view_count" json:"view_count"`
	Status          string    `db:"status" json:"status"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	FactoryName     string    `db:"factory_name" json:"factory_name"`
	FactoryImageURL *string   `db:"factory_image_url" json:"factory_image_url,omitempty"`
	FactoryRating   *float64  `db:"factory_rating" json:"factory_rating,omitempty"`
	FactoryVerified bool      `db:"factory_verified" json:"factory_verified"`
	CategoryName    *string   `db:"category_name" json:"category_name,omitempty"`
	SubCategoryName *string   `db:"sub_category_name" json:"sub_category_name,omitempty"`
}

// ShowcaseByFactoryItem is the list payload for GET /factories/:id/showcases.
// Does not include factory info (caller already knows the factory context).
type ShowcaseByFactoryItem struct {
	ShowcaseID      int64     `db:"showcase_id" json:"showcase_id"`
	ContentType     string    `db:"content_type" json:"content_type"`
	Title           string    `db:"title" json:"title"`
	Excerpt         *string   `db:"excerpt" json:"excerpt,omitempty"`
	ImageURL        *string   `db:"image_url" json:"image_url,omitempty"`
	CategoryID      *int64    `db:"category_id" json:"category_id,omitempty"`
	SubCategoryID   *int64    `db:"sub_category_id" json:"sub_category_id,omitempty"`
	MinOrder        *int      `db:"min_order" json:"min_order,omitempty"`
	LeadTimeDays    *int      `db:"lead_time_days" json:"lead_time_days,omitempty"`
	PriceRange      *string   `db:"price_range" json:"price_range,omitempty"`
	LikesCount      int       `db:"likes_count" json:"likes_count"`
	Status          string    `db:"status" json:"status"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	CategoryName    *string   `db:"category_name" json:"category_name,omitempty"`
	SubCategoryName *string   `db:"sub_category_name" json:"sub_category_name,omitempty"`
}

// ShowcaseSpec is a single key-value specification row for a PD showcase.
type ShowcaseSpec struct {
	SpecID     int64  `db:"spec_id" json:"spec_id"`
	ShowcaseID int64  `db:"showcase_id" json:"showcase_id,omitempty"`
	SpecKey    string `db:"spec_key" json:"spec_key"`
	SpecValue  string `db:"spec_value" json:"spec_value"`
	SortOrder  int    `db:"sort_order" json:"sort_order"`
}

// ShowcaseSpecInput is the write model for PUT /showcases/:id/specs.
type ShowcaseSpecInput struct {
	SpecKey   string `json:"spec_key"`
	SpecValue string `json:"spec_value"`
	SortOrder int    `json:"sort_order"`
}

// ShowcaseImage is a gallery image belonging to a showcase.
type ShowcaseImage struct {
	ImageID    int64   `db:"image_id" json:"image_id"`
	ShowcaseID int64   `db:"showcase_id" json:"showcase_id,omitempty"`
	ImageURL   string  `db:"image_url" json:"image_url"`
	SortOrder  int     `db:"sort_order" json:"sort_order"`
	Caption    *string `db:"caption" json:"caption"`
}

// ShowcaseSectionItem is a single item inside a ShowcaseSection.
type ShowcaseSectionItem struct {
	ItemID      int64   `db:"item_id" json:"item_id"`
	Title       *string `db:"title" json:"title"`
	Description string  `db:"description" json:"description"`
	IconName    *string `db:"icon_name" json:"icon_name"`
	SortOrder   int     `db:"sort_order" json:"sort_order"`
}

// ShowcaseSection is a highlight or checklist section on a showcase detail page.
type ShowcaseSection struct {
	SectionID    int64                 `db:"section_id" json:"section_id"`
	SectionType  string                `db:"section_type" json:"section_type"`
	SectionTitle string                `db:"section_title" json:"section_title"`
	SortOrder    int                   `db:"sort_order" json:"sort_order"`
	Items        []ShowcaseSectionItem `db:"-" json:"items"`
}

// ShowcaseDetail is the full detail payload for GET /showcases/:id.
type ShowcaseDetail struct {
	ShowcaseID            int64     `db:"showcase_id" json:"showcase_id"`
	FactoryID             int64     `db:"factory_id" json:"factory_id"`
	ContentType           string    `db:"content_type" json:"content_type"`
	Title                 string    `db:"title" json:"title"`
	Excerpt               *string   `db:"excerpt" json:"excerpt,omitempty"`
	Description           *string   `db:"description" json:"description,omitempty"`
	ImageURL              *string   `db:"image_url" json:"image_url,omitempty"`
	CategoryID            *int64    `db:"category_id" json:"category_id,omitempty"`
	SubCategoryID         *int64    `db:"sub_category_id" json:"sub_category_id,omitempty"`
	MinOrder              *int      `db:"min_order" json:"min_order,omitempty"`
	LeadTimeDays          *int      `db:"lead_time_days" json:"lead_time_days,omitempty"`
	PriceRange            *string   `db:"price_range" json:"price_range,omitempty"`
	LikesCount            int       `db:"likes_count" json:"likes_count"`
	Status                string    `db:"status" json:"status"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
	FactoryName           string    `db:"factory_name" json:"factory_name"`
	FactoryImageURL       *string   `db:"factory_image_url" json:"factory_image_url,omitempty"`
	FactoryRating         *float64  `db:"factory_rating" json:"factory_rating,omitempty"`
	FactoryVerified       bool      `db:"factory_verified" json:"factory_verified"`
	FactorySpecialization *string   `db:"factory_specialization" json:"factory_specialization,omitempty"`
	FactoryReviewCount    *int      `db:"factory_review_count" json:"factory_review_count,omitempty"`
	ProvinceName          *string   `db:"province_name" json:"province_name,omitempty"`
	CategoryName          *string   `db:"category_name" json:"category_name,omitempty"`
	SubCategoryName       *string   `db:"sub_category_name" json:"sub_category_name,omitempty"`
	Images   []ShowcaseImage   `db:"-" json:"images"`
	Specs    []ShowcaseSpec    `db:"-" json:"specs"`
	Sections []ShowcaseSection `db:"-" json:"sections"`
}

// ShowcaseSectionInput is the write model for PUT /showcases/:id/sections.
type ShowcaseSectionInput struct {
	SectionType  string                    `json:"section_type"`
	SectionTitle string                    `json:"section_title"`
	SortOrder    int                       `json:"sort_order"`
	Items        []ShowcaseSectionItemInput `json:"items"`
}

// ShowcaseSectionItemInput is a single item inside ShowcaseSectionInput.
type ShowcaseSectionItemInput struct {
	Title       *string `json:"title"`
	Description string  `json:"description"`
	IconName    *string `json:"icon_name"`
	SortOrder   int     `json:"sort_order"`
}

type PromoSlide struct {
	SlideID  int64  `db:"slide_id" json:"slide_id"`
	Title    string `db:"title" json:"title"`
	Subtitle string `db:"subtitle" json:"subtitle"`
	Code     string `db:"code" json:"code"`
	ImageURL string `db:"image_url" json:"image_url"`
	Status   string `db:"status" json:"status"`
}

type ShowcaseAnalytics struct {
	ShowcaseID      int64   `json:"showcase_id"`
	FactoryID       int64   `json:"factory_id"`
	Title           string  `json:"title"`
	ContentType     string  `json:"content_type"`
	LikesCount      int     `json:"likes_count"`
	ViewCount       int64   `json:"view_count"`
	EngagementScore float64 `json:"engagement_score"`
}

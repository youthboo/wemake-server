package domain

import "time"

type FactoryShowcase struct {
	ShowcaseID    int64     `db:"showcase_id" json:"showcase_id"`
	FactoryID     int64     `db:"factory_id" json:"factory_id"`
	ContentType   string    `db:"content_type" json:"content_type"`
	Title         string    `db:"title" json:"title"`
	Excerpt       *string   `db:"excerpt" json:"excerpt,omitempty"`
	ImageURL      *string   `db:"image_url" json:"image_url,omitempty"`
	CategoryID    *int64    `db:"category_id" json:"category_id,omitempty"`
	SubCategoryID *int64    `db:"sub_category_id" json:"sub_category_id,omitempty"`
	MinOrder      *int      `db:"min_order" json:"min_order,omitempty"`
	LeadTimeDays  *int      `db:"lead_time_days" json:"lead_time_days,omitempty"`
	LikesCount    int       `db:"likes_count" json:"likes_count"`
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
	LikesCount      int       `db:"likes_count" json:"likes_count"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	FactoryName     string    `db:"factory_name" json:"factory_name"`
	FactoryImageURL *string   `db:"factory_image_url" json:"factory_image_url,omitempty"`
	FactoryRating   *float64  `db:"factory_rating" json:"factory_rating,omitempty"`
	FactoryVerified bool      `db:"factory_verified" json:"factory_verified"`
	CategoryName    *string   `db:"category_name" json:"category_name,omitempty"`
	SubCategoryName *string   `db:"sub_category_name" json:"sub_category_name,omitempty"`
}

type PromoSlide struct {
	SlideID  int64  `db:"slide_id" json:"slide_id"`
	Title    string `db:"title" json:"title"`
	Subtitle string `db:"subtitle" json:"subtitle"`
	Code     string `db:"code" json:"code"`
	ImageURL string `db:"image_url" json:"image_url"`
	Status   string `db:"status" json:"status"`
}

package domain

import "time"

type FactoryShowcase struct {
	ShowcaseID   int64     `db:"showcase_id" json:"showcase_id"`
	FactoryID    int64     `db:"factory_id" json:"factory_id"`
	ContentType  string    `db:"content_type" json:"content_type"`
	Title        string    `db:"title" json:"title"`
	Excerpt      string    `db:"excerpt" json:"excerpt"`
	ImageURL     string    `db:"image_url" json:"image_url"`
	CategoryID   *int64    `db:"category_id" json:"category_id,omitempty"`
	MinOrder     *int      `db:"min_order" json:"min_order,omitempty"`
	LeadTimeDays *int      `db:"lead_time_days" json:"lead_time_days,omitempty"`
	LikesCount   int       `db:"likes_count" json:"likes_count"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type PromoSlide struct {
	SlideID  int64  `db:"slide_id" json:"slide_id"`
	Title    string `db:"title" json:"title"`
	Subtitle string `db:"subtitle" json:"subtitle"`
	Code     string `db:"code" json:"code"`
	ImageURL string `db:"image_url" json:"image_url"`
	Status   string `db:"status" json:"status"`
}

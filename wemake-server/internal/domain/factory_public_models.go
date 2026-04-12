package domain

import "time"

// FactoryListItem is the JSON shape for GET /api/v1/factories (Explore listing).
type FactoryListItem struct {
	FactoryID       int64    `json:"factory_id" db:"factory_id"`
	FactoryName     string   `json:"factory_name" db:"factory_name"`
	FactoryTypeID   int64    `json:"factory_type_id" db:"factory_type_id"`
	FactoryTypeName *string  `json:"factory_type_name,omitempty" db:"factory_type_name"`
	Specialization  *string  `json:"specialization,omitempty" db:"specialization"`
	Rating          *float64 `json:"rating,omitempty" db:"rating"`
	ReviewCount     int64    `json:"review_count" db:"review_count"`
	MinOrder        *int     `json:"min_order,omitempty" db:"min_order"`
	LeadTimeDesc    *string  `json:"lead_time_desc,omitempty" db:"lead_time_desc"`
	IsVerified      bool     `json:"is_verified" db:"is_verified"`
	CompletedOrders int64    `json:"completed_orders" db:"completed_orders"`
	ImageURL        *string  `json:"image_url,omitempty" db:"image_url"`
	Description     *string  `json:"description,omitempty" db:"description"`
	PriceRange      *string  `json:"price_range,omitempty" db:"price_range"`
	ProvinceID      *int64   `json:"province_id,omitempty" db:"province_id"`
	ProvinceName    *string  `json:"province_name,omitempty" db:"province_name"`
}

type FactoryProfileCategory struct {
	CategoryID int64  `db:"category_id" json:"category_id"`
	Name       string `db:"name" json:"name"`
}

type FactoryProfileSubCategory struct {
	SubCategoryID int64  `db:"sub_category_id" json:"sub_category_id"`
	CategoryID    int64  `db:"category_id" json:"category_id"`
	Name          string `db:"name" json:"name"`
}

type FactoryProfileCertificate struct {
	CertID       int64  `db:"cert_id" json:"cert_id"`
	CertName     string `db:"cert_name" json:"cert_name"`
	VerifyStatus string `db:"verify_status" json:"verify_status"`
}

type FactoryProfileReview struct {
	ReviewID  int64     `json:"review_id"`
	UserID    int64     `json:"user_id"`
	Rating    int       `json:"rating"`
	Comment   *string   `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	FirstName *string   `json:"first_name,omitempty"`
	LastName  *string   `json:"last_name,omitempty"`
}

// FactoryPublicDetail is GET /api/v1/factories/:id aggregate for FE profile page.
type FactoryPublicDetail struct {
	FactoryID       int64                       `json:"factory_id"`
	FactoryName     string                      `json:"factory_name"`
	FactoryTypeID   int64                       `json:"factory_type_id"`
	FactoryTypeName *string                     `json:"factory_type_name,omitempty"`
	TaxID           *string                     `json:"tax_id,omitempty"`
	Specialization  *string                     `json:"specialization,omitempty"`
	MinOrder        *int                        `json:"min_order,omitempty"`
	LeadTimeDesc    *string                     `json:"lead_time_desc,omitempty"`
	IsVerified      bool                        `json:"is_verified"`
	Rating          *float64                    `json:"rating,omitempty"`
	ReviewCount     int64                       `json:"review_count"`
	CompletedOrders int64                       `json:"completed_orders"`
	ImageURL        *string                     `json:"image_url,omitempty"`
	Description     *string                     `json:"description,omitempty"`
	PriceRange      *string                     `json:"price_range,omitempty"`
	ProvinceID      *int64                      `json:"province_id,omitempty"`
	ProvinceName    *string                     `json:"province_name,omitempty"`
	Categories      []FactoryProfileCategory    `json:"categories"`
	SubCategories   []FactoryProfileSubCategory `json:"sub_categories"`
	Certificates    []FactoryProfileCertificate `json:"certificates"`
	Reviews         []FactoryProfileReview      `json:"reviews"`
}

type FactoryDashboardCounts struct {
	PendingRFQs              int64 `json:"pending_rfqs"`
	ActiveOrders             int64 `json:"active_orders"`
	PendingProductionUpdates int64 `json:"pending_production_updates"`
	UnreadMessages           int64 `json:"unread_messages"`
	UnreadNotifications      int64 `json:"unread_notifications"`
}

type FactoryDashboardWallet struct {
	GoodFund    float64 `json:"good_fund"`
	PendingFund float64 `json:"pending_fund"`
}

type FactoryDashboard struct {
	FactoryID          int64                           `json:"factory_id"`
	Counts             FactoryDashboardCounts          `json:"counts"`
	Wallet             FactoryDashboardWallet          `json:"wallet"`
	RecentMatchingRFQs []FactoryDashboardRFQItem       `json:"recent_matching_rfqs"`
	RecentOrders       []FactoryDashboardOrderItem     `json:"recent_orders"`
	RecentQuotations   []FactoryDashboardQuotationItem `json:"recent_quotations"`
	RecentShowcases    []FactoryDashboardShowcaseItem  `json:"recent_showcases"`
}

type FactoryDashboardRFQItem struct {
	RFQID         int64      `json:"rfq_id"`
	Title         string     `json:"title"`
	CategoryID    int64      `json:"category_id"`
	SubCategoryID *int64     `json:"sub_category_id,omitempty"`
	Status        string     `json:"status"`
	DeadlineDate  *time.Time `json:"deadline_date,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type FactoryDashboardOrderItem struct {
	OrderID           int64      `json:"order_id"`
	QuoteID           int64      `json:"quote_id"`
	UserID            int64      `json:"user_id"`
	Status            string     `json:"status"`
	TotalAmount       float64    `json:"total_amount"`
	EstimatedDelivery *time.Time `json:"estimated_delivery,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type FactoryDashboardQuotationItem struct {
	QuoteID       int64     `json:"quote_id"`
	RFQID         int64     `json:"rfq_id"`
	Status        string    `json:"status"`
	PricePerPiece float64   `json:"price_per_piece"`
	LeadTimeDays  int64     `json:"lead_time_days"`
	LogTimestamp  time.Time `json:"log_timestamp"`
}

type FactoryDashboardShowcaseItem struct {
	ShowcaseID    int64     `json:"showcase_id"`
	ContentType   string    `json:"content_type"`
	Title         string    `json:"title"`
	CategoryID    *int64    `json:"category_id,omitempty"`
	SubCategoryID *int64    `json:"sub_category_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

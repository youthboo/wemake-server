package domain

import "time"

// FactoryAnalytics is the payload for GET /factories/me/analytics.
type FactoryAnalytics struct {
	FactoryID       int64   `json:"factory_id"`
	TotalOrders     int64   `json:"total_orders"`
	CompletedOrders int64   `json:"completed_orders"`
	ActiveOrders    int64   `json:"active_orders"`
	CancelledOrders int64   `json:"cancelled_orders"`
	TotalRevenue    float64 `json:"total_revenue"`
	TotalQuotations int64   `json:"total_quotations"`
	AcceptedQuotes  int64   `json:"accepted_quotes"`
	PendingQuotes   int64   `json:"pending_quotes"`
	TotalShowcases  int64   `json:"total_showcases"`
	TotalViews      int64   `json:"total_views"`
	TotalLikes      int64   `json:"total_likes"`
	AverageRating   float64 `json:"average_rating"`
	TotalReviews    int64   `json:"total_reviews"`
}

// FactoryListItem is the JSON shape for GET /api/v1/factories (Explore listing).
type FactoryListItem struct {
	FactoryID          int64       `json:"factory_id" db:"factory_id"`
	FactoryName        string      `json:"factory_name" db:"factory_name"`
	Specialization     *string     `json:"specialization,omitempty" db:"specialization"`
	Rating             *float64    `json:"rating,omitempty" db:"rating"`
	ReviewCount        int64       `json:"review_count" db:"review_count"`
	LeadTimeDesc       *string     `json:"lead_time_desc,omitempty" db:"lead_time_desc"`
	IsVerified         bool        `json:"is_verified" db:"is_verified"`
	CompletedOrders    int64       `json:"completed_orders" db:"completed_orders"`
	ImageURL           *string     `json:"image_url,omitempty" db:"image_url"`
	BackgroundImageURL *string     `json:"background_image_url,omitempty" db:"background_image_url"`
	Description        *string     `json:"description,omitempty" db:"description"`
	PriceRange         *string     `json:"price_range,omitempty" db:"price_range"`
	ProvinceID         *int64      `json:"province_id,omitempty" db:"province_id"`
	ProvinceName       *string     `json:"province_name,omitempty" db:"province_name"`
	// Tags contains the factory's main category names (from map_factory_categories →
	// lbi_categories). Used by the FE for keyword search and card display.
	Tags               StringArray `json:"tags" db:"tags"`
	// CategoryScopes holds distinct lbi_categories.scope values ("PD", "MT") from
	// map_factory_categories — used by the factory-type dropdown on /factory-ideas.
	CategoryScopes     StringArray `json:"category_scopes" db:"category_scopes"`
}

type FactoryProfileCategory struct {
	CategoryID int64  `db:"category_id" json:"category_id"`
	Name       string `db:"name" json:"name"`
}

type FactoryProfileSubCategory struct {
	SubCategoryID int64  `db:"sub_category_id" json:"sub_category_id"`
	CategoryID    int64  `db:"category_id" json:"category_id"`
	// CategoryName is the parent category's display name — required by FE
	// to group sub-categories under their parent on the factory profile page.
	CategoryName string `db:"category_name" json:"category_name"`
	Name         string `db:"name" json:"name"`
	// SubCategoryName is an alias of Name for FE compatibility (FE reads
	// either `sub_category_name` or `name`).
	SubCategoryName string `db:"sub_category_name" json:"sub_category_name"`
}

type FactoryProfileCertificate struct {
	MapID        int64   `db:"map_id" json:"map_id"`
	CertID       int64   `db:"cert_id" json:"cert_id"`
	CertName     string  `db:"cert_name" json:"cert_name"`
	VerifyStatus string  `db:"verify_status" json:"verify_status"`
	DocumentURL  *string `db:"document_url" json:"document_url,omitempty"`
	CertNumber   *string `db:"cert_number" json:"cert_number,omitempty"`
	ExpireDate   *string `db:"expire_date" json:"expire_date,omitempty"`
}

type FactoryProfileReview struct {
	ReviewID  int64     `json:"review_id"`
	UserID    int64     `json:"user_id"`
	Rating    float64   `json:"rating"`
	Comment   *string   `json:"comment,omitempty"`
	ImageURLs StringArray `json:"image_urls"`
	CreatedAt time.Time `json:"created_at"`
	FirstName *string   `json:"first_name,omitempty"`
	LastName  *string   `json:"last_name,omitempty"`
}

// FactoryPublicDetail is GET /api/v1/factories/:id aggregate for FE profile page.
type FactoryPublicDetail struct {
	FactoryID          int64                       `json:"factory_id"`
	FactoryName        string                      `json:"factory_name"`
	TaxID              *string                     `json:"tax_id,omitempty"`
	Specialization     *string                     `json:"specialization,omitempty"`
	LeadTimeDesc       *string                     `json:"lead_time_desc,omitempty"`
	LeadTimeDese       *string                     `json:"lead_time_dese,omitempty"`
	IsVerified         bool                        `json:"is_verified"`
	Rating             *float64                    `json:"rating,omitempty"`
	ReviewCount        int64                       `json:"review_count"`
	CompletedOrders    int64                       `json:"completed_orders"`
	ImageURL           *string                     `json:"image_url,omitempty"`
	BackgroundImageURL *string                     `json:"background_image_url,omitempty"`
	Description        *string                     `json:"description,omitempty"`
	PriceRange         *string                     `json:"price_range,omitempty"`
	ProvinceID         *int64                      `json:"province_id,omitempty"`
	ProvinceName       *string                     `json:"province_name,omitempty"`
	Categories         []FactoryProfileCategory    `json:"categories"`
	SubCategories      []FactoryProfileSubCategory `json:"sub_categories"`
	Certificates       []FactoryProfileCertificate `json:"certificates"`
	Reviews            []FactoryProfileReview      `json:"reviews"`
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
	RFQID         int64     `json:"rfq_id"`
	Title         string    `json:"title"`
	CategoryID    int64     `json:"category_id"`
	SubCategoryID *int64    `json:"sub_category_id,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
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

// PortalOrderItem is a lightweight order row used in the /factories/me/portal response.
// Only the fields needed for chart series and KPI calculation are included.
type PortalOrderItem struct {
	OrderID     int64     `json:"order_id" db:"order_id"`
	FactoryID   int64     `json:"factory_id" db:"factory_id"`
	Status      string    `json:"status" db:"status"`
	TotalAmount float64   `json:"total_amount" db:"total_amount"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// PortalQuotationItem is a lightweight quotation row for the portal response.
type PortalQuotationItem struct {
	QuoteID   int64     `json:"quote_id" db:"quote_id"`
	FactoryID int64     `json:"factory_id" db:"factory_id"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RFQHistoryItem is an RFQ that this factory has actually quoted on, with the
// original RFQ created_at date. Used for the "RFQ received per period" chart series.
type RFQHistoryItem struct {
	RFQID     int64     `json:"rfq_id" db:"rfq_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// FactoryPortal is the aggregate response for GET /factories/me/portal.
// It replaces 6 separate API calls the factory dashboard page previously made.
type FactoryPortal struct {
	Analytics        *FactoryAnalytics               `json:"analytics"`
	Counts           FactoryDashboardCounts           `json:"counts"`
	Wallet           FactoryDashboardWallet           `json:"wallet"`
	// MatchingRFQs is the full list of open RFQs matching this factory's categories,
	// used by the FE to discover new quoting opportunities.
	MatchingRFQs     []FactoryDashboardRFQItem        `json:"matching_rfqs"`
	// RFQHistory is every RFQ this factory has ever submitted a quotation for,
	// with the original RFQ created_at date. Used for the per-period chart series
	// so the "RFQ received" bars reflect real historical timestamps.
	RFQHistory       []RFQHistoryItem                 `json:"rfq_history"`
	// Orders is the full list of the factory's orders (lightweight), used for chart series.
	Orders           []PortalOrderItem               `json:"orders"`
	// Quotations is the full list of the factory's quotations (lightweight), used for chart series.
	Quotations       []PortalQuotationItem           `json:"quotations"`
	// Recent* are the 5-item summary slices shown in the dashboard widget tiles.
	RecentRFQs       []FactoryDashboardRFQItem        `json:"recent_rfqs"`
	RecentOrders     []FactoryDashboardOrderItem      `json:"recent_orders"`
	RecentQuotations []FactoryDashboardQuotationItem  `json:"recent_quotations"`
	RecentShowcases  []FactoryDashboardShowcaseItem   `json:"recent_showcases"`
}

type FactoryDashboardShowcaseItem struct {
	ShowcaseID    int64     `json:"showcase_id"`
	ContentType   string    `json:"content_type"`
	Title         string    `json:"title"`
	CategoryID    *int64    `json:"category_id,omitempty"`
	SubCategoryID *int64    `json:"sub_category_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

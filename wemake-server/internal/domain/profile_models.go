package domain

import "time"

type ProfileSummary struct {
	TotalOrders           int64    `json:"total_orders"`
	PendingOrders         int64    `json:"pending_orders"`
	CompletedOrders       int64    `json:"completed_orders"`
	CancelledOrders       int64    `json:"cancelled_orders"`
	TotalSpendTHB         float64  `json:"total_spend_thb"`
	TotalRFQs             int64    `json:"total_rfqs"`
	ActiveRFQs            int64    `json:"active_rfqs"`
	ReviewsGiven          int64    `json:"reviews_given"`
	AverageRatingReceived *float64 `json:"average_rating_received,omitempty"`
}

type TransactionListItem struct {
	TxID          string    `json:"tx_id"`
	Type          string    `json:"type"`
	TypeLabel     string    `json:"type_label"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Direction     string    `json:"direction"`
	Description   string    `json:"description"`
	ReferenceID   *int64    `json:"reference_id,omitempty"`
	ReferenceType *string   `json:"reference_type,omitempty"`
	Status        string    `json:"status"`
	StatusLabel   string    `json:"status_label"`
	CreatedAt     time.Time `json:"created_at"`
}

type UserReviewListItem struct {
	ReviewID      int64      `json:"review_id"`
	OrderID       *int64     `json:"order_id,omitempty"`
	FactoryID     int64      `json:"factory_id"`
	FactoryName   string     `json:"factory_name"`
	FactoryAvatar *string    `json:"factory_avatar,omitempty"`
	ReviewerName  *string    `json:"reviewer_name,omitempty"`
	Rating        int        `json:"rating"`
	Comment       string     `json:"comment"`
	IsEditable    bool       `json:"is_editable"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type NotificationPreference struct {
	UserID       int64     `db:"user_id" json:"user_id"`
	OrderUpdates bool      `db:"order_updates" json:"order_updates"`
	RFQUpdates   bool      `db:"rfq_updates" json:"rfq_updates"`
	ChatMessages bool      `db:"chat_messages" json:"chat_messages"`
	Promotions   bool      `db:"promotions" json:"promotions"`
	EmailEnabled bool      `db:"email_enabled" json:"email_enabled"`
	PushEnabled  bool      `db:"push_enabled" json:"push_enabled"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type ProfileAddress struct {
	Line1       *string `json:"line1,omitempty"`
	SubDistrict *string `json:"sub_district,omitempty"`
	District    *string `json:"district,omitempty"`
	Province    *string `json:"province,omitempty"`
	PostalCode  *string `json:"postal_code,omitempty"`
}

type ProfileResponse struct {
	UserID         int64           `json:"user_id"`
	Role           string          `json:"role"`
	Email          string          `json:"email"`
	Phone          string          `json:"phone"`
	AvatarURL      *string         `json:"avatar_url,omitempty"`
	Bio            *string         `json:"bio,omitempty"`
	FirstName      *string         `json:"first_name,omitempty"`
	LastName       *string         `json:"last_name,omitempty"`
	FactoryName    *string         `json:"factory_name,omitempty"`
	FactoryTypeID  *int64          `json:"factory_type_id,omitempty"`
	TaxID          *string         `json:"tax_id,omitempty"`
	ProvinceID     *int64          `json:"province_id,omitempty"`
	Specialization *string         `json:"specialization,omitempty"`
	MinOrder       *int64          `json:"min_order,omitempty"`
	LeadTimeDesc   *string         `json:"lead_time_desc,omitempty"`
	IsVerified     *bool           `json:"is_verified,omitempty"`
	VerifiedAt     *time.Time      `json:"verified_at,omitempty"`
	Description    *string         `json:"description,omitempty"`
	PriceRange     *string         `json:"price_range,omitempty"`
	Address        *ProfileAddress `json:"address,omitempty"`
	IsActive       bool            `json:"is_active"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

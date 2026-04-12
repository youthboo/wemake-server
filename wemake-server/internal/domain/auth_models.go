package domain

import "time"

const (
	RoleCustomer = "CT"
	RoleFactory  = "FT"
)

type User struct {
	UserID       int64     `db:"user_id" json:"user_id"`
	Role         string    `db:"role" json:"role"`
	Email        string    `db:"email" json:"email"`
	Phone        string    `db:"phone" json:"phone"`
	PasswordHash string    `db:"password_hash" json:"-"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type CustomerProfile struct {
	UserID    int64  `db:"user_id" json:"user_id"`
	FirstName string `db:"first_name" json:"first_name"`
	LastName  string `db:"last_name" json:"last_name"`
}

type FactoryProfile struct {
	UserID          int64    `db:"user_id" json:"user_id"`
	FactoryName     string   `db:"factory_name" json:"factory_name"`
	FactoryTypeID   int64    `db:"factory_type_id" json:"factory_type_id"`
	TaxID           string   `db:"tax_id" json:"tax_id,omitempty"`
	ProvinceID      *int64   `db:"province_id" json:"province_id,omitempty"`
	Rating          *float64 `db:"rating" json:"rating,omitempty"`
	ReviewCount     int64    `db:"review_count" json:"review_count"`
	Specialization  *string  `db:"specialization" json:"specialization,omitempty"`
	MinOrder        *int64   `db:"min_order" json:"min_order,omitempty"`
	LeadTimeDesc    *string  `db:"lead_time_desc" json:"lead_time_desc,omitempty"`
	IsVerified      bool       `db:"is_verified" json:"is_verified"`
	VerifiedAt      *time.Time `db:"verified_at" json:"verified_at,omitempty"`
	CompletedOrders int64      `db:"completed_orders" json:"completed_orders"`
	ImageURL        *string  `db:"image_url" json:"image_url,omitempty"`
	Description     *string  `db:"description" json:"description,omitempty"`
	PriceRange      *string  `db:"price_range" json:"price_range,omitempty"`
}

type PasswordResetToken struct {
	ID        int64      `db:"id"`
	UserID    int64      `db:"user_id"`
	Token     string     `db:"token"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}

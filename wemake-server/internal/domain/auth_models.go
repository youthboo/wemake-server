package domain

import "time"

const (
	RoleCustomer = "CT"
	RoleFactory  = "FT"
)

type User struct {
	UserID       int64      `db:"user_id" json:"user_id"`
	Role         string     `db:"role" json:"role"`
	Email        string     `db:"email" json:"email"`
	Phone        string     `db:"phone" json:"phone"`
	PasswordHash string     `db:"password_hash" json:"-"`
	IsActive     bool       `db:"is_active" json:"is_active"`
	LogTimestamp *time.Time `db:"log_timestamp" json:"log_timestamp,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

type CustomerProfile struct {
	UserID    int64  `db:"user_id" json:"user_id"`
	FirstName string `db:"first_name" json:"first_name"`
	LastName  string `db:"last_name" json:"last_name"`
}

type FactoryProfile struct {
	UserID        int64  `db:"user_id" json:"user_id"`
	FactoryName   string `db:"factory_name" json:"factory_name"`
	FactoryTypeID int64  `db:"factory_type_id" json:"factory_type_id"`
	TaxID         string `db:"tax_id" json:"tax_id,omitempty"`
}

type PasswordResetToken struct {
	ID        int64      `db:"id"`
	UserID    int64      `db:"user_id"`
	Token     string     `db:"token"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
	CreatedAt time.Time  `db:"created_at"`
}

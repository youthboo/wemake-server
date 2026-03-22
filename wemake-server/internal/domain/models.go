package domain

import "time"

// Factory represents a factory in the system
type Factory struct {
	ID          string    `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Email       string    `db:"email" json:"email"`
	Phone       string    `db:"phone" json:"phone"`
	Address     string    `db:"address" json:"address"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// Entrepreneur represents an entrepreneur in the system
type Entrepreneur struct {
	ID        string    `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Email     string    `db:"email" json:"email"`
	Phone     string    `db:"phone" json:"phone"`
	Company   string    `db:"company" json:"company"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Connection represents a connection between factory and entrepreneur
type Connection struct {
	ID              string    `db:"id" json:"id"`
	FactoryID       string    `db:"factory_id" json:"factory_id"`
	EntrepreneurID  string    `db:"entrepreneur_id" json:"entrepreneur_id"`
	Status          string    `db:"status" json:"status"` // pending, approved, rejected
	Message         string    `db:"message" json:"message"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

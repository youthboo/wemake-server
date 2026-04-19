package domain

import "time"

type Conversation struct {
	ConvID         int64     `db:"conv_id" json:"conv_id"`
	CustomerID     int64     `db:"customer_id" json:"customer_id"`
	FactoryID      int64     `db:"factory_id" json:"factory_id"`
	FactoryName    string    `db:"factory_name" json:"factory_name"`
	FactoryImage   string    `db:"factory_image" json:"factory_image"`
	LastMessage    string    `db:"last_message" json:"last_message"`
	UnreadCustomer int       `db:"unread_customer" json:"unread_customer"`
	UnreadFactory  int       `db:"unread_factory" json:"unread_factory"`
	HasQuote       bool      `db:"has_quote" json:"has_quote"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

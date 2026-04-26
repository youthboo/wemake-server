package domain

import "time"

type Conversation struct {
	ConvID           int64     `db:"conv_id" json:"conv_id"`
	CustomerID       int64     `db:"customer_id" json:"customer_id"`
	FactoryID        int64     `db:"factory_id" json:"factory_id"`
	SourceShowcaseID *int64    `db:"source_showcase_id" json:"source_showcase_id,omitempty"`
	ConvType         string    `db:"conv_type" json:"conv_type"`
	FactoryName      string    `db:"factory_name" json:"factory_name"`
	FactoryImage     string    `db:"factory_image" json:"factory_image"`
	LastMessage      string    `db:"last_message" json:"last_message"`
	UnreadCustomer   int       `db:"unread_customer" json:"unread_customer"`
	UnreadFactory    int       `db:"unread_factory" json:"unread_factory"`
	HasQuote         bool      `db:"has_quote" json:"has_quote"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

type ConversationRow struct {
	ConvID                int64     `db:"conv_id"`
	CustomerID            int64     `db:"customer_id"`
	FactoryID             int64     `db:"factory_id"`
	SourceShowcaseID      *int64    `db:"source_showcase_id"`
	ConvType              string    `db:"conv_type"`
	LastMessage           *string   `db:"last_message"`
	UnreadCustomer        int       `db:"unread_customer"`
	UnreadFactory         int       `db:"unread_factory"`
	HasQuote              bool      `db:"has_quote"`
	UpdatedAt             time.Time `db:"updated_at"`
	CustomerFirstName     *string   `db:"customer_first_name"`
	CustomerLastName      *string   `db:"customer_last_name"`
	FactoryName           *string   `db:"factory_name"`
	FactoryImageURL       *string   `db:"factory_image_url"`
	FactoryIsVerified     *bool     `db:"factory_is_verified"`
	FactorySpecialization *string   `db:"factory_specialization"`
}

type ConversationResponse struct {
	ConvID             int64             `json:"conv_id"`
	CustomerID         int64             `json:"customer_id"`
	FactoryID          int64             `json:"factory_id"`
	SourceShowcaseID   *int64            `json:"source_showcase_id,omitempty"`
	ConvType           string            `json:"conv_type"`
	LastMessage        string            `json:"last_message"`
	UnreadCustomer     int               `json:"unread_customer"`
	UnreadFactory      int               `json:"unread_factory"`
	HasQuote           bool              `json:"has_quote"`
	UpdatedAt          time.Time         `json:"updated_at"`
	Customer           CustomerPartyInfo `json:"customer"`
	Factory            FactoryPartyInfo  `json:"factory"`
	ViewerRole         *string           `json:"viewer_role,omitempty"`
	CounterpartyUserID *int64            `json:"counterparty_user_id,omitempty"`
}

type CustomerPartyInfo struct {
	UserID      int64  `json:"user_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DisplayName string `json:"display_name"`
}

type FactoryPartyInfo struct {
	UserID         int64  `json:"user_id"`
	FactoryName    string `json:"factory_name"`
	ImageURL       string `json:"image_url"`
	IsVerified     bool   `json:"is_verified"`
	Specialization string `json:"specialization"`
}

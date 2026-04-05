package domain

import "time"

type Quotation struct {
	QuotationID      int64     `db:"quote_id" json:"quote_id"`
	RFQID            int64     `db:"rfq_id" json:"rfq_id"`
	FactoryID        int64     `db:"factory_id" json:"factory_id"`
	PricePerPiece    float64   `db:"price_per_piece" json:"price_per_piece"`
	MoldCost         float64   `db:"mold_cost" json:"mold_cost"`
	LeadTimeDays     int64     `db:"lead_time_days" json:"lead_time_days"`
	ShippingMethodID int64     `db:"shipping_method_id" json:"shipping_method_id"`
	Status           string    `db:"status" json:"status"`
	CreateTime       time.Time `db:"create_time" json:"create_time"`
	LogTimestamp     time.Time `db:"log_timestamp" json:"log_timestamp"`
}

type Order struct {
	OrderID            int64        `db:"order_id" json:"order_id"`
	QuotationID        int64        `db:"quote_id" json:"quote_id"`
	UserID             int64        `db:"user_id" json:"user_id"`
	FactoryID          int64        `db:"factory_id" json:"factory_id"`
	TotalAmount        float64      `db:"total_amount" json:"total_amount"`
	DepositAmount      float64      `db:"deposit_amount" json:"deposit_amount"`
	Status             string       `db:"status" json:"status"`
	EstimatedDelivery  *time.Time `db:"estimated_delivery" json:"estimated_delivery,omitempty"`
	CreatedAt          time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time    `db:"updated_at" json:"updated_at"`
}

type ProductionUpdate struct {
	UpdateID    int64     `db:"update_id" json:"update_id"`
	OrderID     int64     `db:"order_id" json:"order_id"`
	StepID      int64     `db:"step_id" json:"step_id"`
	Status      string    `db:"status" json:"status"`
	Description string    `db:"description" json:"description"`
	ImageURL    string    `db:"image_url" json:"image_url,omitempty"`
	UpdateDate  time.Time `db:"update_date" json:"update_date"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type Message struct {
	MessageID     string    `db:"message_id" json:"message_id"`
	ReferenceType string    `db:"reference_type" json:"reference_type"`
	ReferenceID   int64     `db:"reference_id" json:"reference_id"`
	SenderID      int64     `db:"sender_id" json:"sender_id"`
	ReceiverID    int64     `db:"receiver_id" json:"receiver_id"`
	Content       string    `db:"content" json:"content"`
	AttachmentURL string    `db:"attachment_url" json:"attachment_url,omitempty"`
	ConvID        *int64    `db:"conv_id" json:"conv_id,omitempty"`
	MessageType   string    `db:"message_type" json:"message_type"`
	QuoteData     *string   `db:"quote_data" json:"quote_data,omitempty"`
	IsRead        bool      `db:"is_read" json:"is_read"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type MessageThread struct {
	ReferenceType string    `db:"reference_type" json:"reference_type"`
	ReferenceID   int64     `db:"reference_id" json:"reference_id"`
	LastMessage   string    `db:"last_message" json:"last_message"`
	LastMessageAt time.Time `db:"last_message_at" json:"last_message_at"`
}

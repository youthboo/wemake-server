package domain

import (
	"encoding/json"
	"time"
)

type Quotation struct {
	QuotationID      int64      `db:"quote_id" json:"quote_id"`
	RFQID            int64      `db:"rfq_id" json:"rfq_id"`
	FactoryID        int64      `db:"factory_id" json:"factory_id"`
	PricePerPiece    float64    `db:"price_per_piece" json:"price_per_piece"`
	MoldCost         float64    `db:"mold_cost" json:"mold_cost"`
	LeadTimeDays     int64      `db:"lead_time_days" json:"lead_time_days"`
	ShippingMethodID int64      `db:"shipping_method_id" json:"shipping_method_id"`
	Status           string     `db:"status" json:"status"`
	CreateTime       time.Time  `db:"create_time" json:"create_time"`
	LogTimestamp     time.Time  `db:"log_timestamp" json:"log_timestamp"`
	Version          int        `db:"version" json:"version"`
	IsLocked         bool       `db:"is_locked" json:"is_locked"`
	LastEditedAt     *time.Time `db:"last_edited_at" json:"last_edited_at,omitempty"`
	LastEditedBy     *int64     `db:"last_edited_by" json:"last_edited_by,omitempty"`
}

type QuotationHistoryEntry struct {
	HistoryID        int64     `db:"history_id" json:"history_id"`
	QuoteID          int64     `db:"quote_id" json:"quote_id"`
	EventType        string    `db:"event_type" json:"event_type"`
	VersionAfter     int       `db:"version_after" json:"version_after"`
	PricePerPiece    *float64  `db:"price_per_piece" json:"price_per_piece,omitempty"`
	MoldCost         *float64  `db:"mold_cost" json:"mold_cost,omitempty"`
	LeadTimeDays     *int64    `db:"lead_time_days" json:"lead_time_days,omitempty"`
	ShippingMethodID *int64    `db:"shipping_method_id" json:"shipping_method_id,omitempty"`
	Status           *string   `db:"status" json:"status,omitempty"`
	Reason           *string   `db:"reason" json:"reason,omitempty"`
	EditedBy         *int64    `db:"edited_by" json:"edited_by,omitempty"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

type OrderActivityEntry struct {
	ActivityID  int64           `db:"activity_id" json:"activity_id"`
	OrderID     int64           `db:"order_id" json:"order_id"`
	ActorUserID *int64          `db:"actor_user_id" json:"actor_user_id,omitempty"`
	EventCode   string          `db:"event_code" json:"event_code"`
	Payload     json.RawMessage `db:"payload" json:"payload,omitempty"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
}

type Order struct {
	OrderID           int64      `db:"order_id" json:"order_id"`
	QuotationID       int64      `db:"quote_id" json:"quote_id"`
	UserID            int64      `db:"user_id" json:"user_id"`
	FactoryID         int64      `db:"factory_id" json:"factory_id"`
	TotalAmount       float64    `db:"total_amount" json:"total_amount"`
	DepositAmount     float64    `db:"deposit_amount" json:"deposit_amount"`
	Status            string     `db:"status" json:"status"`
	EstimatedDelivery *time.Time `db:"estimated_delivery" json:"estimated_delivery,omitempty"`
	TrackingNo        *string    `db:"tracking_no" json:"tracking_no,omitempty"`
	Courier           *string    `db:"courier" json:"courier,omitempty"`
	ShippedAt         *time.Time `db:"shipped_at" json:"shipped_at,omitempty"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updated_at"`
}

type OrderFactorySummary struct {
	FactoryID int64  `json:"factory_id"`
	Name      string `json:"name"`
}

type OrderNextAction struct {
	Actor      string     `json:"actor"`
	Type       string     `json:"type"`
	Amount     float64    `json:"amount"`
	Currency   string     `json:"currency"`
	DueDate    *time.Time `json:"due_date,omitempty"`
	CTAURL     string     `json:"cta_url,omitempty"`
	CTALabelTH string     `json:"cta_label_th,omitempty"`
}

type RfqImage struct {
	ImageID  string `db:"image_id" json:"image_id"`
	ImageURL string `db:"image_url" json:"image_url"`
}

type RfqNested struct {
	RfqID          int64      `json:"rfq_id"`
	Title          string     `json:"title"`
	Details        string     `json:"details"`
	Quantity       int64      `json:"quantity"`
	UnitName       string     `json:"unit_name"`
	BudgetPerPiece float64    `json:"budget_per_piece"`
	CategoryID     int64      `json:"category_id"`
	CategoryName   string     `json:"category_name"`
	DeadlineDate   *time.Time `json:"deadline_date,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	Images         []RfqImage `json:"images"`
}

type QuoteNested struct {
	QuoteID       int64   `json:"quote_id"`
	PricePerPiece float64 `json:"price_per_piece"`
	MoldCost      float64 `json:"mold_cost"`
	LeadTimeDays  int64   `json:"lead_time_days"`
}

// OrderDetailResponse extends the legacy order payload with FE-ready action state.
type OrderDetailResponse struct {
	OrderID           int64                      `json:"order_id"`
	QuotationID       int64                      `json:"quote_id"`
	UserID            int64                      `json:"user_id"`
	FactoryID         int64                      `json:"factory_id"`
	TotalAmount       float64                    `json:"total_amount"`
	DepositAmount     float64                    `json:"deposit_amount"`
	Status            string                     `json:"status"`
	StatusLabelTH     string                     `json:"status_label_th"`
	PaymentType       *string                    `json:"payment_type,omitempty"`
	Currency          string                     `json:"currency"`
	Factory           OrderFactorySummary        `json:"factory"`
	CustomerUserID    int64                      `json:"customer_user_id"`
	EstimatedDelivery *time.Time                 `json:"estimated_delivery,omitempty"`
	TrackingNo        *string                    `json:"tracking_no,omitempty"`
	Courier           *string                    `json:"courier,omitempty"`
	ShippedAt         *time.Time                 `json:"shipped_at,omitempty"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
	NextAction        *OrderNextAction           `json:"next_action"`
	PaymentSchedule   []OrderPaymentScheduleItem `json:"payment_schedule"`
	RFQ               RfqNested                  `json:"rfq"`
	Quotation         QuoteNested                `json:"quotation"`
}

type ProductionUpdate struct {
	UpdateID        int64       `db:"update_id" json:"update_id"`
	OrderID         int64       `db:"order_id" json:"order_id"`
	StepID          int64       `db:"step_id" json:"step_id"`
	StepCode        string      `db:"step_code" json:"step_code"`
	StepNameTH      string      `db:"step_name_th" json:"step_name_th"`
	StepNameEN      string      `db:"step_name_en" json:"step_name_en"`
	SortOrder       int64       `db:"sort_order" json:"sort_order"`
	Status          string      `db:"status" json:"status"`
	Description     string      `db:"description" json:"description"`
	ImageURLs       StringArray `db:"image_urls" json:"image_urls"`
	CompletedAt     *time.Time  `db:"completed_at" json:"completed_at,omitempty"`
	RejectedReason  *string     `db:"rejected_reason" json:"rejected_reason,omitempty"`
	UpdatedByUserID *int64      `db:"updated_by_user_id" json:"updated_by_user_id,omitempty"`
	LastUpdatedAt   *time.Time  `db:"last_updated_at" json:"last_updated_at,omitempty"`
	CreatedAt       time.Time   `db:"created_at" json:"created_at"`
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

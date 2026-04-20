package domain

import "time"

type PaymentStage string

const (
	PaymentStageDeposit    PaymentStage = "DEPOSIT"
	PaymentStageProduction PaymentStage = "PRODUCTION"
	PaymentStageDelivery   PaymentStage = "DELIVERY"
)

// PaymentSchedule represents an installment payment schedule row for an order.
type PaymentSchedule struct {
	ScheduleID    int64      `db:"schedule_id" json:"schedule_id"`
	OrderID       int64      `db:"order_id" json:"order_id"`
	InstallmentNo int        `db:"installment_no" json:"installment_no"`
	DueDate       time.Time  `db:"due_date" json:"due_date"`
	Amount        float64    `db:"amount" json:"amount"`
	Status        string     `db:"status" json:"status"` // PE, PD, OD
	PaidAt        *time.Time `db:"paid_at" json:"paid_at,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}

// OrderPaymentScheduleItem is the FE-facing schedule shape in GET /orders/:id.
type OrderPaymentScheduleItem struct {
	Stage           PaymentStage `json:"stage"`
	Percent         float64      `json:"percent"`
	Amount          float64      `json:"amount"`
	Status          string       `json:"status"`
	DueDate         *time.Time   `json:"due_date,omitempty"`
	PaidAt          *time.Time   `json:"paid_at,omitempty"`
	TriggeredByStep *string      `json:"triggered_by_step,omitempty"`
}

// Settlement represents a factory payout record.
type Settlement struct {
	SettlementID int64      `db:"settlement_id" json:"settlement_id"`
	FactoryID    int64      `db:"factory_id" json:"factory_id"`
	OrderID      *int64     `db:"order_id" json:"order_id,omitempty"`
	Amount       float64    `db:"amount" json:"amount"`
	Status       string     `db:"status" json:"status"` // PE, PR, CP, FL
	SettledAt    *time.Time `db:"settled_at" json:"settled_at,omitempty"`
	Note         *string    `db:"note" json:"note,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

// TopupIntent represents a PromptPay QR top-up request.
type TopupIntent struct {
	IntentID    string     `db:"intent_id" json:"intent_id"`
	WalletID    int64      `db:"wallet_id" json:"wallet_id"`
	Amount      float64    `db:"amount" json:"amount"`
	QRPayload   *string    `db:"qr_payload" json:"qr_payload,omitempty"`
	Status      string     `db:"status" json:"status"` // PE, CP, EX, FL
	ExpiresAt   *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	ConfirmedAt *time.Time `db:"confirmed_at" json:"confirmed_at,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}

// WithdrawalRequest represents a factory withdrawal request.
type WithdrawalRequest struct {
	RequestID     int64      `db:"request_id" json:"request_id"`
	WalletID      int64      `db:"wallet_id" json:"wallet_id"`
	FactoryID     int64      `db:"factory_id" json:"factory_id"`
	Amount        float64    `db:"amount" json:"amount"`
	BankAccountNo string     `db:"bank_account_no" json:"bank_account_no"`
	BankName      string     `db:"bank_name" json:"bank_name"`
	AccountName   string     `db:"account_name" json:"account_name"`
	Status        string     `db:"status" json:"status"` // PE, AP, RJ, CP
	ProcessedAt   *time.Time `db:"processed_at" json:"processed_at,omitempty"`
	Note          *string    `db:"note" json:"note,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

// Dispute represents an order dispute.
type Dispute struct {
	DisputeID  int64      `db:"dispute_id" json:"dispute_id"`
	OrderID    int64      `db:"order_id" json:"order_id"`
	OpenedBy   int64      `db:"opened_by" json:"opened_by"`
	Reason     string     `db:"reason" json:"reason"`
	Status     string     `db:"status" json:"status"` // OP, RS, CL
	Resolution *string    `db:"resolution" json:"resolution,omitempty"`
	ResolvedAt *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`
}

// QuotationTemplate represents a factory's reusable quotation template.
type QuotationTemplate struct {
	TemplateID       int64     `db:"template_id" json:"template_id"`
	FactoryID        int64     `db:"factory_id" json:"factory_id"`
	TemplateName     string    `db:"template_name" json:"template_name"`
	PricePerPiece    *float64  `db:"price_per_piece" json:"price_per_piece,omitempty"`
	MoldCost         *float64  `db:"mold_cost" json:"mold_cost,omitempty"`
	LeadTimeDays     *int      `db:"lead_time_days" json:"lead_time_days,omitempty"`
	ShippingMethodID *int64    `db:"shipping_method_id" json:"shipping_method_id,omitempty"`
	Note             *string   `db:"note" json:"note,omitempty"`
	IsActive         bool      `db:"is_active" json:"is_active"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

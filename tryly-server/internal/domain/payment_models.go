package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type PaymentStage string

const (
	PaymentStageDeposit     PaymentStage = "DEPOSIT"
	PaymentStageProduction  PaymentStage = "PRODUCTION"
	PaymentStageDelivery    PaymentStage = "DELIVERY"
	PaymentStageFullPayment PaymentStage = "FULL_PAYMENT"
)

// PaymentSchedule represents an installment payment schedule row for an order.
type PaymentSchedule struct {
	ScheduleID    int64      `db:"schedule_id" json:"schedule_id"`
	OrderID       int64      `db:"order_id" json:"order_id"`
	InstallmentNo int        `db:"installment_no" json:"installment_no"`
	DueDate       time.Time  `db:"due_date" json:"due_date"`
	Amount        decimal.Decimal `db:"amount" json:"amount"`
	Status        string          `db:"status" json:"status"` // PE, PD, OD
	PaidAt        *time.Time `db:"paid_at" json:"paid_at,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}

// OrderPaymentScheduleItem is the FE-facing schedule shape in GET /orders/:id.
type OrderPaymentScheduleItem struct {
	Stage           PaymentStage `json:"stage"`
	Percent         decimal.Decimal `json:"percent"`
	Amount          decimal.Decimal `json:"amount"`
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
	Amount       decimal.Decimal `db:"amount" json:"amount"`
	Status       string          `db:"status" json:"status"` // PE, PR, CP, FL
	SettledAt    *time.Time `db:"settled_at" json:"settled_at,omitempty"`
	Note         *string    `db:"note" json:"note,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

// TopupIntent represents a PromptPay QR top-up request.
type TopupIntent struct {
	IntentID    string     `db:"intent_id" json:"intent_id"`
	WalletID    int64      `db:"wallet_id" json:"wallet_id"`
	Amount      decimal.Decimal `db:"amount" json:"amount"`
	QRPayload   *string         `db:"qr_payload" json:"qr_payload,omitempty"`
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
	Amount        decimal.Decimal `db:"amount" json:"amount"`
	BankAccountNo string          `db:"bank_account_no" json:"bank_account_no"`
	BankName      string     `db:"bank_name" json:"bank_name"`
	AccountName   string     `db:"account_name" json:"account_name"`
	Status        string     `db:"status" json:"status"` // PE, AP, RJ, CP
	ProcessedAt   *time.Time `db:"processed_at" json:"processed_at,omitempty"`
	ProcessedBy   *int64     `db:"processed_by" json:"processed_by,omitempty"`
	Note          *string    `db:"note" json:"note,omitempty"`
	SlipURL       *string    `db:"slip_url" json:"slip_url,omitempty"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at" json:"updated_at"`
}

// Dispute represents an order dispute.
type Dispute struct {
	DisputeID     int64       `db:"dispute_id" json:"dispute_id"`
	OrderID       int64       `db:"order_id" json:"order_id"`
	OpenedBy      int64       `db:"opened_by" json:"opened_by"`
	Category      string      `db:"category" json:"category"` // NR=ไม่ได้รับสินค้า, ND=ไม่ตรงปก, OT=อื่นๆ
	Reason        string      `db:"reason" json:"reason"`
	EvidenceURLs  StringArray `db:"evidence_urls" json:"evidence_urls"`
	RefundAccount     *string `db:"refund_account" json:"refund_account,omitempty"`
	RefundAccountName *string `db:"refund_account_name" json:"refund_account_name,omitempty"`
	ContactEmail      *string `db:"contact_email" json:"contact_email,omitempty"`
	ContactPhone      *string `db:"contact_phone" json:"contact_phone,omitempty"`
	ReturnTrackingNo   *string     `db:"return_tracking_no" json:"return_tracking_no,omitempty"`
	ReturnCourier      *string     `db:"return_courier" json:"return_courier,omitempty"`
	ReturnNote         *string     `db:"return_note" json:"return_note,omitempty"`
	ReturnEvidenceURLs StringArray `db:"return_evidence_urls" json:"return_evidence_urls"`
	ReturnRequestedAt  *time.Time  `db:"return_requested_at" json:"return_requested_at,omitempty"`
	ReturnSubmittedAt  *time.Time  `db:"return_submitted_at" json:"return_submitted_at,omitempty"`
	PriorOrderStatus *string `db:"prior_order_status" json:"prior_order_status,omitempty"`
	Status        string      `db:"status" json:"status"` // OP, RF (refunded), RJ (rejected)
	Resolution    *string     `db:"resolution" json:"resolution,omitempty"`
	RefundAmount  *decimal.Decimal `db:"refund_amount" json:"refund_amount,omitempty"`
	RefundSlipURL *string     `db:"refund_slip_url" json:"refund_slip_url,omitempty"`
	ResolvedBy    *int64      `db:"resolved_by" json:"resolved_by,omitempty"`
	ResolvedAt    *time.Time  `db:"resolved_at" json:"resolved_at,omitempty"`
	CreatedAt     time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time   `db:"updated_at" json:"updated_at"`
}

// QuotationTemplate represents a factory's reusable quotation template.
type QuotationTemplate struct {
	TemplateID       int64     `db:"template_id" json:"template_id"`
	FactoryID        int64     `db:"factory_id" json:"factory_id"`
	TemplateName     string    `db:"template_name" json:"template_name" validate:"notblank"`
	PricePerPiece    *decimal.Decimal `db:"price_per_piece" json:"price_per_piece,omitempty"`
	MoldCost         *decimal.Decimal `db:"mold_cost" json:"mold_cost,omitempty"`
	LeadTimeDays     *int      `db:"lead_time_days" json:"lead_time_days,omitempty"`
	ShippingMethodID *int64    `db:"shipping_method_id" json:"shipping_method_id,omitempty"`
	Note             *string   `db:"note" json:"note,omitempty"`
	IsActive         bool      `db:"is_active" json:"is_active"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

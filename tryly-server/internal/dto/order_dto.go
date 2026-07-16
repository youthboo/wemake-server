package dto

// Order Request DTOs
type CreateOrderFromQuoteRequest struct {
	QuotationID int64 `json:"quote_id" validate:"gt=0"`
}

type CreateOrderRequest struct {
	QuotationID int64  `json:"quotation_id" validate:"gt=0"`
	Quantity    int64  `json:"quantity" validate:"gt=0"`
	AddressID   int64  `json:"address_id" validate:"gt=0"`
	Notes       string `json:"notes"`
}

type BulkCheckoutRequest struct {
	OrderItems []BulkCheckoutItem `json:"order_items"`
	AddressID  int64              `json:"address_id" validate:"gt=0"`
}

type BulkCheckoutItem struct {
	QuotationID int64 `json:"quotation_id" validate:"gt=0"`
	Quantity    int64 `json:"quantity" validate:"gt=0"`
}

type BulkCheckoutItemInput struct {
	QuotationID int64  `json:"quotation_id"`
	AddressID   int64  `json:"address_id"`
	PaymentType string `json:"payment_type"`
}

type BulkCheckoutBodyRequest struct {
	Items          []BulkCheckoutItemInput `json:"items"`
	IdempotencyKey string                  `json:"idempotency_key"`
}

type ShipOrderRequest struct {
	TrackingNo string `json:"tracking_no" validate:"notblank"`
	Courier    string `json:"courier" validate:"notblank"`
}

type ConfirmReceiptRequest struct {
	Note       string  `json:"note"`
	ReceivedAt *string `json:"received_at"`
}

type CancelOrderRequest struct {
	Reason string `json:"reason" validate:"notblank"`
}

type MarkShippedRequest struct {
	TrackingNo string `json:"tracking_no"`
	Courier    string `json:"courier"`
}

type CreateDisputeRequest struct {
	Category    string   `json:"category" validate:"notblank"` // NR=ไม่ได้รับสินค้า, ND=ไม่ตรงปก, OT=อื่นๆ
	Description string   `json:"description" validate:"notblank"`
	ImageURLs   []string `json:"image_urls"` // รูปหลักฐาน (อัปโหลดผ่าน /media ก่อน)
	// บัญชีปลายทางสำหรับโอนเงินคืน + ช่องทางติดต่อกลับ
	RefundAccount     string `json:"refund_account" validate:"notblank"`      // เลขบัญชี / พร้อมเพย์
	RefundAccountName string `json:"refund_account_name" validate:"notblank"` // ชื่อบัญชี
	ContactEmail      string `json:"contact_email"`
	ContactPhone      string `json:"contact_phone" validate:"notblank"`
}

type PatchDisputeStatusRequest struct {
	Status   string  `json:"status" validate:"notblank"`
	Comments *string `json:"comments"`
}

// SubmitReturnRequest — ลูกค้าแนบหลักฐานการส่งสินค้าคืน (RT → RC)
type SubmitReturnRequest struct {
	TrackingNo string   `json:"tracking_no"`
	Courier    string   `json:"courier"`
	Note       string   `json:"note"`
	ImageURLs  []string `json:"image_urls"` // บิล/หลักฐานการจัดส่ง
}

// ResolveDisputeRequest — superadmin ตัดสิน ticket
type ResolveDisputeRequest struct {
	Action        string  `json:"action" validate:"notblank"` // "request_return" | "refund" | "reject"
	Resolution    string  `json:"resolution"`                 // หมายเหตุ
	RefundAmount  float64 `json:"refund_amount"`              // 0 = เต็มจำนวน; >0 = คืนบางส่วน
	RefundSlipURL string  `json:"refund_slip_url"`            // required เมื่อ refund
}

type VerifyPaymentRequest struct {
	ProofOfPayment string `json:"proof_of_payment" validate:"notblank"`
}

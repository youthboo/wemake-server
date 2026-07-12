package dto

// Wallet DTOs
type PayDepositRequest struct {
	Type           string  `json:"type"`
	Amount         float64 `json:"amount"`
	PaymentMethod  string  `json:"payment_method"`
	IdempotencyKey string  `json:"idempotency_key"`
}

type TopupIntentRequest struct {
	Amount   float64 `json:"amount" validate:"gt=0"`
	Currency string  `json:"currency"`
	Method   string  `json:"method" validate:"notblank"`
}

type ConfirmTopupIntentRequest struct {
	TransactionID  string `json:"transaction_id" validate:"notblank"`
	ProofOfPayment string `json:"proof_of_payment"`
}

type WithdrawalRequest struct {
	Amount           float64 `json:"amount" validate:"gt=0"`
	BankAccountID    *int64  `json:"bank_account_id"`
	BankName         *string `json:"bank_name"`
	AccountNumber    *string `json:"account_number"`
	AccountHolderName *string `json:"account_holder_name"`
}

type PatchWithdrawalStatusRequest struct {
	Status   string  `json:"status" validate:"notblank"`
	Comments *string `json:"comments"`
	// SlipURL — สลิปโอนเงินจาก superadmin (อัปโหลดผ่าน /media/upload ก่อน); บังคับเมื่อ status = CP
	SlipURL *string `json:"slip_url"`
}

// Payment Schedule DTOs
type CreatePaymentScheduleRequest struct {
	InstallmentNo int     `json:"installment_no"`
	DueDate       string  `json:"due_date"` // YYYY-MM-DD
	Amount        float64 `json:"amount"`
}

type PatchPaymentScheduleStatusRequest struct {
	Status string `json:"status"`
	Notes  *string `json:"notes"`
}

// Settlement DTOs
type CreateSettlementRequest struct {
	FactoryID int64  `json:"factory_id" validate:"gt=0"`
	Amount    float64 `json:"amount" validate:"gt=0"`
	Period    string  `json:"period"` // e.g., "2024-01", "2024-Q1"
	Notes     *string `json:"notes"`
}

type PatchSettlementStatusRequest struct {
	Status  string `json:"status" validate:"notblank"`
	ReferenceNo *string `json:"reference_no"`
	Notes   *string `json:"notes"`
}

// Transaction DTOs
type CreateTransactionRequest struct {
	Type        string   `json:"type" validate:"notblank"` // "payment", "refund", "topup", etc.
	Amount      float64  `json:"amount" validate:"gt=0"`
	Description string   `json:"description"`
	Reference   *string  `json:"reference"`
}

type PatchTransactionStatusRequest struct {
	Status string `json:"status" validate:"notblank"`
	Notes  *string `json:"notes"`
}

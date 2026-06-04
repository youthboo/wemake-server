package domain

const (
	OrderStatusWaitSlip            = "WS" // รอลูกค้าแนบสลีป
	OrderStatusWaitApprove         = "WA" // รอโรงงานตรวจสอบสลีป
	OrderStatusPaymentPending      = "PP"
	OrderStatusPaymentExpired      = "PE"
	OrderStatusPaymentDone         = "PD"
	OrderStatusProduction          = "PR"
	OrderStatusWaitingFinalPayment = "WF"
	OrderStatusQualityCheck        = "QC"
	OrderStatusShipping            = "SH"
	OrderStatusDelivered           = "DL"
	OrderStatusAccepted            = "AC"
	OrderStatusComplete            = "CP"
	OrderStatusCancelled           = "CN"
	OrderStatusCancelledByCustomer = "CC"
)

const (
	QuotationStatusPending  = "PE"
	QuotationStatusAccepted = "AC"
	QuotationStatusPrepared = "PD"
	QuotationStatusDeclined = "DC"
	QuotationStatusRejected = "RJ"
	QuotationStatusExpired  = "EX"
)

const (
	RFQStatusOpen      = "OP"
	RFQStatusInReview  = "IR"
	RFQStatusClosed    = "CL"
	RFQStatusDismissed = "DM"
)

const (
	PaymentTypeDeposit = "DP"
	PaymentTypeFull    = "FP"
)

const (
	PaymentScheduleStatusPending = "PE"
	PaymentScheduleStatusPaid    = "PD"
	PaymentScheduleStatusOverdue = "OD"
)

const (
	TransactionStatusSubmitted = "ST"
	TransactionStatusProcessed = "PT"
	TransactionStatusRejected  = "RJ"
)

const (
	SettlementStatusPending  = "PE"
	SettlementStatusApproved = "AP"
	SettlementStatusRejected = "RJ"
	SettlementStatusComplete = "CP"
)

const (
	TopupStatusPending    = "PE"
	TopupStatusProcessing = "PR"
	TopupStatusCompleted  = "CP"
)

const (
	WithdrawalStatusApproved = "AP"
	WithdrawalStatusRejected = "RJ"
	WithdrawalStatusComplete = "CP"
)

const (
	CatalogScopeProduct  = "PD"
	CatalogScopeMaterial = "MT"
	CatalogScopeAll      = "ALL"
)

const (
	DefaultQuotationValidityDays = 14
	DefaultQuotationTermsDays    = 30
	DefaultDepositScheduleDays   = 3
	DefaultBOQValidityDays       = 14
)

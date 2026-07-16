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
	OrderStatusDisputed            = "DP" // ลูกค้าเปิด ticket ร้องเรียน/ขอคืนเงิน
	OrderStatusRefunded            = "RF" // คืนเงินเต็มจำนวนสำเร็จ (terminal) — แยกจากยกเลิกธรรมดา
)

const (
	DisputeStatusOpen         = "OP" // รอ superadmin ตรวจสอบ
	DisputeStatusReturnWait   = "RT" // อนุมัติแล้ว รอลูกค้าส่งสินค้าคืน
	DisputeStatusReturnCheck  = "RC" // ลูกค้าส่งคืนแล้ว รอเจ้าหน้าที่ตรวจรับ
	DisputeStatusRefunded     = "RF" // คืนเงินแล้ว (เต็ม/บางส่วน)
	DisputeStatusRejected     = "RJ" // ปฏิเสธคำร้อง

	DisputeCategoryNotReceived = "NR" // ไม่ได้รับสินค้า
	DisputeCategoryNotAsDesc   = "ND" // สินค้าไม่ตรงปก
	DisputeCategoryOther       = "OT" // อื่นๆ
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

package status

import (
	"strings"

	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
)

func NormalizeCode(value string) string {
	return domainutil.NormalizeStatus(value)
}

func NormalizeOrder(value string) string {
	switch NormalizeCode(value) {
	case "CC":
		return "CN"
	default:
		return NormalizeCode(value)
	}
}

func IsValidOrder(value string) bool {
	switch NormalizeOrder(value) {
	case domain.OrderStatusWaitSlip,
		domain.OrderStatusWaitApprove,
		domain.OrderStatusPaymentPending,
		domain.OrderStatusPaymentExpired,
		domain.OrderStatusPaymentDone,
		domain.OrderStatusProduction,
		domain.OrderStatusWaitingFinalPayment,
		domain.OrderStatusQualityCheck,
		domain.OrderStatusShipping,
		domain.OrderStatusDelivered,
		domain.OrderStatusAccepted,
		domain.OrderStatusComplete,
		domain.OrderStatusRefunded,
		domain.OrderStatusCancelled:
		return true
	default:
		return false
	}
}

func OrderLabelTH(value string) string {
	switch NormalizeOrder(value) {
	case domain.OrderStatusWaitSlip:
		return "รอแนบสลีป"
	case domain.OrderStatusWaitApprove:
		return "รอยืนยันสลีป"
	case domain.OrderStatusPaymentPending:
		return "รอชำระเงิน"
	case domain.OrderStatusPaymentExpired:
		return "หมดกำหนดชำระ"
	case domain.OrderStatusPaymentDone:
		return "ชำระเงินแล้ว รอเริ่มผลิต"
	case domain.OrderStatusProduction:
		return "กำลังผลิต"
	case domain.OrderStatusQualityCheck:
		return "ตรวจสอบคุณภาพ"
	case domain.OrderStatusShipping:
		return "จัดส่งแล้ว"
	case domain.OrderStatusComplete:
		return "เสร็จสิ้น"
	case domain.OrderStatusRefunded:
		return "คืนเงินแล้ว"
	case domain.OrderStatusCancelled:
		return "ยกเลิก"
	default:
		return NormalizeOrder(value)
	}
}

func IsDepositPaidOrBeyondOrder(value string) bool {
	switch NormalizeOrder(value) {
	case domain.OrderStatusPaymentDone,
		domain.OrderStatusProduction,
		domain.OrderStatusQualityCheck,
		domain.OrderStatusShipping,
		domain.OrderStatusComplete:
		return true
	default:
		return false
	}
}

func IsDepositExpiredOrder(value string) bool {
	return NormalizeOrder(value) == domain.OrderStatusPaymentExpired
}

func IsCancellableOrder(value string) bool {
	switch NormalizeOrder(value) {
	case domain.OrderStatusWaitSlip,
		domain.OrderStatusWaitApprove,
		domain.OrderStatusPaymentExpired,
		domain.OrderStatusPaymentPending,
		domain.OrderStatusProduction,
		domain.OrderStatusWaitingFinalPayment:
		return true
	default:
		return false
	}
}

func IsValidQuotation(value string) bool {
	switch NormalizeCode(value) {
	case domain.QuotationStatusPending,
		domain.QuotationStatusAccepted,
		domain.QuotationStatusPrepared,
		domain.QuotationStatusDeclined,
		domain.QuotationStatusRejected,
		domain.QuotationStatusExpired:
		return true
	default:
		return false
	}
}

func IsValidRFQ(value string) bool {
	switch NormalizeCode(value) {
	case domain.RFQStatusOpen,
		domain.RFQStatusInReview,
		domain.RFQStatusClosed,
		domain.RFQStatusDismissed:
		return true
	default:
		return false
	}
}

func IsValidTransaction(value string) bool {
	switch NormalizeCode(value) {
	case domain.TransactionStatusSubmitted,
		domain.TransactionStatusProcessed,
		domain.TransactionStatusRejected:
		return true
	default:
		return false
	}
}

func IsValidSettlement(value string) bool {
	switch NormalizeCode(value) {
	case domain.SettlementStatusPending,
		domain.SettlementStatusApproved,
		domain.SettlementStatusRejected,
		domain.SettlementStatusComplete:
		return true
	default:
		return false
	}
}

func IsValidWithdrawal(value string) bool {
	switch NormalizeCode(value) {
	case domain.WithdrawalStatusApproved,
		domain.WithdrawalStatusRejected,
		domain.WithdrawalStatusComplete:
		return true
	default:
		return false
	}
}

func IsValidPaymentSchedulePatchStatus(value string) bool {
	switch NormalizeCode(value) {
	case domain.PaymentScheduleStatusPaid, domain.PaymentScheduleStatusOverdue:
		return true
	default:
		return false
	}
}

func IsPreProductionOrder(value string) bool {
	switch NormalizeOrder(value) {
	case "CF", domain.OrderStatusWaitSlip, domain.OrderStatusWaitApprove, domain.OrderStatusPaymentExpired, domain.OrderStatusPaymentPending:
		return true
	default:
		return false
	}
}

func IsProductionLockedOrder(value string) bool {
	switch NormalizeOrder(value) {
	case domain.OrderStatusWaitSlip,
		domain.OrderStatusWaitApprove,
		domain.OrderStatusPaymentPending,
		domain.OrderStatusPaymentExpired,
		domain.OrderStatusCancelled,
		domain.OrderStatusComplete:
		return true
	default:
		return false
	}
}

func IsProductionReadLockedOrder(value string) bool {
	switch NormalizeOrder(value) {
	case domain.OrderStatusWaitSlip,
		domain.OrderStatusWaitApprove,
		domain.OrderStatusPaymentPending,
		domain.OrderStatusPaymentExpired,
		domain.OrderStatusCancelled:
		return true
	default:
		return false
	}
}

func ProductionLockReason(value string) string {
	switch NormalizeOrder(value) {
	case domain.OrderStatusWaitSlip:
		return "WAITING_SLIP"
	case domain.OrderStatusWaitApprove:
		return "WAITING_SLIP_APPROVAL"
	case domain.OrderStatusPaymentPending:
		return "PENDING_DEPOSIT"
	case domain.OrderStatusPaymentExpired:
		return "DEPOSIT_EXPIRED"
	case domain.OrderStatusCancelled:
		return "ORDER_CANCELLED"
	default:
		return "UNKNOWN"
	}
}

func FrontendRFQ(value string, offerCount int64) string {
	switch NormalizeCode(value) {
	case "CC":
		return "cancelled"
	case "CL":
		return "completed"
	case "OP":
		if offerCount > 0 {
			return "offers_received"
		}
		return "pending"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func FrontendOrder(value string) string {
	switch NormalizeOrder(value) {
	case domain.OrderStatusWaitSlip:
		return "waiting_slip"
	case domain.OrderStatusWaitApprove:
		return "waiting_approval"
	case domain.OrderStatusPaymentPending:
		return "pending_payment"
	case domain.OrderStatusPaymentDone, domain.OrderStatusProduction, domain.OrderStatusQualityCheck, domain.OrderStatusWaitingFinalPayment:
		return "in_production"
	case domain.OrderStatusShipping:
		return "shipped"
	case domain.OrderStatusComplete:
		return "completed"
	case domain.OrderStatusRefunded:
		return "refunded"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func FrontendQuotation(value string) string {
	switch NormalizeCode(value) {
	case domain.QuotationStatusPrepared:
		return "pending"
	case domain.QuotationStatusAccepted:
		return "accepted"
	case domain.QuotationStatusRejected:
		return "rejected"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

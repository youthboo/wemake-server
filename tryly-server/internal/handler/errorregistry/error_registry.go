package errorregistry

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/dbutil"
	"github.com/yourusername/wemake/internal/helper"
	factoryrepo "github.com/yourusername/wemake/internal/repository/factory"
	boqservice "github.com/yourusername/wemake/internal/service/boq"
	messageservice "github.com/yourusername/wemake/internal/service/message"
	orderservice "github.com/yourusername/wemake/internal/service/order"
	platformservice "github.com/yourusername/wemake/internal/service/platform_config"
	quotationservice "github.com/yourusername/wemake/internal/service/quotation"
)

func CreateOrderErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		orderservice.ErrQuotationRejected:          helper.ErrorMessage(fiber.StatusBadRequest, "quotation was rejected"),
		orderservice.ErrQuotationInvalidState:      helper.ErrorMessage(fiber.StatusBadRequest, "quotation has invalid state"),
		orderservice.ErrInsufficientGoodFund:       helper.ErrorMessage(fiber.StatusBadRequest, "insufficient good fund balance"),
		orderservice.ErrOrderAlreadyExistsForQuote: helper.ErrorMessage(fiber.StatusConflict, "order already exists for this quotation"),
		orderservice.ErrInvalidOrderTotal:          helper.ErrorMessage(fiber.StatusBadRequest, "order total must be non-negative"),
	}
}

func BulkCheckoutErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		orderservice.ErrRFQLocked:                  helper.ErrorMessage(fiber.StatusLocked, "rfq is locked"),
		orderservice.ErrQuotationInvalidState:      helper.ErrorMessage(fiber.StatusConflict, "quotation has invalid state"),
		orderservice.ErrSelfTransaction:            helper.ErrorMessage(fiber.StatusBadRequest, "cannot transact with self"),
		orderservice.ErrInvalidQuotationSet:        helper.ErrorMessage(fiber.StatusBadRequest, "invalid quotation set"),
		orderservice.ErrPaymentTypeInvalid:         helper.ErrorMessage(fiber.StatusBadRequest, "invalid payment type"),
		orderservice.ErrNotQuotationParty:          helper.ErrorMessage(fiber.StatusForbidden, "not authorized for this quotation"),
		orderservice.ErrOrderAlreadyExistsForQuote: helper.ErrorMessage(fiber.StatusConflict, "order already exists for this quotation"),
		orderservice.ErrInvalidOrderTotal:          helper.ErrorMessage(fiber.StatusBadRequest, "order total must be non-negative"),
		orderservice.ErrNoOrdersCreated:            helper.ErrorMessage(fiber.StatusBadRequest, "no valid orders created from quotations"),
	}
}

func CreatePaymentErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		orderservice.ErrDepositAlreadyPaid:    helper.ErrorBody(fiber.StatusUnprocessableEntity, fiber.Map{"error": fiber.Map{"code": "DEPOSIT_ALREADY_PAID", "message": "deposit already recorded"}}),
		orderservice.ErrDepositExpired:        helper.ErrorBody(fiber.StatusGone, fiber.Map{"error": fiber.Map{"code": "DEPOSIT_EXPIRED", "message": "deposit payment window has expired"}}),
		orderservice.ErrPaymentTypeInvalid:    helper.ErrorMessage(fiber.StatusBadRequest, "invalid payment type"),
		orderservice.ErrPaymentAmountMismatch: helper.ErrorMessage(fiber.StatusBadRequest, "payment amount does not match"),
		orderservice.ErrPaymentAlreadyExists:  helper.ErrorMessage(fiber.StatusBadRequest, "payment already exists"),
	}
}

func VerifyPaymentErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		orderservice.ErrDepositAlreadyPaid:    helper.ErrorBody(fiber.StatusUnprocessableEntity, fiber.Map{"error": fiber.Map{"code": "DEPOSIT_ALREADY_PAID", "message": "deposit already recorded"}}),
		orderservice.ErrDepositExpired:        helper.ErrorBody(fiber.StatusGone, fiber.Map{"error": fiber.Map{"code": "DEPOSIT_EXPIRED", "message": "deposit payment window has expired"}}),
		orderservice.ErrPaymentTypeInvalid:    helper.ErrorMessage(fiber.StatusBadRequest, "invalid payment type"),
		orderservice.ErrPaymentAmountMismatch: helper.ErrorMessage(fiber.StatusBadRequest, "payment amount does not match"),
		orderservice.ErrPaymentStateInvalid:   helper.ErrorMessage(fiber.StatusBadRequest, "invalid payment state"),
		orderservice.ErrInsufficientGoodFund:  helper.ErrorMessage(fiber.StatusBadRequest, "insufficient good fund balance"),
	}
}

func ConfirmReceiptErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		orderservice.ErrConfirmReceiptInvalidStatus: helper.ErrorMessage(fiber.StatusConflict, "order cannot be marked as received at this stage"),
		orderservice.ErrConfirmReceiptNotAllowed:    helper.ErrorMessage(fiber.StatusUnprocessableEntity, "confirm receipt not allowed for this order"),
	}
}

func PatchProfileErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{}
}

func AddCategoryErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		factoryrepo.ErrDuplicateFactoryCategory: helper.ErrorMessage(fiber.StatusConflict, "category already linked to this factory"),
		factoryrepo.ErrInvalidFactoryCategory:   helper.ErrorMessage(fiber.StatusBadRequest, "invalid category_id"),
	}
}

func ReplaceCategoriesErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		factoryrepo.ErrInvalidFactoryCategory: helper.ErrorMessage(fiber.StatusBadRequest, "invalid category_id"),
	}
}

func AddSubCategoryErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		factoryrepo.ErrDuplicateFactorySubCategory: helper.ErrorMessage(fiber.StatusConflict, "sub-category already linked"),
		factoryrepo.ErrInvalidFactorySubCategory:   helper.ErrorMessage(fiber.StatusBadRequest, "invalid sub_category_id"),
	}
}

func ReplaceSubCategoriesErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		factoryrepo.ErrInvalidFactorySubCategory: helper.ErrorMessage(fiber.StatusBadRequest, "invalid sub_category_id"),
	}
}

func CreateMessageErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		messageservice.ErrInvalidReferenceType:         helper.ErrorMessage(fiber.StatusBadRequest, "reference_type must be one of RQ, OD, PD, PM, ID"),
		messageservice.ErrReferencePairRequired:        helper.ErrorMessage(fiber.StatusBadRequest, "reference_type and reference_id must be provided together"),
		messageservice.ErrReferenceNotFound:            helper.ErrorMessage(fiber.StatusBadRequest, "reference_id not found for the given reference_type"),
		messageservice.ErrSenderReceiverSame:           helper.ErrorMessage(fiber.StatusBadRequest, "receiver_id must differ from sender_id"),
		messageservice.ErrConversationNotAccessible:    helper.ErrorMessage(fiber.StatusBadRequest, "conv_id must be a conversation the sender belongs to"),
		messageservice.ErrConversationReceiverMismatch: helper.ErrorMessage(fiber.StatusBadRequest, "receiver_id must match the other participant in conv_id"),
		messageservice.ErrInvalidMessageType:           helper.ErrorMessage(fiber.StatusBadRequest, "message_type must be one of TX, QT, IM, BQ"),
		messageservice.ErrQuoteDataRequired:            helper.ErrorMessage(fiber.StatusBadRequest, "quote_data is required when message_type is QT"),
	}
}

func ListByReferenceErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		messageservice.ErrInvalidReferenceType: helper.ErrorMessage(fiber.StatusBadRequest, "reference_type must be one of RQ, OD, PD, PM, ID"),
	}
}

func BOQErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		boqservice.ErrBOQNotFound:       helper.ErrorMessage(fiber.StatusNotFound, "BOQ_NOT_FOUND"),
		boqservice.ErrBOQForbidden:      helper.ErrorMessage(fiber.StatusForbidden, "BOQ_FORBIDDEN"),
		boqservice.ErrBOQInvalidItems:   helper.ErrorMessage(fiber.StatusBadRequest, "BOQ_INVALID_ITEMS"),
		boqservice.ErrBOQExpired:        helper.ErrorMessage(fiber.StatusUnprocessableEntity, "BOQ_EXPIRED"),
		boqservice.ErrBOQAlreadyHandled: helper.ErrorMessage(fiber.StatusConflict, "BOQ_ALREADY_HANDLED"),
		boqservice.ErrBOQInvalidState:   helper.ErrorMessage(fiber.StatusConflict, "BOQ_INVALID_STATE"),
	}
}

func CreateConfigErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		platformservice.ErrPlatformConfigValidation: helper.ErrorMessage(fiber.StatusBadRequest, "invalid platform config payload"),
	}
}

func UpdateConfigErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		platformservice.ErrPlatformConfigValidation: helper.ErrorMessage(fiber.StatusBadRequest, "invalid platform config payload"),
		platformservice.ErrPlatformConfigNotFound:   helper.ErrorMessage(fiber.StatusNotFound, "platform config not found"),
	}
}

func DeleteConfigErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		platformservice.ErrPlatformConfigValidation: helper.ErrorMessage(fiber.StatusBadRequest, "failed to delete platform config"),
		platformservice.ErrPlatformDefaultDelete:    helper.ErrorMessage(fiber.StatusBadRequest, "failed to delete platform config"),
		platformservice.ErrPlatformConfigNotFound:   helper.ErrorMessage(fiber.StatusNotFound, "platform config not found"),
	}
}

func CreateQuotationErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		quotationservice.ErrFactorySuspended:        helper.ErrorMessage(fiber.StatusForbidden, "factory account is suspended"),
		quotationservice.ErrFactoryHighlightInvalid: helper.ErrorMessage(fiber.StatusBadRequest, "HIGHLIGHT_TOO_LONG"),
		quotationservice.ErrPaymentTermsInvalid:     helper.ErrorMessage(fiber.StatusBadRequest, "invalid payment terms"),
		quotationservice.ErrInvalidShippingMethod:   helper.ErrorMessage(fiber.StatusBadRequest, "invalid shipping method"),
		quotationservice.ErrInvalidLineItem:         helper.ErrorMessage(fiber.StatusBadRequest, "invalid line item"),
	}
}

func GetQuotationErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{}
}

func ListHistoryErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{}
}

func PatchQuotationErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		quotationservice.ErrQuotationLocked:         helper.ErrorMessage(fiber.StatusConflict, "LOCKED"),
		quotationservice.ErrNotQuotationParty:       helper.ErrorMessage(fiber.StatusForbidden, "not authorized to modify this quotation"),
		quotationservice.ErrFactoryHighlightInvalid: helper.ErrorMessage(fiber.StatusBadRequest, "HIGHLIGHT_TOO_LONG"),
		quotationservice.ErrInvalidShippingMethod:   helper.ErrorMessage(fiber.StatusBadRequest, "invalid shipping method"),
	}
}

func CreateDetailedErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{
		quotationservice.ErrFactorySuspended:        helper.ErrorMessage(fiber.StatusForbidden, "factory account is suspended"),
		quotationservice.ErrFactoryHighlightInvalid: helper.ErrorMessage(fiber.StatusBadRequest, "HIGHLIGHT_TOO_LONG"),
	}
}

func CanViewErrorMap() map[error]helper.ErrorResponse {
	return map[error]helper.ErrorResponse{}
}

func IsNotFoundError(err error) bool {
	return dbutil.IsNotFoundError(err)
}

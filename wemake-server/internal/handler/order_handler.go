package handler

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type OrderHandler struct {
	service *service.OrderService
	auth    *service.AuthService
}

func NewOrderHandler(orderService *service.OrderService, authService *service.AuthService) *OrderHandler {
	return &OrderHandler{service: orderService, auth: authService}
}

func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	type reqBody struct {
		QuotationID int64 `json:"quote_id"`
	}
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if req.QuotationID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "quote_id is required"})
	}
	order, err := h.service.CreateFromQuotation(req.QuotationID, userID)
	if err != nil {
		if errors.Is(err, service.ErrQuotationRejected) || errors.Is(err, service.ErrQuotationInvalidState) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrInsufficientGoodFund) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrOrderAlreadyExistsForQuote) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":  "failed to create order",
			"detail": err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(order)
}

func (h *OrderHandler) BulkCheckout(c *fiber.Ctx) error {
	type reqBody struct {
		Items          []service.BulkCheckoutItemInput `json:"items"`
		IdempotencyKey string                          `json:"idempotency_key"`
	}
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	rfqID, err := c.ParamsInt("rfq_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	result, err := h.service.BulkCheckout(service.BulkCheckoutInput{
		RFQID:          int64(rfqID),
		UserID:         userID,
		Items:          req.Items,
		IdempotencyKey: req.IdempotencyKey,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRFQLocked):
			return c.Status(fiber.StatusLocked).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrQuotationInvalidState):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "QUOTATION_NOT_PENDING"})
		case errors.Is(err, service.ErrSelfTransaction):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrInvalidQuotationSet), errors.Is(err, service.ErrPaymentTypeInvalid):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrNotQuotationParty):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrOrderAlreadyExistsForQuote):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to bulk checkout"})
		}
	}
	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *OrderHandler) ListOrders(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	if factoryParam := strings.TrimSpace(c.Query("factory_id")); factoryParam != "" {
		if u.Role != domain.RoleFactory {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "factory role required"})
		}
		if !strings.EqualFold(factoryParam, "me") {
			factoryID, parseErr := strconv.ParseInt(factoryParam, 10, 64)
			if parseErr != nil || factoryID <= 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
			}
			if factoryID != userID {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "factory_id must match authenticated factory"})
			}
		}
	}
	status := strings.TrimSpace(c.Query("status"))
	var rfqID *int64
	if raw := strings.TrimSpace(c.Query("rfq_id")); raw != "" {
		parsed, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || parsed <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
		}
		rfqID = &parsed
	}
	items, err := h.service.List(userID, u.Role, status, rfqID, c.Query("request_kind"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch orders"})
	}
	return c.JSON(items)
}

func (h *OrderHandler) GetOrder(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	detail, err := h.service.GetDetailByID(int64(orderID), userID, u.Role)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch order"})
	}
	return c.JSON(detail)
}

func (h *OrderHandler) ListActivity(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	if _, err := h.service.GetByID(int64(orderID), userID, u.Role); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		}
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to verify order"})
	}
	items, err := h.service.ListActivity(int64(orderID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch activity"})
	}
	return c.JSON(items)
}

func (h *OrderHandler) PatchOrderStatus(c *fiber.Ctx) error {
	type reqBody struct {
		Status string `json:"status"`
	}
	uid, authErr := getUserIDFromHeader(c)
	var actor *int64
	if authErr == nil {
		actor = &uid
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	status := strings.TrimSpace(strings.ToUpper(req.Status))
	validOrderStatuses := map[string]struct{}{
		"PP": {}, "PR": {}, "WF": {}, "QC": {}, "SH": {}, "DL": {}, "AC": {}, "CP": {}, "CC": {},
	}
	if _, ok := validOrderStatuses[status]; !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be PP, PR, WF, QC, SH, DL, AC, CP, or CC"})
	}
	if err := h.service.UpdateStatus(int64(orderID), status, actor); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update order status"})
	}
	return c.JSON(fiber.Map{"message": "order status updated"})
}

func (h *OrderHandler) CancelOrder(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	if err := h.service.Cancel(int64(orderID), userID, u.Role); err != nil {
		if errors.Is(err, service.ErrOrderCannotBeCancelled) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to cancel order"})
	}
	return c.JSON(fiber.Map{"message": "order cancelled"})
}

func (h *OrderHandler) MarkShipped(c *fiber.Ctx) error {
	type reqBody struct {
		TrackingNo string `json:"tracking_no"`
		Courier    string `json:"courier"`
	}
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	if u.Role != domain.RoleFactory {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "factory role required"})
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if err := h.service.MarkShipped(int64(orderID), userID, req.TrackingNo, req.Courier); err != nil {
		if errors.Is(err, service.ErrShipOrderInvalid) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to mark order as shipped"})
	}
	item, err := h.service.GetByID(int64(orderID), userID, u.Role)
	if err != nil {
		return c.JSON(fiber.Map{"message": "order marked as shipped"})
	}
	return c.JSON(item)
}

func (h *OrderHandler) CreatePayment(c *fiber.Ctx) error {
	type reqBody struct {
		Type   string  `json:"type"`
		Amount float64 `json:"amount"`
	}
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil || orderID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	item, err := h.service.CreatePayment(int64(orderID), userID, u.Role, req.Type, req.Amount)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDepositAlreadyPaid):
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": fiber.Map{"code": "DEPOSIT_ALREADY_PAID", "message": "deposit already recorded"},
			})
		case errors.Is(err, service.ErrDepositExpired):
			return c.Status(fiber.StatusGone).JSON(fiber.Map{
				"error": fiber.Map{"code": "DEPOSIT_EXPIRED", "message": "deposit payment window has expired"},
			})
		case errors.Is(err, service.ErrPaymentTypeInvalid),
			errors.Is(err, service.ErrPaymentAmountMismatch),
			errors.Is(err, service.ErrPaymentAlreadyExists):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		case repository.IsNotFoundError(err):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create payment"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *OrderHandler) VerifyPayment(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil || orderID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	txID := strings.TrimSpace(c.Params("tx_id"))
	if txID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tx_id"})
	}
	item, err := h.service.VerifyPayment(int64(orderID), userID, u.Role, txID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDepositAlreadyPaid):
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
				"error": fiber.Map{"code": "DEPOSIT_ALREADY_PAID", "message": "deposit already recorded"},
			})
		case errors.Is(err, service.ErrDepositExpired):
			return c.Status(fiber.StatusGone).JSON(fiber.Map{
				"error": fiber.Map{"code": "DEPOSIT_EXPIRED", "message": "deposit payment window has expired"},
			})
		case errors.Is(err, service.ErrPaymentTypeInvalid),
			errors.Is(err, service.ErrPaymentAmountMismatch),
			errors.Is(err, service.ErrPaymentStateInvalid),
			errors.Is(err, service.ErrInsufficientGoodFund):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		case repository.IsNotFoundError(err):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "payment not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to verify payment"})
	}
	return c.JSON(item)
}

func (h *OrderHandler) ConfirmReceipt(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil || orderID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	type reqBody struct {
		Note       string  `json:"note"`
		ReceivedAt *string `json:"received_at"`
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	var receivedAt *time.Time
	if req.ReceivedAt != nil && strings.TrimSpace(*req.ReceivedAt) != "" {
		t, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(*req.ReceivedAt))
		if parseErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "received_at must be RFC3339 datetime"})
		}
		receivedAt = &t
	}

	result, err := h.service.ConfirmReceipt(int64(orderID), userID, u.Role, service.ConfirmReceiptInput{
		Note:       req.Note,
		ReceivedAt: receivedAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrForbidden):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		case errors.Is(err, service.ErrConfirmReceiptInvalidStatus):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrConfirmReceiptNotAllowed):
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		case repository.IsNotFoundError(err):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to confirm receipt"})
		}
	}
	return c.JSON(result)
}

func (h *OrderHandler) GetReviewState(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil || orderID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	item, err := h.service.GetReviewState(int64(orderID), userID, u.Role)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrForbidden):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		case repository.IsNotFoundError(err):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch review state"})
		}
	}
	return c.JSON(item)
}

func (h *OrderHandler) CreateReview(c *fiber.Ctx) error {
	type reqBody struct {
		Rating    int                `json:"rating"`
		Comment   string             `json:"comment"`
		ImageURLs domain.StringArray `json:"image_urls"`
	}
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil || orderID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	item, err := h.service.CreateReview(int64(orderID), userID, u.Role, service.CreateOrderReviewInput{
		Rating:    req.Rating,
		Comment:   req.Comment,
		ImageURLs: req.ImageURLs,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrForbidden):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
		case errors.Is(err, service.ErrReviewRatingInvalid), errors.Is(err, service.ErrReviewCommentInvalid), errors.Is(err, service.ErrReviewImagesInvalid):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrReviewOrderNotCompleted):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, service.ErrReviewAlreadyExists):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		case repository.IsNotFoundError(err):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create review"})
		}
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

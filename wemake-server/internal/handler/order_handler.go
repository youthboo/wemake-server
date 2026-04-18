package handler

import (
	"errors"
	"strconv"
	"strings"

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
		if err == service.ErrQuotationNotAccepted {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrInsufficientGoodFund) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrOrderAlreadyExistsForQuote) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create order"})
	}
	return c.Status(fiber.StatusCreated).JSON(order)
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
	items, err := h.service.List(userID, u.Role, status)
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
	item, err := h.service.GetByID(int64(orderID), userID, u.Role)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch order"})
	}
	return c.JSON(item)
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

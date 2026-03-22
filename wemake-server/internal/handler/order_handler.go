package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type OrderHandler struct {
	service *service.OrderService
}

func NewOrderHandler(service *service.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create order"})
	}
	return c.Status(fiber.StatusCreated).JSON(order)
}

func (h *OrderHandler) ListOrders(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	status := strings.TrimSpace(c.Query("status"))
	items, err := h.service.ListByUserID(userID, status)
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
	orderID, err := c.ParamsInt("order_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	item, err := h.service.GetByID(int64(orderID), userID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch order"})
	}
	return c.JSON(item)
}

func (h *OrderHandler) PatchOrderStatus(c *fiber.Ctx) error {
	type reqBody struct {
		Status string `json:"status"`
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
	if status != "PR" && status != "QC" && status != "SH" && status != "CP" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be PR, QC, SH or CP"})
	}
	if err := h.service.UpdateStatus(int64(orderID), status); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update order status"})
	}
	return c.JSON(fiber.Map{"message": "order status updated"})
}

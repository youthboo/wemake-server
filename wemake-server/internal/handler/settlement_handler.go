package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type SettlementHandler struct {
	service *service.SettlementService
}

func NewSettlementHandler(svc *service.SettlementService) *SettlementHandler {
	return &SettlementHandler{service: svc}
}

// GET /settlements
func (h *SettlementHandler) List(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	items, err := h.service.ListByFactoryID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch settlements"})
	}
	return c.JSON(items)
}

// GET /settlements/:settlement_id
func (h *SettlementHandler) GetByID(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	settlementID, err := strconv.ParseInt(c.Params("settlement_id"), 10, 64)
	if err != nil || settlementID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid settlement_id"})
	}
	item, err := h.service.GetByID(settlementID, userID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "settlement not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch settlement"})
	}
	return c.JSON(item)
}

// POST /settlements — create a settlement record (factory or system initiated)
func (h *SettlementHandler) Create(c *fiber.Ctx) error {
	type reqBody struct {
		OrderID *int64  `json:"order_id"`
		Amount  float64 `json:"amount"`
		Note    *string `json:"note"`
	}
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "amount must be greater than 0"})
	}
	item, err := h.service.Create(userID, req.OrderID, req.Amount, req.Note)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create settlement"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

// PATCH /settlements/:settlement_id/status
func (h *SettlementHandler) PatchStatus(c *fiber.Ctx) error {
	type reqBody struct {
		Status string `json:"status"`
	}
	settlementID, err := strconv.ParseInt(c.Params("settlement_id"), 10, 64)
	if err != nil || settlementID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid settlement_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status != "PR" && status != "CP" && status != "FL" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be PR, CP, or FL"})
	}
	if err := h.service.UpdateStatus(settlementID, status); err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "settlement not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update settlement status"})
	}
	return c.JSON(fiber.Map{"message": "settlement status updated"})
}

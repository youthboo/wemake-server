package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type TransactionHandler struct {
	service *service.TransactionService
}

func NewTransactionHandler(service *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{service: service}
}

func (h *TransactionHandler) CreateTransaction(c *fiber.Ctx) error {
	type reqBody struct {
		WalletID int64   `json:"wallet_id"`
		OrderID  *int64  `json:"order_id"`
		Type     string  `json:"type"`
		Amount   float64 `json:"amount"`
		Status   string  `json:"status"`
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if req.WalletID <= 0 || strings.TrimSpace(req.Type) == "" || strings.TrimSpace(req.Status) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "wallet_id, type, status are required"})
	}

	item := &domain.Transaction{
		WalletID: req.WalletID,
		OrderID:  req.OrderID,
		Type:     req.Type,
		Amount:   req.Amount,
		Status:   req.Status,
	}
	if err := h.service.Create(item); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create transaction"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *TransactionHandler) ListTransactions(c *fiber.Ctx) error {
	filters := repository.TransactionFilters{}

	if raw := c.Query("wallet_id"); raw != "" {
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid wallet_id"})
		}
		filters.WalletID = &val
	}
	if raw := c.Query("order_id"); raw != "" {
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
		}
		filters.OrderID = &val
	}
	if raw := c.Query("type"); raw != "" {
		filters.Type = &raw
	}
	if raw := c.Query("status"); raw != "" {
		filters.Status = &raw
	}

	items, err := h.service.List(filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch transactions"})
	}
	return c.JSON(items)
}

func (h *TransactionHandler) PatchTransactionStatus(c *fiber.Ctx) error {
	type reqBody struct {
		Status string `json:"status"`
	}
	txID := c.Params("tx_id")
	if strings.TrimSpace(txID) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid tx_id"})
	}

	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	status := strings.TrimSpace(strings.ToUpper(req.Status))
	if status != "ST" && status != "PT" && status != "RJ" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be ST, PT or RJ"})
	}

	if err := h.service.PatchStatus(txID, status); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update transaction status"})
	}
	return c.JSON(fiber.Map{"message": "transaction status updated"})
}

package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type WalletHandler struct {
	service *service.WalletService
}

func NewWalletHandler(service *service.WalletService) *WalletHandler {
	return &WalletHandler{service: service}
}

func (h *WalletHandler) GetMyWallet(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	wallet, err := h.service.GetByUserID(userID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "wallet not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch wallet"})
	}
	return c.JSON(wallet)
}

func (h *WalletHandler) ListMyTransactions(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	var orderID *int64
	if raw := c.Query("order_id"); raw != "" {
		val, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
		}
		orderID = &val
	}
	var txType *string
	if raw := c.Query("type"); raw != "" {
		txType = &raw
	}
	var status *string
	if raw := c.Query("status"); raw != "" {
		status = &raw
	}
	items, err := h.service.ListTransactionsByUserID(userID, orderID, txType, status)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "wallet not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch transactions"})
	}
	return c.JSON(items)
}

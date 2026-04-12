package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type WithdrawalHandler struct {
	service *service.WithdrawalService
}

func NewWithdrawalHandler(svc *service.WithdrawalService) *WithdrawalHandler {
	return &WithdrawalHandler{service: svc}
}

// POST /wallets/withdraw
func (h *WithdrawalHandler) Create(c *fiber.Ctx) error {
	type reqBody struct {
		Amount        float64 `json:"amount"`
		BankAccountNo string  `json:"bank_account_no"`
		BankName      string  `json:"bank_name"`
		AccountName   string  `json:"account_name"`
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
	if strings.TrimSpace(req.BankAccountNo) == "" || strings.TrimSpace(req.BankName) == "" || strings.TrimSpace(req.AccountName) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "bank_account_no, bank_name, and account_name are required"})
	}
	item, err := h.service.Create(userID, req.Amount, req.BankAccountNo, req.BankName, req.AccountName)
	if err != nil {
		if errors.Is(err, service.ErrInsufficientFunds) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "wallet not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create withdrawal request"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

// GET /wallets/withdraw
func (h *WithdrawalHandler) List(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	items, err := h.service.ListByFactoryID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch withdrawal requests"})
	}
	return c.JSON(items)
}

// PATCH /wallets/withdraw/:request_id/status
func (h *WithdrawalHandler) PatchStatus(c *fiber.Ctx) error {
	type reqBody struct {
		Status string  `json:"status"`
		Note   *string `json:"note"`
	}
	requestID, err := strconv.ParseInt(c.Params("request_id"), 10, 64)
	if err != nil || requestID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if err := h.service.UpdateStatus(requestID, req.Status, req.Note); err != nil {
		if errors.Is(err, service.ErrInvalidWithdrawalStatus) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "withdrawal request not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update withdrawal status"})
	}
	return c.JSON(fiber.Map{"message": "withdrawal status updated"})
}

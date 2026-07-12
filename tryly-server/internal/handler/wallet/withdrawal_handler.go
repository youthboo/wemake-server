package wallet

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/dto"
	"github.com/yourusername/wemake/internal/helper"
	walletrepo "github.com/yourusername/wemake/internal/repository/wallet"
	walletservice "github.com/yourusername/wemake/internal/service/wallet"
)

type WithdrawalHandler struct {
	service *walletservice.WithdrawalService
}

func NewWithdrawalHandler(svc *walletservice.WithdrawalService) *WithdrawalHandler {
	return &WithdrawalHandler{service: svc}
}

// POST /wallets/withdraw
func (h *WithdrawalHandler) Create(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	var req dto.WithdrawalRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	bankAccountNo := helper.DereferenceString(req.AccountNumber, "")
	bankName := helper.DereferenceString(req.BankName, "")
	accountName := helper.DereferenceString(req.AccountHolderName, "")
	item, err := h.service.Create(userID, req.Amount, bankAccountNo, bankName, accountName)
	if err != nil {
		return helper.MapServiceError(c, err, withdrawalCreateFallback, withdrawalCreateResponses)
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

// GET /wallets/withdraw
func (h *WithdrawalHandler) List(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	items, err := h.service.ListByFactoryID(userID)
	if err != nil {
		return helper.InternalServerError(c, "failed to fetch withdrawal requests")
	}
	return c.JSON(items)
}

// PATCH /wallets/withdraw/:request_id/status
func (h *WithdrawalHandler) PatchStatus(c *fiber.Ctx) error {
	requestID, err := helper.ParsePositiveInt64Param(c, "request_id")
	if err != nil {
		return helper.BadRequestError(c, "invalid request_id")
	}
	var req dto.PatchWithdrawalStatusRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	var processedBy *int64
	if actorID := helper.OptionalActorID(c); actorID > 0 {
		processedBy = &actorID
	}
	if err := h.service.UpdateStatus(requestID, req.Status, req.Comments, req.SlipURL, processedBy); err != nil {
		return helper.MapServiceError(c, err, withdrawalPatchStatusFallback, withdrawalPatchStatusResponses)
	}
	return c.JSON(fiber.Map{"message": "withdrawal status updated"})
}

var withdrawalCreateFallback = helper.ErrorMessage(fiber.StatusInternalServerError, "failed to create withdrawal request")

var withdrawalCreateResponses = map[error]helper.ErrorResponse{
	walletservice.ErrInsufficientFunds: helper.ErrorMessage(fiber.StatusBadRequest, walletservice.ErrInsufficientFunds.Error()),
	sql.ErrNoRows:                      helper.ErrorMessage(fiber.StatusNotFound, "wallet not found"),
}

var withdrawalPatchStatusFallback = helper.ErrorMessage(fiber.StatusInternalServerError, "failed to update withdrawal status")

var withdrawalPatchStatusResponses = map[error]helper.ErrorResponse{
	walletservice.ErrInvalidWithdrawalStatus:      helper.ErrorMessage(fiber.StatusBadRequest, walletservice.ErrInvalidWithdrawalStatus.Error()),
	walletservice.ErrSlipRequiredForComplete:      helper.ErrorMessage(fiber.StatusBadRequest, walletservice.ErrSlipRequiredForComplete.Error()),
	walletrepo.ErrWithdrawalAlreadyProcessed:      helper.ErrorMessage(fiber.StatusBadRequest, "คำขอถอนเงินนี้ถูกดำเนินการแล้ว"),
	sql.ErrNoRows:                                 helper.ErrorMessage(fiber.StatusNotFound, "withdrawal request not found"),
}

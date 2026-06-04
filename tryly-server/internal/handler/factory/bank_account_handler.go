package factory

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/helper"
	factoryrepo "github.com/yourusername/wemake/internal/repository/factory"
)

type BankAccountHandler struct {
	repo *factoryrepo.BankAccountRepository
}

func NewBankAccountHandler(repo *factoryrepo.BankAccountRepository) *BankAccountHandler {
	return &BankAccountHandler{repo: repo}
}

// List GET /api/factories/me/bank-accounts
func (h *BankAccountHandler) List(c *fiber.Ctx) error {
	factoryID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	items, err := h.repo.ListByFactory(factoryID)
	if err != nil {
		return helper.JSONInternal(c, "failed to list bank accounts")
	}
	return c.JSON(fiber.Map{"accounts": items})
}

// Create POST /api/factories/me/bank-accounts
func (h *BankAccountHandler) Create(c *fiber.Ctx) error {
	factoryID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}

	var body struct {
		BankName      string `json:"bank_name"`
		AccountNumber string `json:"account_number"`
		AccountName   string `json:"account_name"`
		IsDefault     *bool  `json:"is_default"`
	}
	if err := c.BodyParser(&body); err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	body.BankName = strings.TrimSpace(body.BankName)
	body.AccountNumber = strings.TrimSpace(body.AccountNumber)
	body.AccountName = strings.TrimSpace(body.AccountName)

	if body.BankName == "" || body.AccountNumber == "" || body.AccountName == "" {
		return helper.JSONError(c, fiber.StatusBadRequest, "bank_name, account_number, account_name are required")
	}

	// If first account, force default
	isDefault := true
	if body.IsDefault != nil {
		isDefault = *body.IsDefault
	}
	count, _ := h.repo.CountByFactory(factoryID)
	if count == 0 {
		isDefault = true
	}

	item, err := h.repo.Create(factoryID, body.BankName, body.AccountNumber, body.AccountName, isDefault)
	if err != nil {
		return helper.JSONInternal(c, "failed to create bank account")
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

// Update PATCH /api/factories/me/bank-accounts/:account_id
func (h *BankAccountHandler) Update(c *fiber.Ctx) error {
	factoryID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	accountID, err := helper.ParsePositiveInt64Param(c, "account_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid account_id")
	}

	// Verify ownership
	existing, err := h.repo.GetByID(accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusNotFound, "bank account not found")
		}
		return helper.JSONInternal(c, "failed to fetch bank account")
	}
	if existing.FactoryID != factoryID {
		return helper.JSONError(c, fiber.StatusForbidden, "forbidden")
	}

	var body struct {
		BankName      *string `json:"bank_name"`
		AccountNumber *string `json:"account_number"`
		AccountName   *string `json:"account_name"`
		IsDefault     *bool   `json:"is_default"`
	}
	if err := c.BodyParser(&body); err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if body.BankName != nil {
		v := strings.TrimSpace(*body.BankName)
		body.BankName = &v
	}
	if body.AccountNumber != nil {
		v := strings.TrimSpace(*body.AccountNumber)
		body.AccountNumber = &v
	}
	if body.AccountName != nil {
		v := strings.TrimSpace(*body.AccountName)
		body.AccountName = &v
	}

	item, err := h.repo.Update(accountID, factoryID, body.BankName, body.AccountNumber, body.AccountName, body.IsDefault)
	if err != nil {
		return helper.JSONInternal(c, "failed to update bank account")
	}
	return c.JSON(item)
}

// Delete DELETE /api/factories/me/bank-accounts/:account_id
func (h *BankAccountHandler) Delete(c *fiber.Ctx) error {
	factoryID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	accountID, err := helper.ParsePositiveInt64Param(c, "account_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid account_id")
	}

	existing, err := h.repo.GetByID(accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusNotFound, "bank account not found")
		}
		return helper.JSONInternal(c, "failed to fetch bank account")
	}
	if existing.FactoryID != factoryID {
		return helper.JSONError(c, fiber.StatusForbidden, "forbidden")
	}

	// If deleting default and there are other accounts, block
	if existing.IsDefault {
		count, _ := h.repo.CountByFactory(factoryID)
		if count > 1 {
			return helper.JSONError(c, fiber.StatusBadRequest, "ไม่สามารถลบบัญชีหลักได้ กรุณาตั้งบัญชีอื่นเป็นหลักก่อน")
		}
	}

	if err := h.repo.Delete(accountID); err != nil {
		return helper.JSONInternal(c, "failed to delete bank account")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// GetPublicDefault GET /api/factories/:factory_id/bank-account (public for customer/admin)
func (h *BankAccountHandler) GetPublicDefault(c *fiber.Ctx) error {
	factoryID, err := helper.ParsePositiveInt64Param(c, "factory_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid factory_id")
	}

	item, err := h.repo.GetDefault(factoryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusNotFound, "โรงงานยังไม่ตั้งค่าบัญชีธนาคาร")
		}
		return helper.JSONInternal(c, "failed to fetch bank account")
	}
	return c.JSON(item)
}

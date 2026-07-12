package tconfig

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/helper"
	tconfigrepo "github.com/yourusername/wemake/internal/repository/tconfig"
)

type TConfigHandler struct {
	repo *tconfigrepo.TConfigRepository
}

func NewTConfigHandler(repo *tconfigrepo.TConfigRepository) *TConfigHandler {
	return &TConfigHandler{repo: repo}
}

// GetAll returns all tconfig key-value pairs as a flat map.
// GET /admin/tconfig
func (h *TConfigHandler) GetAll(c *fiber.Ctx) error {
	items, err := h.repo.GetAll()
	if err != nil {
		return helper.WriteAPIError(c, helper.InternalServerAPIError("FETCH_TCONFIG_FAILED", "failed to fetch config"))
	}
	result := make(map[string]string, len(items))
	for _, item := range items {
		result[item.Key] = item.Value
	}
	return c.JSON(fiber.Map{"configs": result})
}

// publicConfigKeys — whitelist ของ key ที่เปิดเผยได้โดยไม่ต้อง auth
// (frontend ใช้ branch payment flow + แสดงบัญชี Tryly ใน escrow mode)
var publicConfigKeys = []string{
	"config_payment",
	"tryly_bank_name",
	"tryly_bank_account_no",
	"tryly_account_holder",
	"tryly_promptpay",
}

// GetPublic returns only whitelisted config keys.
// GET /api/v1/configs/public (no auth required)
func (h *TConfigHandler) GetPublic(c *fiber.Ctx) error {
	items, err := h.repo.GetAll()
	if err != nil {
		return helper.WriteAPIError(c, helper.InternalServerAPIError("FETCH_TCONFIG_FAILED", "failed to fetch config"))
	}
	allowed := make(map[string]bool, len(publicConfigKeys))
	for _, k := range publicConfigKeys {
		allowed[k] = true
	}
	result := make(map[string]string)
	for _, item := range items {
		if allowed[item.Key] {
			result[item.Key] = item.Value
		}
	}
	return c.JSON(fiber.Map{"configs": result})
}

// PatchBulk upserts one or more key-value pairs.
// PATCH /admin/tconfig
// Body: { "platform_name": "...", "rfq_expired": "30" }
func (h *TConfigHandler) PatchBulk(c *fiber.Ctx) error {
	var body map[string]string
	if err := c.BodyParser(&body); err != nil || len(body) == 0 {
		return helper.WriteAPIError(c, helper.BadRequestAPIError("INVALID_BODY", "request body must be a non-empty key-value map"))
	}
	if err := h.repo.SetBulk(body); err != nil {
		return helper.WriteAPIError(c, helper.InternalServerAPIError("UPDATE_TCONFIG_FAILED", "failed to update config"))
	}
	return helper.WriteSuccess(c, "config updated", nil)
}

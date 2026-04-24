package handler

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/service"
)

type PlatformConfigHandler struct {
	service *service.PlatformConfigService
	auth    *service.AuthService
}

func NewPlatformConfigHandler(service *service.PlatformConfigService, auth *service.AuthService) *PlatformConfigHandler {
	return &PlatformConfigHandler{service: service, auth: auth}
}

func (h *PlatformConfigHandler) requireAdmin(c *fiber.Ctx) (int64, error) {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return 0, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	role := strings.ToUpper(strings.TrimSpace(u.Role))
	if role != "AD" && role != "ADMIN" {
		return 0, fiber.NewError(fiber.StatusForbidden, "admin role required")
	}
	return userID, nil
}

func (h *PlatformConfigHandler) GetActive(c *fiber.Ctx) error {
	if _, err := h.requireAdmin(c); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	item, err := h.service.GetActive()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch platform config"})
	}
	return c.JSON(item)
}

func (h *PlatformConfigHandler) ListHistory(c *fiber.Ctx) error {
	if _, err := h.requireAdmin(c); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	items, err := h.service.ListHistory()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch platform config history"})
	}
	return c.JSON(items)
}

func (h *PlatformConfigHandler) Create(c *fiber.Ctx) error {
	type reqBody struct {
		DefaultCommissionRate float64  `json:"default_commission_rate"`
		PromoCommissionRate   *float64 `json:"promo_commission_rate"`
		PromoStartAt          *string  `json:"promo_start_at"`
		PromoEndAt            *string  `json:"promo_end_at"`
		PromoLabel            *string  `json:"promo_label"`
		VatRate               float64  `json:"vat_rate"`
		CurrencyCode          string   `json:"currency_code"`
	}
	userID, err := h.requireAdmin(c)
	if err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	var promoStartAt *time.Time
	if req.PromoStartAt != nil && strings.TrimSpace(*req.PromoStartAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.PromoStartAt))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "promo_start_at must be RFC3339"})
		}
		promoStartAt = &parsed
	}
	var promoEndAt *time.Time
	if req.PromoEndAt != nil && strings.TrimSpace(*req.PromoEndAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.PromoEndAt))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "promo_end_at must be RFC3339"})
		}
		promoEndAt = &parsed
	}
	cfg := &domain.PlatformConfig{
		DefaultCommissionRate: req.DefaultCommissionRate,
		PromoCommissionRate:   req.PromoCommissionRate,
		PromoStartAt:          promoStartAt,
		PromoEndAt:            promoEndAt,
		PromoLabel:            req.PromoLabel,
		VatRate:               req.VatRate,
		CurrencyCode:          req.CurrencyCode,
		CreatedBy:             &userID,
	}
	if err := h.service.CreateVersion(cfg); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create platform config"})
	}
	return c.Status(fiber.StatusCreated).JSON(cfg)
}

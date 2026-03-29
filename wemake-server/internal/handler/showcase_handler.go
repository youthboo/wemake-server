package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/service"
)

type ShowcaseHandler struct {
	service *service.ShowcaseService
}

func NewShowcaseHandler(service *service.ShowcaseService) *ShowcaseHandler {
	return &ShowcaseHandler{service: service}
}

func (h *ShowcaseHandler) List(c *fiber.Ctx) error {
	contentType := c.Query("type", "")
	items, err := h.service.ListAll(contentType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch showcases"})
	}
	return c.JSON(items)
}

func (h *ShowcaseHandler) Create(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var req domain.FactoryShowcase
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	req.FactoryID = userID
	if err := h.service.Create(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create showcase"})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

func (h *ShowcaseHandler) ListPromoSlides(c *fiber.Ctx) error {
	items, err := h.service.ListPromoSlides()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch promo slides"})
	}
	return c.JSON(items)
}

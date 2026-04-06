package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/service"
)

// Allowed values for GET /showcases?type= (matches factory_showcases.content_type)
var showcaseTypeQueryAllowed = map[string]struct{}{
	"PD": {}, "PM": {}, "ID": {},
}

type ShowcaseHandler struct {
	service *service.ShowcaseService
}

func NewShowcaseHandler(service *service.ShowcaseService) *ShowcaseHandler {
	return &ShowcaseHandler{service: service}
}

func (h *ShowcaseHandler) List(c *fiber.Ctx) error {
	contentType := strings.TrimSpace(c.Query("type", ""))
	if contentType != "" {
		if _, ok := showcaseTypeQueryAllowed[contentType]; !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid query type: use PD (product), PM (promotion), or ID (idea); omit type for all showcases",
			})
		}
	}
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

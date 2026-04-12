package handler

import (
	"database/sql"
	"errors"
	"strconv"
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
	items, err := h.service.ListExplore(contentType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch showcases"})
	}
	return c.JSON(items)
}

func (h *ShowcaseHandler) ListByFactory(c *fiber.Ctx) error {
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil || factoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
	}
	contentType := strings.TrimSpace(c.Query("type", ""))
	if contentType != "" {
		if _, ok := showcaseTypeQueryAllowed[contentType]; !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid query type: use PD, PM, or ID; omit type for all showcases for this factory",
			})
		}
	}
	items, err := h.service.ListExploreByFactory(int64(factoryID), contentType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch showcases"})
	}
	return c.JSON(items)
}

func (h *ShowcaseHandler) Patch(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid showcase_id"})
	}
	var req domain.FactoryShowcase
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	req.ShowcaseID = showcaseID
	req.FactoryID = userID
	if err := h.service.Update(&req); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "showcase not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update showcase"})
	}
	out, err := h.service.GetByID(showcaseID, userID)
	if err != nil {
		return c.JSON(req)
	}
	return c.JSON(out)
}

func (h *ShowcaseHandler) Delete(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid showcase_id"})
	}
	if err := h.service.Delete(showcaseID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "showcase not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete showcase"})
	}
	return c.SendStatus(fiber.StatusNoContent)
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

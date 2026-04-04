package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/service"
)

type CatalogHandler struct {
	service *service.CatalogService
}

func NewCatalogHandler(service *service.CatalogService) *CatalogHandler {
	return &CatalogHandler{service: service}
}

func (h *CatalogHandler) GetCategories(c *fiber.Ctx) error {
	items, err := h.service.GetCategories()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch categories"})
	}
	return c.JSON(items)
}

func (h *CatalogHandler) GetSubCategories(c *fiber.Ctx) error {
	categoryID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || categoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid category id"})
	}

	items, err := h.service.GetSubCategories(categoryID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch sub-categories"})
	}
	return c.JSON(items)
}

func (h *CatalogHandler) GetUnits(c *fiber.Ctx) error {
	items, err := h.service.GetUnits()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch units"})
	}
	return c.JSON(items)
}

package handler

import (
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

func (h *CatalogHandler) GetUnits(c *fiber.Ctx) error {
	items, err := h.service.GetUnits()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch units"})
	}
	return c.JSON(items)
}

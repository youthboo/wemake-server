package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/service"
)

type ProductionHandler struct {
	service *service.ProductionService
}

func NewProductionHandler(service *service.ProductionService) *ProductionHandler {
	return &ProductionHandler{service: service}
}

func (h *ProductionHandler) CreateUpdate(c *fiber.Ctx) error {
	type reqBody struct {
		StepID      int64  `json:"step_id"`
		Description string `json:"description"`
		ImageURL    string `json:"image_url"`
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if req.StepID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "step_id is required"})
	}
	item := &domain.ProductionUpdate{
		OrderID:     int64(orderID),
		StepID:      req.StepID,
		Description: req.Description,
		ImageURL:    req.ImageURL,
	}
	if err := h.service.Create(item); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create production update"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *ProductionHandler) ListUpdates(c *fiber.Ctx) error {
	orderID, err := c.ParamsInt("order_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	items, err := h.service.ListByOrderID(int64(orderID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch production updates"})
	}
	return c.JSON(items)
}

func (h *ProductionHandler) PatchUpdate(c *fiber.Ctx) error {
	type reqBody struct {
		Description *string `json:"description"`
		ImageURL    *string `json:"image_url"`
	}
	updateID, err := c.ParamsInt("update_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid update_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if err := h.service.Patch(int64(updateID), req.Description, req.ImageURL); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to patch production update"})
	}
	return c.JSON(fiber.Map{"message": "production update updated"})
}

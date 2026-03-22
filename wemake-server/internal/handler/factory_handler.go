package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/service"
)

type FactoryHandler struct {
	service *service.FactoryService
}

func NewFactoryHandler(service *service.FactoryService) *FactoryHandler {
	return &FactoryHandler{service: service}
}

func (h *FactoryHandler) CreateFactory(c *fiber.Ctx) error {
	type CreateFactoryRequest struct {
		Name        string `json:"name" validate:"required"`
		Email       string `json:"email" validate:"required,email"`
		Phone       string `json:"phone" validate:"required"`
		Address     string `json:"address" validate:"required"`
		Description string `json:"description"`
	}

	var req CreateFactoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	factory, err := h.service.CreateFactory(req.Name, req.Email, req.Phone, req.Address, req.Description)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create factory"})
	}

	return c.Status(fiber.StatusCreated).JSON(factory)
}

func (h *FactoryHandler) GetFactory(c *fiber.Ctx) error {
	id := c.Params("id")
	factory, err := h.service.GetFactoryByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Factory not found"})
	}
	return c.JSON(factory)
}

func (h *FactoryHandler) GetAllFactories(c *fiber.Ctx) error {
	factories, err := h.service.GetAllFactories()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch factories"})
	}
	return c.JSON(factories)
}

func (h *FactoryHandler) UpdateFactory(c *fiber.Ctx) error {
	type UpdateFactoryRequest struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		Address     string `json:"address"`
		Description string `json:"description"`
	}

	id := c.Params("id")
	var req UpdateFactoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request"})
	}

	factory, err := h.service.UpdateFactory(id, req.Name, req.Email, req.Phone, req.Address, req.Description)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update factory"})
	}

	return c.JSON(factory)
}

func (h *FactoryHandler) DeleteFactory(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.service.DeleteFactory(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete factory"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

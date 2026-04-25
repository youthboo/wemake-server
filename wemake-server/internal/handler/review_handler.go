package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/service"
)

type ReviewHandler struct {
	service *service.ReviewService
}

func NewReviewHandler(service *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{service: service}
}

func (h *ReviewHandler) ListByFactory(c *fiber.Ctx) error {
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
	}
	items, err := h.service.ListByFactoryID(int64(factoryID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to sort reviews"})
	}
	return c.JSON(items)
}

func (h *ReviewHandler) GetSummaryByFactory(c *fiber.Ctx) error {
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
	}
	item, err := h.service.GetSummaryByFactoryID(int64(factoryID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to summarize reviews"})
	}
	return c.JSON(item)
}

func (h *ReviewHandler) Create(c *fiber.Ctx) error {
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
	}
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req domain.FactoryReview
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	req.FactoryID = int64(factoryID)
	req.UserID = userID

	if err := h.service.Create(&req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create review"})
	}
	return c.Status(fiber.StatusCreated).JSON(req)
}

package handler

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type FrontendHandler struct {
	service *service.FrontendService
}

func NewFrontendHandler(service *service.FrontendService) *FrontendHandler {
	return &FrontendHandler{service: service}
}

func (h *FrontendHandler) GetBootstrap(c *fiber.Ctx) error {
	userID := getOptionalUserIDFromHeader(c)
	log.Printf("[DEBUG] GetBootstrap handler: userID=%d", userID)

	item, err := h.service.GetBootstrap(userID)
	if err != nil {
		log.Printf("[ERROR] GetBootstrap service failed: %v (type: %T)", err, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch bootstrap data",
			"debug": fmt.Sprintf("%v", err),
		})
	}

	log.Printf("[SUCCESS] GetBootstrap returned: currentUser=%v, categories=%d, factories=%d",
		item.CurrentUser != nil, len(item.Categories), len(item.Factories))
	return c.JSON(item)
}

func (h *FrontendHandler) GetCurrentUser(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user context"})
	}

	item, err := h.service.GetCurrentUser(userID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch current user"})
	}

	return c.JSON(item)
}

func (h *FrontendHandler) ListFactories(c *fiber.Ctx) error {
	items, err := h.service.ListFactories()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch factories"})
	}
	return c.JSON(items)
}

func (h *FrontendHandler) GetFactoryDetail(c *fiber.Ctx) error {
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
	}

	item, err := h.service.GetFactoryDetail(int64(factoryID))
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "factory not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch factory detail"})
	}
	return c.JSON(item)
}

func (h *FrontendHandler) GetRFQDetail(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user context"})
	}

	rfqID, err := c.ParamsInt("rfq_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
	}

	item, err := h.service.GetRFQDetail(userID, int64(rfqID))
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "rfq not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch rfq detail"})
	}
	return c.JSON(item)
}

func (h *FrontendHandler) GetOrderDetail(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user context"})
	}

	orderID, err := c.ParamsInt("order_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}

	item, err := h.service.GetOrderDetail(userID, int64(orderID))
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch order detail"})
	}
	return c.JSON(item)
}

func (h *FrontendHandler) ListThreads(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user context"})
	}

	items, err := h.service.ListThreads(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch message threads"})
	}
	return c.JSON(items)
}

func (h *FrontendHandler) GetMockData(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user context"})
	}

	item, err := h.service.GetMockData(userID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch frontend mock data"})
	}
	return c.JSON(item)
}

func (h *FrontendHandler) GetProducts(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 8)
	categoryID := c.Query("category_id", "")

	items, err := h.service.GetProducts(limit, categoryID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch products"})
	}
	return c.JSON(items)
}

func (h *FrontendHandler) GetPromotions(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 4)

	items, err := h.service.GetPromotions(limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch promotions"})
	}
	return c.JSON(items)
}

func (h *FrontendHandler) GetPromoCodes(c *fiber.Ctx) error {
	items, err := h.service.GetPromoCodes()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch promo codes"})
	}
	return c.JSON(items)
}

func (h *FrontendHandler) GetExplore(c *fiber.Ctx) error {
	userID := getOptionalUserIDFromHeader(c)
	item, err := h.service.GetExploreData(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch explore data"})
	}
	return c.JSON(item)
}

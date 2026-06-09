package frontend

import (
	"fmt"
	"github.com/yourusername/wemake/internal/helper"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/logger"
	frontendservice "github.com/yourusername/wemake/internal/service/frontend"
)

type FrontendHandler struct {
	service *frontendservice.FrontendService
}

func NewFrontendHandler(service *frontendservice.FrontendService) *FrontendHandler {
	return &FrontendHandler{service: service}
}

func (h *FrontendHandler) GetBootstrap(c *fiber.Ctx) error {
	userID := helper.OptionalActorID(c)
	logger.Debug("frontend bootstrap requested", "user_id", userID)

	item, err := h.service.GetBootstrap(userID)
	if err != nil {
		logger.Error("frontend bootstrap failed", "user_id", userID, "err", err, "err_type", fmt.Sprintf("%T", err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to fetch bootstrap data",
			"debug": fmt.Sprintf("%v", err),
		})
	}

	return c.JSON(item)
}

func (h *FrontendHandler) GetCurrentUser(c *fiber.Ctx) error {
	userID, err := requireFrontendUserID(c)
	if err != nil {
		return err
	}

	item, err := h.service.GetCurrentUser(userID)
	if err != nil {
		return helper.WriteServiceError(c, err, "failed to fetch current user", helper.NotFoundCase(helper.ErrNotFound, "user not found"))
	}

	return c.JSON(item)
}

func (h *FrontendHandler) ListFactories(c *fiber.Ctx) error {
	scope := helper.QueryString(c, "scope") // optional: "PD" or "MT"
	var items []domain.FrontendFactoryCard
	var err error
	if scope != "" {
		items, err = h.service.ListFactories(scope)
	} else {
		items, err = h.service.ListFactories()
	}
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch factories")
	}
	return c.JSON(items)
}

func (h *FrontendHandler) GetFactoryDetail(c *fiber.Ctx) error {
	factoryID, err := helper.RequireInt64Param(c, "factory_id")
	if err != nil {
		return err
	}

	item, err := h.service.GetFactoryDetail(int64(factoryID))
	if err != nil {
		return helper.WriteServiceError(c, err, "failed to fetch factory detail", helper.NotFoundCase(helper.ErrNotFound, "factory not found"))
	}
	return c.JSON(item)
}

func (h *FrontendHandler) GetRFQDetail(c *fiber.Ctx) error {
	userID, err := requireFrontendUserID(c)
	if err != nil {
		return err
	}

	rfqID, err := helper.RequireInt64Param(c, "rfq_id")
	if err != nil {
		return err
	}

	item, err := h.service.GetRFQDetail(userID, int64(rfqID))
	if err != nil {
		return helper.WriteServiceError(c, err, "failed to fetch rfq detail", helper.NotFoundCase(helper.ErrNotFound, "rfq not found"))
	}
	return c.JSON(item)
}

func (h *FrontendHandler) GetOrderDetail(c *fiber.Ctx) error {
	userID, err := requireFrontendUserID(c)
	if err != nil {
		return err
	}

	orderID, err := helper.RequireInt64Param(c, "order_id")
	if err != nil {
		return err
	}

	item, err := h.service.GetOrderDetail(userID, int64(orderID))
	if err != nil {
		return helper.WriteServiceError(c, err, "failed to fetch order detail", helper.NotFoundCase(helper.ErrNotFound, "order not found"))
	}
	return c.JSON(item)
}

func (h *FrontendHandler) ListThreads(c *fiber.Ctx) error {
	userID, err := requireFrontendUserID(c)
	if err != nil {
		return err
	}

	items, err := h.service.ListThreads(userID)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch message threads")
	}
	return c.JSON(items)
}

func (h *FrontendHandler) GetMockData(c *fiber.Ctx) error {
	userID, err := requireFrontendUserID(c)
	if err != nil {
		return err
	}

	item, err := h.service.GetMockData(userID)
	if err != nil {
		return helper.WriteServiceError(c, err, "failed to fetch frontend mock data", helper.NotFoundCase(helper.ErrNotFound, "user not found"))
	}
	return c.JSON(item)
}

func (h *FrontendHandler) GetProducts(c *fiber.Ctx) error {
	query := helper.QueryParams(c)
	limit := query.Int("limit", 8)
	categoryID := query.String("category_id")

	items, err := h.service.GetProducts(limit, categoryID)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch products")
	}
	return c.JSON(items)
}

func (h *FrontendHandler) GetPromotions(c *fiber.Ctx) error {
	limit := helper.QueryParams(c).Int("limit", 4)

	items, err := h.service.GetPromotions(limit)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch promotions")
	}
	return c.JSON(items)
}

func (h *FrontendHandler) GetPromoCodes(c *fiber.Ctx) error {
	items, err := h.service.GetPromoCodes()
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch promo codes")
	}
	return c.JSON(items)
}

func (h *FrontendHandler) GetExplore(c *fiber.Ctx) error {
	userID := helper.OptionalActorID(c)
	item, err := h.service.GetExploreData(userID)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch explore data")
	}
	return c.JSON(item)
}

func requireFrontendUserID(c *fiber.Ctx) (int64, error) {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return 0, err
	}
	return userID, nil
}

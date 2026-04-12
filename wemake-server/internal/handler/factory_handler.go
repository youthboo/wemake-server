package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

const errMsgInvalidFactoryID = "invalid factory_id"

type FactoryHandler struct {
	service *service.FactoryService
}

func NewFactoryHandler(service *service.FactoryService) *FactoryHandler {
	return &FactoryHandler{service: service}
}

func (h *FactoryHandler) List(c *fiber.Ctx) error {
	items, err := h.service.ListPublic()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch factories"})
	}
	return c.JSON(items)
}

func (h *FactoryHandler) GetByID(c *fiber.Ctx) error {
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil || factoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errMsgInvalidFactoryID})
	}
	item, err := h.service.GetPublicDetail(int64(factoryID))
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "factory not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch factory"})
	}
	return c.JSON(item)
}

func (h *FactoryHandler) ListCategories(c *fiber.Ctx) error {
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil || factoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errMsgInvalidFactoryID})
	}
	ok, err := h.service.FactoryExistsActive(int64(factoryID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to verify factory"})
	}
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "factory not found"})
	}
	items, err := h.service.ListFactoryCategories(int64(factoryID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch categories"})
	}
	return c.JSON(items)
}

type addFactoryCategoryBody struct {
	CategoryID int64 `json:"category_id"`
}

func (h *FactoryHandler) AddCategory(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil || factoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errMsgInvalidFactoryID})
	}
	if int64(factoryID) != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	var body addFactoryCategoryBody
	if err := c.BodyParser(&body); err != nil || body.CategoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body must include category_id (positive integer)"})
	}
	err = h.service.AddFactoryCategory(int64(factoryID), body.CategoryID)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateFactoryCategory) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "category already linked to this factory"})
		}
		if errors.Is(err, repository.ErrInvalidFactoryCategory) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid category_id"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add category"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"factory_id":  factoryID,
		"category_id": body.CategoryID,
	})
}

func (h *FactoryHandler) RemoveCategory(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil || factoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errMsgInvalidFactoryID})
	}
	if int64(factoryID) != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	categoryID, err := strconv.ParseInt(c.Params("category_id"), 10, 64)
	if err != nil || categoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid category_id"})
	}
	err = h.service.RemoveFactoryCategory(int64(factoryID), categoryID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "mapping not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to remove category"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

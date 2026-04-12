package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

const errMsgInvalidFactoryID = "invalid factory_id"

type FactoryHandler struct {
	service *service.FactoryService
	auth    *service.AuthService
}

func NewFactoryHandler(service *service.FactoryService, authService *service.AuthService) *FactoryHandler {
	return &FactoryHandler{service: service, auth: authService}
}

func (h *FactoryHandler) GetMe(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user context"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	if u.Role != domain.RoleFactory {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "factory role required"})
	}
	item, err := h.service.GetPublicDetail(userID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "factory profile not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch factory"})
	}
	return c.JSON(item)
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

func (h *FactoryHandler) ListSubCategories(c *fiber.Ctx) error {
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
	items, err := h.service.ListFactorySubCategories(int64(factoryID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch sub-categories"})
	}
	return c.JSON(items)
}

type addFactorySubCategoryBody struct {
	SubCategoryID int64 `json:"sub_category_id"`
}

func (h *FactoryHandler) AddSubCategory(c *fiber.Ctx) error {
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
	var body addFactorySubCategoryBody
	if err := c.BodyParser(&body); err != nil || body.SubCategoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body must include sub_category_id (positive integer)"})
	}
	err = h.service.AddFactorySubCategory(int64(factoryID), body.SubCategoryID)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateFactorySubCategory) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "sub-category already linked"})
		}
		if errors.Is(err, repository.ErrInvalidFactorySubCategory) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sub_category_id"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add sub-category"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"factory_id":      factoryID,
		"sub_category_id": body.SubCategoryID,
	})
}

func (h *FactoryHandler) RemoveSubCategory(c *fiber.Ctx) error {
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
	subID, err := strconv.ParseInt(c.Params("sub_category_id"), 10, 64)
	if err != nil || subID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sub_category_id"})
	}
	err = h.service.RemoveFactorySubCategory(int64(factoryID), subID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "mapping not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to remove sub-category"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

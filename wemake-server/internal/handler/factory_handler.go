package handler

import (
	"errors"
	"strconv"
	"strings"

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

type replaceFactoryCategoriesBody struct {
	CategoryIDs []int64 `json:"category_ids"`
}

type replaceFactorySubCategoriesBody struct {
	SubCategoryIDs []int64 `json:"sub_category_ids"`
}

func validatePositiveUniqueIDs(ids []int64) ([]int64, bool) {
	if len(ids) == 0 {
		return nil, false
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, false
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, true
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

func (h *FactoryHandler) ReplaceCategories(c *fiber.Ctx) error {
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
	var body replaceFactoryCategoriesBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	categoryIDs, ok := validatePositiveUniqueIDs(body.CategoryIDs)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "body must include category_ids with at least one positive integer"})
	}
	if err := h.service.ReplaceFactoryCategories(int64(factoryID), categoryIDs); err != nil {
		if errors.Is(err, repository.ErrInvalidFactoryCategory) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid category_id"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to replace categories"})
	}
	items, err := h.service.ListFactoryCategories(int64(factoryID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "categories updated but failed to fetch latest data"})
	}
	return c.JSON(fiber.Map{
		"factory_id": factoryID,
		"categories": items,
	})
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

func (h *FactoryHandler) ReplaceSubCategories(c *fiber.Ctx) error {
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
	var body replaceFactorySubCategoriesBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}

	subCategoryIDs := make([]int64, 0, len(body.SubCategoryIDs))
	if len(body.SubCategoryIDs) > 0 {
		var ok bool
		subCategoryIDs, ok = validatePositiveUniqueIDs(body.SubCategoryIDs)
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "sub_category_ids must contain only positive integers"})
		}
	}

	if err := h.service.ReplaceFactorySubCategories(int64(factoryID), subCategoryIDs); err != nil {
		if errors.Is(err, repository.ErrInvalidFactorySubCategory) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sub_category_id"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to replace sub-categories"})
	}
	items, err := h.service.ListFactorySubCategories(int64(factoryID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "sub-categories updated but failed to fetch latest data"})
	}
	return c.JSON(fiber.Map{
		"factory_id":     factoryID,
		"sub_categories": items,
	})
}

func (h *FactoryHandler) GetDashboard(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user context"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	if strings.TrimSpace(strings.ToUpper(u.Role)) != domain.RoleFactory {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "factory role required"})
	}
	item, err := h.service.GetDashboard(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch dashboard"})
	}
	return c.JSON(item)
}

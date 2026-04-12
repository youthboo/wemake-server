package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/service"
)

type MasterHandler struct {
	service *service.MasterService
}

func NewMasterHandler(service *service.MasterService) *MasterHandler {
	return &MasterHandler{service: service}
}

func (h *MasterHandler) GetProvinces(c *fiber.Ctx) error {
	items, err := h.service.GetProvinces()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch provinces"})
	}
	return c.JSON(items)
}

func (h *MasterHandler) GetDistricts(c *fiber.Ctx) error {
	var provinceID *int64
	if raw := c.Query("province_id"); raw != "" {
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid province_id"})
		}
		provinceID = &val
	}
	items, err := h.service.GetDistricts(provinceID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch districts"})
	}
	return c.JSON(items)
}

func (h *MasterHandler) GetSubDistricts(c *fiber.Ctx) error {
	var districtID *int64
	if raw := c.Query("district_id"); raw != "" {
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid district_id"})
		}
		districtID = &val
	}
	items, err := h.service.GetSubDistricts(districtID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch sub districts"})
	}
	return c.JSON(items)
}

func (h *MasterHandler) GetFactoryTypes(c *fiber.Ctx) error {
	items, err := h.service.GetFactoryTypes()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch factory types"})
	}
	return c.JSON(items)
}

func (h *MasterHandler) GetProductCategories(c *fiber.Ctx) error {
	var parentID *int64
	if raw := c.Query("parent_category_id"); raw != "" {
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid parent_category_id"})
		}
		parentID = &val
	}
	items, err := h.service.GetProductCategories(parentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch product categories"})
	}
	return c.JSON(items)
}

// GetCategories is an alias for product master list (same payload as GET /master/product-categories).
func (h *MasterHandler) GetCategories(c *fiber.Ctx) error {
	items, err := h.service.GetProductCategories(nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch categories"})
	}
	return c.JSON(items)
}

func (h *MasterHandler) GetProductionSteps(c *fiber.Ctx) error {
	var factoryTypeID *int64
	if raw := c.Query("factory_type_id"); raw != "" {
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_type_id"})
		}
		factoryTypeID = &val
	}
	items, err := h.service.GetProductionSteps(factoryTypeID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch production steps"})
	}
	return c.JSON(items)
}

func (h *MasterHandler) GetUnits(c *fiber.Ctx) error {
	items, err := h.service.GetUnits()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch master units"})
	}
	return c.JSON(items)
}

func (h *MasterHandler) GetShippingMethods(c *fiber.Ctx) error {
	items, err := h.service.GetShippingMethods()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch shipping methods"})
	}
	return c.JSON(items)
}

func (h *MasterHandler) GetCertificates(c *fiber.Ctx) error {
	items, err := h.service.GetCertificates()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch certificates"})
	}
	return c.JSON(items)
}

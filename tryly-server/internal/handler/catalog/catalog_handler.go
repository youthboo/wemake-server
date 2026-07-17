package catalog

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/helper"
	catalogservice "github.com/yourusername/wemake/internal/service/catalog"
)

type CatalogHandler struct {
	service *catalogservice.CatalogService
}

func NewCatalogHandler(service *catalogservice.CatalogService) *CatalogHandler {
	return &CatalogHandler{service: service}
}

func (h *CatalogHandler) GetCategories(c *fiber.Ctx) error {
	scope := helper.QueryString(c, "scope")
	limit := helper.QueryParams(c).Int("limit", 0)

	if helper.QueryString(c, "include_sub") == "true" {
		items, err := h.service.GetCategoriesWithSubs(scope, limit)
		if err != nil {
			return helper.JSONInternal(c, "failed to fetch categories")
		}
		return c.JSON(items)
	}

	if limit > 0 && scope == "" {
		scope = domain.CatalogScopeAll
	}
	items, err := h.service.GetCategories(scope, limit)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch categories")
	}
	return c.JSON(items)
}

func (h *CatalogHandler) GetLBICategories(c *fiber.Ctx) error {
	// Support filtering by hub_id (new) or scope (legacy)
	if hubIDStr := helper.QueryString(c, "hub_id"); hubIDStr != "" {
		hubID, err := helper.ParsePositiveInt64Param(c, "hub_id")
		if err != nil || hubID <= 0 {
			// fallback: parse from query string directly
			var id int64
			if _, scanErr := fmt.Sscanf(hubIDStr, "%d", &id); scanErr != nil || id <= 0 {
				return helper.BadRequestError(c, "INVALID_HUB_ID")
			}
			hubID = id
		}
		items, err := h.service.GetCategoriesByHub(hubID)
		if err != nil {
			return helper.JSONInternal(c, "failed to fetch categories")
		}
		return c.JSON(fiber.Map{"categories": items})
	}
	scope := domainutil.NormalizeStatus(helper.QueryString(c, "scope"))
	if scope == "" {
		scope = domain.CatalogScopeProduct
	}
	if !domainutil.StatusIn(scope, domain.CatalogScopeProduct, domain.CatalogScopeMaterial, domain.CatalogScopeAll) {
		return helper.BadRequestError(c, "INVALID_SCOPE")
	}
	items, err := h.service.GetCategories(scope, 0)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch categories")
	}
	return c.JSON(fiber.Map{"categories": items})
}

func (h *CatalogHandler) GetHubs(c *fiber.Ctx) error {
	scope := domainutil.NormalizeStatus(helper.QueryString(c, "scope"))
	if scope != "" && !domainutil.StatusIn(scope, domain.CatalogScopeProduct, domain.CatalogScopeMaterial, domain.CatalogScopeAll) {
		return helper.BadRequestError(c, "INVALID_SCOPE")
	}
	hubs, err := h.service.GetHubs(scope)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch hubs")
	}
	return c.JSON(fiber.Map{"hubs": hubs})
}

func (h *CatalogHandler) GetSubCategories(c *fiber.Ctx) error {
	categoryID, err := helper.ParsePositiveInt64Param(c, "id")
	if err != nil || categoryID <= 0 {
		return helper.BadRequestError(c, "invalid category id")
	}

	items, err := h.service.GetSubCategories(categoryID)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch sub-categories")
	}
	return c.JSON(items)
}

func (h *CatalogHandler) GetAllLBISubCategories(c *fiber.Ctx) error {
	scope := domainutil.NormalizeStatus(helper.QueryString(c, "scope"))
	if scope != "" && !domainutil.StatusIn(scope, domain.CatalogScopeProduct, domain.CatalogScopeMaterial, domain.CatalogScopeAll) {
		return helper.BadRequestError(c, "INVALID_SCOPE")
	}
	items, err := h.service.GetAllSubCategories(scope)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch sub-categories")
	}
	return c.JSON(items)
}

func (h *CatalogHandler) GetUnits(c *fiber.Ctx) error {
	items, err := h.service.GetUnits()
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch units")
	}
	return c.JSON(items)
}

// GetHubShowcases returns showcases grouped by hub.
// GET /api/v1/hubs/showcases           → all hubs
// GET /api/v1/hubs/:hub_id/showcases   → single hub
func (h *CatalogHandler) GetHubShowcases(c *fiber.Ctx) error {
	var hubID int64
	if idStr := c.Params("hub_id"); idStr != "" {
		id, err := helper.ParsePositiveInt64Param(c, "hub_id")
		if err != nil {
			return helper.BadRequestError(c, "INVALID_HUB_ID")
		}
		hubID = id
	}
	limit := helper.QueryParams(c).Int("limit", 8)
	groups, err := h.service.GetHubShowcases(hubID, limit)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch hub showcases")
	}
	return c.JSON(fiber.Map{"hubs": groups})
}

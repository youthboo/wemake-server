package platformconfig

import (
	"errors"

	"github.com/yourusername/wemake/internal/dto"
	handlerregistry "github.com/yourusername/wemake/internal/handler/errorregistry"
	"github.com/yourusername/wemake/internal/helper"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	authservice "github.com/yourusername/wemake/internal/service/auth"
	platformservice "github.com/yourusername/wemake/internal/service/platform_config"
)

type PlatformConfigHandler struct {
	service *platformservice.PlatformConfigService
	auth    *authservice.AuthService
}

func NewPlatformConfigHandler(service *platformservice.PlatformConfigService, auth *authservice.AuthService) *PlatformConfigHandler {
	return &PlatformConfigHandler{service: service, auth: auth}
}

func (h *PlatformConfigHandler) GetActive(c *fiber.Ctx) error {
	item, err := h.service.GetActive()
	if err != nil {
		return helper.WriteAPIError(c, helper.InternalServerAPIError("FETCH_CONFIG_FAILED", "failed to fetch platform config"))
	}
	return c.JSON(item)
}

func (h *PlatformConfigHandler) ListHistory(c *fiber.Ctx) error {
	items, err := h.service.ListHistory()
	if err != nil {
		return helper.WriteAPIError(c, helper.InternalServerAPIError("FETCH_HISTORY_FAILED", "failed to fetch platform config history"))
	}
	return helper.WriteListResponse(c, items, len(items))
}

func (h *PlatformConfigHandler) ListAll(c *fiber.Ctx) error {
	items, err := h.service.ListAll()
	if err != nil {
		return helper.WriteAPIError(c, helper.InternalServerAPIError("FETCH_CONFIGS_FAILED", "failed to fetch platform configs"))
	}
	return c.JSON(fiber.Map{
		"configs": items,
		"total":   len(items),
	})
}

// GetDefaultComm returns the platform_config row where label = 'default_comm'.
// GET /admin/platform-configs/default-comm
func (h *PlatformConfigHandler) GetDefaultComm(c *fiber.Ctx) error {
	item, err := h.service.GetDefaultComm()
	if err != nil {
		return helper.WriteAPIError(c, helper.NotFoundAPIError("DEFAULT_COMM_NOT_FOUND", "default commission config not found"))
	}
	return c.JSON(item)
}

func (h *PlatformConfigHandler) Create(c *fiber.Ctx) error {
	userID, err := requirePlatformConfigActorID(c)
	if err != nil {
		return err
	}
	var req dto.CreateConfigVersionRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	promoStartAt, err := helper.ParseOptionalRFC3339Value(req.PromoStartAt, "promo_start_at")
	if err != nil {
		return helper.WriteAPIError(c, helper.BadRequestAPIError("INVALID_PROMO_START_AT", "promo_start_at must be RFC3339"))
	}
	promoEndAt, err := helper.ParseOptionalRFC3339Value(req.PromoEndAt, "promo_end_at")
	if err != nil {
		return helper.WriteAPIError(c, helper.BadRequestAPIError("INVALID_PROMO_END_AT", "promo_end_at must be RFC3339"))
	}
	cfg := &domain.PlatformConfig{
		DefaultCommissionRate: req.DefaultCommissionRate,
		PromoCommissionRate:   req.PromoCommissionRate,
		PromoStartAt:          promoStartAt,
		PromoEndAt:            promoEndAt,
		PromoLabel:            req.PromoLabel,
		VatRate:               req.VatRate,
		CurrencyCode:          req.CurrencyCode,
		CreatedBy:             &userID,
	}
	if err := h.service.CreateVersion(cfg); err != nil {
		return helper.WriteAPIError(c, helper.InternalServerAPIError("CREATE_VERSION_FAILED", "failed to create platform config version"))
	}
	c.Status(fiber.StatusCreated)
	return c.JSON(cfg)
}

func (h *PlatformConfigHandler) CreateConfig(c *fiber.Ctx) error {
	actorID, err := requirePlatformConfigActorID(c)
	if err != nil {
		return err
	}
	var req domain.CreatePlatformConfigRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	ip := c.IP()
	item, err := h.service.CreateConfig(req, actorID, &ip)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to create platform config"), handlerregistry.CreateConfigErrorMap())
	}
	c.Status(fiber.StatusCreated)
	return c.JSON(item)
}

func (h *PlatformConfigHandler) UpdateConfig(c *fiber.Ctx) error {
	configID, err := helper.ParsePositiveInt64Param(c, "config_id")
	if err != nil || configID <= 0 {
		return helper.WriteAPIError(c, helper.BadRequestAPIError("INVALID_CONFIG_ID", "invalid config_id"))
	}
	actorID, err := requirePlatformConfigActorID(c)
	if err != nil {
		return err
	}
	var req domain.UpdatePlatformConfigRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	ip := c.IP()
	item, err := h.service.UpdateConfig(configID, req, actorID, &ip)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to update platform config"), handlerregistry.UpdateConfigErrorMap())
	}
	return c.JSON(item)
}

func (h *PlatformConfigHandler) DeleteConfig(c *fiber.Ctx) error {
	configID, err := helper.ParsePositiveInt64Param(c, "config_id")
	if err != nil || configID <= 0 {
		return helper.WriteAPIError(c, helper.BadRequestAPIError("INVALID_CONFIG_ID", "invalid config_id"))
	}
	actorID, err := requirePlatformConfigActorID(c)
	if err != nil {
		return err
	}
	ip := c.IP()
	err = h.service.DeleteConfig(configID, actorID, &ip)
	if err != nil {
		if errors.Is(err, platformservice.ErrPlatformConfigInUse) {
			count, _ := h.service.FactoriesUsingConfig(configID)
			msg := "ไม่สามารถลบได้ มีโรงงาน " + strconv.Itoa(count) + " แห่งกำลังใช้ config นี้อยู่"
			return helper.WriteAPIError(c, helper.ConflictAPIError("CONFIG_IN_USE", msg))
		}
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to delete platform config"), handlerregistry.DeleteConfigErrorMap())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func requirePlatformConfigActorID(c *fiber.Ctx) (int64, error) {
	return helper.RequireAPIUserID(c, helper.UnauthorizedAPIError("UNAUTHORIZED", "unauthorized"))
}

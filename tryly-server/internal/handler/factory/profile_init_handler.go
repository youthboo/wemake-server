package factory

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/helper"
	"github.com/yourusername/wemake/internal/logger"
	authservice "github.com/yourusername/wemake/internal/service/auth"
	factoryservice "github.com/yourusername/wemake/internal/service/factory"
)

type ProfileInitHandler struct {
	profileInit *factoryservice.ProfileInitService
	auth        *authservice.AuthService
}

func NewProfileInitHandler(profileInit *factoryservice.ProfileInitService, auth *authservice.AuthService) *ProfileInitHandler {
	return &ProfileInitHandler{profileInit: profileInit, auth: auth}
}

// GET /factories/me/profile-init
func (h *ProfileInitHandler) GetProfileInit(c *fiber.Ctx) error {
	userID, _, err := helper.RequireAPIFactoryUser(
		c, h.auth,
		helper.BadRequestAPIError("INVALID_USER_CONTEXT", "invalid user context"),
		helper.UnauthorizedAPIError("USER_NOT_FOUND", "user not found"),
		helper.ForbiddenAPIError("FACTORY_ROLE_REQUIRED", "factory role required"),
	)
	if err != nil {
		return err
	}

	resp, err := h.profileInit.GetProfileInit(userID)
	if err != nil {
		logger.Error("GetProfileInit failed for userID=%d: %v", userID, err)
		return helper.WriteAPIError(c, helper.InternalServerAPIError("FETCH_FACTORY_FAILED", "failed to fetch factory profile"))
	}
	return c.JSON(resp)
}

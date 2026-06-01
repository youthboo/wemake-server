package profile

import (
	"strconv"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/dto"
	"github.com/yourusername/wemake/internal/helper"
	mediautil "github.com/yourusername/wemake/internal/media"
	profileservice "github.com/yourusername/wemake/internal/service/profile"
)

type ProfileHandler struct {
	service       *profileservice.ProfileService
	publicBaseURL string
	cld           *cloudinary.Cloudinary
}

func NewProfileHandler(service *profileservice.ProfileService, publicBaseURL string, cld *cloudinary.Cloudinary) *ProfileHandler {
	return &ProfileHandler{service: service, publicBaseURL: strings.TrimRight(publicBaseURL, "/"), cld: cld}
}

func (h *ProfileHandler) GetProfile(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	item, err := h.service.GetProfile(userID)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch profile")
	}
	return c.JSON(item)
}

func (h *ProfileHandler) UpdateProfile(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	role := helper.OptionalRoleFromContext(c)
	if role == "" {
		profile, err := h.service.GetProfile(userID)
		if err != nil {
			return helper.JSONInternal(c, "failed to resolve profile")
		}
		role = profile.Role
	}
	switch role {
	case domain.RoleFactory:
		var req dto.UpdateProfileRequest
		if err := helper.ParseBody(c, &req, "invalid payload"); err != nil {
			return err
		}
		item, err := h.service.UpdateFactoryProfile(userID, req.Phone, nil, &domain.FactoryProfile{
			Description:    nil,
			Specialization: nil,
			LeadTimeDesc:   nil,
			PriceRange:     nil,
		})
		if err != nil {
			return helper.BadRequestError(c, "invalid profile input")
		}
		return c.JSON(item)
	default:
		var req dto.UpdateProfileRequest
		if err := helper.ParseBody(c, &req, "invalid payload"); err != nil {
			return err
		}
		item, err := h.service.UpdateCustomerProfile(userID, req.Phone, nil, &domain.CustomerProfile{
			FirstName: req.FirstName, LastName: req.LastName, AddressLine1: nil,
			SubDistrict: nil, District: nil, Province: nil, PostalCode: nil,
		})
		if err != nil {
			return helper.BadRequestError(c, "invalid profile input")
		}
		return c.JSON(item)
	}
}

func (h *ProfileHandler) UploadAvatar(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	result, err := mediautil.SaveUploadedFile(c, mediautil.UploadOptions{
		FileNamePrefix:        "avatar_" + strconv.FormatInt(userID, 10) + "_" + uuid.NewString(),
		Folder:                "wemake/avatars",
		PublicBaseURL:         h.publicBaseURL,
		MaxSize:               5 * 1024 * 1024,
		RequiredMessage:       "file is required",
		MaxSizeMessage:        "file exceeds 5MB",
		CloudUploadMessage:    "failed to upload avatar",
		CloudUploadLogMessage: "cloudinary avatar upload failed",
		Cloudinary:            h.cld,
		CloudinaryLogFields:   []interface{}{"user_id", userID},
	})
	if err != nil {
		if uploadErr, ok := err.(*mediautil.UploadError); ok {
			return c.Status(uploadErr.Status).JSON(fiber.Map{"error": uploadErr.Message})
		}
		return helper.JSONInternal(c, "failed to save file")
	}
	item, err := h.service.UpdateAvatar(userID, result.URL)
	if err != nil {
		return helper.JSONInternal(c, "failed to update avatar")
	}
	return c.JSON(fiber.Map{"avatar_url": item.AvatarURL})
}

func (h *ProfileHandler) ChangePassword(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	var req dto.ChangePasswordRequest
	if err := helper.ParseBody(c, &req, "invalid payload"); err != nil {
		return err
	}
	if err := h.service.ChangePassword(userID, req.OldPassword, req.NewPassword, ""); err != nil {
		if err == profileservice.ErrProfileUnauthorized {
			return helper.UnauthorizedError(c, "current password is incorrect")
		}
		return helper.BadRequestError(c, "invalid password input")
	}
	return c.JSON(fiber.Map{"message": "password changed successfully"})
}

func (h *ProfileHandler) GetSummary(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	role := helper.OptionalRoleFromContext(c)
	if role == "" {
		profile, err := h.service.GetProfile(userID)
		if err != nil {
			return helper.JSONInternal(c, "failed to resolve profile")
		}
		role = profile.Role
	}
	item, err := h.service.GetSummary(userID, role)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch summary")
	}
	return c.JSON(item)
}

func (h *ProfileHandler) ListTransactions(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	page, limit := helper.PageLimit(c, helper.DefaultPageSize)
	items, total, totalIn, totalOut, err := h.service.ListTransactions(userID, page, limit, helper.QueryString(c, "type"), helper.QueryString(c, "status"))
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch transactions")
	}
	totalPages := int64((int(total) + limit - 1) / limit)
	return c.JSON(fiber.Map{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
		"data":        items,
		"summary": fiber.Map{
			"total_in":  totalIn,
			"total_out": totalOut,
			"net":       totalIn - totalOut,
		},
	})
}

func (h *ProfileHandler) ListMyReviews(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	page, limit := helper.PageLimit(c, helper.DefaultPageSize)
	items, total, err := h.service.ListMyReviews(userID, page, limit)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch reviews")
	}
	return c.JSON(fiber.Map{"page": page, "limit": limit, "total": total, "data": items})
}

func (h *ProfileHandler) ListReceivedReviews(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	page, limit := helper.PageLimit(c, helper.DefaultPageSize)
	role := helper.OptionalRoleFromContext(c)
	items, total, err := h.service.ListReceivedReviews(userID, role, page, limit)
	if err != nil {
		return helper.ForbiddenError(c, "factory role required")
	}
	return c.JSON(fiber.Map{"page": page, "limit": limit, "total": total, "data": items})
}

func (h *ProfileHandler) GetNotifPrefs(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	item, err := h.service.GetNotificationPreference(userID)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch preferences")
	}
	return c.JSON(item)
}

func (h *ProfileHandler) UpdateNotifPrefs(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	var req domain.NotificationPreference
	if err := helper.ParseBody(c, &req, "invalid payload"); err != nil {
		return err
	}
	item, err := h.service.UpdateNotificationPreference(userID, &req)
	if err != nil {
		return helper.JSONInternal(c, "failed to update preferences")
	}
	return c.JSON(item)
}

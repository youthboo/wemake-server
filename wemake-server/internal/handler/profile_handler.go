package handler

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/service"
)

type ProfileHandler struct {
	service       *service.ProfileService
	publicBaseURL string
	cld           *cloudinary.Cloudinary
}

func NewProfileHandler(service *service.ProfileService, publicBaseURL string, cld *cloudinary.Cloudinary) *ProfileHandler {
	return &ProfileHandler{service: service, publicBaseURL: strings.TrimRight(publicBaseURL, "/"), cld: cld}
}

func (h *ProfileHandler) GetProfile(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	item, err := h.service.GetProfile(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch profile"})
	}
	return c.JSON(item)
}

func (h *ProfileHandler) UpdateProfile(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	role := getOptionalRoleFromContext(c)
	if role == "" {
		profile, err := h.service.GetProfile(userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to resolve profile"})
		}
		role = profile.Role
	}
	switch role {
	case domain.RoleFactory:
		var req struct {
			Phone          string  `json:"phone"`
			Bio            *string `json:"bio"`
			Description    *string `json:"description"`
			Specialization *string `json:"specialization"`
			MinOrder       *int64  `json:"min_order"`
			LeadTimeDesc   *string `json:"lead_time_desc"`
			PriceRange     *string `json:"price_range"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
		}
		item, err := h.service.UpdateFactoryProfile(userID, req.Phone, req.Bio, &domain.FactoryProfile{
			Description:    req.Description,
			Specialization: req.Specialization,
			MinOrder:       req.MinOrder,
			LeadTimeDesc:   req.LeadTimeDesc,
			PriceRange:     req.PriceRange,
		})
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid profile input"})
		}
		return c.JSON(item)
	default:
		var req struct {
			Phone        string  `json:"phone"`
			Bio          *string `json:"bio"`
			FirstName    string  `json:"first_name"`
			LastName     string  `json:"last_name"`
			AddressLine1 *string `json:"address_line1"`
			SubDistrict  *string `json:"sub_district"`
			District     *string `json:"district"`
			Province     *string `json:"province"`
			PostalCode   *string `json:"postal_code"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
		}
		item, err := h.service.UpdateCustomerProfile(userID, req.Phone, req.Bio, &domain.CustomerProfile{
			FirstName: req.FirstName, LastName: req.LastName, AddressLine1: req.AddressLine1,
			SubDistrict: req.SubDistrict, District: req.District, Province: req.Province, PostalCode: req.PostalCode,
		})
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid profile input"})
		}
		return c.JSON(item)
	}
}

func (h *ProfileHandler) UploadAvatar(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file is required"})
	}
	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "file exceeds 5MB"})
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		ext = ".jpg"
	}
	fileName := "avatar_" + strconv.FormatInt(userID, 10) + "_" + uuid.NewString() + ext
	var avatarURL string
	if h.cld != nil {
		src, err := file.Open()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to read upload"})
		}
		defer src.Close()
		result, err := h.cld.Upload.Upload(context.Background(), src, uploader.UploadParams{
			Folder:       "wemake/avatars",
			PublicID:     strings.TrimSuffix(fileName, ext),
			ResourceType: "auto",
		})
		if err != nil {
			log.Printf("cloudinary upload avatar: %v", err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "failed to upload avatar"})
		}
		avatarURL = result.SecureURL
	} else {
		saveDir := "./uploads"
		savePath := filepath.Join(saveDir, fileName)
		if err := os.MkdirAll(saveDir, 0755); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create uploads directory"})
		}
		if err := c.SaveFile(file, savePath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
		}
		baseURL := h.publicBaseURL
		if baseURL == "" {
			baseURL = c.BaseURL()
		}
		avatarURL = baseURL + "/uploads/" + fileName
	}
	item, err := h.service.UpdateAvatar(userID, avatarURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update avatar"})
	}
	return c.JSON(fiber.Map{"avatar_url": item.AvatarURL})
}

func (h *ProfileHandler) ChangePassword(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
		ConfirmPassword string `json:"confirm_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	if err := h.service.ChangePassword(userID, req.CurrentPassword, req.NewPassword, req.ConfirmPassword); err != nil {
		if err == service.ErrProfileUnauthorized {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "current password is incorrect"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid password input"})
	}
	return c.JSON(fiber.Map{"message": "password changed successfully"})
}

func (h *ProfileHandler) GetSummary(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	role := getOptionalRoleFromContext(c)
	if role == "" {
		profile, err := h.service.GetProfile(userID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to resolve profile"})
		}
		role = profile.Role
	}
	item, err := h.service.GetSummary(userID, role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch summary"})
	}
	return c.JSON(item)
}

func (h *ProfileHandler) ListTransactions(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	page := maxIntQuery(c.QueryInt("page", 1), 1)
	limit := clampInt(c.QueryInt("limit", 20), 1, 100)
	items, total, totalIn, totalOut, err := h.service.ListTransactions(userID, page, limit, c.Query("type"), c.Query("status"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch transactions"})
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
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	page := maxIntQuery(c.QueryInt("page", 1), 1)
	limit := clampInt(c.QueryInt("limit", 20), 1, 100)
	items, total, err := h.service.ListMyReviews(userID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch reviews"})
	}
	return c.JSON(fiber.Map{"page": page, "limit": limit, "total": total, "data": items})
}

func (h *ProfileHandler) ListReceivedReviews(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	page := maxIntQuery(c.QueryInt("page", 1), 1)
	limit := clampInt(c.QueryInt("limit", 20), 1, 100)
	role := getOptionalRoleFromContext(c)
	items, total, err := h.service.ListReceivedReviews(userID, role, page, limit)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "factory role required"})
	}
	return c.JSON(fiber.Map{"page": page, "limit": limit, "total": total, "data": items})
}

func (h *ProfileHandler) GetNotifPrefs(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	item, err := h.service.GetNotificationPreference(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch preferences"})
	}
	return c.JSON(item)
}

func (h *ProfileHandler) UpdateNotifPrefs(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var req domain.NotificationPreference
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid payload"})
	}
	item, err := h.service.UpdateNotificationPreference(userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update preferences"})
	}
	return c.JSON(item)
}

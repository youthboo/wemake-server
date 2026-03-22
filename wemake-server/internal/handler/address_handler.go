package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/service"
)

type AddressHandler struct {
	service *service.AddressService
}

func NewAddressHandler(service *service.AddressService) *AddressHandler {
	return &AddressHandler{service: service}
}

func (h *AddressHandler) ListAddresses(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}

	addresses, err := h.service.ListByUserID(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch addresses"})
	}
	return c.JSON(addresses)
}

func (h *AddressHandler) CreateAddress(c *fiber.Ctx) error {
	type createAddressRequest struct {
		AddressType   string `json:"address_type"`
		AddressDetail string `json:"address_detail"`
		SubDistrictID int64  `json:"sub_district_id"`
		DistrictID    int64  `json:"district_id"`
		ProvinceID    int64  `json:"province_id"`
		ZipCode       string `json:"zip_code"`
		IsDefault     bool   `json:"is_default"`
	}

	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}

	var req createAddressRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	if strings.TrimSpace(req.AddressType) == "" || strings.TrimSpace(req.AddressDetail) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "address_type and address_detail are required"})
	}

	address := &domain.Address{
		UserID:        userID,
		AddressType:   strings.TrimSpace(strings.ToUpper(req.AddressType)),
		AddressDetail: strings.TrimSpace(req.AddressDetail),
		SubDistrictID: req.SubDistrictID,
		DistrictID:    req.DistrictID,
		ProvinceID:    req.ProvinceID,
		ZipCode:       strings.TrimSpace(req.ZipCode),
		IsDefault:     req.IsDefault,
	}

	if err := h.service.Create(address); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create address"})
	}
	return c.Status(fiber.StatusCreated).JSON(address)
}

func (h *AddressHandler) PatchAddress(c *fiber.Ctx) error {
	type patchAddressRequest struct {
		AddressType   *string `json:"address_type"`
		AddressDetail *string `json:"address_detail"`
		SubDistrictID *int64  `json:"sub_district_id"`
		DistrictID    *int64  `json:"district_id"`
		ProvinceID    *int64  `json:"province_id"`
		ZipCode       *string `json:"zip_code"`
		IsDefault     *bool   `json:"is_default"`
	}

	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}

	addressID, err := c.ParamsInt("address_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid address_id"})
	}

	var req patchAddressRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	fields := map[string]interface{}{}
	if req.AddressType != nil {
		fields["address_type"] = strings.TrimSpace(strings.ToUpper(*req.AddressType))
	}
	if req.AddressDetail != nil {
		fields["address_detail"] = strings.TrimSpace(*req.AddressDetail)
	}
	if req.SubDistrictID != nil {
		fields["sub_district_id"] = *req.SubDistrictID
	}
	if req.DistrictID != nil {
		fields["district_id"] = *req.DistrictID
	}
	if req.ProvinceID != nil {
		fields["province_id"] = *req.ProvinceID
	}
	if req.ZipCode != nil {
		fields["zip_code"] = strings.TrimSpace(*req.ZipCode)
	}
	if req.IsDefault != nil {
		fields["is_default"] = *req.IsDefault
	}

	if err := h.service.Patch(userID, int64(addressID), fields); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to patch address"})
	}
	return c.JSON(fiber.Map{"message": "address updated"})
}

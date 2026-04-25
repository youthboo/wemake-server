package handler

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type RFQHandler struct {
	service *service.RFQService
	auth    *service.AuthService
}

func NewRFQHandler(rfqService *service.RFQService, authService *service.AuthService) *RFQHandler {
	return &RFQHandler{service: rfqService, auth: authService}
}

func (h *RFQHandler) CreateRFQ(c *fiber.Ctx) error {
	type createRFQRequest struct {
		CategoryID             int64         `json:"category_id"`
		SubCategoryID          *int64        `json:"sub_category_id"`
		Title                  string        `json:"title"`
		Description            string        `json:"description"`
		Quantity               int64         `json:"quantity"`
		Unit                   string        `json:"unit"`
		Details                string        `json:"details"`
		AddressID              int64         `json:"address_id"`
		ShippingMethodID       *int64        `json:"shipping_method_id"`
		MaterialGrade          *string       `json:"material_grade"`
		TargetUnitPrice        *float64      `json:"target_unit_price"`
		TargetLeadTimeDays     *int          `json:"target_lead_time_days"`
		RequiredDeliveryDate   *string       `json:"required_delivery_date"`
		DeliveryAddressID      *int64        `json:"delivery_address_id"`
		CertificationsRequired []string      `json:"certifications_required"`
		SampleRequired         bool          `json:"sample_required"`
		SampleQty              *int          `json:"sample_qty"`
		InspectionType         *string       `json:"inspection_type"`
		ReferenceImages        []string      `json:"reference_images"`
	}

	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}

	var req createRFQRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	details := strings.TrimSpace(req.Details)
	if details == "" {
		details = strings.TrimSpace(req.Description)
	}

	if req.CategoryID <= 0 || req.AddressID <= 0 || req.Quantity <= 0 || strings.TrimSpace(req.Title) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "category_id, title, quantity, and address_id are required"})
	}

	rfq := &domain.RFQ{
		UserID:                 userID,
		CategoryID:             req.CategoryID,
		SubCategoryID:          req.SubCategoryID,
		Title:                  req.Title,
		Quantity:               req.Quantity,
		Details:                details,
		AddressID:              req.AddressID,
		ShippingMethodID:       req.ShippingMethodID,
		MaterialGrade:          req.MaterialGrade,
		TargetUnitPrice:        req.TargetUnitPrice,
		TargetLeadTimeDays:     req.TargetLeadTimeDays,
		DeliveryAddressID:      req.DeliveryAddressID,
		CertificationsRequired: req.CertificationsRequired,
		SampleRequired:         req.SampleRequired,
		SampleQty:              req.SampleQty,
		InspectionType:         req.InspectionType,
		ReferenceImages:        req.ReferenceImages,
	}
	if rfq.DeliveryAddressID == nil {
		rfq.DeliveryAddressID = &rfq.AddressID
	}
	if req.RequiredDeliveryDate != nil && strings.TrimSpace(*req.RequiredDeliveryDate) != "" {
		d, err := time.Parse("2006-01-02", strings.TrimSpace(*req.RequiredDeliveryDate))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "required_delivery_date must be YYYY-MM-DD"})
		}
		rfq.RequiredDeliveryDate = &d
	}

	if err := h.service.Create(rfq); err != nil {
		if err == service.ErrInvalidSubCategory || err == service.ErrInvalidShippingMethod || err == service.ErrMaxRFQReferenceImages || err == service.ErrRFQInspectionInvalid || err == service.ErrRFQDetailsRequired {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create rfq"})
	}
	return c.Status(fiber.StatusCreated).JSON(rfq)
}

func (h *RFQHandler) PatchRFQ(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	rfqID, err := c.ParamsInt("rfq_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
	}
	type patchRFQRequest struct {
		CategoryID             int64         `json:"category_id"`
		SubCategoryID          *int64        `json:"sub_category_id"`
		Title                  string        `json:"title"`
		Description            string        `json:"description"`
		Quantity               int64         `json:"quantity"`
		Unit                   string        `json:"unit"`
		Details                string        `json:"details"`
		AddressID              int64         `json:"address_id"`
		ShippingMethodID       *int64        `json:"shipping_method_id"`
		MaterialGrade          *string       `json:"material_grade"`
		TargetUnitPrice        *float64      `json:"target_unit_price"`
		TargetLeadTimeDays     *int          `json:"target_lead_time_days"`
		RequiredDeliveryDate   *string       `json:"required_delivery_date"`
		DeliveryAddressID      *int64        `json:"delivery_address_id"`
		CertificationsRequired []string      `json:"certifications_required"`
		SampleRequired         bool          `json:"sample_required"`
		SampleQty              *int          `json:"sample_qty"`
		InspectionType         *string       `json:"inspection_type"`
		ReferenceImages        []string      `json:"reference_images"`
	}
	var req patchRFQRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	details := strings.TrimSpace(req.Details)
	if details == "" {
		details = strings.TrimSpace(req.Description)
	}
	rfq := &domain.RFQ{
		CategoryID: req.CategoryID, SubCategoryID: req.SubCategoryID, Title: req.Title, Quantity: req.Quantity,
		Details: details, AddressID: req.AddressID,
		ShippingMethodID: req.ShippingMethodID,
		MaterialGrade: req.MaterialGrade, TargetUnitPrice: req.TargetUnitPrice,
		TargetLeadTimeDays: req.TargetLeadTimeDays,
		DeliveryAddressID: req.DeliveryAddressID, CertificationsRequired: req.CertificationsRequired, SampleRequired: req.SampleRequired,
		SampleQty: req.SampleQty, InspectionType: req.InspectionType, ReferenceImages: req.ReferenceImages,
	}
	if req.RequiredDeliveryDate != nil && strings.TrimSpace(*req.RequiredDeliveryDate) != "" {
		d, err := time.Parse("2006-01-02", strings.TrimSpace(*req.RequiredDeliveryDate))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "required_delivery_date must be YYYY-MM-DD"})
		}
		rfq.RequiredDeliveryDate = &d
	}
	if err := h.service.Patch(userID, int64(rfqID), rfq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(rfq)
}

func (h *RFQHandler) ListRFQs(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	status := c.Query("status")
	rfqs, err := h.service.ListByUserID(userID, status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch rfqs"})
	}
	return c.JSON(rfqs)
}

func (h *RFQHandler) ListMatching(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	if u.Role != domain.RoleFactory {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "factory role required"})
	}
	status := c.Query("status")
	items, err := h.service.ListMatchingForFactory(userID, status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch matching rfqs"})
	}
	return c.JSON(items)
}

func (h *RFQHandler) GetRFQ(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	rfqID, err := c.ParamsInt("rfq_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
	}

	rfq, err := h.service.GetForViewer(userID, u.Role, int64(rfqID))
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "rfq not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch rfq"})
	}

	return c.JSON(fiber.Map{"rfq": rfq})
}

func (h *RFQHandler) CancelRFQ(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	rfqID, err := c.ParamsInt("rfq_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
	}

	if err := h.service.Cancel(userID, int64(rfqID)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to cancel rfq"})
	}
	return c.JSON(fiber.Map{"message": "rfq canceled"})
}

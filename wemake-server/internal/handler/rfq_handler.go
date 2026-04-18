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
		CategoryID       int64    `json:"category_id"`
		SubCategoryID    *int64   `json:"sub_category_id"`
		Title            string   `json:"title"`
		Quantity         int64    `json:"quantity"`
		UnitID           int64    `json:"unit_id"`
		BudgetPerPiece   float64  `json:"budget_per_piece"`
		Details          string   `json:"details"`
		AddressID        int64    `json:"address_id"`
		ShippingMethodID *int64   `json:"shipping_method_id"`
		DeadlineDate     *string  `json:"deadline_date"`
		ImageURLs        []string `json:"image_urls"`
	}

	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}

	var req createRFQRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	if req.CategoryID <= 0 || req.UnitID <= 0 || req.AddressID <= 0 || req.Quantity <= 0 || strings.TrimSpace(req.Title) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "category_id, title, quantity, unit_id, and address_id are required"})
	}

	rfq := &domain.RFQ{
		UserID:           userID,
		CategoryID:       req.CategoryID,
		SubCategoryID:    req.SubCategoryID,
		Title:            req.Title,
		Quantity:         req.Quantity,
		UnitID:           req.UnitID,
		BudgetPerPiece:   req.BudgetPerPiece,
		Details:          req.Details,
		AddressID:        req.AddressID,
		ShippingMethodID: req.ShippingMethodID,
		ImageURLs:        domain.JSONStringArray(req.ImageURLs),
	}
	if req.DeadlineDate != nil && strings.TrimSpace(*req.DeadlineDate) != "" {
		d, err := time.Parse("2006-01-02", strings.TrimSpace(*req.DeadlineDate))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "deadline_date must be YYYY-MM-DD"})
		}
		rfq.DeadlineDate = &d
	}

	if err := h.service.Create(rfq); err != nil {
		if err == service.ErrInvalidSubCategory || err == service.ErrInvalidShippingMethod || err == service.ErrMaxRFQImages {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create rfq"})
	}
	return c.Status(fiber.StatusCreated).JSON(rfq)
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

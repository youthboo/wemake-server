package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type RFQHandler struct {
	service *service.RFQService
}

func NewRFQHandler(service *service.RFQService) *RFQHandler {
	return &RFQHandler{service: service}
}

func (h *RFQHandler) CreateRFQ(c *fiber.Ctx) error {
	type createRFQRequest struct {
		CategoryID     int64   `json:"category_id"`
		Title          string  `json:"title"`
		Quantity       int64   `json:"quantity"`
		UnitID         int64   `json:"unit_id"`
		BudgetPerPiece float64 `json:"budget_per_piece"`
		Details        string  `json:"details"`
		AddressID      int64   `json:"address_id"`
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
		UserID:         userID,
		CategoryID:     req.CategoryID,
		Title:          req.Title,
		Quantity:       req.Quantity,
		UnitID:         req.UnitID,
		BudgetPerPiece: req.BudgetPerPiece,
		Details:        req.Details,
		AddressID:      req.AddressID,
	}

	if err := h.service.Create(rfq); err != nil {
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

func (h *RFQHandler) GetRFQ(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	rfqID, err := c.ParamsInt("rfq_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
	}

	rfq, images, err := h.service.GetByID(userID, int64(rfqID))
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "rfq not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch rfq"})
	}

	return c.JSON(fiber.Map{
		"rfq":    rfq,
		"images": images,
	})
}

func (h *RFQHandler) AddRFQImage(c *fiber.Ctx) error {
	type addImageRequest struct {
		ImageURL string `json:"image_url"`
	}
	rfqID, err := c.ParamsInt("rfq_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
	}

	var req addImageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if strings.TrimSpace(req.ImageURL) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "image_url is required"})
	}

	image, err := h.service.AddImage(int64(rfqID), req.ImageURL)
	if err != nil {
		if err == service.ErrMaxRFQImages {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add image"})
	}
	return c.Status(fiber.StatusCreated).JSON(image)
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

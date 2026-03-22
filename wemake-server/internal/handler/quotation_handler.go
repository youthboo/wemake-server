package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type QuotationHandler struct {
	service *service.QuotationService
}

func NewQuotationHandler(service *service.QuotationService) *QuotationHandler {
	return &QuotationHandler{service: service}
}

func (h *QuotationHandler) CreateQuotation(c *fiber.Ctx) error {
	type reqBody struct {
		FactoryID        int64   `json:"factory_id"`
		PricePerPiece    float64 `json:"price_per_piece"`
		MoldCost         float64 `json:"mold_cost"`
		LeadTimeDays     int64   `json:"lead_time_days"`
		ShippingMethodID int64   `json:"shipping_method_id"`
	}

	rfqID, err := c.ParamsInt("rfq_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
	}

	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}

	if req.FactoryID <= 0 || req.PricePerPiece <= 0 || req.LeadTimeDays <= 0 || req.ShippingMethodID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "factory_id, price_per_piece, lead_time_days, shipping_method_id are required"})
	}

	item := &domain.Quotation{
		RFQID:            int64(rfqID),
		FactoryID:        req.FactoryID,
		PricePerPiece:    req.PricePerPiece,
		MoldCost:         req.MoldCost,
		LeadTimeDays:     req.LeadTimeDays,
		ShippingMethodID: req.ShippingMethodID,
	}
	if err := h.service.Create(item); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create quotation"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *QuotationHandler) ListQuotationsByRFQ(c *fiber.Ctx) error {
	rfqID, err := c.ParamsInt("rfq_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
	}
	items, err := h.service.ListByRFQID(int64(rfqID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch quotations"})
	}
	return c.JSON(items)
}

func (h *QuotationHandler) GetQuotation(c *fiber.Ctx) error {
	quotationID, err := c.ParamsInt("quotation_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid quotation_id"})
	}
	item, err := h.service.GetByID(int64(quotationID))
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "quotation not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch quotation"})
	}
	return c.JSON(item)
}

func (h *QuotationHandler) PatchQuotationStatus(c *fiber.Ctx) error {
	type reqBody struct {
		Status string `json:"status"`
	}
	quotationID, err := c.ParamsInt("quotation_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid quotation_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	status := strings.TrimSpace(strings.ToUpper(req.Status))
	if status != "AC" && status != "RJ" && status != "PD" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be PD, AC or RJ"})
	}
	if err := h.service.UpdateStatus(int64(quotationID), status); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update quotation status"})
	}
	return c.JSON(fiber.Map{"message": "quotation status updated"})
}

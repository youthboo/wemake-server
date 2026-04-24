package handler

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type QuotationHandler struct {
	service *service.QuotationService
	auth    *service.AuthService
}

func NewQuotationHandler(quotationService *service.QuotationService, authService *service.AuthService) *QuotationHandler {
	return &QuotationHandler{service: quotationService, auth: authService}
}

func (h *QuotationHandler) CreateQuotation(c *fiber.Ctx) error {
	type reqBody struct {
		FactoryID        int64   `json:"factory_id"`
		PricePerPiece    float64 `json:"price_per_piece"`
		MoldCost         float64 `json:"mold_cost"`
		LeadTimeDays     int64   `json:"lead_time_days"`
		ShippingMethodID int64   `json:"shipping_method_id"`
	}

	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
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
	if req.FactoryID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "factory_id must match authenticated user"})
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

func (h *QuotationHandler) ListMine(c *fiber.Ctx) error {
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
	items, err := h.service.ListMine(userID, status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch quotations"})
	}
	return c.JSON(items)
}

func (h *QuotationHandler) ListCollection(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	factoryParam := strings.TrimSpace(c.Query("factory_id"))
	if factoryParam == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "factory_id query is required"})
	}
	if u.Role != domain.RoleFactory {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "factory role required"})
	}
	var factoryID int64
	if strings.EqualFold(factoryParam, "me") {
		factoryID = userID
	} else {
		parsed, parseErr := strconv.ParseInt(factoryParam, 10, 64)
		if parseErr != nil || parsed <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
		}
		if parsed != userID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "factory_id must match authenticated factory"})
		}
		factoryID = parsed
	}
	items, err := h.service.ListMine(factoryID, c.Query("status"))
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

func (h *QuotationHandler) ListHistory(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	u, err := h.auth.GetUserByID(userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	quotationID, err := c.ParamsInt("quotation_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid quotation_id"})
	}
	ok, err := h.service.CanView(int64(quotationID), userID, u.Role)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "quotation not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to authorize"})
	}
	if !ok {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "not authorized"})
	}
	items, err := h.service.ListRevisionChain(int64(quotationID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch history"})
	}
	return c.JSON(items)
}

func (h *QuotationHandler) Preview(c *fiber.Ctx) error {
	type reqBody struct {
		Items           []domain.QuotationItem `json:"items"`
		DiscountAmount  float64                `json:"discount_amount"`
		ShippingCost    float64                `json:"shipping_cost"`
		PackagingCost   float64                `json:"packaging_cost"`
		ToolingMoldCost float64                `json:"tooling_mold_cost"`
	}
	if _, err := getUserIDFromHeader(c); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	item, err := h.service.Preview(req.Items, req.DiscountAmount, req.ShippingCost, req.PackagingCost, req.ToolingMoldCost)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(item)
}

func (h *QuotationHandler) CreateDetailed(c *fiber.Ctx) error {
	type reqBody struct {
		RFQID                int64                  `json:"rfq_id"`
		Items                []domain.QuotationItem `json:"items"`
		DiscountAmount       float64                `json:"discount_amount"`
		ShippingCost         float64                `json:"shipping_cost"`
		ShippingMethod       *string                `json:"shipping_method"`
		PackagingCost        float64                `json:"packaging_cost"`
		ToolingMoldCost      float64                `json:"tooling_mold_cost"`
		LeadTimeDays         *int64                 `json:"lead_time_days"`
		ProductionStartDate  *string                `json:"production_start_date"`
		DeliveryDate         *string                `json:"delivery_date"`
		Incoterms            *string                `json:"incoterms"`
		PaymentTerms         *string                `json:"payment_terms"`
		ValidityDays         int                    `json:"validity_days"`
		WarrantyPeriodMonths *int                   `json:"warranty_period_months"`
	}
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	item := &domain.Quotation{
		RFQID:                req.RFQID,
		FactoryID:            userID,
		Items:                req.Items,
		DiscountAmount:       req.DiscountAmount,
		ShippingCost:         req.ShippingCost,
		ShippingMethod:       req.ShippingMethod,
		PackagingCost:        req.PackagingCost,
		ToolingMoldCost:      req.ToolingMoldCost,
		Incoterms:            req.Incoterms,
		PaymentTerms:         req.PaymentTerms,
		ValidityDays:         req.ValidityDays,
		WarrantyPeriodMonths: req.WarrantyPeriodMonths,
	}
	if req.LeadTimeDays != nil {
		item.LeadTimeDays = *req.LeadTimeDays
	}
	if req.ProductionStartDate != nil && strings.TrimSpace(*req.ProductionStartDate) != "" {
		d, err := time.Parse("2006-01-02", strings.TrimSpace(*req.ProductionStartDate))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "production_start_date must be YYYY-MM-DD"})
		}
		item.ProductionStartDate = &d
	}
	if req.DeliveryDate != nil && strings.TrimSpace(*req.DeliveryDate) != "" {
		d, err := time.Parse("2006-01-02", strings.TrimSpace(*req.DeliveryDate))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "delivery_date must be YYYY-MM-DD"})
		}
		item.DeliveryDate = &d
	}
	if err := h.service.CreateDetailed(item); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *QuotationHandler) CreateRevision(c *fiber.Ctx) error {
	parentID, err := c.ParamsInt("quotation_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid quotation_id"})
	}
	type reqBody struct {
		Items                []domain.QuotationItem `json:"items"`
		DiscountAmount       float64                `json:"discount_amount"`
		ShippingCost         float64                `json:"shipping_cost"`
		ShippingMethod       *string                `json:"shipping_method"`
		PackagingCost        float64                `json:"packaging_cost"`
		ToolingMoldCost      float64                `json:"tooling_mold_cost"`
		LeadTimeDays         *int64                 `json:"lead_time_days"`
		ProductionStartDate  *string                `json:"production_start_date"`
		DeliveryDate         *string                `json:"delivery_date"`
		Incoterms            *string                `json:"incoterms"`
		PaymentTerms         *string                `json:"payment_terms"`
		ValidityDays         int                    `json:"validity_days"`
		WarrantyPeriodMonths *int                   `json:"warranty_period_months"`
	}
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	item := &domain.Quotation{
		FactoryID:            userID,
		Items:                req.Items,
		DiscountAmount:       req.DiscountAmount,
		ShippingCost:         req.ShippingCost,
		ShippingMethod:       req.ShippingMethod,
		PackagingCost:        req.PackagingCost,
		ToolingMoldCost:      req.ToolingMoldCost,
		Incoterms:            req.Incoterms,
		PaymentTerms:         req.PaymentTerms,
		ValidityDays:         req.ValidityDays,
		WarrantyPeriodMonths: req.WarrantyPeriodMonths,
	}
	if req.LeadTimeDays != nil {
		item.LeadTimeDays = *req.LeadTimeDays
	}
	if err := h.service.CreateRevision(int64(parentID), userID, item); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *QuotationHandler) Accept(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	quotationID, err := c.ParamsInt("quotation_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid quotation_id"})
	}
	order, err := h.service.Accept(int64(quotationID), userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(order)
}

func (h *QuotationHandler) Reject(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	quotationID, err := c.ParamsInt("quotation_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid quotation_id"})
	}
	if err := h.service.Reject(int64(quotationID), userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "quotation rejected"})
}

func (h *QuotationHandler) PatchQuotation(c *fiber.Ctx) error {
	type reqBody struct {
		PricePerPiece    float64 `json:"price_per_piece"`
		MoldCost         float64 `json:"mold_cost"`
		LeadTimeDays     int64   `json:"lead_time_days"`
		ShippingMethodID int64   `json:"shipping_method_id"`
		Reason           string  `json:"reason"`
	}
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid X-User-ID header"})
	}
	quotationID, err := c.ParamsInt("quotation_id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid quotation_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	item, err := h.service.PatchBody(int64(quotationID), userID, req.PricePerPiece, req.MoldCost, req.LeadTimeDays, req.ShippingMethodID, req.Reason)
	if err != nil {
		if errors.Is(err, service.ErrQuotationPatchReason) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrQuotationLocked) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrNotQuotationParty) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrInvalidShippingMethod) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update quotation"})
	}
	return c.JSON(item)
}

func (h *QuotationHandler) PatchQuotationStatus(c *fiber.Ctx) error {
	type reqBody struct {
		Status string `json:"status"`
	}
	userID, err := getUserIDFromHeader(c)
	var editor *int64
	if err == nil {
		editor = &userID
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
	if err := h.service.UpdateStatus(int64(quotationID), status, editor); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update quotation status"})
	}
	return c.JSON(fiber.Map{"message": "quotation status updated"})
}

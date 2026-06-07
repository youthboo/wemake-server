package quotation

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/dbutil"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/dto"
	handlerregistry "github.com/yourusername/wemake/internal/handler/errorregistry"
	"github.com/yourusername/wemake/internal/helper"
	authservice "github.com/yourusername/wemake/internal/service/auth"
	quotationservice "github.com/yourusername/wemake/internal/service/quotation"
)

type QuotationHandler struct {
	service *quotationservice.QuotationService
	auth    *authservice.AuthService
}

func NewQuotationHandler(quotationService *quotationservice.QuotationService, authService *authservice.AuthService) *QuotationHandler {
	return &QuotationHandler{service: quotationService, auth: authService}
}

func (h *QuotationHandler) CreateQuotation(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}

	rfqID, err := helper.RequireInt64Param(c, "rfq_id")
	if err != nil {
		return err
	}

	var req dto.CreateQuotationRequest
	if err := helper.ParseAndValidateBody(c, &req, map[string]string{
		"FactoryID":     "factory_id and lead_time_days are required; price_per_piece must be >= 0",
		"PricePerPiece": "factory_id and lead_time_days are required; price_per_piece must be >= 0",
		"LeadTimeDays":  "factory_id and lead_time_days are required; price_per_piece must be >= 0",
	}); err != nil {
		return err
	}
	if req.FactoryID != userID {
		return helper.ForbiddenError(c, "factory_id must match authenticated user")
	}

	// tooling_mold_cost takes precedence over legacy mold_cost
	moldCost := req.MoldCost
	if req.ToolingMoldCost > 0 {
		moldCost = req.ToolingMoldCost
	}
	validityDays := req.ValidityDays
	if validityDays <= 0 {
		validityDays = domain.DefaultQuotationValidityDays
	}

	item := &domain.Quotation{
		RFQID:            int64(rfqID),
		FactoryID:        req.FactoryID,
		PricePerPiece:    helper.MoneyDecimal(req.PricePerPiece),
		MoldCost:         helper.MoneyDecimal(moldCost),
		ToolingMoldCost:  helper.MoneyDecimal(req.ToolingMoldCost),
		ShippingCost:     helper.MoneyDecimal(req.ShippingCost),
		PackagingCost:    helper.MoneyDecimal(req.PackagingCost),
		LeadTimeDays:     req.LeadTimeDays,
		ValidityDays:     validityDays,
		ShippingMethodID: req.ShippingMethodID,
		PaymentTerms:     req.PaymentTerms,
		ImageURLs:        req.ImageURLs,
		FactoryHighlight: req.FactoryHighlight,
		FactoryQty:       req.FactoryQty,
		FactoryUnitID:    req.FactoryUnitID,
	}
	if err := h.service.Create(item); err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to create quotation"), handlerregistry.CreateQuotationErrorMap())
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *QuotationHandler) ListQuotationsByRFQ(c *fiber.Ctx) error {
	rfqID, err := helper.RequireInt64Param(c, "rfq_id")
	if err != nil {
		return err
	}
	items, err := h.service.ListByRFQID(int64(rfqID))
	if err != nil {
		return helper.InternalServerError(c, "failed to fetch quotations")
	}
	rfqMeta := fiber.Map{"rfq_id": int64(rfqID)}
	if len(items) > 0 {
		rfqMeta["request_kind"] = items[0].RequestKind
		rfqMeta["sample_qty"] = items[0].SampleQty
		rfqMeta["status"] = items[0].RFQStatus
	}
	return c.JSON(fiber.Map{
		"rfq":        rfqMeta,
		"quotations": items,
	})
}

func (h *QuotationHandler) ListMine(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	status := helper.QueryString(c, "status")
	items, err := h.service.ListMine(userID, status)
	if err != nil {
		return helper.InternalServerError(c, "failed to fetch quotations")
	}
	return c.JSON(items)
}

func (h *QuotationHandler) ListCollection(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	factoryID, err := helper.RequireQueryMatchingSelfOrID(c, "factory_id", userID, "factory_id query is required", "factory_id must match authenticated factory")
	if err != nil {
		return err
	}
	items, err := h.service.ListMine(factoryID, helper.QueryString(c, "status"))
	if err != nil {
		return helper.InternalServerError(c, "failed to fetch quotations")
	}
	return c.JSON(items)
}

func (h *QuotationHandler) GetQuotation(c *fiber.Ctx) error {
	quotationID, err := helper.RequireInt64Param(c, "quotation_id")
	if err != nil {
		return err
	}
	item, err := h.service.GetByID(int64(quotationID))
	if err != nil {
		if dbutil.IsNotFoundError(err) {
			return helper.WriteAPIError(c, helper.NotFoundAPIError("QUOTATION_NOT_FOUND", "quotation not found"))
		}
		return helper.WriteAPIError(c, helper.InternalServerAPIError("FETCH_FAILED", "failed to fetch quotation"))
	}
	return c.JSON(item)
}

func (h *QuotationHandler) ListHistory(c *fiber.Ctx) error {
	userID, u, err := helper.RequireUser(c, h.auth)
	if err != nil {
		return err
	}
	quotationID, err := helper.RequireInt64Param(c, "quotation_id")
	if err != nil {
		return err
	}
	ok, err := h.service.CanView(int64(quotationID), userID, u.Role)
	if err != nil {
		if dbutil.IsNotFoundError(err) {
			return helper.WriteAPIError(c, helper.NotFoundAPIError("QUOTATION_NOT_FOUND", "quotation not found"))
		}
		return helper.WriteAPIError(c, helper.InternalServerAPIError("AUTH_FAILED", "failed to authorize"))
	}
	if !ok {
		return helper.ForbiddenError(c, "not authorized")
	}
	items, err := h.service.ListHistory(int64(quotationID))
	if err != nil {
		return helper.InternalServerError(c, "failed to fetch history")
	}
	return c.JSON(items)
}

func (h *QuotationHandler) Preview(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	var req dto.PreviewQuotationRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	item, err := h.service.Preview(req.Items, req.DiscountAmount, req.ShippingCost, req.PackagingCost, req.ToolingMoldCost, &userID)
	if err != nil {
		return helper.BadRequestError(c, err.Error())
	}
	return c.JSON(item)
}

func (h *QuotationHandler) CreateDetailed(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	var req dto.CreateDetailedQuotationRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	item := &domain.Quotation{
		RFQID:                req.RFQID,
		FactoryID:            userID,
		Items:                req.Items,
		DiscountAmount:       helper.MoneyDecimal(req.DiscountAmount),
		ShippingCost:         helper.MoneyDecimal(req.ShippingCost),
		ShippingMethod:       req.ShippingMethod,
		PackagingCost:        helper.MoneyDecimal(req.PackagingCost),
		ToolingMoldCost:      helper.MoneyDecimal(req.ToolingMoldCost),
		Incoterms:            req.Incoterms,
		PaymentTerms:         req.PaymentTerms,
		ValidityDays:         req.ValidityDays,
		WarrantyPeriodMonths: req.WarrantyPeriodMonths,
		FactoryHighlight:     req.FactoryHighlight,
		FactoryNote:          req.FactoryNote,
		FactoryQty:           req.FactoryQty,
		FactoryUnitID:        req.FactoryUnitID,
	}
	helper.AssignIfNotNil(&item.LeadTimeDays, req.LeadTimeDays)
	if req.ShippingMethodID != nil && *req.ShippingMethodID > 0 {
		item.ShippingMethodID = *req.ShippingMethodID
	}
	if helper.DereferenceString(req.ProductionStartDate, "") != "" {
		d, err := helper.ParseDate(*req.ProductionStartDate, "production_start_date")
		if err != nil {
			return helper.BadRequestError(c, "production_start_date must be YYYY-MM-DD")
		}
		item.ProductionStartDate = &d
	}
	if helper.DereferenceString(req.DeliveryDate, "") != "" {
		d, err := helper.ParseDate(*req.DeliveryDate, "delivery_date")
		if err != nil {
			return helper.BadRequestError(c, "delivery_date must be YYYY-MM-DD")
		}
		item.DeliveryDate = &d
	}
	if err := h.service.CreateDetailed(item); err != nil {
		slog.Error("CreateDetailed failed", "error", err, "rfq_id", item.RFQID, "factory_id", item.FactoryID)
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusBadRequest, "failed to create detailed quotation"), handlerregistry.CreateDetailedErrorMap())
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *QuotationHandler) CreateRevision(c *fiber.Ctx) error {
	parentID, err := helper.RequireInt64Param(c, "quotation_id")
	if err != nil {
		return err
	}
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	var req dto.CreateRevisionQuotationRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	item := &domain.Quotation{
		FactoryID:            userID,
		Items:                req.Items,
		DiscountAmount:       helper.MoneyDecimal(req.DiscountAmount),
		ShippingCost:         helper.MoneyDecimal(req.ShippingCost),
		ShippingMethod:       req.ShippingMethod,
		PackagingCost:        helper.MoneyDecimal(req.PackagingCost),
		ToolingMoldCost:      helper.MoneyDecimal(req.ToolingMoldCost),
		Incoterms:            req.Incoterms,
		PaymentTerms:         req.PaymentTerms,
		ValidityDays:         req.ValidityDays,
		WarrantyPeriodMonths: req.WarrantyPeriodMonths,
	}
	helper.AssignIfNotNil(&item.LeadTimeDays, req.LeadTimeDays)
	if err := h.service.CreateRevision(int64(parentID), userID, item); err != nil {
		return helper.BadRequestError(c, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *QuotationHandler) Accept(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	quotationID, err := helper.RequireInt64Param(c, "quotation_id")
	if err != nil {
		return err
	}
	order, err := h.service.Accept(int64(quotationID), userID)
	if err != nil {
		return helper.BadRequestError(c, err.Error())
	}
	return c.JSON(order)
}

func (h *QuotationHandler) Reject(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	quotationID, err := helper.RequireInt64Param(c, "quotation_id")
	if err != nil {
		return err
	}
	if err := h.service.Reject(int64(quotationID), userID); err != nil {
		return helper.BadRequestError(c, err.Error())
	}
	return c.JSON(fiber.Map{"message": "quotation rejected"})
}

func (h *QuotationHandler) PatchQuotation(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	quotationID, err := helper.RequireInt64Param(c, "quotation_id")
	if err != nil {
		return err
	}
	var req dto.PatchQuotationBodyRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	// tooling_mold_cost takes precedence over legacy mold_cost
	moldCost := req.MoldCost
	if req.ToolingMoldCost > 0 {
		moldCost = req.ToolingMoldCost
	}
	item, err := h.service.PatchBody(
		int64(quotationID),
		userID,
		req.PricePerPiece,
		moldCost,
		req.ShippingCost,
		req.PackagingCost,
		req.ToolingMoldCost,
		req.LeadTimeDays,
		req.ShippingMethodID,
		req.PaymentTerms,
		req.FactoryHighlight,
		req.Reason,
		req.ValidityDays,
		req.FactoryNote,
		req.FactoryQty, req.FactoryUnitID,
	)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to update quotation"), handlerregistry.PatchQuotationErrorMap())
	}
	// Update image_urls if provided in request
	if req.ImageURLs != nil {
		if imgErr := h.service.UpdateImageURLs(int64(quotationID), req.ImageURLs); imgErr != nil {
			// non-fatal — log but return existing item
			_ = imgErr
		} else {
			item.ImageURLs = req.ImageURLs
		}
	}
	return c.JSON(item)
}

// PatchFactoryNote updates only factory_note — no lock/status check.
// PATCH /quotations/:quotation_id/factory-note
func (h *QuotationHandler) PatchFactoryNote(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	quotationID, err := helper.RequireInt64Param(c, "quotation_id")
	if err != nil {
		return err
	}
	var body struct {
		FactoryNote *string `json:"factory_note"`
	}
	if err := c.BodyParser(&body); err != nil {
		return helper.BadRequestError(c, "invalid request body")
	}
	if err := h.service.PatchFactoryNote(int64(quotationID), userID, body.FactoryNote); err != nil {
		if dbutil.IsNotFoundError(err) {
			return helper.WriteAPIError(c, helper.NotFoundAPIError("QUOTATION_NOT_FOUND", "quotation not found"))
		}
		return helper.InternalServerError(c, "failed to update factory note")
	}
	return c.JSON(fiber.Map{"message": "factory note updated"})
}

func (h *QuotationHandler) PatchQuotationStatus(c *fiber.Ctx) error {
	var editor *int64
	if userID := helper.OptionalActorID(c); userID > 0 {
		editor = &userID
	}
	quotationID, err := helper.RequireInt64Param(c, "quotation_id")
	if err != nil {
		return err
	}
	var req dto.PatchQuotationStatusRequest
	if err := helper.ParseAndValidateBody(c, &req, map[string]string{
		"Status": "status must be PD, AC, RJ or EX",
	}); err != nil {
		return err
	}
	status := domainutil.NormalizeStatus(req.Status)
	v := domain.NewValidationCollector()
	v.AddIf(!domainutil.StatusIn(status, domain.QuotationStatusAccepted, domain.QuotationStatusRejected, domain.QuotationStatusPrepared, domain.QuotationStatusExpired), "status", "must be PD, AC, RJ or EX")
	if err := helper.ValidateRequest(c, v); err != nil {
		return err
	}
	if err := h.service.UpdateStatus(int64(quotationID), status, editor); err != nil {
		return helper.InternalServerError(c, "failed to update quotation status")
	}
	return c.JSON(fiber.Map{"message": "quotation status updated"})
}

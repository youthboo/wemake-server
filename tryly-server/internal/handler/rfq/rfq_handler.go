package rfq

import (
	"database/sql"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/lib/pq"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/dto"
	"github.com/yourusername/wemake/internal/helper"
	authservice "github.com/yourusername/wemake/internal/service/auth"
	quotationservice "github.com/yourusername/wemake/internal/service/quotation"
	rfqservice "github.com/yourusername/wemake/internal/service/rfq"
)

type RFQHandler struct {
	service          *rfqservice.RFQService
	quotationService *quotationservice.QuotationService
	auth             *authservice.AuthService
}

func NewRFQHandler(rfqService *rfqservice.RFQService, quotationService *quotationservice.QuotationService, authService *authservice.AuthService) *RFQHandler {
	return &RFQHandler{service: rfqService, quotationService: quotationService, auth: authService}
}

var rfqCreateErrorMap = map[error]helper.ErrorResponse{
	rfqservice.ErrInvalidSubCategory:    helper.ErrorMessage(fiber.StatusBadRequest, rfqservice.ErrInvalidSubCategory.Error()),
	rfqservice.ErrInvalidCategory:       helper.ErrorMessage(fiber.StatusBadRequest, rfqservice.ErrInvalidCategory.Error()),
	rfqservice.ErrInvalidShippingMethod: helper.ErrorMessage(fiber.StatusBadRequest, rfqservice.ErrInvalidShippingMethod.Error()),
	rfqservice.ErrMaxRFQReferenceImages: helper.ErrorMessage(fiber.StatusBadRequest, rfqservice.ErrMaxRFQReferenceImages.Error()),
	rfqservice.ErrRFQDetailsRequired:    helper.ErrorMessage(fiber.StatusBadRequest, rfqservice.ErrRFQDetailsRequired.Error()),
	rfqservice.ErrRFQDetailsTooShort:    helper.ErrorMessage(fiber.StatusBadRequest, rfqservice.ErrRFQDetailsTooShort.Error()),
	rfqservice.ErrRFQKindInvalid:        helper.ErrorMessage(fiber.StatusBadRequest, rfqservice.ErrRFQKindInvalid.Error()),
	rfqservice.ErrRFQSampleQtyInvalid:   helper.ErrorMessage(fiber.StatusBadRequest, rfqservice.ErrRFQSampleQtyInvalid.Error()),
	rfqservice.ErrRFQWrongScope:         helper.ErrorMessage(fiber.StatusBadRequest, rfqservice.ErrRFQWrongScope.Error()),
}

var rfqPreviewErrorMap = map[error]helper.ErrorResponse{
	rfqservice.ErrRFQKindInvalid:     helper.ErrorMessage(fiber.StatusBadRequest, "INVALID_KIND"),
	rfqservice.ErrRFQWrongScope:      helper.ErrorMessage(fiber.StatusBadRequest, "WRONG_SCOPE"),
	rfqservice.ErrInvalidSubCategory: helper.ErrorMessage(fiber.StatusNotFound, "CATEGORY_NOT_FOUND"),
	rfqservice.ErrInvalidCategory:    helper.ErrorMessage(fiber.StatusNotFound, "CATEGORY_NOT_FOUND"),
}

var rfqDismissErrorMap = map[error]helper.ErrorResponse{
	rfqservice.ErrHasActiveQuotation: helper.ErrorMessage(fiber.StatusConflict, "HAS_ACTIVE_QUOTATION"),
	rfqservice.ErrQuotationAccepted:  helper.ErrorMessage(fiber.StatusConflict, "QUOTATION_ACCEPTED"),
	sql.ErrNoRows:                    helper.ErrorMessage(fiber.StatusNotFound, "RFQ_NOT_FOUND"),
}

var rfqNotFoundErrorMap = map[error]helper.ErrorResponse{
	sql.ErrNoRows: helper.ErrorMessage(fiber.StatusNotFound, "rfq not found"),
}

var rfqNotFoundCodeErrorMap = map[error]helper.ErrorResponse{
	sql.ErrNoRows: helper.ErrorMessage(fiber.StatusNotFound, "RFQ_NOT_FOUND"),
}

func (h *RFQHandler) CreateRFQ(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}

	var req dto.CreateRFQRequest
	if err := helper.ParseAndValidateBody(c, &req, map[string]string{
		"CategoryID": "category_id, title, and quantity are required",
		"Title":      "category_id, title, and quantity are required",
		"Quantity":   "category_id, title, and quantity are required",
	}); err != nil {
		return err
	}

	details := helper.DereferenceString(&req.Details, "")
	if details == "" {
		details = helper.DereferenceString(&req.Description, "")
	}

	rfq := &domain.RFQ{
		UserID:                 userID,
		CategoryID:             req.CategoryID,
		SubCategoryID:          req.SubCategoryID,
		Title:                  req.Title,
		Quantity:               req.Quantity,
		UnitID:                 req.UnitID,
		Details:                details,
		AddressID:              req.AddressID,
		ShippingMethodID:       req.ShippingMethodID,
		MaterialGrade:          req.MaterialGrade,
		TargetPrice:            helper.MoneyDecimalPtrFromFloat64(req.TargetPrice),
		TargetLeadTimeDays:     req.TargetLeadTimeDays,
		DeliveryAddressID:      req.DeliveryAddressID,
		CertificationsRequired: req.CertificationsRequired,
		ReferenceImages:        req.ReferenceImages,
		RequestKind:            req.RequestKind,
		Targeting:              req.Targeting,
		TargetFactoryIDs:       req.FactoryIDs,
	}
	if rfq.DeliveryAddressID == nil {
		rfq.DeliveryAddressID = &rfq.AddressID
	}
	if helper.DereferenceString(req.RequiredDeliveryDate, "") != "" {
		d, err := helper.ParseDate(*req.RequiredDeliveryDate, "required_delivery_date")
		if err != nil {
			return helper.BadRequestError(c, "required_delivery_date must be YYYY-MM-DD")
		}
		rfq.RequiredDeliveryDate = &d
	}

	if err := h.service.Create(rfq); err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to create rfq"), rfqCreateErrorMap)
	}
	domain.EnrichRFQBudgetFields(rfq)
	return c.Status(fiber.StatusCreated).JSON(rfq)
}

// UpdateRFQTargets replaces the target factory list for a specific-targeting RFQ.
// PUT /api/v1/rfqs/:rfq_id/targets
func (h *RFQHandler) UpdateRFQTargets(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	rfqID, err := helper.RequireInt64Param(c, "rfq_id")
	if err != nil {
		return err
	}
	var req dto.UpdateRFQTargetsRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}

	if err := h.service.UpdateTargets(userID, rfqID, req.FactoryIDs); err != nil {
		return helper.MapServiceError(c, err,
			helper.ErrorMessage(fiber.StatusInternalServerError, "failed to update targets"),
			map[error]helper.ErrorResponse{
				rfqservice.ErrRFQNotEditable: helper.ErrorMessage(fiber.StatusConflict, "RFQ_NOT_EDITABLE"),
				sql.ErrNoRows:                helper.ErrorMessage(fiber.StatusNotFound, "RFQ_NOT_FOUND"),
			},
		)
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *RFQHandler) PatchRFQ(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	rfqID, err := helper.RequireInt64Param(c, "rfq_id")
	if err != nil {
		return err
	}
	var req dto.PatchRFQRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	details := helper.DereferenceString(req.Details, "")
	if details == "" && req.Description != nil {
		details = helper.DereferenceString(req.Description, "")
	}
	rfq := &domain.RFQ{
		Details: details,
	}
	if req.CategoryID != nil {
		rfq.CategoryID = *req.CategoryID
	}
	if req.SubCategoryID != nil {
		rfq.SubCategoryID = req.SubCategoryID
	}
	if req.Title != nil {
		rfq.Title = *req.Title
	}
	if req.Quantity != nil {
		rfq.Quantity = *req.Quantity
	}
	if req.UnitID != nil {
		rfq.UnitID = req.UnitID
	}
	if req.MaterialGrade != nil {
		rfq.MaterialGrade = req.MaterialGrade
	}
	if req.TargetPrice != nil {
		rfq.TargetPrice = helper.MoneyDecimalPtrFromFloat64(req.TargetPrice)
	}
	if req.TargetLeadTimeDays != nil {
		rfq.TargetLeadTimeDays = req.TargetLeadTimeDays
	}
	if req.ReferenceImages != nil {
		rfq.ReferenceImages = pq.StringArray(req.ReferenceImages)
	}
	if helper.DereferenceString(req.RequiredDeliveryDate, "") != "" {
		d, err := helper.ParseDate(*req.RequiredDeliveryDate, "required_delivery_date")
		if err != nil {
			return helper.BadRequestError(c, "required_delivery_date must be YYYY-MM-DD")
		}
		rfq.RequiredDeliveryDate = &d
	}
	if err := h.service.Patch(userID, int64(rfqID), rfq); err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusBadRequest, err.Error()), rfqCreateErrorMap)
	}
	domain.EnrichRFQBudgetFields(rfq)
	return c.JSON(rfq)
}

func (h *RFQHandler) ListRFQs(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	query := helper.QueryParams(c)
	status := query.String("status")
	rfqs, err := h.service.ListByUserID(userID, status)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch rfqs")
	}
	kind := domainutil.NormalizeStatus(query.String("kind"))
	if kind != "" {
		if kind != domain.RequestKindProduction && kind != domain.RequestKindProductSample && kind != domain.RequestKindMaterialSample {
			return helper.BadRequestError(c, "INVALID_KIND")
		}
		filtered := make([]domain.RFQ, 0, len(rfqs))
		for _, item := range rfqs {
			if strings.EqualFold(item.RequestKind, kind) {
				filtered = append(filtered, item)
			}
		}
		rfqs = filtered
	}
	return c.JSON(rfqs)
}

func (h *RFQHandler) PreviewFactories(c *fiber.Ctx) error {
	query := helper.QueryParams(c)
	kind := query.String("kind")
	if kind == "" {
		return helper.BadRequestError(c, "INVALID_KIND")
	}
	categoryID := query.RequiredPositiveInt64("category_id")
	if query.Err() != nil {
		return helper.BadRequestError(c, "MISSING_CATEGORY")
	}
	subCategoryID := query.OptionalPositiveInt64("sub_category_id")
	if err := query.Err(); err != nil {
		return err
	}
	result, err := h.service.PreviewFactories(kind, categoryID, subCategoryID)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to preview factories"), rfqPreviewErrorMap)
	}
	return c.JSON(result)
}

func (h *RFQHandler) ListMatching(c *fiber.Ctx) error {
	userID, _, err := helper.RequireFactoryUser(c, h.auth)
	if err != nil {
		return err
	}
	query := helper.QueryParams(c)
	status := query.String("status")
	showDismissed := strings.EqualFold(query.String("show_dismissed"), "true")
	items, err := h.service.ListMatchingForFactory(userID, status, query.String("kind"), showDismissed)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to fetch matching rfqs"), map[error]helper.ErrorResponse{
			rfqservice.ErrRFQKindInvalid: helper.ErrorMessage(fiber.StatusBadRequest, "INVALID_KIND"),
		})
	}
	return c.JSON(items)
}

func (h *RFQHandler) DismissRFQ(c *fiber.Ctx) error {
	userID, _, err := helper.RequireFactoryUser(c, h.auth)
	if err != nil {
		return err
	}
	rfqID, err := helper.RequireInt64Param(c, "rfq_id")
	if err != nil {
		return err
	}
	item, created, err := h.service.DismissRFQ(userID, int64(rfqID))
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to dismiss rfq"), rfqDismissErrorMap)
	}
	status := fiber.StatusOK
	if created {
		status = fiber.StatusCreated
	}
	return c.Status(status).JSON(fiber.Map{
		"rfq_id":       item.RFQID,
		"dismissed":    true,
		"dismissed_at": item.DismissedAt,
	})
}

func (h *RFQHandler) UndismissRFQ(c *fiber.Ctx) error {
	userID, _, err := helper.RequireFactoryUser(c, h.auth)
	if err != nil {
		return err
	}
	rfqID, err := helper.RequireInt64Param(c, "rfq_id")
	if err != nil {
		return err
	}
	if err := h.service.UndismissRFQ(userID, int64(rfqID)); err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to undismiss rfq"), rfqNotFoundCodeErrorMap)
	}
	return c.JSON(fiber.Map{"rfq_id": int64(rfqID), "dismissed": false})
}

func (h *RFQHandler) GetRFQ(c *fiber.Ctx) error {
	userID, u, err := helper.RequireUser(c, h.auth)
	if err != nil {
		return err
	}
	if u == nil {
		return helper.JSONError(c, 401, "user not found")
	}
	rfqID, err := helper.RequireInt64Param(c, "rfq_id")
	if err != nil {
		return err
	}

	rfq, err := h.service.GetForViewer(userID, u.Role, int64(rfqID))
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to fetch rfq"), rfqNotFoundErrorMap)
	}

	return c.JSON(fiber.Map{"rfq": rfq})
}

// GetDetail handles GET /rfqs/:rfq_id/detail
// Returns RFQ + quotations + per-quote history in a single call.
func (h *RFQHandler) GetDetail(c *fiber.Ctx) error {
	userID, u, err := helper.RequireUser(c, h.auth)
	if err != nil {
		return err
	}
	if u == nil {
		return helper.JSONError(c, 401, "user not found")
	}
	rfqID, err := helper.RequireInt64Param(c, "rfq_id")
	if err != nil {
		return err
	}

	rfq, err := h.service.GetForViewer(userID, u.Role, rfqID)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to fetch rfq"), rfqNotFoundErrorMap)
	}

	quotes, err := h.quotationService.ListByRFQID(rfqID)
	if err != nil {
		quotes = []domain.Quotation{}
	}
	if quotes == nil {
		quotes = []domain.Quotation{}
	}

	quoteHistories, err := h.quotationService.HistoriesForQuotes(quotes)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to fetch quotation histories"), rfqNotFoundErrorMap)
	}

	return c.JSON(domain.RFQDetailBundle{
		RFQ:            rfq,
		Quotations:     quotes,
		QuoteHistories: quoteHistories,
	})
}

func (h *RFQHandler) CancelRFQ(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	rfqID, err := helper.RequireInt64Param(c, "rfq_id")
	if err != nil {
		return err
	}

	if err := h.service.Cancel(userID, int64(rfqID)); err != nil {
		return helper.InternalServerError(c, "failed to cancel rfq")
	}
	return c.JSON(fiber.Map{"message": "rfq canceled"})
}

// CloseRFQ lets the customer manually close (stop accepting new quotes) an open RFQ.
// PATCH /rfqs/:rfq_id/close
func (h *RFQHandler) CloseRFQ(c *fiber.Ctx) error {
	userID, err := helper.RequireUserID(c)
	if err != nil {
		return err
	}
	rfqID, err := helper.RequireInt64Param(c, "rfq_id")
	if err != nil {
		return err
	}

	if err := h.service.Close(userID, int64(rfqID)); err != nil {
		return helper.InternalServerError(c, "failed to close rfq")
	}
	return c.JSON(fiber.Map{"message": "rfq closed"})
}

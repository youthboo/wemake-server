package showcase

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yourusername/wemake/internal/helper"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/dto"
	"github.com/yourusername/wemake/internal/logger"
	showcaseservice "github.com/yourusername/wemake/internal/service/showcase"
)

// Allowed values for GET /showcases?type= (matches factory_showcases.content_type)
var showcaseTypeQueryAllowed = map[string]struct{}{
	"PD": {}, "PM": {}, "ID": {}, "MT": {},
}

// showcaseStatusAllowed are valid values for the status field.
var showcaseStatusAllowed = map[string]struct{}{
	"AC": {}, "DR": {}, "HI": {}, "AR": {},
}

const (
	errInvalidShowcaseID  = "invalid showcase_id"
	errShowcaseNotFound   = "showcase not found"
	errFetchShowcases     = "failed to fetch showcases"
	errInvalidPayload     = "invalid payload"
	errNotYourShowcase    = "not your showcase"
	errInvalidStatus      = "invalid status: use AC, DR, HI, or AR"
	errInvalidTypeQuery   = "invalid query type: use PD (product), PM (promotion), ID (idea), or MT (material); omit type for all showcases"
	errInvalidTypeFactory = "invalid query type: use PD, PM, ID, or MT; omit type for all showcases for this factory"
)

type ShowcaseHandler struct {
	service *showcaseservice.ShowcaseService
}

func NewShowcaseHandler(service *showcaseservice.ShowcaseService) *ShowcaseHandler {
	return &ShowcaseHandler{service: service}
}

type linkedShowcaseObject struct {
	ImageURL  string `json:"image_url"`
	SortOrder int    `json:"sort_order"`
	IsCover   bool   `json:"is_cover"`
}

func parseLinkedShowcases(raw json.RawMessage) (*[]string, *domain.ShowcaseValidationDetail) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	// Format 1: ["https://...", "123"]
	var asStrings []string
	if err := helper.UnmarshalJSON(raw, &asStrings); err == nil {
		out := make([]string, 0, len(asStrings))
		for _, item := range asStrings {
			v := strings.TrimSpace(item)
			if v == "" {
				continue
			}
			out = append(out, v)
		}
		return &out, nil
	}

	// Format 1b: [31, "32", "https://..."] mixed numeric/string array
	var asMixed []interface{}
	if err := helper.UnmarshalJSON(raw, &asMixed); err == nil {
		out := make([]string, 0, len(asMixed))
		for _, item := range asMixed {
			switch v := item.(type) {
			case string:
				trimmed := strings.TrimSpace(v)
				if trimmed != "" {
					out = append(out, trimmed)
				}
			case float64:
				if v > 0 {
					out = append(out, strconv.FormatInt(int64(v), 10))
				}
			default:
				return nil, &domain.ShowcaseValidationDetail{
					Field:   "linked_showcases",
					Message: "all entries must be HTTPS URLs or numeric showcase IDs",
				}
			}
		}
		return &out, nil
	}

	// Format 2: [{ "image_url": "...", "sort_order": 1, "is_cover": true }]
	var asObjects []linkedShowcaseObject
	if err := helper.UnmarshalJSON(raw, &asObjects); err == nil {
		type normalized struct {
			URL      string
			Sort     int
			Cover    bool
			Position int
		}
		norm := make([]normalized, 0, len(asObjects))
		for idx, item := range asObjects {
			url := strings.TrimSpace(item.ImageURL)
			if url == "" {
				continue
			}
			norm = append(norm, normalized{
				URL:      url,
				Sort:     item.SortOrder,
				Cover:    item.IsCover,
				Position: idx,
			})
		}
		sort.SliceStable(norm, func(i, j int) bool {
			if norm[i].Cover != norm[j].Cover {
				return norm[i].Cover
			}
			if norm[i].Sort != norm[j].Sort {
				return norm[i].Sort < norm[j].Sort
			}
			return norm[i].Position < norm[j].Position
		})
		out := make([]string, 0, len(norm))
		for _, item := range norm {
			out = append(out, item.URL)
		}
		return &out, nil
	}

	return nil, &domain.ShowcaseValidationDetail{
		Field:   "linked_showcases",
		Message: fmt.Sprintf("%s", "must be an array of strings, numbers, or objects with image_url"),
	}
}

func parseShowcaseDate(raw *string, field string) (*time.Time, *domain.ShowcaseValidationDetail) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	t, err := helper.ParseDate(trimmed, field)
	if err != nil {
		return nil, &domain.ShowcaseValidationDetail{Field: field, Message: "must use YYYY-MM-DD format"}
	}
	return &t, nil
}

func showcaseWriteRequestToInput(r dto.ShowcaseWriteRequest) (domain.ShowcaseWriteInput, []domain.ShowcaseValidationDetail) {
	var details []domain.ShowcaseValidationDetail
	typeValue := r.Type
	if typeValue == nil {
		typeValue = r.ContentType
	}
	startDate, detail := parseShowcaseDate(r.StartDate, "start_date")
	if detail != nil {
		details = append(details, *detail)
	}
	endDate, detail := parseShowcaseDate(r.EndDate, "end_date")
	if detail != nil {
		details = append(details, *detail)
	}
	linkedShowcases, detail := parseLinkedShowcases(r.LinkedShowcases)
	if detail != nil {
		details = append(details, *detail)
	}
	return domain.ShowcaseWriteInput{
		ContentType:     typeValue,
		Status:          r.Status,
		Title:           r.Title,
		CategoryID:      r.CategoryID,
		SubCategoryID:   r.SubCategoryID,
		MOQ:             r.MOQ,
		UnitID:          r.UnitID,
		LeadTimeDays:    r.LeadTimeDays,
		BasePrice:       r.BasePrice,
		PromoPrice:      r.PromoPrice,
		StartDate:       startDate,
		EndDate:         endDate,
		Content:         r.Content,
		LinkedShowcases: linkedShowcases,
		Excerpt:         r.Excerpt,
		ImageURL:        r.ImageURL,
		Tags:            r.Tags,
	}, details
}

func writeShowcaseError(c *fiber.Ctx, err error, fallback string) error {
	return writeShowcaseMappedError(c, err, fallback, showcaseNotFoundErrorMap)
}

func writeShowcaseMappedError(c *fiber.Ctx, err error, fallback string, responses map[error]helper.ErrorResponse) error {
	var validationErr domain.ShowcaseValidationError
	if errors.As(err, &validationErr) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"details": validationErr.Details,
		})
	}
	if !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, domain.ErrForbidden) && !errors.Is(err, domain.ErrImageLimitExceeded) {
		logger.Error("showcase handler failed", "fallback", fallback, "err", err)
	}
	return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, fallback), responses)
}

var showcaseNotFoundErrorMap = map[error]helper.ErrorResponse{
	sql.ErrNoRows: helper.ErrorMessage(fiber.StatusNotFound, errShowcaseNotFound),
}

var showcaseImageCreateErrorMap = map[error]helper.ErrorResponse{
	sql.ErrNoRows:                helper.ErrorMessage(fiber.StatusNotFound, errShowcaseNotFound),
	domain.ErrForbidden:          helper.ErrorMessage(fiber.StatusForbidden, errNotYourShowcase),
	domain.ErrImageLimitExceeded: helper.ErrorMessage(fiber.StatusUnprocessableEntity, domain.ErrImageLimitExceeded.Error()),
}

var showcaseOwnerNotFoundErrorMap = map[error]helper.ErrorResponse{
	sql.ErrNoRows:       helper.ErrorMessage(fiber.StatusNotFound, errShowcaseNotFound),
	domain.ErrForbidden: helper.ErrorMessage(fiber.StatusForbidden, errNotYourShowcase),
}

var showcaseImageNotFoundErrorMap = map[error]helper.ErrorResponse{
	sql.ErrNoRows: helper.ErrorMessage(fiber.StatusNotFound, "image not found"),
}

var showcaseSectionNotFoundErrorMap = map[error]helper.ErrorResponse{
	sql.ErrNoRows: helper.ErrorMessage(fiber.StatusNotFound, "section not found"),
}

// parseContentTypeQuery validates the ?type= query param (shared by List handlers).
func parseContentTypeQuery(c *fiber.Ctx, invalidMsg string) (string, error) {
	ct := helper.QueryString(c, "type")
	if ct != "" {
		if _, ok := showcaseTypeQueryAllowed[ct]; !ok {
			return "", errors.New(invalidMsg)
		}
	}
	return ct, nil
}

func (h *ShowcaseHandler) List(c *fiber.Ctx) error {
	if typesParam := helper.QueryString(c, "types"); typesParam != "" {
		rawTypes := strings.Split(typesParam, ",")
		validTypes := make([]string, 0, len(rawTypes))
		for _, t := range rawTypes {
			t = strings.ToUpper(strings.TrimSpace(t))
			if _, ok := showcaseTypeQueryAllowed[t]; !ok {
				return helper.BadRequestError(c, errInvalidTypeQuery)
			}
			validTypes = append(validTypes, t)
		}

		// Paginated flat list: ?types=PD,MT&page=1&limit=20
		if page := helper.QueryParams(c).Int("page", 0); page > 0 {
			limit := helper.QueryParams(c).Int("limit", 20)
			keyword := helper.QueryString(c, "q")
			sort := helper.QueryString(c, "sort")
			catID, _ := helper.ParseOptionalPositiveInt64Query(c, "category")
			subCatID, _ := helper.ParseOptionalPositiveInt64Query(c, "sub_cat")
			result, err := h.service.ListPaginated(domain.ShowcasePaginatedFilter{
				Types:         validTypes,
				Keyword:       keyword,
				CategoryID:    catID,
				SubCategoryID: subCatID,
				Sort:          sort,
				Limit:         limit,
				Page:          page,
			})
			if err != nil {
				return helper.JSONInternal(c, errFetchShowcases)
			}
			return c.JSON(result)
		}

		// Grouped mode: ?types=PD,MT&limit=8
		limit := helper.QueryParams(c).Int("limit", 8)
		grouped, err := h.service.GetHomeShowcases(validTypes, limit)
		if err != nil {
			return helper.JSONInternal(c, errFetchShowcases)
		}
		return c.JSON(grouped)
	}

	contentType, err := parseContentTypeQuery(c, errInvalidTypeQuery)
	if err != nil {
		return helper.BadRequestError(c, err.Error())
	}
	status := domainutil.NormalizeStatus(helper.QueryString(c, "status"))
	if status != "" {
		if _, ok := showcaseStatusAllowed[status]; !ok {
			return helper.BadRequestError(c, errInvalidStatus)
		}
	}
	factoryID, err := helper.QuerySelfOrID(c, "factory_id")
	if err != nil {
		return err
	}
	var categoryID *int64
	categoryID, err = helper.ParseOptionalPositiveInt64Query(c, "category_id")
	if err != nil {
		return helper.BadRequestError(c, "invalid category_id")
	}
	var subCategoryID *int64
	subCategoryID, err = helper.ParseOptionalPositiveInt64Query(c, "sub_category_id")
	if err != nil {
		return helper.BadRequestError(c, "invalid sub_category_id")
	}
	viewerID := helper.OptionalActorID(c)
	items, err := h.service.ListStructured(domain.ShowcaseListFilter{
		Type:          contentType,
		FactoryID:     factoryID,
		Status:        status,
		CategoryID:    categoryID,
		SubCategoryID: subCategoryID,
		ViewerID:      viewerID,
	})
	if err != nil {
		return helper.JSONInternal(c, errFetchShowcases)
	}
	return c.JSON(items)
}

// GetDetail handles GET /showcases/:showcase_id
func (h *ShowcaseHandler) GetDetail(c *fiber.Ctx) error {
	showcaseID, err := helper.ParsePositiveInt64Param(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	detail, err := h.service.GetDetail(showcaseID)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to fetch showcase"), showcaseNotFoundErrorMap)
	}
	// Non-owner sees only active showcases
	callerID := helper.OptionalActorID(c)
	if detail.Status != "AC" && callerID != detail.FactoryID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
	}
	return c.JSON(detail)
}

// ListByFactory handles GET /factories/:factory_id/showcases
func (h *ShowcaseHandler) ListByFactory(c *fiber.Ctx) error {
	factoryID, err := helper.RequirePathID(c, "factory_id")
	if err != nil {
		return err
	}
	contentType, err := parseContentTypeQuery(c, errInvalidTypeFactory)
	if err != nil {
		return helper.BadRequestError(c, err.Error())
	}
	callerID := helper.OptionalActorID(c)
	items, err := h.service.GetShowcasesByFactory(int64(factoryID), contentType, callerID)
	if err != nil {
		logger.Error("showcase list by factory failed", "factory_id", factoryID, "caller_id", callerID, "content_type", contentType, "err", err)
		return helper.JSONInternal(c, errFetchShowcases)
	}
	return c.JSON(items)
}

func (h *ShowcaseHandler) Create(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	var req dto.ShowcaseWriteRequest
	if err := helper.ParseJSONBody(c, &req, errInvalidPayload); err != nil {
		return err
	}
	input, details := showcaseWriteRequestToInput(req)
	if len(details) > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "VALIDATION_ERROR", "details": details})
	}
	out, err := h.service.CreateStructured(userID, input)
	if err != nil {
		return writeShowcaseError(c, err, "failed to create showcase")
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func (h *ShowcaseHandler) Patch(c *fiber.Ctx) error {
	return h.updateStructured(c, false)
}

func (h *ShowcaseHandler) Put(c *fiber.Ctx) error {
	return h.updateStructured(c, true)
}

func (h *ShowcaseHandler) updateStructured(c *fiber.Ctx, replace bool) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	var req dto.ShowcaseWriteRequest
	if err := helper.ParseJSONBody(c, &req, errInvalidPayload); err != nil {
		return err
	}
	input, details := showcaseWriteRequestToInput(req)
	if len(details) > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "VALIDATION_ERROR", "details": details})
	}
	out, err := h.service.UpdateStructured(showcaseID, userID, input, replace)
	if err != nil {
		return writeShowcaseError(c, err, "failed to update showcase")
	}
	return c.JSON(out)
}

func (h *ShowcaseHandler) PatchStatus(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	var req struct {
		Status string `json:"status" validate:"notblank,statuscode=AC DR HI AR"`
	}
	if err := helper.ParseJSONBody(c, &req, errInvalidPayload); err != nil {
		return err
	}
	if err := helper.ValidateStruct(c, &req, map[string]string{"Status": errInvalidStatus}); err != nil {
		return err
	}
	status := domainutil.NormalizeStatus(req.Status)
	if _, ok := showcaseStatusAllowed[status]; !ok {
		return helper.BadRequestError(c, errInvalidStatus)
	}
	if err := h.service.UpdateStatus(showcaseID, userID, status); err != nil {
		return writeShowcaseError(c, err, "failed to update showcase status")
	}
	out, err := h.service.GetByID(showcaseID, userID)
	if err != nil {
		return c.JSON(fiber.Map{"showcase_id": showcaseID, "status": status})
	}
	return c.JSON(out)
}

func (h *ShowcaseHandler) Delete(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	if err := h.service.Delete(showcaseID, userID); err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to delete showcase"), showcaseNotFoundErrorMap)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ShowcaseHandler) GetAnalytics(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	item, err := h.service.GetAnalytics(showcaseID, userID)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to fetch analytics"), showcaseNotFoundErrorMap)
	}
	return c.JSON(item)
}

// RecordView handles POST /showcases/:showcase_id/view — increment view count
func (h *ShowcaseHandler) RecordView(c *fiber.Ctx) error {
	showcaseID, err := helper.ParsePositiveInt64Param(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	_ = h.service.RecordView(showcaseID) // fire-and-forget; don't surface errors to caller
	return c.JSON(fiber.Map{"message": "view recorded"})
}

func (h *ShowcaseHandler) ListPromoSlides(c *fiber.Ctx) error {
	items, err := h.service.ListPromoSlides()
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch promo slides")
	}
	return c.JSON(items)
}

func (h *ShowcaseHandler) ListHomePromoSlides(c *fiber.Ctx) error {
	limit := helper.QueryParams(c).Int("limit", 5)
	items, err := h.service.ListHomePromoSlides(limit)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch promo slides")
	}
	return c.JSON(items)
}

// CreateImage handles POST /showcases/:showcase_id/images
func (h *ShowcaseHandler) CreateImage(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	var req struct {
		ImageURL  string  `json:"image_url" validate:"notblank"`
		SortOrder int     `json:"sort_order"`
		Caption   *string `json:"caption"`
	}
	if err := helper.ParseAndValidateBodyWithMessage(c, &req, map[string]string{"ImageURL": "image_url is required"}, errInvalidPayload); err != nil {
		return err
	}
	img := &domain.ShowcaseImage{
		ShowcaseID: showcaseID,
		ImageURL:   req.ImageURL,
		SortOrder:  req.SortOrder,
		Caption:    req.Caption,
	}
	if err := h.service.CreateImage(img, userID); err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to add image"), showcaseImageCreateErrorMap)
	}
	return c.Status(fiber.StatusCreated).JSON(img)
}

// ListImages handles GET /showcases/:showcase_id/images
func (h *ShowcaseHandler) ListImages(c *fiber.Ctx) error {
	showcaseID, err := helper.ParsePositiveInt64Param(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	callerID := helper.OptionalActorID(c)
	images, err := h.service.ListImages(showcaseID, callerID)
	if err != nil {
		logger.Error("showcase images fetch failed", "showcase_id", showcaseID, "caller_id", callerID, "err", err)
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to fetch images"), showcaseNotFoundErrorMap)
	}
	return c.JSON(fiber.Map{"images": images})
}

// DeleteImage handles DELETE /showcases/:showcase_id/images/:image_id
func (h *ShowcaseHandler) DeleteImage(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	imageID, err := helper.ParsePositiveInt64Param(c, "image_id")
	if err != nil || imageID <= 0 {
		return helper.BadRequestError(c, "invalid image_id")
	}
	if err := h.service.DeleteImage(showcaseID, imageID, userID); err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to delete image"), showcaseImageNotFoundErrorMap)
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}

// GetSections handles GET /showcases/:showcase_id/sections
func (h *ShowcaseHandler) GetSections(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	sections, err := h.service.GetSections(showcaseID, userID)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to fetch sections"), showcaseOwnerNotFoundErrorMap)
	}
	return c.JSON(fiber.Map{"sections": sections})
}

// BulkReplaceSections handles PUT /showcases/:showcase_id/sections
func (h *ShowcaseHandler) BulkReplaceSections(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	var req struct {
		Sections []domain.ShowcaseSectionInput `json:"sections"`
	}
	if err := helper.ParseBody(c, &req, errInvalidPayload); err != nil {
		return err
	}
	if len(req.Sections) > 10 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "max 10 sections per showcase"})
	}
	if err := h.service.ValidateSectionInputs(req.Sections); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.service.BulkReplaceSections(showcaseID, userID, req.Sections); err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to update sections"), showcaseOwnerNotFoundErrorMap)
	}
	return c.JSON(fiber.Map{"message": "sections updated"})
}

// GetSpecs handles GET /showcases/:showcase_id/specs
func (h *ShowcaseHandler) GetSpecs(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	specs, err := h.service.GetSpecs(showcaseID, userID)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to fetch specs"), showcaseOwnerNotFoundErrorMap)
	}
	return c.JSON(fiber.Map{"specs": specs})
}

// BulkReplaceSpecs handles PUT /showcases/:showcase_id/specs
func (h *ShowcaseHandler) BulkReplaceSpecs(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	var req struct {
		Specs []domain.ShowcaseSpecInput `json:"specs"`
	}
	if err := helper.ParseBody(c, &req, errInvalidPayload); err != nil {
		return err
	}
	for _, s := range req.Specs {
		if helper.DereferenceString(&s.SpecKey, "") == "" {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "spec_key is required"})
		}
		if helper.DereferenceString(&s.SpecValue, "") == "" {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "spec_value is required"})
		}
	}
	if err := h.service.BulkReplaceSpecs(showcaseID, userID, req.Specs); err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to update specs"), showcaseOwnerNotFoundErrorMap)
	}
	return c.JSON(fiber.Map{"message": "specs updated"})
}

// PatchImage handles PATCH /showcases/:showcase_id/images/:image_id
func (h *ShowcaseHandler) PatchImage(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	imageID, err := helper.ParsePositiveInt64Param(c, "image_id")
	if err != nil || imageID <= 0 {
		return helper.BadRequestError(c, "invalid image_id")
	}
	var req struct {
		SortOrder *int    `json:"sort_order"`
		Caption   *string `json:"caption"`
	}
	if err := helper.ParseBody(c, &req, errInvalidPayload); err != nil {
		return err
	}
	img, err := h.service.PatchImage(showcaseID, imageID, userID, req.SortOrder, req.Caption)
	if err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to update image"), showcaseImageNotFoundErrorMap)
	}
	return c.JSON(img)
}

// DeleteSection handles DELETE /showcases/:showcase_id/sections/:section_id
func (h *ShowcaseHandler) DeleteSection(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	showcaseID, err := helper.RequirePathID(c, "showcase_id")
	if err != nil {
		return helper.BadRequestError(c, errInvalidShowcaseID)
	}
	sectionID, err := helper.ParsePositiveInt64Param(c, "section_id")
	if err != nil || sectionID <= 0 {
		return helper.BadRequestError(c, "invalid section_id")
	}
	if err := h.service.DeleteSection(showcaseID, sectionID, userID); err != nil {
		return helper.MapServiceError(c, err, helper.ErrorMessage(fiber.StatusInternalServerError, "failed to delete section"), showcaseSectionNotFoundErrorMap)
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}

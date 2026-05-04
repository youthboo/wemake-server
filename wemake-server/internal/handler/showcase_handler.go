package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/service"
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
	service *service.ShowcaseService
}

func NewShowcaseHandler(service *service.ShowcaseService) *ShowcaseHandler {
	return &ShowcaseHandler{service: service}
}

type showcaseWriteRequest struct {
	Type            *string         `json:"type"`
	ContentType     *string         `json:"content_type"`
	Status          *string         `json:"status"`
	Title           *string         `json:"title"`
	CategoryID      *int64          `json:"category_id"`
	SubCategoryID   *int64          `json:"sub_category_id"`
	MOQ             *int            `json:"moq"`
	LeadTimeDays    *int            `json:"lead_time_days"`
	BasePrice       *float64        `json:"base_price"`
	PromoPrice      *float64        `json:"promo_price"`
	StartDate       *string         `json:"start_date"`
	EndDate         *string         `json:"end_date"`
	Content         *string         `json:"content"`
	LinkedShowcases json.RawMessage `json:"linked_showcases"`
	Excerpt         *string         `json:"excerpt"`
	ImageURL        *string         `json:"image_url"`
	Tags            *[]string       `json:"tags"`
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
	if err := json.Unmarshal(raw, &asStrings); err == nil {
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
	if err := json.Unmarshal(raw, &asMixed); err == nil {
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
	if err := json.Unmarshal(raw, &asObjects); err == nil {
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
	t, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, &domain.ShowcaseValidationDetail{Field: field, Message: "must use YYYY-MM-DD format"}
	}
	return &t, nil
}

func (r showcaseWriteRequest) toInput() (domain.ShowcaseWriteInput, []domain.ShowcaseValidationDetail) {
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
	var validationErr domain.ShowcaseValidationError
	if errors.As(err, &validationErr) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error":   "VALIDATION_ERROR",
			"details": validationErr.Details,
		})
	}
	if errors.Is(err, sql.ErrNoRows) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
	}
	log.Printf("[showcase] %s: %v", fallback, err)
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": fallback})
}

// parseContentTypeQuery validates the ?type= query param (shared by List handlers).
func parseContentTypeQuery(c *fiber.Ctx, invalidMsg string) (string, error) {
	ct := strings.TrimSpace(c.Query("type", ""))
	if ct != "" {
		if _, ok := showcaseTypeQueryAllowed[ct]; !ok {
			return "", errors.New(invalidMsg)
		}
	}
	return ct, nil
}

// listByFactoryParam handles the factory_id query-param branch inside List.
func (h *ShowcaseHandler) listByFactoryParam(c *fiber.Ctx, factoryParam, contentType string) error {
	if strings.EqualFold(factoryParam, "me") {
		userID, err := getUserIDFromHeader(c)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		items, err := h.service.ListExploreByFactory(userID, contentType)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": errFetchShowcases})
		}
		return c.JSON(items)
	}
	factoryID, err := strconv.ParseInt(factoryParam, 10, 64)
	if err != nil || factoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
	}
	items, err := h.service.ListExploreByFactory(factoryID, contentType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": errFetchShowcases})
	}
	return c.JSON(items)
}

func (h *ShowcaseHandler) List(c *fiber.Ctx) error {
	contentType, err := parseContentTypeQuery(c, errInvalidTypeQuery)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	status := strings.TrimSpace(strings.ToUpper(c.Query("status", "")))
	if status != "" {
		if _, ok := showcaseStatusAllowed[status]; !ok {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidStatus})
		}
	}
	var factoryID *int64
	if factoryParam := strings.TrimSpace(c.Query("factory_id", "")); factoryParam != "" {
		if strings.EqualFold(factoryParam, "me") {
			userID, err := getUserIDFromHeader(c)
			if err != nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
			}
			factoryID = &userID
		} else {
			parsed, err := strconv.ParseInt(factoryParam, 10, 64)
			if err != nil || parsed <= 0 {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
			}
			factoryID = &parsed
		}
	}
	var categoryID *int64
	if raw := strings.TrimSpace(c.Query("category_id", "")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid category_id"})
		}
		categoryID = &parsed
	}
	var subCategoryID *int64
	if raw := strings.TrimSpace(c.Query("sub_category_id", "")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid sub_category_id"})
		}
		subCategoryID = &parsed
	}
	viewerID, _ := getUserIDFromHeader(c)
	items, err := h.service.ListStructured(domain.ShowcaseListFilter{
		Type:          contentType,
		FactoryID:     factoryID,
		Status:        status,
		CategoryID:    categoryID,
		SubCategoryID: subCategoryID,
		ViewerID:      viewerID,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": errFetchShowcases})
	}
	return c.JSON(items)
}

// GetDetail handles GET /showcases/:showcase_id
func (h *ShowcaseHandler) GetDetail(c *fiber.Ctx) error {
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	detail, err := h.service.GetDetail(showcaseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch showcase"})
	}
	// Non-owner sees only active showcases
	callerID, _ := getUserIDFromHeader(c)
	if detail.Status != "AC" && callerID != detail.FactoryID {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
	}
	return c.JSON(detail)
}

// ListByFactory handles GET /factories/:factory_id/showcases
func (h *ShowcaseHandler) ListByFactory(c *fiber.Ctx) error {
	factoryID, err := c.ParamsInt("factory_id")
	if err != nil || factoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
	}
	contentType, err := parseContentTypeQuery(c, errInvalidTypeFactory)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	callerID, _ := getUserIDFromHeader(c)
	items, err := h.service.GetShowcasesByFactory(int64(factoryID), contentType, callerID)
	if err != nil {
		log.Printf("[showcase] ListByFactory error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": errFetchShowcases})
	}
	return c.JSON(items)
}

func (h *ShowcaseHandler) Create(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	var req showcaseWriteRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidPayload})
	}
	input, details := req.toInput()
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
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	var req showcaseWriteRequest
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidPayload})
	}
	input, details := req.toInput()
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
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidPayload})
	}
	status := strings.TrimSpace(strings.ToUpper(req.Status))
	if _, ok := showcaseStatusAllowed[status]; !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidStatus})
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
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	if err := h.service.Delete(showcaseID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete showcase"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *ShowcaseHandler) GetAnalytics(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	item, err := h.service.GetAnalytics(showcaseID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch analytics"})
	}
	return c.JSON(item)
}

// RecordView handles POST /showcases/:showcase_id/view — increment view count
func (h *ShowcaseHandler) RecordView(c *fiber.Ctx) error {
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	_ = h.service.RecordView(showcaseID) // fire-and-forget; don't surface errors to caller
	return c.JSON(fiber.Map{"message": "view recorded"})
}

func (h *ShowcaseHandler) ListPromoSlides(c *fiber.Ctx) error {
	items, err := h.service.ListPromoSlides()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch promo slides"})
	}
	return c.JSON(items)
}

// CreateImage handles POST /showcases/:showcase_id/images
func (h *ShowcaseHandler) CreateImage(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	var req struct {
		ImageURL  string  `json:"image_url"`
		SortOrder int     `json:"sort_order"`
		Caption   *string `json:"caption"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidPayload})
	}
	if strings.TrimSpace(req.ImageURL) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "image_url is required"})
	}
	img := &domain.ShowcaseImage{
		ShowcaseID: showcaseID,
		ImageURL:   req.ImageURL,
		SortOrder:  req.SortOrder,
		Caption:    req.Caption,
	}
	if err := h.service.CreateImage(img, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
		}
		if errors.Is(err, domain.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": errNotYourShowcase})
		}
		if errors.Is(err, domain.ErrImageLimitExceeded) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to add image"})
	}
	return c.Status(fiber.StatusCreated).JSON(img)
}

// ListImages handles GET /showcases/:showcase_id/images
func (h *ShowcaseHandler) ListImages(c *fiber.Ctx) error {
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	callerID, _ := getUserIDFromHeader(c)
	images, err := h.service.ListImages(showcaseID, callerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
		}
		log.Printf("[showcase] failed to fetch images: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch images"})
	}
	return c.JSON(fiber.Map{"images": images})
}

// DeleteImage handles DELETE /showcases/:showcase_id/images/:image_id
func (h *ShowcaseHandler) DeleteImage(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	imageID, err := strconv.ParseInt(c.Params("image_id"), 10, 64)
	if err != nil || imageID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid image_id"})
	}
	if err := h.service.DeleteImage(showcaseID, imageID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "image not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete image"})
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}

// GetSections handles GET /showcases/:showcase_id/sections
func (h *ShowcaseHandler) GetSections(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	sections, err := h.service.GetSections(showcaseID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
		}
		if errors.Is(err, domain.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": errNotYourShowcase})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch sections"})
	}
	return c.JSON(fiber.Map{"sections": sections})
}

// BulkReplaceSections handles PUT /showcases/:showcase_id/sections
func (h *ShowcaseHandler) BulkReplaceSections(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	var req struct {
		Sections []domain.ShowcaseSectionInput `json:"sections"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidPayload})
	}
	if len(req.Sections) > 10 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "max 10 sections per showcase"})
	}
	if err := validateSectionInputs(req.Sections); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}
	if err := h.service.BulkReplaceSections(showcaseID, userID, req.Sections); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
		}
		if errors.Is(err, domain.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": errNotYourShowcase})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update sections"})
	}
	return c.JSON(fiber.Map{"message": "sections updated"})
}

// GetSpecs handles GET /showcases/:showcase_id/specs
func (h *ShowcaseHandler) GetSpecs(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	specs, err := h.service.GetSpecs(showcaseID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
		}
		if errors.Is(err, domain.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": errNotYourShowcase})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch specs"})
	}
	return c.JSON(fiber.Map{"specs": specs})
}

// BulkReplaceSpecs handles PUT /showcases/:showcase_id/specs
func (h *ShowcaseHandler) BulkReplaceSpecs(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	var req struct {
		Specs []domain.ShowcaseSpecInput `json:"specs"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidPayload})
	}
	for _, s := range req.Specs {
		if strings.TrimSpace(s.SpecKey) == "" {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "spec_key is required"})
		}
		if strings.TrimSpace(s.SpecValue) == "" {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "spec_value is required"})
		}
	}
	if err := h.service.BulkReplaceSpecs(showcaseID, userID, req.Specs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": errShowcaseNotFound})
		}
		if errors.Is(err, domain.ErrForbidden) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": errNotYourShowcase})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update specs"})
	}
	return c.JSON(fiber.Map{"message": "specs updated"})
}

// PatchImage handles PATCH /showcases/:showcase_id/images/:image_id
func (h *ShowcaseHandler) PatchImage(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	imageID, err := strconv.ParseInt(c.Params("image_id"), 10, 64)
	if err != nil || imageID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid image_id"})
	}
	var req struct {
		SortOrder *int    `json:"sort_order"`
		Caption   *string `json:"caption"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidPayload})
	}
	img, err := h.service.PatchImage(showcaseID, imageID, userID, req.SortOrder, req.Caption)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "image not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update image"})
	}
	return c.JSON(img)
}

// DeleteSection handles DELETE /showcases/:showcase_id/sections/:section_id
func (h *ShowcaseHandler) DeleteSection(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	showcaseID, err := strconv.ParseInt(c.Params("showcase_id"), 10, 64)
	if err != nil || showcaseID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": errInvalidShowcaseID})
	}
	sectionID, err := strconv.ParseInt(c.Params("section_id"), 10, 64)
	if err != nil || sectionID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid section_id"})
	}
	if err := h.service.DeleteSection(showcaseID, sectionID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "section not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete section"})
	}
	return c.JSON(fiber.Map{"message": "deleted"})
}

func validateSectionInputs(sections []domain.ShowcaseSectionInput) error {
	for _, sec := range sections {
		if sec.SectionType != "highlight" && sec.SectionType != "checklist" {
			return errors.New("section_type must be 'highlight' or 'checklist'")
		}
		if strings.TrimSpace(sec.SectionTitle) == "" {
			return errors.New("section_title is required")
		}
		if len(sec.Items) > 20 {
			return errors.New("max 20 items per section")
		}
		for _, item := range sec.Items {
			if strings.TrimSpace(item.Description) == "" {
				return errors.New("item description is required")
			}
		}
	}
	return nil
}

package handler

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/service"
)

type ProductionHandler struct {
	service *service.ProductionService
}

func NewProductionHandler(service *service.ProductionService) *ProductionHandler {
	return &ProductionHandler{service: service}
}

func (h *ProductionHandler) ListSteps(c *fiber.Ctx) error {
	var factoryTypeID *int64
	if raw := strings.TrimSpace(c.Query("factory_type_id", "")); raw != "" {
		parsed, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || parsed <= 0 {
			return productionError(c, fiber.StatusBadRequest, "INVALID_FACTORY_TYPE_ID", "invalid factory_type_id", nil)
		}
		factoryTypeID = &parsed
	}
	steps, err := h.service.ListSteps(factoryTypeID)
	if err != nil {
		return productionInternalError(c, err)
	}
	c.Set("Cache-Control", "public, max-age=3600")
	return c.JSON(fiber.Map{"steps": steps})
}

func (h *ProductionHandler) ListUpdates(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return productionError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil {
		return productionError(c, fiber.StatusBadRequest, "INVALID_ORDER_ID", "invalid order_id", nil)
	}
	item, err := h.service.ListByOrderID(int64(orderID), userID)
	if err != nil {
		return productionServiceError(c, err)
	}
	return c.JSON(item)
}

func (h *ProductionHandler) CreateUpdate(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return productionError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil {
		return productionError(c, fiber.StatusBadRequest, "INVALID_ORDER_ID", "invalid order_id", nil)
	}
	type reqBody struct {
		StepID                int64    `json:"step_id"`
		Status                string   `json:"status"`
		Description           string   `json:"description"`
		ImageURLs             []string `json:"image_urls"`
		ConfirmPaymentTrigger bool     `json:"confirm_payment_trigger"`
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return productionError(c, fiber.StatusBadRequest, "INVALID_PAYLOAD", "invalid request payload", nil)
	}
	result, err := h.service.Upsert(int64(orderID), userID, service.ProductionWriteInput{
		StepID:                 req.StepID,
		Status:                 req.Status,
		Description:            req.Description,
		ImageURLs:              req.ImageURLs,
		ConfirmPaymentTrigger:  req.ConfirmPaymentTrigger,
		HeaderPaymentConfirmed: strings.EqualFold(strings.TrimSpace(c.Get("X-Confirm-Payment-Trigger")), "true"),
	})
	if err != nil {
		return productionServiceError(c, err)
	}
	return c.JSON(result)
}

func (h *ProductionHandler) RejectUpdate(c *fiber.Ctx) error {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return productionError(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", nil)
	}
	updateID, err := c.ParamsInt("update_id")
	if err != nil {
		return productionError(c, fiber.StatusBadRequest, "INVALID_UPDATE_ID", "invalid update_id", nil)
	}
	type reqBody struct {
		RejectedReason string `json:"rejected_reason"`
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return productionError(c, fiber.StatusBadRequest, "INVALID_PAYLOAD", "invalid request payload", nil)
	}
	item, err := h.service.Reject(int64(updateID), userID, req.RejectedReason)
	if err != nil {
		return productionServiceError(c, err)
	}
	return c.JSON(item)
}

func productionServiceError(c *fiber.Ctx, err error) error {
	if errors.Is(err, sql.ErrNoRows) || service.IsNotFound(err) {
		return productionError(c, fiber.StatusNotFound, "NOT_FOUND", "resource not found", nil)
	}
	if rule, ok := service.AsProductionRuleError(err); ok {
		switch {
		case errors.Is(rule, service.ErrProductionNotOrderFactory):
			return productionError(c, fiber.StatusForbidden, "NOT_ORDER_FACTORY", "factory caller does not own the order", rule.Details)
		case errors.Is(rule, service.ErrProductionNotOrderCustomer):
			return productionError(c, fiber.StatusForbidden, "NOT_ORDER_CUSTOMER", "customer caller does not own the order", rule.Details)
		case errors.Is(rule, service.ErrProductionOrderLocked):
			return productionError(c, fiber.StatusConflict, "ORDER_STATE_INVALID", "order is locked for production updates", rule.Details)
		case errors.Is(rule, service.ErrProductionAnotherStepInProgress):
			return productionError(c, fiber.StatusConflict, "ANOTHER_STEP_IN_PROGRESS", "another step is already in progress", rule.Details)
		case errors.Is(rule, service.ErrProductionInvalidStateTransition):
			return productionError(c, fiber.StatusConflict, "STEP_LOCKED", "invalid or locked step transition", rule.Details)
		case errors.Is(rule, service.ErrProductionDownstreamInFlight):
			return productionError(c, fiber.StatusConflict, "DOWNSTREAM_IN_FLIGHT", "cannot reject because downstream steps are already active", rule.Details)
		case errors.Is(rule, service.ErrProductionStepOrderViolation):
			return productionError(c, fiber.StatusConflict, "STEP_LOCKED", "previous step must be completed first", rule.Details)
		case errors.Is(rule, service.ErrProductionInsufficientEvidence):
			return productionError(c, fiber.StatusUnprocessableEntity, "INSUFFICIENT_EVIDENCE", "insufficient evidence", rule.Details)
		case errors.Is(rule, service.ErrProductionPaymentConfirmRequired):
			return productionError(c, fiber.StatusUnprocessableEntity, "PAYMENT_CONFIRMATION_REQUIRED", "payment confirmation required", rule.Details)
		case errors.Is(rule, service.ErrProductionInvalidStep):
			return productionError(c, fiber.StatusUnprocessableEntity, "INVALID_STEP_ID", "step_id must reference an active production step", rule.Details)
		case errors.Is(rule, service.ErrProductionStepIDRequired):
			return productionError(c, fiber.StatusUnprocessableEntity, "STEP_ID_REQUIRED", "step_id is required", rule.Details)
		case errors.Is(rule, service.ErrProductionInvalidStatus):
			return productionError(c, fiber.StatusUnprocessableEntity, "INVALID_STATUS", "status must be IP or CD", rule.Details)
		case errors.Is(rule, service.ErrProductionMaxImages):
			return productionError(c, fiber.StatusUnprocessableEntity, "MAX_5_IMAGES", "image_urls can contain at most 5 items", rule.Details)
		case errors.Is(rule, service.ErrProductionInvalidImageFormat):
			return productionError(c, fiber.StatusUnprocessableEntity, "INVALID_IMAGE_FORMAT", "image_urls must be a non-empty array of unique URL strings", rule.Details)
		case errors.Is(rule, service.ErrProductionInvalidImageURL):
			return productionError(c, fiber.StatusUnprocessableEntity, "INVALID_IMAGE_URL", "image_urls must be unique HTTP/HTTPS URLs", rule.Details)
		case errors.Is(rule, service.ErrProductionDescriptionTooLong):
			return productionError(c, fiber.StatusUnprocessableEntity, "INVALID_DESCRIPTION", "description must be 2000 characters or fewer", rule.Details)
		case errors.Is(rule, service.ErrProductionReasonRequired):
			return productionError(c, fiber.StatusUnprocessableEntity, "REASON_REQUIRED", "rejected_reason is required", rule.Details)
		}
	}
	return productionInternalError(c, err)
}

func productionInternalError(c *fiber.Ctx, err error) error {
	return productionError(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", service.ExplainProductionError(err), nil)
}

func productionError(c *fiber.Ctx, status int, code, message string, details map[string]interface{}) error {
	errBody := fiber.Map{
		"code":    code,
		"message": message,
	}
	if len(details) > 0 {
		errBody["details"] = details
	}
	return c.Status(status).JSON(fiber.Map{"error": errBody})
}

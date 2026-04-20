package handler

import (
	"database/sql"
	"errors"
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
	steps, err := h.service.ListSteps()
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
			return productionError(c, fiber.StatusConflict, "ORDER_LOCKED", "order is cancelled, closed, or completed", rule.Details)
		case errors.Is(rule, service.ErrProductionAnotherStepInProgress):
			return productionError(c, fiber.StatusConflict, "ANOTHER_STEP_IN_PROGRESS", "another step is already in progress", rule.Details)
		case errors.Is(rule, service.ErrProductionInvalidStateTransition):
			return productionError(c, fiber.StatusConflict, "INVALID_STATE_TRANSITION", "invalid state transition", rule.Details)
		case errors.Is(rule, service.ErrProductionDownstreamInFlight):
			return productionError(c, fiber.StatusConflict, "DOWNSTREAM_IN_FLIGHT", "cannot reject because downstream steps are already active", rule.Details)
		case errors.Is(rule, service.ErrProductionStepOrderViolation):
			return productionError(c, fiber.StatusUnprocessableEntity, "STEP_ORDER_VIOLATION", "previous step must be completed first", rule.Details)
		case errors.Is(rule, service.ErrProductionInsufficientEvidence):
			return productionError(c, fiber.StatusUnprocessableEntity, "INSUFFICIENT_EVIDENCE", "insufficient evidence", rule.Details)
		case errors.Is(rule, service.ErrProductionPaymentConfirmRequired):
			return productionError(c, fiber.StatusUnprocessableEntity, "PAYMENT_CONFIRMATION_REQUIRED", "payment confirmation required", rule.Details)
		case errors.Is(rule, service.ErrProductionInvalidStep):
			return productionError(c, fiber.StatusUnprocessableEntity, "INVALID_STEP", "step_id must reference an active production step", rule.Details)
		case errors.Is(rule, service.ErrProductionInvalidImageURL):
			return productionError(c, fiber.StatusUnprocessableEntity, "INVALID_IMAGE_URL", "image_urls must be unique HTTPS URLs", rule.Details)
		case errors.Is(rule, service.ErrProductionDescriptionTooLong):
			return productionError(c, fiber.StatusUnprocessableEntity, "DESCRIPTION_TOO_LONG", "description must be 500 characters or fewer", rule.Details)
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

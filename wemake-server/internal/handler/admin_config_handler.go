package handler

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
)

type AdminConfigHandler struct {
	commission *repository.CommissionRepository
	audit      *repository.AdminAuditRepository
}

func NewAdminConfigHandler(commission *repository.CommissionRepository, audit *repository.AdminAuditRepository) *AdminConfigHandler {
	return &AdminConfigHandler{commission: commission, audit: audit}
}

func (h *AdminConfigHandler) ListRules(c *fiber.Ctx) error {
	var factoryID *int64
	if v := strings.TrimSpace(c.Query("factory_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
		}
		factoryID = &id
	}
	items, err := h.commission.ListRules(factoryID, !strings.EqualFold(c.Query("active_only", "true"), "false"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch commission rules"})
	}
	return c.JSON(fiber.Map{"data": items})
}

func (h *AdminConfigHandler) CreateRule(c *fiber.Ctx) error {
	var req struct {
		FactoryID     int64   `json:"factory_id"`
		RatePercent   float64 `json:"rate_percent"`
		EffectiveFrom *string `json:"effective_from"`
		EffectiveTo   *string `json:"effective_to"`
		Note          *string `json:"note"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if req.FactoryID <= 0 || req.RatePercent < 0 || req.RatePercent > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid commission rule payload"})
	}
	from := time.Now().UTC()
	if req.EffectiveFrom != nil && strings.TrimSpace(*req.EffectiveFrom) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.EffectiveFrom))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "effective_from must be RFC3339"})
		}
		from = t
	}
	var to *time.Time
	if req.EffectiveTo != nil && strings.TrimSpace(*req.EffectiveTo) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.EffectiveTo))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "effective_to must be RFC3339"})
		}
		to = &t
	}
	actorID, _ := getUserIDFromHeader(c)
	factoryID := req.FactoryID
	item := &domain.CommissionRule{FactoryID: &factoryID, RatePercent: req.RatePercent, EffectiveFrom: from, EffectiveTo: to, Note: req.Note, CreatedBy: actorID}
	if err := h.commission.CreateRule(item); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create commission rule"})
	}
	h.insertAudit(actorID, "COMMISSION_RULE_CREATE", "commission_rule", item.RuleID, item, c.IP())
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *AdminConfigHandler) DeleteRule(c *fiber.Ctx) error {
	ruleID, err := strconv.ParseInt(c.Params("rule_id"), 10, 64)
	if err != nil || ruleID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rule_id"})
	}
	item, err := h.commission.DeactivateRule(ruleID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to deactivate commission rule"})
	}
	actorID, _ := getUserIDFromHeader(c)
	h.insertAudit(actorID, "COMMISSION_RULE_DEACTIVATE", "commission_rule", ruleID, item, c.IP())
	return c.JSON(fiber.Map{"rule_id": ruleID, "effective_to": item.EffectiveTo})
}

func (h *AdminConfigHandler) ListExemptions(c *fiber.Ctx) error {
	items, err := h.commission.ListExemptions(!strings.EqualFold(c.Query("active_only", "true"), "false"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch commission exemptions"})
	}
	return c.JSON(fiber.Map{"data": items})
}

func (h *AdminConfigHandler) CreateExemption(c *fiber.Ctx) error {
	var req struct {
		FactoryID int64   `json:"factory_id"`
		Reason    string  `json:"reason"`
		ExpiresAt *string `json:"expires_at"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if req.FactoryID <= 0 || strings.TrimSpace(req.Reason) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "factory_id and reason are required"})
	}
	if exists, _ := h.commission.ActiveExemptionExists(req.FactoryID); exists {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "factory already has an active exemption"})
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.ExpiresAt))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "expires_at must be RFC3339"})
		}
		expiresAt = &t
	}
	actorID, _ := getUserIDFromHeader(c)
	item := &domain.CommissionExemption{FactoryID: req.FactoryID, Reason: strings.TrimSpace(req.Reason), ExpiresAt: expiresAt, CreatedBy: actorID}
	if err := h.commission.CreateExemption(item); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create commission exemption"})
	}
	h.insertAudit(actorID, "COMMISSION_EXEMPTION_CREATE", "commission_exemption", item.ExemptionID, item, c.IP())
	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *AdminConfigHandler) DeleteExemption(c *fiber.Ctx) error {
	exemptionID, err := strconv.ParseInt(c.Params("exemption_id"), 10, 64)
	if err != nil || exemptionID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid exemption_id"})
	}
	actorID, _ := getUserIDFromHeader(c)
	item, err := h.commission.RevokeExemption(exemptionID, actorID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to revoke commission exemption"})
	}
	h.insertAudit(actorID, "COMMISSION_EXEMPTION_REVOKE", "commission_exemption", exemptionID, item, c.IP())
	return c.JSON(fiber.Map{"exemption_id": exemptionID, "revoked_at": item.RevokedAt, "revoked_by": item.RevokedBy})
}

func (h *AdminConfigHandler) ListAuditLog(c *fiber.Ctx) error {
	filter := domain.AdminAuditFilter{
		Action:     strings.TrimSpace(c.Query("action")),
		TargetType: strings.TrimSpace(c.Query("target_type")),
		Page:       c.QueryInt("page", 1),
		PageSize:   c.QueryInt("page_size", 20),
	}
	if v := strings.TrimSpace(c.Query("actor_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid actor_id"})
		}
		filter.ActorID = &id
	}
	if v := strings.TrimSpace(c.Query("date_from")); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "date_from must be YYYY-MM-DD"})
		}
		filter.DateFrom = &t
	}
	if v := strings.TrimSpace(c.Query("date_to")); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "date_to must be YYYY-MM-DD"})
		}
		filter.DateTo = &t
	}
	items, total, err := h.audit.List(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch audit log"})
	}
	return c.JSON(fiber.Map{"data": items, "pagination": domain.Pagination{Page: maxInt(filter.Page, 1), PageSize: normalizePageSize(filter.PageSize), Total: total}})
}

func (h *AdminConfigHandler) insertAudit(actorID int64, action, targetType string, targetID int64, payload interface{}, ip string) {
	raw, _ := json.Marshal(payload)
	_ = h.audit.Insert(&domain.AdminAuditLog{ActorID: actorID, Action: action, TargetType: targetType, TargetID: strconv.FormatInt(targetID, 10), Payload: raw, IPAddress: &ip})
}

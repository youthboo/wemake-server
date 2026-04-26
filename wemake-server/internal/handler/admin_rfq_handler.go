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

type AdminRFQHandler struct {
	repo  *repository.RFQRepository
	audit *repository.AdminAuditRepository
}

func NewAdminRFQHandler(repo *repository.RFQRepository, audit *repository.AdminAuditRepository) *AdminRFQHandler {
	return &AdminRFQHandler{repo: repo, audit: audit}
}

func (h *AdminRFQHandler) List(c *fiber.Ctx) error {
	filter := domain.AdminRFQFilter{
		Status:   strings.TrimSpace(c.Query("status")),
		Search:   strings.TrimSpace(c.Query("search")),
		Page:     c.QueryInt("page", 1),
		PageSize: c.QueryInt("page_size", 20),
	}
	if v := strings.TrimSpace(c.Query("user_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user_id"})
		}
		filter.UserID = &id
	}
	if v := strings.TrimSpace(c.Query("category_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid category_id"})
		}
		filter.CategoryID = &id
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
	items, total, err := h.repo.ListAdmin(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch rfqs"})
	}
	return c.JSON(fiber.Map{"data": items, "pagination": domain.Pagination{Page: maxInt(filter.Page, 1), PageSize: normalizePageSize(filter.PageSize), Total: total}})
}

func (h *AdminRFQHandler) GetByID(c *fiber.Ctx) error {
	rfqID, err := strconv.ParseInt(c.Params("rfq_id"), 10, 64)
	if err != nil || rfqID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
	}
	item, err := h.repo.GetAdminDetail(rfqID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "rfq not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch rfq"})
	}
	return c.JSON(item)
}

func (h *AdminRFQHandler) PatchStatus(c *fiber.Ctx) error {
	rfqID, err := strconv.ParseInt(c.Params("rfq_id"), 10, 64)
	if err != nil || rfqID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid rfq_id"})
	}
	var req struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	status := strings.ToUpper(strings.TrimSpace(req.Status))
	if status != "CL" && status != "CC" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "status must be CL or CC"})
	}
	if err := h.repo.UpdateStatusAdmin(rfqID, status); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update rfq status"})
	}
	actorID, _ := getUserIDFromHeader(c)
	payload, _ := json.Marshal(map[string]interface{}{"status": status, "reason": req.Reason})
	ip := c.IP()
	_ = h.audit.Insert(&domain.AdminAuditLog{ActorID: actorID, Action: "RFQ_STATUS_CHANGE", TargetType: "rfq", TargetID: strconv.FormatInt(rfqID, 10), Payload: payload, IPAddress: &ip})
	return c.JSON(fiber.Map{"rfq_id": rfqID, "status": status})
}

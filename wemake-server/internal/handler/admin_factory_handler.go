package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type AdminFactoryHandler struct {
	repo    *repository.FactoryRepository
	service *service.AdminFactoryService
}

func NewAdminFactoryHandler(repo *repository.FactoryRepository, service *service.AdminFactoryService) *AdminFactoryHandler {
	return &AdminFactoryHandler{repo: repo, service: service}
}

func (h *AdminFactoryHandler) List(c *fiber.Ctx) error {
	filter := domain.AdminFactoryFilter{
		ApprovalStatus: strings.TrimSpace(c.Query("approval_status")),
		Search:         strings.TrimSpace(c.Query("search")),
		Page:           c.QueryInt("page", 1),
		PageSize:       c.QueryInt("page_size", 20),
	}
	if v := strings.TrimSpace(c.Query("is_verified")); v != "" {
		isVerified := strings.EqualFold(v, "true") || v == "1"
		filter.IsVerified = &isVerified
	}
	items, total, err := h.repo.ListAdmin(filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch factories"})
	}
	return c.JSON(fiber.Map{"data": items, "pagination": domain.Pagination{Page: maxInt(filter.Page, 1), PageSize: normalizePageSize(filter.PageSize), Total: total}})
}

func (h *AdminFactoryHandler) GetByID(c *fiber.Ctx) error {
	factoryID, err := strconv.ParseInt(c.Params("factory_id"), 10, 64)
	if err != nil || factoryID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid factory_id"})
	}
	item, err := h.service.HydrateAdminDetail(factoryID)
	if err != nil {
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "factory not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch factory"})
	}
	return c.JSON(item)
}

func (h *AdminFactoryHandler) Approve(c *fiber.Ctx) error {
	return h.mutateFactoryState(c, h.service.Approve)
}

func (h *AdminFactoryHandler) Reject(c *fiber.Ctx) error {
	return h.mutateFactoryReasonState(c, h.service.Reject)
}

func (h *AdminFactoryHandler) Suspend(c *fiber.Ctx) error {
	return h.mutateFactoryReasonState(c, h.service.Suspend)
}

func (h *AdminFactoryHandler) Unsuspend(c *fiber.Ctx) error {
	return h.mutateFactoryState(c, h.service.Unsuspend)
}

func (h *AdminFactoryHandler) PatchVerification(c *fiber.Ctx) error {
	factoryID, actorID, err := parseFactoryActor(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	var req struct {
		IsVerified bool   `json:"is_verified"`
		Note       string `json:"note"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	ip := c.IP()
	if err := h.service.ToggleVerification(factoryID, actorID, req.IsVerified, req.Note, &ip); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	item, _ := h.service.HydrateAdminDetail(factoryID)
	return c.JSON(fiber.Map{
		"factory_id":   factoryID,
		"is_verified":  item.IsVerified,
		"verified_at":  item.VerifiedAt,
		"verified_by":  item.VerifiedBy,
		"approval_status": item.ApprovalStatus,
	})
}

func (h *AdminFactoryHandler) mutateFactoryState(c *fiber.Ctx, fn func(int64, int64, string, *string) error) error {
	factoryID, actorID, err := parseFactoryActor(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	var req struct{ Note string `json:"note"` }
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	ip := c.IP()
	if err := fn(factoryID, actorID, req.Note, &ip); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	item, _ := h.service.HydrateAdminDetail(factoryID)
	return c.JSON(fiber.Map{"factory_id": factoryID, "approval_status": item.ApprovalStatus, "is_verified": item.IsVerified, "verified_at": item.VerifiedAt, "verified_by": item.VerifiedBy})
}

func (h *AdminFactoryHandler) mutateFactoryReasonState(c *fiber.Ctx, fn func(int64, int64, string, *string) error) error {
	factoryID, actorID, err := parseFactoryActor(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	var req struct{ Reason string `json:"reason"` }
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	ip := c.IP()
	if err := fn(factoryID, actorID, req.Reason, &ip); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	item, _ := h.service.HydrateAdminDetail(factoryID)
	return c.JSON(fiber.Map{"factory_id": factoryID, "approval_status": item.ApprovalStatus, "is_verified": item.IsVerified, "rejection_reason": item.RejectionReason})
}

func parseFactoryActor(c *fiber.Ctx) (int64, int64, error) {
	factoryID, err := strconv.ParseInt(c.Params("factory_id"), 10, 64)
	if err != nil || factoryID <= 0 {
		return 0, 0, fiber.NewError(fiber.StatusBadRequest, "invalid factory_id")
	}
	actorID, err := getUserIDFromHeader(c)
	if err != nil {
		return 0, 0, fiber.NewError(fiber.StatusUnauthorized, "unauthorized")
	}
	return factoryID, actorID, nil
}

package admin

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/dto"
	"github.com/yourusername/wemake/internal/helper"
	adminrepo "github.com/yourusername/wemake/internal/repository/admin"
	adminservice "github.com/yourusername/wemake/internal/service/admin"
)

type AdminFactoryHandler struct {
	repo    *adminrepo.AdminFactoryRepository
	service *adminservice.AdminFactoryService
}

func NewAdminFactoryHandler(repo *adminrepo.AdminFactoryRepository, service *adminservice.AdminFactoryService) *AdminFactoryHandler {
	return &AdminFactoryHandler{repo: repo, service: service}
}

func (h *AdminFactoryHandler) List(c *fiber.Ctx) error {
	query := helper.QueryParams(c)
	page, pageSize := query.PageSize(helper.DefaultPageSize)
	filter := domain.AdminFactoryFilter{
		ApprovalStatus: query.String("approval_status"),
		Search:         query.String("search"),
		Page:           page,
		PageSize:       pageSize,
	}
	if v := query.String("is_verified"); v != "" {
		isVerified := strings.EqualFold(v, "true") || v == "1"
		filter.IsVerified = &isVerified
	}
	items, total, err := h.repo.ListAdmin(filter)
	if err != nil {
		return helper.InternalServerError(c, "failed to fetch factories")
	}
	return helper.PaginatedResponse(c, items, filter.Page, filter.PageSize, total)
}

func (h *AdminFactoryHandler) GetByID(c *fiber.Ctx) error {
	factoryID, err := helper.ParsePositiveInt64Param(c, "factory_id")
	if err != nil {
		return helper.BadRequest(c, "invalid factory_id")
	}
	item, err := h.service.HydrateAdminDetail(factoryID)
	if err != nil {
		return helper.WriteServiceError(c, err, "failed to fetch factory", helper.NotFoundCase(helper.ErrNotFound, "factory not found"))
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
		return helper.BadRequestError(c, err.Error())
	}
	var req dto.PatchFactoryVerificationRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	ip := c.IP()
	isVerified := req.TaxIDVerified != nil && *req.TaxIDVerified
	note := helper.DereferenceString(req.Notes, "")
	if err := h.service.ToggleVerification(factoryID, actorID, isVerified, note, &ip); err != nil {
		return helper.BadRequestError(c, err.Error())
	}
	item, _ := h.service.HydrateAdminDetail(factoryID)
	return c.JSON(fiber.Map{
		"factory_id":      factoryID,
		"is_verified":     item.IsVerified,
		"verified_at":     item.VerifiedAt,
		"verified_by":     item.VerifiedBy,
		"approval_status": item.ApprovalStatus,
	})
}

func (h *AdminFactoryHandler) GetFactoryConfig(c *fiber.Ctx) error {
	factoryID, err := helper.ParsePositiveInt64Param(c, "factory_id")
	if err != nil {
		return helper.BadRequest(c, "invalid factory_id")
	}
	item, err := h.service.GetFactoryConfig(factoryID)
	if err != nil {
		return helper.WriteServiceError(c, err, "failed to fetch factory config", helper.NotFoundCase(adminservice.ErrFactoryNotFound, "factory not found"))
	}
	return c.JSON(item)
}

func (h *AdminFactoryHandler) AssignFactoryConfig(c *fiber.Ctx) error {
	factoryID, actorID, err := parseFactoryActor(c)
	if err != nil {
		status := fiber.StatusBadRequest
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) && fiberErr.Code == fiber.StatusUnauthorized {
			status = fiber.StatusUnauthorized
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	var req domain.AssignFactoryConfigRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	ip := c.IP()
	item, err := h.service.AssignFactoryConfig(factoryID, actorID, req, &ip)
	if err != nil {
		switch {
		case errors.Is(err, adminservice.ErrFactoryNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "factory not found"})
		case errors.Is(err, adminservice.ErrFactoryConfigMissing):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "platform config not found"})
		default:
			return helper.InternalServerError(c, "failed to assign factory config")
		}
	}
	return c.JSON(item)
}

func (h *AdminFactoryHandler) mutateFactoryState(c *fiber.Ctx, fn func(int64, int64, string, *string) error) error {
	factoryID, actorID, err := parseFactoryActor(c)
	if err != nil {
		return helper.BadRequestError(c, err.Error())
	}
	var req dto.ApproveFactoryRequest
	_ = c.BodyParser(&req) // body is optional — notes is optional
	ip := c.IP()
	note := helper.DereferenceString(req.Notes, "")
	if err := fn(factoryID, actorID, note, &ip); err != nil {
		return helper.BadRequestError(c, err.Error())
	}
	item, _ := h.service.HydrateAdminDetail(factoryID)
	return c.JSON(fiber.Map{"factory_id": factoryID, "approval_status": item.ApprovalStatus, "is_verified": item.IsVerified, "verified_at": item.VerifiedAt, "verified_by": item.VerifiedBy})
}

func (h *AdminFactoryHandler) mutateFactoryReasonState(c *fiber.Ctx, fn func(int64, int64, string, *string) error) error {
	factoryID, actorID, err := parseFactoryActor(c)
	if err != nil {
		return helper.BadRequestError(c, err.Error())
	}
	var req dto.RejectFactoryRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	ip := c.IP()
	if err := fn(factoryID, actorID, req.Reason, &ip); err != nil {
		return helper.BadRequestError(c, err.Error())
	}
	item, _ := h.service.HydrateAdminDetail(factoryID)
	return c.JSON(fiber.Map{"factory_id": factoryID, "approval_status": item.ApprovalStatus, "is_verified": item.IsVerified, "rejection_reason": item.RejectionReason})
}

// PatchCertificateStatus handles PATCH /admin/factories/:factory_id/certificates/:map_id
// Body: { "verify_status": "AP"|"RJ"|"PE" }
func (h *AdminFactoryHandler) PatchCertificateStatus(c *fiber.Ctx) error {
	factoryID, err := helper.ParsePositiveInt64Param(c, "factory_id")
	if err != nil || factoryID <= 0 {
		return helper.BadRequest(c, "invalid factory_id")
	}
	mapID, err := helper.ParsePositiveInt64Param(c, "map_id")
	if err != nil || mapID <= 0 {
		return helper.BadRequest(c, "invalid map_id")
	}
	var body struct {
		VerifyStatus string `json:"verify_status"`
	}
	if err := helper.RequireBody(c, &body); err != nil {
		return err
	}
	allowed := map[string]bool{"AP": true, "RJ": true, "PE": true}
	if !allowed[body.VerifyStatus] {
		return helper.BadRequest(c, "verify_status must be AP, RJ, or PE")
	}
	if err := h.repo.PatchCertificateStatus(factoryID, mapID, body.VerifyStatus); err != nil {
		return helper.InternalServerError(c, "failed to update certificate status")
	}
	return c.JSON(fiber.Map{
		"factory_id":    factoryID,
		"map_id":        mapID,
		"verify_status": body.VerifyStatus,
	})
}

func parseFactoryActor(c *fiber.Ctx) (int64, int64, error) {
	factoryID, err := helper.ParsePositiveInt64Param(c, "factory_id")
	if err != nil || factoryID <= 0 {
		return 0, 0, fiber.NewError(fiber.StatusBadRequest, "invalid factory_id")
	}
	actorID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return 0, 0, err
	}
	return factoryID, actorID, nil
}

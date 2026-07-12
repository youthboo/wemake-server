package admin

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	domainstatus "github.com/yourusername/wemake/internal/domain/status"
	"github.com/yourusername/wemake/internal/domainutil"
	"github.com/yourusername/wemake/internal/dto"
	"github.com/yourusername/wemake/internal/helper"
	adminrepo "github.com/yourusername/wemake/internal/repository/admin"
	walletrepo "github.com/yourusername/wemake/internal/repository/wallet"
	orderservice "github.com/yourusername/wemake/internal/service/order"
)

type AdminOrderHandler struct {
	repo            *adminrepo.AdminOrderRepository
	service         *orderservice.OrderService
	withdrawal      *walletrepo.WithdrawalRepository
	adminWithdrawal *adminrepo.AdminWithdrawalRepository
	dispute         *walletrepo.DisputeRepository
	adminDispute    *adminrepo.AdminDisputeRepository
	audit           *adminrepo.AdminAuditRepository
}

func NewAdminOrderHandler(repo *adminrepo.AdminOrderRepository, service *orderservice.OrderService, withdrawal *walletrepo.WithdrawalRepository, adminWithdrawal *adminrepo.AdminWithdrawalRepository, dispute *walletrepo.DisputeRepository, adminDispute *adminrepo.AdminDisputeRepository, audit *adminrepo.AdminAuditRepository) *AdminOrderHandler {
	return &AdminOrderHandler{repo: repo, service: service, withdrawal: withdrawal, adminWithdrawal: adminWithdrawal, dispute: dispute, adminDispute: adminDispute, audit: audit}
}

func (h *AdminOrderHandler) List(c *fiber.Ctx) error {
	query := helper.QueryParams(c)
	page, pageSize := query.PageSize(helper.DefaultPageSize)
	filter := domain.AdminOrderFilter{
		Status:    query.String("status"),
		Search:    query.String("search"),
		Page:      page,
		PageSize:  pageSize,
		FactoryID: query.OptionalPositiveInt64("factory_id"),
		UserID:    query.OptionalPositiveInt64("user_id"),
		DateFrom:  query.OptionalDate("date_from"),
		DateTo:    query.OptionalDate("date_to"),
	}
	if err := query.Err(); err != nil {
		return err
	}
	items, total, err := h.repo.ListAdmin(filter)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch orders")
	}
	return helper.PaginatedResponse(c, items, filter.Page, filter.PageSize, total)
}

func (h *AdminOrderHandler) GetByID(c *fiber.Ctx) error {
	orderID, err := helper.ParsePositiveInt64Param(c, "order_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid order_id")
	}
	detail, err := h.service.GetAdminDetailByID(orderID)
	if err != nil {
		return helper.WriteServiceError(c, err, "failed to fetch order", helper.NotFoundCase(helper.ErrNotFound, "order not found"))
	}
	finance, err := h.repo.GetAdminFinance(orderID)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch order finance")
	}
	return c.JSON(domain.AdminOrderDetailResponse{OrderDetailResponse: detail, AdminFinance: *finance})
}

func (h *AdminOrderHandler) PatchStatus(c *fiber.Ctx) error {
	orderID, err := helper.ParsePositiveInt64Param(c, "order_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid order_id")
	}
	var req dto.PatchOrderStatusRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	status := domainstatus.NormalizeOrder(req.Status)
	v := domain.NewValidationCollector()
	v.AddIf(!domainstatus.IsValidOrder(status), "status", "is invalid")
	if err := helper.ValidateRequest(c, v); err != nil {
		return err
	}
	if err := h.service.UpdateStatus(orderID, status, nil); err != nil {
		return helper.JSONInternal(c, "failed to update order status")
	}
	actorID := helper.OptionalActorID(c)
	notes := helper.DereferenceString(req.Notes, "")
	payload, _ := json.Marshal(map[string]interface{}{"status": status, "notes": notes})
	ip := c.IP()
	_ = h.audit.Insert(&domain.AdminAuditLog{ActorID: actorID, Action: "ORDER_STATUS_CHANGE", TargetType: "order", TargetID: strconv.FormatInt(orderID, 10), Payload: payload, IPAddress: &ip})
	return c.JSON(fiber.Map{"order_id": orderID, "status": status})
}

func (h *AdminOrderHandler) ListWithdrawals(c *fiber.Ctx) error {
	query := helper.QueryParams(c)
	page, pageSize := query.PageSize(helper.DefaultPageSize)
	factoryID := query.OptionalPositiveInt64("factory_id")
	if err := query.Err(); err != nil {
		return err
	}
	items, total, err := h.adminWithdrawal.ListAdmin(query.String("status"), factoryID, page, pageSize)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch withdrawals")
	}
	return helper.PaginatedResponse(c, items, page, pageSize, total)
}

func (h *AdminOrderHandler) PatchWithdrawal(c *fiber.Ctx) error {
	requestID, err := helper.ParsePositiveInt64Param(c, "request_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid request_id")
	}
	var req dto.PatchWithdrawalStatusRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	status := domainutil.NormalizeStatus(req.Status)
	v := domain.NewValidationCollector()
	v.AddIf(!domainutil.StatusIn(status, domain.WithdrawalStatusApproved, domain.WithdrawalStatusRejected, domain.WithdrawalStatusComplete), "status", "must be AP, RJ, or CP")
	// เมื่อโอนเสร็จ (CP) superadmin ต้องแนบสลิปโอนเงิน (อัปโหลดผ่าน /media/upload ก่อน)
	v.AddIf(status == domain.WithdrawalStatusComplete && (req.SlipURL == nil || *req.SlipURL == ""), "slip_url", "is required when completing a withdrawal")
	if err := helper.ValidateRequest(c, v); err != nil {
		return err
	}
	actorID := helper.OptionalActorID(c)
	var processedBy *int64
	if actorID > 0 {
		processedBy = &actorID
	}
	if err := h.withdrawal.UpdateStatus(requestID, status, req.Comments, req.SlipURL, processedBy); err != nil {
		if errors.Is(err, walletrepo.ErrWithdrawalAlreadyProcessed) {
			return helper.JSONError(c, fiber.StatusBadRequest, "คำขอถอนเงินนี้ถูกดำเนินการแล้ว")
		}
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusNotFound, "withdrawal request not found")
		}
		return helper.JSONInternal(c, "failed to update withdrawal")
	}
	payload, _ := json.Marshal(map[string]interface{}{"status": status, "comments": req.Comments})
	ip := c.IP()
	_ = h.audit.Insert(&domain.AdminAuditLog{ActorID: actorID, Action: "WITHDRAWAL_STATUS_CHANGE", TargetType: "withdrawal", TargetID: strconv.FormatInt(requestID, 10), Payload: payload, IPAddress: &ip})
	return c.JSON(fiber.Map{"request_id": requestID, "status": status, "processed_at": time.Now().UTC()})
}

func (h *AdminOrderHandler) ListDisputes(c *fiber.Ctx) error {
	query := helper.QueryParams(c)
	page, pageSize := query.PageSize(helper.DefaultPageSize)
	orderID := query.OptionalPositiveInt64("order_id")
	if err := query.Err(); err != nil {
		return err
	}
	items, total, err := h.adminDispute.ListAdmin(query.String("status"), orderID, page, pageSize)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch disputes")
	}
	return helper.PaginatedResponse(c, items, page, pageSize, total)
}

func (h *AdminOrderHandler) PatchDispute(c *fiber.Ctx) error {
	disputeID, err := helper.ParsePositiveInt64Param(c, "dispute_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid dispute_id")
	}
	var req dto.PatchDisputeStatusRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	status := domainutil.NormalizeStatus(req.Status)
	v := domain.NewValidationCollector()
	v.AddIf(!domainutil.StatusIn(status, "RS", "CL"), "status", "must be RS or CL")
	if err := helper.ValidateRequest(c, v); err != nil {
		return err
	}
	if err := h.dispute.UpdateStatus(disputeID, status, req.Comments); err != nil {
		return helper.JSONInternal(c, "failed to update dispute")
	}
	actorID := helper.OptionalActorID(c)
	payload, _ := json.Marshal(map[string]interface{}{"status": status, "comments": req.Comments})
	ip := c.IP()
	_ = h.audit.Insert(&domain.AdminAuditLog{ActorID: actorID, Action: "DISPUTE_STATUS_CHANGE", TargetType: "dispute", TargetID: strconv.FormatInt(disputeID, 10), Payload: payload, IPAddress: &ip})
	item, _ := h.dispute.GetByID(disputeID)
	return c.JSON(item)
}

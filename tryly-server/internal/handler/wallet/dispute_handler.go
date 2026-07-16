package wallet

import (
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/dto"
	"github.com/yourusername/wemake/internal/helper"
	walletrepo "github.com/yourusername/wemake/internal/repository/wallet"
	walletservice "github.com/yourusername/wemake/internal/service/wallet"
)

type DisputeHandler struct {
	service *walletservice.DisputeService
}

func NewDisputeHandler(svc *walletservice.DisputeService) *DisputeHandler {
	return &DisputeHandler{service: svc}
}

// POST /orders/:order_id/disputes
func (h *DisputeHandler) Create(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	orderID, err := helper.RequireInt64Param(c, "order_id")
	if err != nil {
		return err
	}
	var req dto.CreateDisputeRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	item, err := h.service.Create(int64(orderID), userID, req.Category, req.Description, req.ImageURLs,
		walletservice.DisputeContact{
			RefundAccount:     req.RefundAccount,
			RefundAccountName: req.RefundAccountName,
			ContactEmail:      req.ContactEmail,
			ContactPhone:      req.ContactPhone,
		})
	if err != nil {
		return helper.MapServiceError(c, err, disputeCreateFallback, disputeCreateResponses)
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

// POST /orders/:order_id/disputes/return — customer attaches return-shipping evidence
func (h *DisputeHandler) SubmitReturn(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	orderID, err := helper.RequireInt64Param(c, "order_id")
	if err != nil {
		return err
	}
	cur, err := h.service.GetByOrderID(int64(orderID))
	if err != nil {
		return helper.MapServiceError(c, err, disputeGetFallback, disputeGetResponses)
	}
	var req dto.SubmitReturnRequest
	if err := helper.RequireBody(c, &req); err != nil {
		return err
	}
	if err := h.service.SubmitReturn(cur.DisputeID, userID, req.TrackingNo, req.Courier, req.Note, req.ImageURLs); err != nil {
		return helper.MapServiceError(c, err, disputeCreateFallback, disputeSubmitReturnResponses)
	}
	item, _ := h.service.GetByOrderID(int64(orderID))
	return c.JSON(item)
}

// GET /orders/:order_id/disputes
func (h *DisputeHandler) GetByOrderID(c *fiber.Ctx) error {
	orderID, err := helper.RequireInt64Param(c, "order_id")
	if err != nil {
		return err
	}
	item, err := h.service.GetByOrderID(int64(orderID))
	if err != nil {
		return helper.MapServiceError(c, err, disputeGetFallback, disputeGetResponses)
	}
	return c.JSON(item)
}

var disputeCreateFallback = helper.ErrorMessage(fiber.StatusInternalServerError, "failed to create dispute")

var disputeCreateResponses = map[error]helper.ErrorResponse{
	walletrepo.ErrOrderNotDisputable:        helper.ErrorMessage(fiber.StatusBadRequest, "ไม่สามารถเปิดคำร้องได้ในสถานะคำสั่งซื้อปัจจุบัน"),
	walletrepo.ErrDisputeWindowExpired:      helper.ErrorMessage(fiber.StatusBadRequest, "เกินระยะเวลา 14 วันหลังคำสั่งซื้อเสร็จสมบูรณ์ ไม่สามารถเปิดคำร้องได้แล้ว"),
	walletrepo.ErrAlreadyReviewed:           helper.ErrorMessage(fiber.StatusBadRequest, "คำสั่งซื้อนี้ถูกรีวิวไปแล้ว ไม่สามารถเปิดคำร้องได้"),
	walletrepo.ErrNotOrderOwner:             helper.ErrorMessage(fiber.StatusForbidden, "เฉพาะเจ้าของคำสั่งซื้อเท่านั้นที่เปิดคำร้องได้"),
	walletservice.ErrInvalidDisputeCategory: helper.ErrorMessage(fiber.StatusBadRequest, "กรุณาเลือกเหตุผล (ไม่ได้รับสินค้า / สินค้าไม่ตรงปก / อื่นๆ)"),
	sql.ErrNoRows:                           helper.ErrorMessage(fiber.StatusNotFound, "ไม่พบคำสั่งซื้อ"),
}

var disputeGetFallback = helper.ErrorMessage(fiber.StatusInternalServerError, "failed to fetch dispute")

var disputeGetResponses = map[error]helper.ErrorResponse{
	sql.ErrNoRows: helper.ErrorMessage(fiber.StatusNotFound, "dispute not found"),
}

var disputeSubmitReturnResponses = map[error]helper.ErrorResponse{
	walletrepo.ErrNotOrderOwner:     helper.ErrorMessage(fiber.StatusForbidden, "เฉพาะเจ้าของคำสั่งซื้อเท่านั้น"),
	walletrepo.ErrReturnNotExpected: helper.ErrorMessage(fiber.StatusBadRequest, "คำร้องนี้ยังไม่อยู่ในขั้นตอนส่งคืนสินค้า"),
	sql.ErrNoRows:                   helper.ErrorMessage(fiber.StatusNotFound, "ไม่พบคำร้อง"),
}

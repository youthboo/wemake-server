package payment

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourusername/wemake/internal/helper"
	"github.com/yourusername/wemake/internal/mailer"
	mediapkg "github.com/yourusername/wemake/internal/media"
	orderrepo "github.com/yourusername/wemake/internal/repository/order"
	tconfigrepo "github.com/yourusername/wemake/internal/repository/tconfig"
	walletrepo "github.com/yourusername/wemake/internal/repository/wallet"
)

type SlipHandler struct {
	slips   *orderrepo.SlipRepository
	wallets *walletrepo.WalletRepository
	tconfig *tconfigrepo.TConfigRepository
	mail    *mailer.Mailer
	cld     *cloudinary.Cloudinary
}

func NewSlipHandler(slips *orderrepo.SlipRepository, wallets *walletrepo.WalletRepository, tconfig *tconfigrepo.TConfigRepository, mail *mailer.Mailer, cld *cloudinary.Cloudinary) *SlipHandler {
	return &SlipHandler{slips: slips, wallets: wallets, tconfig: tconfig, mail: mail, cld: cld}
}

// AttachSlip POST /api/orders/:order_id/slip — customer uploads payment slip
func (h *SlipHandler) AttachSlip(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	orderID, err := helper.ParsePositiveInt64Param(c, "order_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid order_id")
	}

	// Verify ownership + state
	own, err := h.slips.GetOrderOwnership(orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusNotFound, "order not found")
		}
		return helper.JSONInternal(c, "failed to fetch order")
	}
	if own.CustomerID != userID {
		return helper.JSONError(c, fiber.StatusForbidden, "forbidden")
	}
	if own.SlipStatus != "PE" && own.SlipStatus != "RJ" {
		return helper.JSONError(c, fiber.StatusBadRequest, "ไม่สามารถแนบสลีปได้ในสถานะปัจจุบัน")
	}

	// Upload file
	result, err := mediapkg.SaveUploadedFile(c, mediapkg.UploadOptions{
		FieldName:             "file",
		FileNamePrefix:        uuid.NewString(),
		Folder:                "wemake/payment-slips",
		MaxSize:               5 * 1024 * 1024,
		CloudUploadMessage:    "failed to upload slip",
		CloudUploadLogMessage: "cloudinary payment slip upload failed",
		Cloudinary:            h.cld,
		CloudinaryLogFields:   []interface{}{"order_id", orderID, "user_id", userID},
	})
	if err != nil {
		if uploadErr, ok := err.(*mediapkg.UploadError); ok {
			return c.Status(uploadErr.Status).JSON(fiber.Map{"error": uploadErr.Message})
		}
		return helper.JSONError(c, fiber.StatusBadRequest, "กรุณาแนบไฟล์รูปสลีป")
	}

	note := strings.TrimSpace(c.FormValue("note"))

	// Get or create wallet (for transaction FK — kept for backward compat)
	walletID, err := h.wallets.EnsureWalletDirect(userID)
	if err != nil {
		return helper.JSONInternal(c, "failed to ensure wallet")
	}

	if err := h.slips.AttachSlip(orderID, walletID, own.TotalAmount, result.URL, note); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusBadRequest, "ไม่สามารถแนบสลีปได้ในสถานะปัจจุบัน")
		}
		return helper.JSONInternal(c, "failed to attach slip")
	}

	info, _ := h.slips.GetSlipInfo(orderID)

	// E1: email factory — customer attached a payment slip
	if h.mail != nil {
		factoryEmail := h.mail.UserEmail(own.FactoryID)
		if factoryEmail != "" {
			factoryName := h.mail.FactoryName(own.FactoryID)
			webURL := h.mail.WebURL()
			h.mail.SendAsync("SLIP_ATTACHED", factoryEmail, map[string]string{
				"OrderID":     fmt.Sprintf("%d", orderID),
				"FactoryName": factoryName,
				"Amount":      fmt.Sprintf("%.2f", own.TotalAmount),
				"Link":        webURL + fmt.Sprintf("/factory/orders/%d", orderID),
			}, "order", orderID)
		}
	}

	return c.JSON(info)
}

// GetSlip GET /api/orders/:order_id/slip
func (h *SlipHandler) GetSlip(c *fiber.Ctx) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	orderID, err := helper.ParsePositiveInt64Param(c, "order_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid order_id")
	}

	own, err := h.slips.GetOrderOwnership(orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusNotFound, "order not found")
		}
		return helper.JSONInternal(c, "failed to fetch order")
	}

	// Customer or factory owner or admin can view
	role := helper.OptionalRoleFromContext(c)
	if own.CustomerID != userID && own.FactoryID != userID && role != "AD" && role != "SA" {
		return helper.JSONError(c, fiber.StatusForbidden, "forbidden")
	}

	info, err := h.slips.GetSlipInfo(orderID)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch slip info")
	}
	return c.JSON(info)
}

// VerifySlip PATCH /api/factory/orders/:order_id/verify-slip — factory approves/rejects
// (direct-pay flow only; blocked when escrow mode is enabled)
func (h *SlipHandler) VerifySlip(c *fiber.Ctx) error {
	if h.tconfig != nil && h.tconfig.IsEscrowMode() {
		return helper.JSONError(c, fiber.StatusForbidden,
			"ระบบอยู่ในโหมดชำระผ่าน Tryly — การตรวจสอบสลีปทำโดยผู้ดูแลระบบ")
	}
	return h.verifySlip(c, false)
}

// VerifySlipAdmin PATCH /api/admin/orders/:order_id/verify-slip — superadmin approves/rejects
// (escrow mode: funds stay held as PT until the order completes)
func (h *SlipHandler) VerifySlipAdmin(c *fiber.Ctx) error {
	return h.verifySlip(c, true)
}

func (h *SlipHandler) verifySlip(c *fiber.Ctx, asAdmin bool) error {
	userID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	orderID, err := helper.ParsePositiveInt64Param(c, "order_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid order_id")
	}

	own, err := h.slips.GetOrderOwnership(orderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusNotFound, "order not found")
		}
		return helper.JSONInternal(c, "failed to fetch order")
	}
	if !asAdmin && own.FactoryID != userID {
		return helper.JSONError(c, fiber.StatusForbidden, "forbidden")
	}
	if own.SlipStatus != "ST" {
		return helper.JSONError(c, fiber.StatusBadRequest, "ไม่มีสลีปรอตรวจสอบ")
	}

	var body struct {
		Action string `json:"action"`
		Reason string `json:"reason"`
	}
	if err := c.BodyParser(&body); err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}
	body.Action = strings.TrimSpace(strings.ToLower(body.Action))
	body.Reason = strings.TrimSpace(body.Reason)

	escrow := h.tconfig != nil && h.tconfig.IsEscrowMode()

	switch body.Action {
	case "approve":
		var approveErr error
		if asAdmin && escrow {
			// Escrow: hold funds (BU stays PT) + create factory receivable
			approveErr = h.slips.ApproveSlipEscrow(orderID, userID)
		} else {
			approveErr = h.slips.ApproveSlip(orderID, userID)
		}
		if approveErr != nil {
			if errors.Is(approveErr, sql.ErrNoRows) {
				return helper.JSONError(c, fiber.StatusBadRequest, "ไม่สามารถ approve ได้ในสถานะปัจจุบัน")
			}
			return helper.JSONInternal(c, "failed to approve slip")
		}
		// E2: email customer — factory approved the payment slip
		if h.mail != nil {
			customerEmail := h.mail.UserEmail(own.CustomerID)
			if customerEmail != "" {
				factoryName := h.mail.FactoryName(own.FactoryID)
				webURL := h.mail.WebURL()
				h.mail.SendAsync("SLIP_APPROVED", customerEmail, map[string]string{
					"OrderID":     fmt.Sprintf("%d", orderID),
					"FactoryName": factoryName,
					"Link":        webURL + fmt.Sprintf("/orders/%d", orderID),
				}, "order", orderID)
			}
		}
	case "reject":
		if body.Reason == "" {
			return helper.JSONError(c, fiber.StatusBadRequest, "กรุณาระบุเหตุผลในการปฏิเสธ")
		}
		if err := h.slips.RejectSlip(orderID, body.Reason); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return helper.JSONError(c, fiber.StatusBadRequest, "ไม่สามารถ reject ได้ในสถานะปัจจุบัน")
			}
			return helper.JSONInternal(c, "failed to reject slip")
		}
	default:
		return helper.JSONError(c, fiber.StatusBadRequest, "action must be 'approve' or 'reject'")
	}

	info, _ := h.slips.GetSlipInfo(orderID)
	return c.JSON(info)
}

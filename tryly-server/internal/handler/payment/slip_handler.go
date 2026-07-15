package payment

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourusername/wemake/internal/helper"
	"github.com/yourusername/wemake/internal/mailer"
	mediapkg "github.com/yourusername/wemake/internal/media"
	orderrepo "github.com/yourusername/wemake/internal/repository/order"
	tconfigrepo "github.com/yourusername/wemake/internal/repository/tconfig"
	walletrepo "github.com/yourusername/wemake/internal/repository/wallet"
	"github.com/yourusername/wemake/internal/slipok"
)

type SlipHandler struct {
	slips          *orderrepo.SlipRepository
	wallets        *walletrepo.WalletRepository
	tconfig        *tconfigrepo.TConfigRepository
	mail           *mailer.Mailer
	cld            *cloudinary.Cloudinary
	slipok         *slipok.Client
	autoApproveCap float64
}

func NewSlipHandler(slips *orderrepo.SlipRepository, wallets *walletrepo.WalletRepository, tconfig *tconfigrepo.TConfigRepository, mail *mailer.Mailer, cld *cloudinary.Cloudinary, slipokClient *slipok.Client, autoApproveCap float64) *SlipHandler {
	return &SlipHandler{slips: slips, wallets: wallets, tconfig: tconfig, mail: mail, cld: cld, slipok: slipokClient, autoApproveCap: autoApproveCap}
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

	// Escrow mode: try to auto-verify the slip via SlipOK immediately (sync).
	// This is best-effort: on any failure the slip simply stays 'ST' (pending)
	// for manual admin review — the customer can close the page and the factory
	// can still proceed once the slip is approved (item 4: resilient callback).
	if h.tconfig != nil && h.tconfig.IsEscrowMode() && h.slipok.Enabled() {
		h.autoVerifySlip(orderID, own, result.URL)
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

// autoVerifySlip runs SlipOK verification for a freshly-attached slip (escrow).
// Best-effort: never fails the request — it only upgrades the slip to approved
// when every condition passes, otherwise leaves it 'ST' for manual review.
func (h *SlipHandler) autoVerifySlip(orderID int64, own *orderrepo.OrderOwnership, imageURL string) {
	res := h.slipok.VerifyByURL(imageURL)
	switch res.Status {
	case slipok.StatusOK:
		if reason := h.checkSlipConditions(own, res); reason == "" {
			verifiedBy, err := h.slips.PlatformUserID()
			if err != nil {
				log.Printf("[slip] auto-verify order=%d: platform user lookup failed: %v", orderID, err)
				_ = h.slips.RecordSlipVerifyResult(orderID, "pending", "", res.TransferredAt, res.Raw)
				return
			}
			if err := h.slips.AutoApproveSlipEscrow(orderID, verifiedBy, res.TransRef, res.TransferredAt, res.Raw); err != nil {
				if errors.Is(err, orderrepo.ErrDuplicateSlip) {
					log.Printf("[slip] auto-verify order=%d: duplicate bank ref %q", orderID, res.TransRef)
					_ = h.slips.RecordSlipVerifyResult(orderID, "failed", res.TransRef+"#dup", res.TransferredAt, res.Raw)
					return
				}
				log.Printf("[slip] auto-verify order=%d: escrow approve failed: %v", orderID, err)
			}
			return
		} else {
			log.Printf("[slip] auto-verify order=%d: condition failed (%s) — pending manual review", orderID, reason)
			_ = h.slips.RecordSlipVerifyResult(orderID, "failed", res.TransRef+"#chk", res.TransferredAt, res.Raw)
		}
	case slipok.StatusRetry:
		log.Printf("[slip] auto-verify order=%d: bank delay/retry (%d min) — pending", orderID, res.DelayMinutes)
		_ = h.slips.RecordSlipVerifyResult(orderID, "pending", "", res.TransferredAt, res.Raw)
	case slipok.StatusUnavailable:
		log.Printf("[slip] auto-verify order=%d: SlipOK unavailable (%s) — pending manual", orderID, res.Message)
		_ = h.slips.RecordSlipVerifyResult(orderID, "pending", "", time.Time{}, res.Raw)
	default: // StatusInvalid
		log.Printf("[slip] auto-verify order=%d: invalid slip (code=%d) — pending manual", orderID, res.Code)
		_ = h.slips.RecordSlipVerifyResult(orderID, "pending", "", res.TransferredAt, res.Raw)
	}
}

// checkSlipConditions returns "" when the slip may be auto-approved, otherwise a
// short reason. auto-approve requires ALL conditions to pass at once (spec §2).
func (h *SlipHandler) checkSlipConditions(own *orderrepo.OrderOwnership, res *slipok.Result) string {
	// 1. amount matches the order exactly
	if math.Abs(res.Amount-own.TotalAmount) > 0.01 {
		return "amount"
	}
	// 6. amount within the auto-approve cap (high value → force manual)
	if h.autoApproveCap > 0 && res.Amount > h.autoApproveCap {
		return "over_cap"
	}
	// 5. transfer time must be after the order was created and not in the future
	if !res.TransferredAt.IsZero() {
		if res.TransferredAt.Before(own.CreatedAt) {
			return "too_early"
		}
		if res.TransferredAt.After(time.Now().Add(10 * time.Minute)) {
			return "future"
		}
	}
	// 4. receiver = Tryly's central account or PromptPay (masked → normalize compare)
	acc := strings.TrimSpace(h.tconfig.GetValue("tryly_bank_account_no"))
	pp := strings.TrimSpace(h.tconfig.GetValue("tryly_promptpay"))
	matched := false
	if res.ReceiverAccount != "" && acc != "" && slipok.AccountMatches(res.ReceiverAccount, acc) {
		matched = true
	}
	if !matched && res.ReceiverProxy != "" && pp != "" && slipok.AccountMatches(res.ReceiverProxy, pp) {
		matched = true
	}
	if !matched {
		return "receiver"
	}
	return ""
}

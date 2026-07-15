package payment

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
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
	verify := slipVerifyOutcome{Outcome: "pending"}
	if h.tconfig != nil && h.tconfig.IsEscrowMode() && h.slipok.Enabled() {
		// ส่งไฟล์ตรงแบบ multipart — URL ภายใน (localhost/private) SlipOK เข้าถึงไม่ได้
		fileName, fileData := readUploadedSlip(c)
		verify = h.autoVerifySlip(orderID, own, fileName, fileData)
	}

	info, infoErr := h.slips.GetSlipInfo(orderID)
	if infoErr != nil || info == nil {
		info = &orderrepo.SlipInfo{OrderID: orderID, SlipStatus: "ST"}
	}

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

	return c.JSON(fiber.Map{
		"order_id":       orderID,
		"slip_status":    info.SlipStatus,
		"slip_url":       info.SlipURL,
		"slip_note":      info.SlipNote,
		"verified_by":    info.VerifiedBy,
		"verified_at":    info.VerifiedAt,
		"uploaded_at":    info.UploadedAt,
		"verify_outcome": verify.Outcome,
		"verify_reasons": verify.Reasons,
	})
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

// slipVerifyOutcome is returned to the FE so it can render the result screen.
//   - approved: ทุกเงื่อนไขผ่าน → order stamped PD
//   - rejected: สลิปไม่ผ่าน (invalid / เงื่อนไขไม่ตรง / ซ้ำ / ธนาคารหน่วง) → slip
//     ถูก reject (RJ) ให้ลูกค้าแนบสลิปใหม่ได้ทันที
//   - pending: ระบบตรวจใช้ไม่ได้ (unavailable) หรือยอดเกินเพดาน → คง ST รอ admin
type slipVerifyOutcome struct {
	Outcome string   `json:"verify_outcome"`
	Reasons []string `json:"verify_reasons,omitempty"`
}

// readUploadedSlip re-reads the multipart slip file from the request so its
// bytes can be forwarded to SlipOK directly (no public URL required).
func readUploadedSlip(c *fiber.Ctx) (string, []byte) {
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		return "", nil
	}
	f, err := fh.Open()
	if err != nil {
		return "", nil
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 6*1024*1024))
	if err != nil {
		return "", nil
	}
	return fh.Filename, data
}

// autoVerifySlip runs SlipOK verification for a freshly-attached slip (escrow).
// Best-effort: never fails the request. Follows the slip-verification spec:
// separate "this slip is wrong" (reject → re-attach) from "our verifier is
// down" (pending → admin manual fallback), and report ALL failed conditions.
func (h *SlipHandler) autoVerifySlip(orderID int64, own *orderrepo.OrderOwnership, fileName string, fileData []byte) slipVerifyOutcome {
	var res *slipok.Result
	if len(fileData) > 0 {
		res = h.slipok.VerifyByFile(fileName, fileData)
	} else {
		res = &slipok.Result{Status: slipok.StatusUnavailable, Message: "no slip file data"}
	}

	reject := func(reasons ...string) slipVerifyOutcome {
		_ = h.slips.RecordSlipVerifyResult(orderID, "failed", surrogateRef(res.TransRef), res.TransferredAt, res.Raw)
		if err := h.slips.RejectSlip(orderID, strings.Join(reasons, " / ")); err != nil {
			log.Printf("[slip] auto-verify order=%d: reject failed: %v", orderID, err)
		}
		log.Printf("[slip] auto-verify order=%d: rejected (%s)", orderID, strings.Join(reasons, "; "))
		return slipVerifyOutcome{Outcome: "rejected", Reasons: reasons}
	}
	pending := func(reasons ...string) slipVerifyOutcome {
		_ = h.slips.RecordSlipVerifyResult(orderID, "pending", "", res.TransferredAt, res.Raw)
		log.Printf("[slip] auto-verify order=%d: pending manual (%s)", orderID, strings.Join(reasons, "; "))
		return slipVerifyOutcome{Outcome: "pending", Reasons: reasons}
	}

	switch res.Status {
	case slipok.StatusOK:
		hard, soft, capExceeded := h.checkSlipConditions(own, res)
		if len(hard) > 0 {
			// ยอด/บัญชีผู้รับผิด = สลิปใช้กับออเดอร์นี้ไม่ได้จริง → reject ให้แนบใหม่
			// (รวม soft reasons ไปด้วยเพื่อให้ลูกค้าเห็นครบทุกข้อ)
			return reject(append(hard, soft...)...)
		}
		if len(soft) > 0 || capExceeded {
			// สลิปจริง ยอด+บัญชีถูกต้อง แต่ผิดเงื่อนไข "อ่อน" (เวลา) หรือยอดเกินเพดาน
			// → ห้าม reject (ลูกค้าโอนถูกจริง) ให้เข้าคิว admin ตรวจสอบเอง (spec §2)
			reasons := soft
			if capExceeded {
				reasons = append(reasons, "ยอดเกินวงเงินตรวจอัตโนมัติ")
			}
			reasons = append(reasons, "รอเจ้าหน้าที่ตรวจสอบยืนยัน")
			return pending(reasons...)
		}
		verifiedBy, err := h.slips.PlatformUserID()
		if err != nil {
			log.Printf("[slip] auto-verify order=%d: platform user lookup failed: %v", orderID, err)
			return pending("ระบบยืนยันอัตโนมัติขัดข้อง — รอเจ้าหน้าที่ตรวจสอบ")
		}
		if err := h.slips.AutoApproveSlipEscrow(orderID, verifiedBy, res.TransRef, res.TransferredAt, res.Raw); err != nil {
			if errors.Is(err, orderrepo.ErrDuplicateSlip) {
				return reject("สลิปนี้ถูกใช้ยืนยันการชำระเงินไปแล้ว")
			}
			log.Printf("[slip] auto-verify order=%d: escrow approve failed: %v", orderID, err)
			return pending("ระบบยืนยันอัตโนมัติขัดข้อง — รอเจ้าหน้าที่ตรวจสอบ")
		}
		log.Printf("[slip] auto-verify order=%d: approved (ref=%s)", orderID, res.TransRef)
		return slipVerifyOutcome{Outcome: "approved"}
	case slipok.StatusRetry:
		// ธนาคารต้นทางหน่วงข้อมูล — ไม่ใช่สลิปปลอม ให้รอแล้วแนบสลิปเดิมใหม่
		delay := res.DelayMinutes
		if delay <= 0 {
			delay = 15
		}
		return reject(fmt.Sprintf("ธนาคารต้นทางกำลังประมวลผลสลิป กรุณารอประมาณ %d นาทีแล้วแนบสลิปเดิมอีกครั้ง (ไม่ต้องโอนซ้ำ)", delay))
	case slipok.StatusUnavailable:
		// ระบบตรวจของเรามีปัญหา — ห้ามปฏิเสธธุรกรรมของลูกค้า (spec §1.1)
		return pending("ระบบตรวจสลิปไม่พร้อมใช้งานชั่วคราว — เจ้าหน้าที่จะตรวจสอบให้")
	default: // StatusInvalid — รูปไม่ใช่สลิป / QR อ่านไม่ได้ / ไม่มีธุรกรรมจริง
		msg := strings.TrimSpace(res.Message)
		if msg == "" {
			msg = "รูปภาพไม่ใช่สลิปหรืออ่าน QR ไม่ได้"
		}
		return reject(msg + " — กรุณาแนบรูปสลิปใหม่")
	}
}

// surrogateRef stores a suffixed ref for failed slips so the real reference is
// never blocked from being reused on a later valid attempt (spec §6).
func surrogateRef(ref string) string {
	if strings.TrimSpace(ref) == "" {
		return ""
	}
	return ref + "#failed"
}

// checkSlipConditions returns ALL failed conditions (not just the first — spec
// §6), split into:
//   - hard: ยอด/บัญชีผู้รับผิด → สลิปใช้ไม่ได้จริง (reject)
//   - soft: เวลาผิดปกติ บนสลิปจริงที่ยอด+บัญชีถูก → ให้คนตรวจ (manual review)
// plus capExceeded (ยอดเกินเพดาน auto-approve). All empty + cap=false → auto-approve.
func (h *SlipHandler) checkSlipConditions(own *orderrepo.OrderOwnership, res *slipok.Result) (hard []string, soft []string, capExceeded bool) {
	// 1. amount matches the order exactly (HARD — wrong payment)
	if math.Abs(res.Amount-own.TotalAmount) > 0.01 {
		hard = append(hard, fmt.Sprintf("ยอดเงินในสลิป (%.2f บาท) ไม่ตรงกับยอดที่ต้องชำระ (%.2f บาท)", res.Amount, own.TotalAmount))
	}
	// 4. receiver = Tryly's central account or PromptPay (HARD — money went elsewhere)
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
		hard = append(hard, "บัญชีผู้รับเงินในสลิปไม่ใช่บัญชีกลาง Tryly")
	}
	// 5. transfer time must be after the order was created and not in the future
	//    (SOFT — a real correct payment with an odd timestamp goes to manual review,
	//     never an automatic rejection).
	//
	// own.CreatedAt is scanned from orders.created_at, a naive "timestamp without
	// time zone" column that holds Bangkok wall-clock values (see config.GetDSN),
	// but lib/pq mislabels it as UTC on read. Re-anchor to Asia/Bangkok before
	// comparing against res.TransferredAt (correctly zoned by the SlipOK client) —
	// otherwise a same-day slip within ~7h of order creation looks "older" than
	// the order and is wrongly flagged.
	if !res.TransferredAt.IsZero() {
		orderCreatedAt := helper.AsThailandWallClock(own.CreatedAt)
		if res.TransferredAt.Before(orderCreatedAt) {
			soft = append(soft, "เวลาโอนในสลิปอยู่ก่อนการสร้างคำสั่งซื้อ")
		}
		if res.TransferredAt.After(time.Now().Add(10 * time.Minute)) {
			soft = append(soft, "เวลาโอนในสลิปอยู่ในอนาคต")
		}
	}
	// 6. amount within the auto-approve cap (high value → force manual review)
	capExceeded = h.autoApproveCap > 0 && res.Amount > h.autoApproveCap
	return hard, soft, capExceeded
}

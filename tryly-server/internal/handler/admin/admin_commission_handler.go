package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/helper"
	"github.com/yourusername/wemake/internal/mailer"
	adminrepo "github.com/yourusername/wemake/internal/repository/admin"
)

type AdminCommissionHandler struct {
	invoices *adminrepo.CommissionInvoiceRepository
	mail     *mailer.Mailer
}

func NewAdminCommissionHandler(invoices *adminrepo.CommissionInvoiceRepository, mail *mailer.Mailer) *AdminCommissionHandler {
	return &AdminCommissionHandler{invoices: invoices, mail: mail}
}

// GenerateInvoices POST /api/admin/invoices/generate
func (h *AdminCommissionHandler) GenerateInvoices(c *fiber.Ctx) error {
	var body struct {
		Month int `json:"month"`
		Year  int `json:"year"`
	}
	if err := c.BodyParser(&body); err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}
	if body.Month == 0 {
		body.Month = int(time.Now().Month())
	}
	if body.Year == 0 {
		body.Year = time.Now().Year()
	}
	if body.Month < 1 || body.Month > 12 || body.Year < 2020 {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid month/year")
	}

	created, err := h.invoices.GenerateForMonth(body.Month, body.Year)
	if err != nil {
		return helper.JSONInternal(c, "failed to generate invoices")
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"created":      created,
		"period_month": body.Month,
		"period_year":  body.Year,
	})
}

// ListInvoices GET /api/admin/invoices
func (h *AdminCommissionHandler) ListInvoices(c *fiber.Ctx) error {
	month, _ := strconv.Atoi(c.Query("month", "0"))
	year, _ := strconv.Atoi(c.Query("year", "0"))

	items, err := h.invoices.ListAll(month, year)
	if err != nil {
		return helper.JSONInternal(c, "failed to list invoices")
	}
	return c.JSON(fiber.Map{"invoices": items, "total": len(items)})
}

// GetInvoice GET /api/admin/invoices/:invoice_id
func (h *AdminCommissionHandler) GetInvoice(c *fiber.Ctx) error {
	invoiceID, err := helper.ParsePositiveInt64Param(c, "invoice_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid invoice_id")
	}

	invoice, err := h.invoices.GetByID(invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusNotFound, "invoice not found")
		}
		return helper.JSONInternal(c, "failed to fetch invoice")
	}

	items, _ := h.invoices.GetItems(invoiceID)

	return c.JSON(fiber.Map{
		"invoice": invoice,
		"items":   items,
	})
}

// VerifyInvoiceSlip PATCH /api/admin/invoices/:invoice_id/verify
func (h *AdminCommissionHandler) VerifyInvoiceSlip(c *fiber.Ctx) error {
	adminID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	invoiceID, err := helper.ParsePositiveInt64Param(c, "invoice_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid invoice_id")
	}

	var body struct {
		Action string `json:"action"`
	}
	if err := c.BodyParser(&body); err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}
	body.Action = strings.TrimSpace(strings.ToLower(body.Action))

	approve := false
	switch body.Action {
	case "approve":
		approve = true
	case "reject":
		approve = false
	default:
		return helper.JSONError(c, fiber.StatusBadRequest, "action must be 'approve' or 'reject'")
	}

	if err := h.invoices.AdminVerifySlip(invoiceID, adminID, approve); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusBadRequest, "ไม่มีสลีปรอตรวจสอบ")
		}
		return helper.JSONInternal(c, "failed to verify invoice")
	}

	invoice, _ := h.invoices.GetByID(invoiceID)
	return c.JSON(invoice)
}

// SendInvoice POST /api/admin/invoices/:invoice_id/send — mark as sent + email factory
func (h *AdminCommissionHandler) SendInvoice(c *fiber.Ctx) error {
	invoiceID, err := helper.ParsePositiveInt64Param(c, "invoice_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid invoice_id")
	}

	if err := h.invoices.MarkSent(invoiceID); err != nil {
		return helper.JSONInternal(c, "failed to mark invoice as sent")
	}

	invoice, _ := h.invoices.GetByID(invoiceID)
	h.sendCommissionEmail(invoice, invoiceID)
	return c.JSON(invoice)
}

// ResendInvoice POST /api/admin/invoices/:invoice_id/resend — resend email without changing status
func (h *AdminCommissionHandler) ResendInvoice(c *fiber.Ctx) error {
	invoiceID, err := helper.ParsePositiveInt64Param(c, "invoice_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid invoice_id")
	}

	invoice, err := h.invoices.GetByID(invoiceID)
	if err != nil {
		return helper.JSONError(c, fiber.StatusNotFound, "invoice not found")
	}

	if err := h.invoices.UpdateEmailSentAt(invoiceID); err != nil {
		return helper.JSONInternal(c, "failed to update email timestamp")
	}

	h.sendCommissionEmail(invoice, invoiceID)

	invoice, _ = h.invoices.GetByID(invoiceID)
	return c.JSON(invoice)
}

func (h *AdminCommissionHandler) sendCommissionEmail(invoice *adminrepo.CommissionInvoice, invoiceID int64) {
	if h.mail == nil || invoice == nil {
		return
	}
	factoryEmail := h.mail.UserEmail(invoice.FactoryID)
	if factoryEmail == "" {
		return
	}
	factoryName := h.mail.FactoryName(invoice.FactoryID)
	webURL := h.mail.WebURL()
	h.mail.SendAsync("COMMISSION_INVOICE", factoryEmail, map[string]string{
		"FactoryName":      factoryName,
		"PeriodMonth":      fmt.Sprintf("%d", invoice.PeriodMonth),
		"PeriodYear":       fmt.Sprintf("%d", invoice.PeriodYear),
		"TotalOrders":      fmt.Sprintf("%d", invoice.TotalOrders),
		"CommissionAmount": fmt.Sprintf("%.2f", invoice.CommissionAmount),
		"VatAmount":        fmt.Sprintf("%.2f", invoice.VatAmount),
		"GrandTotal":       fmt.Sprintf("%.2f", invoice.GrandTotal),
		"Link":             webURL + fmt.Sprintf("/factory/invoices/%d", invoiceID),
	}, "invoice", invoiceID)
}

// ListEscrowCommissions GET /api/admin/commission/orders
// ค่าคอมมิชชันที่ Tryly ได้รับต่อออเดอร์ (escrow flow)
func (h *AdminCommissionHandler) ListEscrowCommissions(c *fiber.Ctx) error {
	rows, err := h.invoices.ListEscrowCommissions()
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch escrow commissions")
	}
	var total, totalGross float64
	for _, r := range rows {
		total += r.CommissionAmt
		totalGross += r.GrandTotal
	}
	return c.JSON(fiber.Map{
		"orders":            rows,
		"total_commission":  total,
		"total_gross":       totalGross,
		"count":             len(rows),
	})
}

// GetSummary GET /api/admin/commission/summary
func (h *AdminCommissionHandler) GetSummary(c *fiber.Ctx) error {
	month, _ := strconv.Atoi(c.Query("month", "0"))
	year, _ := strconv.Atoi(c.Query("year", "0"))

	summary, err := h.invoices.GetSummary(month, year)
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch commission summary")
	}
	return c.JSON(summary)
}

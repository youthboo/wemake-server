package factory

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yourusername/wemake/internal/helper"
	"github.com/yourusername/wemake/internal/mailer"
	mediapkg "github.com/yourusername/wemake/internal/media"
	adminrepo "github.com/yourusername/wemake/internal/repository/admin"
)

type InvoiceHandler struct {
	invoices *adminrepo.CommissionInvoiceRepository
	mail     *mailer.Mailer
	cld      *cloudinary.Cloudinary
}

func NewInvoiceHandler(invoices *adminrepo.CommissionInvoiceRepository, mail *mailer.Mailer, cld *cloudinary.Cloudinary) *InvoiceHandler {
	return &InvoiceHandler{invoices: invoices, mail: mail, cld: cld}
}

// ListMyInvoices GET /api/factories/me/invoices
func (h *InvoiceHandler) ListMyInvoices(c *fiber.Ctx) error {
	factoryID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}

	items, err := h.invoices.ListByFactory(factoryID)
	if err != nil {
		return helper.JSONInternal(c, "failed to list invoices")
	}
	return c.JSON(fiber.Map{"invoices": items, "total": len(items)})
}

// GetMyInvoice GET /api/factories/me/invoices/:invoice_id
func (h *InvoiceHandler) GetMyInvoice(c *fiber.Ctx) error {
	factoryID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
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
	if invoice.FactoryID != factoryID {
		return helper.JSONError(c, fiber.StatusNotFound, "invoice not found")
	}

	items, _ := h.invoices.GetItems(invoiceID)

	return c.JSON(fiber.Map{
		"invoice": invoice,
		"items":   items,
	})
}

// GetCurrentPeriod GET /api/factories/me/invoices/current-period
func (h *InvoiceHandler) GetCurrentPeriod(c *fiber.Ctx) error {
	factoryID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	now := time.Now()
	summary, err := h.invoices.GetCurrentPeriod(factoryID, int(now.Month()), now.Year())
	if err != nil {
		return helper.JSONInternal(c, "failed to fetch current period")
	}
	return c.JSON(summary)
}

// AttachCommSlip POST /api/factories/me/invoices/:invoice_id/slip
func (h *InvoiceHandler) AttachCommSlip(c *fiber.Ctx) error {
	factoryID, err := helper.RequireAuthenticatedUserID(c)
	if err != nil {
		return err
	}
	invoiceID, err := helper.ParsePositiveInt64Param(c, "invoice_id")
	if err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid invoice_id")
	}

	// Verify ownership
	invoice, err := h.invoices.GetByID(invoiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusNotFound, "invoice not found")
		}
		return helper.JSONInternal(c, "failed to fetch invoice")
	}
	if invoice.FactoryID != factoryID {
		return helper.JSONError(c, fiber.StatusNotFound, "invoice not found")
	}

	status := strings.TrimSpace(invoice.Status)
	if status != "ST" && status != "RJ" {
		return helper.JSONError(c, fiber.StatusBadRequest, "ไม่สามารถแนบสลีปได้ในสถานะปัจจุบัน")
	}

	// Upload file
	result, err := mediapkg.SaveUploadedFile(c, mediapkg.UploadOptions{
		FieldName:             "file",
		FileNamePrefix:        uuid.NewString(),
		Folder:                "wemake/commission-slips",
		MaxSize:               5 * 1024 * 1024,
		CloudUploadMessage:    "failed to upload slip",
		CloudUploadLogMessage: "cloudinary commission slip upload failed",
		Cloudinary:            h.cld,
		CloudinaryLogFields:   []interface{}{"invoice_id", invoiceID, "factory_id", factoryID},
	})
	if err != nil {
		if uploadErr, ok := err.(*mediapkg.UploadError); ok {
			return c.Status(uploadErr.Status).JSON(fiber.Map{"error": uploadErr.Message})
		}
		return helper.JSONError(c, fiber.StatusBadRequest, "กรุณาแนบไฟล์รูปสลีป")
	}

	note := strings.TrimSpace(c.FormValue("note"))

	if err := h.invoices.AttachFactorySlip(invoiceID, result.URL, note); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return helper.JSONError(c, fiber.StatusBadRequest, "ไม่สามารถแนบสลีปได้ในสถานะปัจจุบัน")
		}
		return helper.JSONInternal(c, "failed to attach slip")
	}

	updated, _ := h.invoices.GetByID(invoiceID)

	// E4: email SA — factory attached commission slip
	if h.mail != nil && updated != nil {
		factoryName := h.mail.FactoryName(factoryID)
		webURL := h.mail.WebURL()
		for _, saEmail := range h.mail.SAEmails() {
			h.mail.SendAsync("COMM_SLIP_ATTACHED", saEmail, map[string]string{
				"FactoryName": factoryName,
				"InvoiceID":   fmt.Sprintf("%d", invoiceID),
				"PeriodMonth": fmt.Sprintf("%d", updated.PeriodMonth),
				"PeriodYear":  fmt.Sprintf("%d", updated.PeriodYear),
				"GrandTotal":  fmt.Sprintf("%.2f", updated.GrandTotal),
				"Link":        webURL + fmt.Sprintf("/admin/invoices/%d", invoiceID),
			}, "invoice", invoiceID)
		}
	}

	return c.JSON(updated)
}

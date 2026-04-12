package handler

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

type PaymentScheduleHandler struct {
	service *service.PaymentScheduleService
}

func NewPaymentScheduleHandler(svc *service.PaymentScheduleService) *PaymentScheduleHandler {
	return &PaymentScheduleHandler{service: svc}
}

// GET /orders/:order_id/payment-schedules
func (h *PaymentScheduleHandler) List(c *fiber.Ctx) error {
	orderID, err := c.ParamsInt("order_id")
	if err != nil || orderID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	items, err := h.service.ListByOrderID(int64(orderID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch payment schedules"})
	}
	return c.JSON(items)
}

// POST /orders/:order_id/payment-schedules
func (h *PaymentScheduleHandler) Create(c *fiber.Ctx) error {
	type reqBody struct {
		InstallmentNo int     `json:"installment_no"`
		DueDate       string  `json:"due_date"` // YYYY-MM-DD
		Amount        float64 `json:"amount"`
	}
	orderID, err := c.ParamsInt("order_id")
	if err != nil || orderID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid order_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if req.InstallmentNo <= 0 || strings.TrimSpace(req.DueDate) == "" || req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "installment_no, due_date, and amount are required"})
	}
	dueDate, err := time.Parse("2006-01-02", strings.TrimSpace(req.DueDate))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "due_date must be YYYY-MM-DD"})
	}
	item := &domain.PaymentSchedule{
		OrderID:       int64(orderID),
		InstallmentNo: req.InstallmentNo,
		DueDate:       dueDate,
		Amount:        req.Amount,
	}
	if err := h.service.CreateSchedule(item); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create payment schedule"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

// PATCH /payment-schedules/:schedule_id
func (h *PaymentScheduleHandler) PatchStatus(c *fiber.Ctx) error {
	type reqBody struct {
		Status string `json:"status"`
	}
	scheduleID, err := strconv.ParseInt(c.Params("schedule_id"), 10, 64)
	if err != nil || scheduleID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid schedule_id"})
	}
	var req reqBody
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request payload"})
	}
	if err := h.service.PatchStatus(scheduleID, req.Status); err != nil {
		if errors.Is(err, service.ErrInvalidScheduleStatus) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, sql.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "payment schedule not found"})
		}
		if repository.IsNotFoundError(err) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "payment schedule not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update payment schedule"})
	}
	return c.JSON(fiber.Map{"message": "payment schedule updated"})
}

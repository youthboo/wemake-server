package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/service"
)

type AdminDashboardHandler struct {
	service *service.AdminDashboardService
}

func NewAdminDashboardHandler(service *service.AdminDashboardService) *AdminDashboardHandler {
	return &AdminDashboardHandler{service: service}
}

func (h *AdminDashboardHandler) GetSummary(c *fiber.Ctx) error {
	period := c.Query("period", "month")
	from, to, err := parseAdminPeriod(period, c.Query("date_from"), c.Query("date_to"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	item, err := h.service.GetSummary(from, to, period)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch dashboard summary"})
	}
	return c.JSON(item)
}

func (h *AdminDashboardHandler) GetRevenueChart(c *fiber.Ctx) error {
	granularity := c.Query("granularity", "day")
	from, to, err := parseAdminPeriod("custom", c.Query("date_from"), c.Query("date_to"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	items, err := h.service.GetRevenueChart(from, to, granularity)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch revenue chart"})
	}
	return c.JSON(fiber.Map{"granularity": granularity, "data": items})
}

func (h *AdminDashboardHandler) GetTopFactories(c *fiber.Ctx) error {
	from, to, err := parseAdminPeriod("custom", c.Query("date_from"), c.Query("date_to"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	items, err := h.service.GetTopFactories(from, to, c.QueryInt("limit", 10))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to fetch top factories"})
	}
	return c.JSON(fiber.Map{"data": items})
}

func parseAdminPeriod(period, rawFrom, rawTo string) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	switch period {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, start.Add(24 * time.Hour), nil
	case "week":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start.AddDate(0, 0, -6), start.Add(24 * time.Hour), nil
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0), nil
	case "quarter":
		month := ((int(now.Month())-1)/3)*3 + 1
		start := time.Date(now.Year(), time.Month(month), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 3, 0), nil
	case "year":
		start := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(1, 0, 0), nil
	case "custom":
		if rawFrom == "" || rawTo == "" {
			return time.Time{}, time.Time{}, fiber.NewError(fiber.StatusBadRequest, "date_from and date_to are required")
		}
		from, err := time.Parse("2006-01-02", rawFrom)
		if err != nil {
			return time.Time{}, time.Time{}, fiber.NewError(fiber.StatusBadRequest, "date_from must be YYYY-MM-DD")
		}
		to, err := time.Parse("2006-01-02", rawTo)
		if err != nil {
			return time.Time{}, time.Time{}, fiber.NewError(fiber.StatusBadRequest, "date_to must be YYYY-MM-DD")
		}
		return from, to.Add(24 * time.Hour), nil
	default:
		return time.Time{}, time.Time{}, fiber.NewError(fiber.StatusBadRequest, "invalid period")
	}
}

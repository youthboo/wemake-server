package tconfig

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/helper"
	"github.com/jmoiron/sqlx"
)

type TcronjobHandler struct {
	db *sqlx.DB
}

func NewTcronjobHandler(db *sqlx.DB) *TcronjobHandler {
	return &TcronjobHandler{db: db}
}

type CronJobRow struct {
	JobKey        string  `db:"job_key"        json:"job_key"`
	ScheduleType  string  `db:"schedule_type"  json:"schedule_type"`
	ScheduleValue string  `db:"schedule_value" json:"schedule_value"`
	Hour          int     `db:"hour"           json:"hour"`
	Enabled       bool    `db:"enabled"        json:"enabled"`
	Description   *string `db:"description"    json:"description,omitempty"`
	LastRunAt     *string `db:"last_run_at"    json:"last_run_at,omitempty"`
}

// ListJobs GET /admin/cronjobs
func (h *TcronjobHandler) ListJobs(c *fiber.Ctx) error {
	var rows []CronJobRow
	if err := h.db.Select(&rows, `
		SELECT job_key, schedule_type, schedule_value, hour, enabled, description,
		       last_run_at::text AS last_run_at
		FROM tcronjob ORDER BY job_key
	`); err != nil {
		return helper.JSONInternal(c, "failed to list cronjobs")
	}
	return c.JSON(fiber.Map{"jobs": rows})
}

// UpdateJob PATCH /admin/cronjobs/:job_key
func (h *TcronjobHandler) UpdateJob(c *fiber.Ctx) error {
	jobKey := c.Params("job_key")
	if jobKey == "" {
		return helper.JSONError(c, fiber.StatusBadRequest, "job_key required")
	}
	var body struct {
		ScheduleType  *string `json:"schedule_type"`
		ScheduleValue *string `json:"schedule_value"`
		Hour          *int    `json:"hour"`
		Enabled       *bool   `json:"enabled"`
	}
	if err := c.BodyParser(&body); err != nil {
		return helper.JSONError(c, fiber.StatusBadRequest, "invalid request body")
	}

	res, err := h.db.Exec(`
		UPDATE tcronjob SET
			schedule_type  = COALESCE($2, schedule_type),
			schedule_value = COALESCE($3, schedule_value),
			hour           = COALESCE($4, hour),
			enabled        = COALESCE($5, enabled),
			updated_at     = NOW()
		WHERE job_key = $1
	`, jobKey, body.ScheduleType, body.ScheduleValue, body.Hour, body.Enabled)
	if err != nil {
		return helper.JSONInternal(c, "failed to update cronjob")
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return helper.JSONError(c, fiber.StatusNotFound, "job not found")
	}

	var row CronJobRow
	_ = h.db.Get(&row, `SELECT job_key, schedule_type, schedule_value, hour, enabled, description, last_run_at::text AS last_run_at FROM tcronjob WHERE job_key=$1`, jobKey)
	return c.JSON(row)
}

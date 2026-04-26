package repository

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type AdminAuditRepository struct {
	db *sqlx.DB
}

func NewAdminAuditRepository(db *sqlx.DB) *AdminAuditRepository {
	return &AdminAuditRepository{db: db}
}

func (r *AdminAuditRepository) Insert(log *domain.AdminAuditLog) error {
	return r.db.QueryRow(`
		INSERT INTO admin_audit_log (actor_id, action, target_type, target_id, payload, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING log_id, created_at
	`, log.ActorID, log.Action, log.TargetType, log.TargetID, log.Payload, nullableStringPtr(log.IPAddress)).Scan(&log.LogID, &log.CreatedAt)
}

func (r *AdminAuditRepository) List(filter domain.AdminAuditFilter) ([]domain.AdminAuditLog, int, error) {
	page, pageSize := normalizePage(filter.Page, filter.PageSize)
	where := []string{"1=1"}
	args := make([]interface{}, 0)
	addArg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.ActorID != nil {
		where = append(where, "l.actor_id = "+addArg(*filter.ActorID))
	}
	if filter.Action != "" {
		where = append(where, "l.action = "+addArg(strings.TrimSpace(strings.ToUpper(filter.Action))))
	}
	if filter.TargetType != "" {
		where = append(where, "l.target_type = "+addArg(strings.TrimSpace(filter.TargetType)))
	}
	if filter.DateFrom != nil {
		where = append(where, "l.created_at >= "+addArg(*filter.DateFrom))
	}
	if filter.DateTo != nil {
		where = append(where, "l.created_at < "+addArg(filter.DateTo.Add(24*time.Hour)))
	}
	condition := strings.Join(where, " AND ")

	var total int
	if err := r.db.Get(&total, `SELECT COUNT(*) FROM admin_audit_log l WHERE `+condition, args...); err != nil {
		return nil, 0, err
	}

	items := []domain.AdminAuditLog{}
	args = append(args, pageSize, (page-1)*pageSize)
	if err := r.db.Select(&items, `
		SELECT
			l.log_id,
			l.actor_id,
			u.email AS actor_email,
			l.action,
			l.target_type,
			l.target_id,
			l.payload,
			l.ip_address,
			l.created_at
		FROM admin_audit_log l
		LEFT JOIN users u ON u.user_id = l.actor_id
		WHERE `+condition+`
		ORDER BY l.created_at DESC, l.log_id DESC
		LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args))+`
	`, args...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

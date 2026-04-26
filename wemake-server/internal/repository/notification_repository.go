package repository

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type NotificationRepository struct {
	db *sqlx.DB
}

func NewNotificationRepository(db *sqlx.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) ListByUserID(userID int64) ([]domain.Notification, error) {
	var items []domain.Notification
	query := `SELECT noti_id, user_id, type, title, message, link_to, is_read, read_at, data, reference_id, deleted_at, created_at
		FROM notifications WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`
	err := r.db.Select(&items, query, userID)
	return items, err
}

func (r *NotificationRepository) MarkAsRead(notiID, userID int64) error {
	query := `UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE noti_id = $1 AND user_id = $2 AND deleted_at IS NULL`
	_, err := r.db.Exec(query, notiID, userID)
	return err
}

func (r *NotificationRepository) Create(noti *domain.Notification) error {
	query := `
		INSERT INTO notifications (user_id, type, title, message, link_to, reference_id, data)
		VALUES (:user_id, :type, :title, :message, :link_to, :reference_id, :data)
		RETURNING noti_id, created_at, is_read, read_at
	`
	rows, err := r.db.NamedQuery(query, noti)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&noti.NotiID, &noti.CreatedAt, &noti.IsRead, &noti.ReadAt)
	}
	rows.Close()
	return err
}

func (r *NotificationRepository) ListPaginated(userID int64, page, limit int, unreadOnly bool) ([]domain.Notification, int64, int64, error) {
	offset := (page - 1) * limit
	where := "user_id = $1 AND deleted_at IS NULL"
	args := []interface{}{userID}
	if unreadOnly {
		where += " AND is_read = FALSE"
	}
	var total int64
	if err := r.db.Get(&total, `SELECT COUNT(*) FROM notifications WHERE `+where, args...); err != nil {
		return nil, 0, 0, err
	}
	var unreadCount int64
	if err := r.db.Get(&unreadCount, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE AND deleted_at IS NULL`, userID); err != nil {
		return nil, 0, 0, err
	}
	query := fmt.Sprintf(`SELECT noti_id, user_id, type, title, message, link_to, is_read, read_at, data, reference_id, deleted_at, created_at
		FROM notifications WHERE %s ORDER BY created_at DESC LIMIT $2 OFFSET $3`, where)
	var items []domain.Notification
	if err := r.db.Select(&items, query, userID, limit, offset); err != nil {
		return nil, 0, 0, err
	}
	return items, total, unreadCount, nil
}

func (r *NotificationRepository) GetUnreadCount(userID int64) (int64, error) {
	var count int64
	err := r.db.Get(&count, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE AND deleted_at IS NULL`, userID)
	return count, err
}

func (r *NotificationRepository) MarkAllRead(userID int64) (int64, error) {
	res, err := r.db.Exec(`UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND is_read = FALSE AND deleted_at IS NULL`, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *NotificationRepository) SoftDelete(notiID, userID int64) error {
	_, err := r.db.Exec(`UPDATE notifications SET deleted_at = NOW() WHERE noti_id = $1 AND user_id = $2 AND deleted_at IS NULL`, notiID, userID)
	return err
}

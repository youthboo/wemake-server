package repository

import (
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
	query := `SELECT * FROM notifications WHERE user_id = $1 ORDER BY created_at DESC`
	err := r.db.Select(&items, query, userID)
	return items, err
}

func (r *NotificationRepository) MarkAsRead(notiID, userID int64) error {
	query := `UPDATE notifications SET is_read = TRUE WHERE noti_id = $1 AND user_id = $2`
	_, err := r.db.Exec(query, notiID, userID)
	return err
}

func (r *NotificationRepository) Create(noti *domain.Notification) error {
	query := `
		INSERT INTO notifications (user_id, type, title, message, link_to, reference_id)
		VALUES (:user_id, :type, :title, :message, :link_to, :reference_id)
		RETURNING noti_id, created_at, is_read
	`
	rows, err := r.db.NamedQuery(query, noti)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&noti.NotiID, &noti.CreatedAt, &noti.IsRead)
	}
	rows.Close()
	return err
}

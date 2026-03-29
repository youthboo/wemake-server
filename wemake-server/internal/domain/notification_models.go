package domain

import "time"

type Notification struct {
	NotiID      int64     `db:"noti_id" json:"noti_id"`
	UserID      int64     `db:"user_id" json:"user_id"`
	Type        string    `db:"type" json:"type"`
	Title       string    `db:"title" json:"title"`
	Message     string    `db:"message" json:"message"`
	LinkTo      string    `db:"link_to" json:"link_to"`
	IsRead      bool      `db:"is_read" json:"is_read"`
	ReferenceID *int64    `db:"reference_id" json:"reference_id,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

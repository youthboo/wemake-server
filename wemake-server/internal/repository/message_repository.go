package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type MessageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(item *domain.Message) error {
	query := `
		INSERT INTO messages (message_id, reference_type, reference_id, sender_id, receiver_id, content, attachment_url, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(
		query,
		item.MessageID,
		item.ReferenceType,
		item.ReferenceID,
		item.SenderID,
		item.ReceiverID,
		item.Content,
		item.AttachmentURL,
		item.CreatedAt,
	)
	return err
}

func (r *MessageRepository) ListByReference(referenceType, referenceID string, userID int64) ([]domain.Message, error) {
	var items []domain.Message
	query := `
		SELECT message_id, reference_type, reference_id, sender_id, receiver_id, content, attachment_url, created_at
		FROM messages
		WHERE reference_type = $1 AND reference_id = $2 AND (sender_id = $3 OR receiver_id = $3)
		ORDER BY created_at ASC
	`
	err := r.db.Select(&items, query, referenceType, referenceID, userID)
	return items, err
}

func (r *MessageRepository) ListThreads(userID int64) ([]domain.MessageThread, error) {
	var items []domain.MessageThread
	query := `
		SELECT m.reference_type, m.reference_id, m.content AS last_message, m.created_at AS last_message_at
		FROM messages m
		INNER JOIN (
			SELECT reference_type, reference_id, MAX(created_at) AS max_created_at
			FROM messages
			WHERE sender_id = $1 OR receiver_id = $1
			GROUP BY reference_type, reference_id
		) latest
		ON m.reference_type = latest.reference_type
		   AND m.reference_id = latest.reference_id
		   AND m.created_at = latest.max_created_at
		ORDER BY m.created_at DESC
	`
	err := r.db.Select(&items, query, userID)
	return items, err
}

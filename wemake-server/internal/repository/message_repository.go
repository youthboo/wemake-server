package repository

import (
	"database/sql"
	"fmt"

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
	return r.CreateTx(r.db, item)
}

func (r *MessageRepository) CreateTx(exec interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}, item *domain.Message) error {
	if item.MessageType == "" {
		item.MessageType = "TX"
	}
	query := `
		INSERT INTO messages (message_id, reference_type, reference_id, sender_id, receiver_id, content, attachment_url, created_at, conv_id, message_type, quote_data, boq_rfq_id, is_read)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := exec.Exec(
		query,
		item.MessageID,
		nullableMessageReferenceType(item.ReferenceType),
		nullableMessageReferenceID(item.ReferenceID),
		item.SenderID,
		item.ReceiverID,
		item.Content,
		item.AttachmentURL,
		item.CreatedAt,
		item.ConvID,
		item.MessageType,
		item.QuoteData,
		nullableInt64Value(item.BOQRfqID),
		item.IsRead,
	)
	return err
}

func (r *MessageRepository) ReferenceExists(referenceType string, referenceID int64) (bool, error) {
	var exists bool
	var query string
	var args []interface{}

	switch referenceType {
	case "RQ":
		query = `SELECT EXISTS (SELECT 1 FROM rfqs WHERE rfq_id = $1)`
		args = []interface{}{referenceID}
	case "OD":
		query = `SELECT EXISTS (SELECT 1 FROM orders WHERE order_id = $1)`
		args = []interface{}{referenceID}
	case "PD", "PM", "ID":
		query = `SELECT EXISTS (SELECT 1 FROM factory_showcases WHERE showcase_id = $1 AND content_type = $2)`
		args = []interface{}{referenceID, referenceType}
	default:
		return false, fmt.Errorf("unsupported reference_type: %s", referenceType)
	}

	if err := r.db.Get(&exists, query, args...); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *MessageRepository) ListByReference(referenceType string, referenceID int64, userID int64) ([]domain.Message, error) {
	var items []domain.Message
	query := `
		SELECT m.message_id,
		       COALESCE(m.reference_type, '') AS reference_type,
		       COALESCE(m.reference_id, 0)    AS reference_id,
		       CASE WHEN m.reference_type = 'RQ' THEN rq.title ELSE NULL END AS rfq_title,
		       m.sender_id, m.receiver_id, m.content, m.attachment_url,
		       m.created_at, m.conv_id, m.message_type, m.quote_data, m.boq_rfq_id, m.is_read
		FROM messages m
		LEFT JOIN rfqs rq ON rq.rfq_id = m.reference_id AND m.reference_type = 'RQ'
		WHERE m.reference_type = $1 AND m.reference_id = $2 AND (m.sender_id = $3 OR m.receiver_id = $3)
		ORDER BY m.created_at ASC
	`
	err := r.db.Select(&items, query, referenceType, referenceID, userID)
	return items, err
}

func (r *MessageRepository) ListByConvID(convID int64) ([]domain.Message, error) {
	var items []domain.Message
	query := `
		SELECT m.message_id,
		       COALESCE(m.reference_type, '') AS reference_type,
		       COALESCE(m.reference_id, 0)    AS reference_id,
		       CASE WHEN m.reference_type = 'RQ' THEN rq.title ELSE NULL END AS rfq_title,
		       m.sender_id, m.receiver_id, m.content, m.attachment_url,
		       m.created_at, m.conv_id, m.message_type, m.quote_data, m.boq_rfq_id, m.is_read
		FROM messages m
		LEFT JOIN rfqs rq ON rq.rfq_id = m.reference_id AND m.reference_type = 'RQ'
		WHERE m.conv_id = $1
		ORDER BY m.created_at ASC
	`
	err := r.db.Select(&items, query, convID)
	return items, err
}

func (r *MessageRepository) ListThreads(userID int64) ([]domain.MessageThread, error) {
	var items []domain.MessageThread
	query := `
		SELECT COALESCE(m.reference_type, '') AS reference_type,
		       COALESCE(m.reference_id, 0) AS reference_id,
		       m.content AS last_message,
		       m.created_at AS last_message_at
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

func nullableMessageReferenceType(referenceType string) interface{} {
	if referenceType == "" {
		return nil
	}
	return referenceType
}

func nullableMessageReferenceID(referenceID int64) interface{} {
	if referenceID <= 0 {
		return nil
	}
	return referenceID
}

package message

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
	"github.com/yourusername/wemake/internal/domainutil"
)

type MessageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) DB() *sqlx.DB {
	return r.db
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
	// reference_type = "สิ่งที่ข้อความอ้างถึง" (RQ/OD/PD/PM/ID) — เก็บตรง ๆ
	// แยกจาก message_type ("รูปแบบข้อความ"). reference_id = PK ของ entity นั้น
	query := `
		INSERT INTO messages (reference_type, reference_id, sender_id, receiver_id, content, attachment_url, created_at, conv_id, message_type, quote_data, is_read)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := exec.Exec(
		query,
		domainutil.NullableString(item.ReferenceType),
		domainutil.NullablePositiveInt64(item.ReferenceID),
		item.SenderID,
		item.ReceiverID,
		item.Content,
		item.AttachmentURL,
		item.CreatedAt,
		item.ConvID,
		item.MessageType,
		item.QuoteData,
		item.IsRead,
	)
	return err
}

// ReferenceExists checks whether the entity pointed to by referenceID exists.
// referenceType is derived from message_type at the service layer.
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
		query = `SELECT EXISTS (SELECT 1 FROM factory_showcases WHERE showcase_id = $1)`
		args = []interface{}{referenceID}
	default:
		return false, fmt.Errorf("unsupported reference_type: %s", referenceType)
	}

	if err := r.db.Get(&exists, query, args...); err != nil {
		return false, err
	}
	return exists, nil
}

// referenceTypeFromMessageType derives the logical reference category from
// message_type so we can drive JOINs without a reference_type DB column.
func referenceTypeFromMessageType(mt string) string {
	switch mt {
	case "QT", "rfq_card", "quotation_card":
		return "RQ"
	case "PD", "PM", "ID":
		return mt
	case "OD":
		return "OD"
	default:
		return ""
	}
}

func (r *MessageRepository) ListByReference(referenceType string, referenceID int64, userID int64) ([]domain.Message, error) {
	var items []domain.Message
	// Prefer the stored reference_type; fall back to deriving from message_type
	// for legacy rows (migration 028 backfilled what it could; TX-order rows stay NULL).
	query := `
		SELECT m.message_id,
		       COALESCE(NULLIF(m.reference_type, ''),
		         CASE m.message_type
		           WHEN 'QT'             THEN 'RQ'
		           WHEN 'rfq_card'       THEN 'RQ'
		           WHEN 'quotation_card' THEN 'RQ'
		           WHEN 'PD'             THEN 'PD'
		           WHEN 'PM'             THEN 'PM'
		           WHEN 'ID'             THEN 'ID'
		           ELSE ''
		         END)                     AS reference_type,
		       COALESCE(m.reference_id, 0) AS reference_id,
		       CASE WHEN m.message_type IN ('QT','rfq_card','quotation_card')
		            THEN rq.title ELSE NULL END AS rfq_title,
		       NULL::text               AS reference_title,
		       m.sender_id, m.receiver_id, m.content, m.attachment_url,
		       m.created_at, m.conv_id, m.message_type, m.quote_data, NULL::bigint AS boq_rfq_id, m.is_read
		FROM messages m
		LEFT JOIN rfqs rq ON rq.rfq_id = m.reference_id
		                  AND m.message_type IN ('QT','rfq_card','quotation_card')
		WHERE m.reference_id = $1 AND (m.sender_id = $2 OR m.receiver_id = $2)
		  -- filter by reference_type so order/rfq sharing an id don't collide;
		  -- legacy rows (NULL/'') stay lenient to preserve existing behavior.
		  AND (m.reference_type = $3 OR m.reference_type IS NULL OR m.reference_type = '')
		ORDER BY m.created_at ASC
	`
	err := r.db.Select(&items, query, referenceID, userID, referenceType)
	return items, err
}

func (r *MessageRepository) ListByConvID(convID int64) ([]domain.Message, error) {
	var items []domain.Message
	// Prefer stored reference_type; fall back to deriving from message_type for
	// legacy rows. JOINs stay driven by message_type to fetch related titles.
	query := `
		SELECT m.message_id,
		       COALESCE(NULLIF(m.reference_type, ''),
		         CASE m.message_type
		           WHEN 'QT'             THEN 'RQ'
		           WHEN 'rfq_card'       THEN 'RQ'
		           WHEN 'quotation_card' THEN 'RQ'
		           WHEN 'PD'             THEN 'PD'
		           WHEN 'PM'             THEN 'PM'
		           WHEN 'ID'             THEN 'ID'
		           ELSE ''
		         END)                     AS reference_type,
		       COALESCE(m.reference_id, 0) AS reference_id,
		       rq.title                 AS rfq_title,
		       CASE
		         WHEN m.message_type IN ('QT','rfq_card','quotation_card') THEN rq.title
		         WHEN m.message_type IN ('PD','PM','ID')                   THEN sc.title
		         ELSE NULL
		       END                      AS reference_title,
		       m.sender_id, m.receiver_id, m.content, m.attachment_url,
		       m.created_at, m.conv_id, m.message_type, m.quote_data, NULL::bigint AS boq_rfq_id, m.is_read
		FROM messages m
		LEFT JOIN rfqs rq
		       ON rq.rfq_id = m.reference_id
		      AND m.message_type IN ('QT','rfq_card','quotation_card')
		LEFT JOIN factory_showcases sc
		       ON sc.showcase_id = m.reference_id
		      AND m.message_type IN ('PD','PM','ID')
		WHERE m.conv_id = $1
		ORDER BY m.created_at ASC
	`
	err := r.db.Select(&items, query, convID)
	return items, err
}

func (r *MessageRepository) ListThreads(userID int64) ([]domain.MessageThread, error) {
	var items []domain.MessageThread
	query := `
		SELECT ''::text AS reference_type,
		       COALESCE(m.reference_id, 0) AS reference_id,
		       m.content AS last_message,
		       m.created_at AS last_message_at
		FROM messages m
		INNER JOIN (
			SELECT COALESCE(reference_id, 0) AS reference_id, MAX(created_at) AS max_created_at
			FROM messages
			WHERE sender_id = $1 OR receiver_id = $1
			GROUP BY COALESCE(reference_id, 0)
		) latest
		ON COALESCE(m.reference_id, 0) = latest.reference_id
		   AND m.created_at = latest.max_created_at
		ORDER BY m.created_at DESC
	`
	err := r.db.Select(&items, query, userID)
	return items, err
}

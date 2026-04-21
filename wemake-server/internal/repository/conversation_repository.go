package repository

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type ConversationRepository struct {
	db *sqlx.DB
}

func NewConversationRepository(db *sqlx.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

const conversationPartySelect = `
		c.conv_id,
		c.customer_id,
		c.factory_id,
		c.last_message,
		COALESCE(c.unread_customer, 0) AS unread_customer,
		COALESCE(c.unread_factory, 0) AS unread_factory,
		COALESCE(c.has_quote, false) AS has_quote,
		c.updated_at,
		cust.first_name AS customer_first_name,
		cust.last_name AS customer_last_name,
		fp.factory_name AS factory_name,
		fp.image_url AS factory_image_url,
		fp.is_verified AS factory_is_verified,
		fp.specialization AS factory_specialization`

func (r *ConversationRepository) ListByUserID(userID int64) ([]domain.ConversationRow, error) {
	var items []domain.ConversationRow
	query := `SELECT ` + conversationPartySelect + `
		FROM conversations c
		LEFT JOIN customers cust ON cust.user_id = c.customer_id
		LEFT JOIN factory_profiles fp ON fp.user_id = c.factory_id
		WHERE c.customer_id = $1 OR c.factory_id = $1
		ORDER BY c.updated_at DESC`
	err := r.db.Select(&items, query, userID)
	return items, err
}

func (r *ConversationRepository) GetByID(convID int64) (*domain.ConversationRow, error) {
	var item domain.ConversationRow
	query := `SELECT ` + conversationPartySelect + `
		FROM conversations c
		LEFT JOIN customers cust ON cust.user_id = c.customer_id
		LEFT JOIN factory_profiles fp ON fp.user_id = c.factory_id
		WHERE c.conv_id = $1`
	err := r.db.Get(&item, query, convID)
	return &item, err
}

func (r *ConversationRepository) Create(conv *domain.Conversation) error {
	query := `
		INSERT INTO conversations (customer_id, factory_id)
		VALUES (:customer_id, :factory_id)
		RETURNING conv_id, updated_at
	`
	rows, err := r.db.NamedQuery(query, conv)
	if err != nil {
		return err
	}
	if rows.Next() {
		err = rows.Scan(&conv.ConvID, &conv.UpdatedAt)
	}
	rows.Close()
	return err
}

// MarkAsRead marks all messages in a conversation as read for the given user
// and resets that user's unread counter on the conversation row.
func (r *ConversationRepository) MarkAsRead(convID, userID int64) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var conv domain.Conversation
	query := `SELECT conv_id, customer_id, factory_id FROM conversations WHERE conv_id = $1`
	if err := tx.Get(&conv, query, convID); err != nil {
		return err
	}

	var unreadField string
	switch userID {
	case conv.CustomerID:
		unreadField = "unread_customer"
	case conv.FactoryID:
		unreadField = "unread_factory"
	default:
		return sql.ErrNoRows
	}

	if _, err := tx.Exec(`
		UPDATE messages
		SET is_read = TRUE
		WHERE conv_id = $1 AND receiver_id = $2 AND is_read = FALSE
	`, convID, userID); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		UPDATE conversations
		SET `+unreadField+` = 0
		WHERE conv_id = $1
	`, convID); err != nil {
		return err
	}

	return tx.Commit()
}

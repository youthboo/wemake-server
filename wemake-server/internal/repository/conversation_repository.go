package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type ConversationRepository struct {
	db *sqlx.DB
}

func NewConversationRepository(db *sqlx.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// conversationSelect lists columns explicitly: last_message is nullable in DB but domain uses string.
const conversationSelect = `conv_id, customer_id, factory_id,
		COALESCE(fp.factory_name, '') AS factory_name,
		COALESCE(fp.image_url, '') AS factory_image,
		COALESCE(last_message, '') AS last_message,
		COALESCE(unread_customer, 0) AS unread_customer,
		COALESCE(unread_factory, 0) AS unread_factory,
		COALESCE(has_quote, false) AS has_quote,
		updated_at`

func (r *ConversationRepository) ListByUserID(userID int64) ([]domain.Conversation, error) {
	var items []domain.Conversation
	query := `SELECT ` + conversationSelect + `
		FROM conversations c
		LEFT JOIN factory_profiles fp ON fp.user_id = c.factory_id
		WHERE c.customer_id = $1 OR c.factory_id = $1
		ORDER BY c.updated_at DESC`
	err := r.db.Select(&items, query, userID)
	return items, err
}

func (r *ConversationRepository) GetByID(convID int64) (*domain.Conversation, error) {
	var item domain.Conversation
	query := `SELECT ` + conversationSelect + `
		FROM conversations c
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

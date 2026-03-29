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

func (r *ConversationRepository) ListByUserID(userID int64) ([]domain.Conversation, error) {
	var items []domain.Conversation
	query := `SELECT * FROM conversations WHERE customer_id = $1 OR factory_id = $1 ORDER BY updated_at DESC`
	err := r.db.Select(&items, query, userID)
	return items, err
}

func (r *ConversationRepository) GetByID(convID int64) (*domain.Conversation, error) {
	var item domain.Conversation
	query := `SELECT * FROM conversations WHERE conv_id = $1`
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

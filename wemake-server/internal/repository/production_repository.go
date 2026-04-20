package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type ProductionRepository struct {
	db *sqlx.DB
}

type ProductionOrderContext struct {
	OrderID       int64     `db:"order_id"`
	UserID        int64     `db:"user_id"`
	FactoryID     int64     `db:"factory_id"`
	OrderStatus   string    `db:"status"`
	DepositAmount float64   `db:"deposit_amount"`
	TotalAmount   float64   `db:"total_amount"`
	CreatedAt     time.Time `db:"created_at"`
}

type ProductionUpdateContext struct {
	domain.ProductionUpdate
	OrderUserID    int64 `db:"order_user_id"`
	OrderFactoryID int64 `db:"order_factory_id"`
}

func NewProductionRepository(db *sqlx.DB) *ProductionRepository {
	return &ProductionRepository{db: db}
}

func (r *ProductionRepository) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
}

func (r *ProductionRepository) GetUserRole(userID int64) (string, error) {
	var role string
	if err := r.db.Get(&role, `SELECT role FROM users WHERE user_id = $1`, userID); err != nil {
		return "", err
	}
	return role, nil
}

func (r *ProductionRepository) ListActiveSteps() ([]domain.ProductionStepTemplate, error) {
	var items []domain.ProductionStepTemplate
	query := `
		SELECT
			step_id,
			COALESCE(step_code, '') AS step_code,
			COALESCE(step_name_th, '') AS step_name_th,
			COALESCE(step_name_en, '') AS step_name_en,
			COALESCE(sort_order, sequence) AS sort_order,
			COALESCE(requires_evidence, TRUE) AS requires_evidence,
			COALESCE(min_photos, 1) AS min_photos,
			COALESCE(is_payment_trigger, FALSE) AS is_payment_trigger,
			COALESCE(icon_name, '') AS icon_name,
			COALESCE(description, '') AS description,
			COALESCE(is_active, FALSE) AS is_active
		FROM lbi_production
		WHERE COALESCE(is_active, FALSE) = TRUE
		ORDER BY COALESCE(sort_order, sequence), step_id
	`
	err := r.db.Select(&items, query)
	return items, err
}

func (r *ProductionRepository) ListActiveStepsTx(tx *sqlx.Tx) ([]domain.ProductionStepTemplate, error) {
	var items []domain.ProductionStepTemplate
	query := `
		SELECT
			step_id,
			COALESCE(step_code, '') AS step_code,
			COALESCE(step_name_th, '') AS step_name_th,
			COALESCE(step_name_en, '') AS step_name_en,
			COALESCE(sort_order, sequence) AS sort_order,
			COALESCE(requires_evidence, TRUE) AS requires_evidence,
			COALESCE(min_photos, 1) AS min_photos,
			COALESCE(is_payment_trigger, FALSE) AS is_payment_trigger,
			COALESCE(icon_name, '') AS icon_name,
			COALESCE(description, '') AS description,
			COALESCE(is_active, FALSE) AS is_active
		FROM lbi_production
		WHERE COALESCE(is_active, FALSE) = TRUE
		ORDER BY COALESCE(sort_order, sequence), step_id
	`
	err := tx.Select(&items, query)
	return items, err
}

func (r *ProductionRepository) GetOrderByID(orderID int64) (*ProductionOrderContext, error) {
	var item ProductionOrderContext
	err := r.db.Get(&item, `SELECT order_id, user_id, factory_id, status, deposit_amount, total_amount, created_at FROM orders WHERE order_id = $1`, orderID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ProductionRepository) GetOrderForUpdateTx(tx *sqlx.Tx, orderID int64) (*ProductionOrderContext, error) {
	var item ProductionOrderContext
	err := tx.Get(&item, `SELECT order_id, user_id, factory_id, status, deposit_amount, total_amount, created_at FROM orders WHERE order_id = $1 FOR UPDATE`, orderID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ProductionRepository) ListByOrderID(orderID int64) ([]domain.ProductionUpdate, error) {
	var items []domain.ProductionUpdate
	query := `
		SELECT
			pu.update_id,
			pu.order_id,
			pu.step_id,
			COALESCE(lp.step_code, '') AS step_code,
			COALESCE(lp.step_name_th, '') AS step_name_th,
			COALESCE(lp.step_name_en, '') AS step_name_en,
			COALESCE(lp.sort_order, lp.sequence) AS sort_order,
			pu.status,
			COALESCE(pu.description, '') AS description,
			COALESCE(pu.image_urls, '[]'::jsonb) AS image_urls,
			pu.completed_at,
			pu.rejected_reason,
			pu.updated_by_user_id,
			pu.last_updated_at,
			pu.created_at
		FROM production_updates pu
		INNER JOIN lbi_production lp ON lp.step_id = pu.step_id
		WHERE pu.order_id = $1
		ORDER BY COALESCE(lp.sort_order, lp.sequence), pu.update_id
	`
	err := r.db.Select(&items, query, orderID)
	return items, err
}

func (r *ProductionRepository) ListByOrderIDTx(tx *sqlx.Tx, orderID int64) ([]domain.ProductionUpdate, error) {
	var items []domain.ProductionUpdate
	query := `
		SELECT
			pu.update_id,
			pu.order_id,
			pu.step_id,
			COALESCE(lp.step_code, '') AS step_code,
			COALESCE(lp.step_name_th, '') AS step_name_th,
			COALESCE(lp.step_name_en, '') AS step_name_en,
			COALESCE(lp.sort_order, lp.sequence) AS sort_order,
			pu.status,
			COALESCE(pu.description, '') AS description,
			COALESCE(pu.image_urls, '[]'::jsonb) AS image_urls,
			pu.completed_at,
			pu.rejected_reason,
			pu.updated_by_user_id,
			pu.last_updated_at,
			pu.created_at
		FROM production_updates pu
		INNER JOIN lbi_production lp ON lp.step_id = pu.step_id
		WHERE pu.order_id = $1
		ORDER BY COALESCE(lp.sort_order, lp.sequence), pu.update_id
	`
	err := tx.Select(&items, query, orderID)
	return items, err
}

func (r *ProductionRepository) GetUpdateByID(updateID int64) (*ProductionUpdateContext, error) {
	var item ProductionUpdateContext
	query := `
		SELECT
			pu.update_id,
			pu.order_id,
			pu.step_id,
			COALESCE(lp.step_code, '') AS step_code,
			COALESCE(lp.step_name_th, '') AS step_name_th,
			COALESCE(lp.step_name_en, '') AS step_name_en,
			COALESCE(lp.sort_order, lp.sequence) AS sort_order,
			pu.status,
			COALESCE(pu.description, '') AS description,
			COALESCE(pu.image_urls, '[]'::jsonb) AS image_urls,
			pu.completed_at,
			pu.rejected_reason,
			pu.updated_by_user_id,
			pu.last_updated_at,
			pu.created_at,
			o.user_id AS order_user_id,
			o.factory_id AS order_factory_id
		FROM production_updates pu
		INNER JOIN orders o ON o.order_id = pu.order_id
		INNER JOIN lbi_production lp ON lp.step_id = pu.step_id
		WHERE pu.update_id = $1
	`
	err := r.db.Get(&item, query, updateID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ProductionRepository) GetUpdateByIDForUpdateTx(tx *sqlx.Tx, updateID int64) (*ProductionUpdateContext, error) {
	var item ProductionUpdateContext
	query := `
		SELECT
			pu.update_id,
			pu.order_id,
			pu.step_id,
			COALESCE(lp.step_code, '') AS step_code,
			COALESCE(lp.step_name_th, '') AS step_name_th,
			COALESCE(lp.step_name_en, '') AS step_name_en,
			COALESCE(lp.sort_order, lp.sequence) AS sort_order,
			pu.status,
			COALESCE(pu.description, '') AS description,
			COALESCE(pu.image_urls, '[]'::jsonb) AS image_urls,
			pu.completed_at,
			pu.rejected_reason,
			pu.updated_by_user_id,
			pu.last_updated_at,
			pu.created_at,
			o.user_id AS order_user_id,
			o.factory_id AS order_factory_id
		FROM production_updates pu
		INNER JOIN orders o ON o.order_id = pu.order_id
		INNER JOIN lbi_production lp ON lp.step_id = pu.step_id
		WHERE pu.update_id = $1
		FOR UPDATE
	`
	err := tx.Get(&item, query, updateID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ProductionRepository) UpsertTx(tx *sqlx.Tx, item *domain.ProductionUpdate) error {
	query := `
		INSERT INTO production_updates (
			order_id,
			step_id,
			status,
			description,
			image_urls,
			completed_at,
			rejected_reason,
			updated_by_user_id,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (order_id, step_id) DO UPDATE SET
			status = EXCLUDED.status,
			description = EXCLUDED.description,
			image_urls = EXCLUDED.image_urls,
			completed_at = EXCLUDED.completed_at,
			rejected_reason = EXCLUDED.rejected_reason,
			updated_by_user_id = EXCLUDED.updated_by_user_id
		RETURNING update_id, created_at, last_updated_at
	`
	return tx.QueryRow(
		query,
		item.OrderID,
		item.StepID,
		item.Status,
		item.Description,
		item.ImageURLs,
		nullableTimeValue(item.CompletedAt),
		item.RejectedReason,
		item.UpdatedByUserID,
	).Scan(&item.UpdateID, &item.CreatedAt, &item.LastUpdatedAt)
}

func (r *ProductionRepository) RejectTx(tx *sqlx.Tx, updateID int64, reason string, updatedBy int64) (*domain.ProductionUpdate, error) {
	var item domain.ProductionUpdate
	query := `
		UPDATE production_updates
		SET status = 'RJ',
			completed_at = NULL,
			rejected_reason = $2,
			updated_by_user_id = $3
		WHERE update_id = $1
		RETURNING update_id, order_id, step_id, status, COALESCE(description, '') AS description,
		          COALESCE(image_urls, '[]'::jsonb) AS image_urls,
		          completed_at, rejected_reason, updated_by_user_id, last_updated_at, created_at
	`
	if err := tx.Get(&item, query, updateID, reason, updatedBy); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ProductionRepository) InsertDomainEventTx(tx *sqlx.Tx, eventType string, payload interface{}) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO domain_events (event_type, payload) VALUES ($1, $2)`, eventType, b)
	return err
}

func (r *ProductionRepository) GetUpdateByOrderAndStep(orderID, stepID int64, items []domain.ProductionUpdate) *domain.ProductionUpdate {
	for i := range items {
		if items[i].OrderID == orderID && items[i].StepID == stepID {
			return &items[i]
		}
	}
	return nil
}

func (r *ProductionRepository) GetActiveInProgressStep(items []domain.ProductionUpdate, ignoreStepID int64) *domain.ProductionUpdate {
	for i := range items {
		if items[i].Status == "IP" && items[i].StepID != ignoreStepID {
			return &items[i]
		}
	}
	return nil
}

func (r *ProductionRepository) HasDownstreamInFlight(items []domain.ProductionUpdate, currentSortOrder int64) bool {
	for _, item := range items {
		if item.SortOrder > currentSortOrder && (item.Status == "IP" || item.Status == "CD") {
			return true
		}
	}
	return false
}

func (r *ProductionRepository) StepByID(steps []domain.ProductionStepTemplate, stepID int64) *domain.ProductionStepTemplate {
	for i := range steps {
		if steps[i].StepID == stepID {
			return &steps[i]
		}
	}
	return nil
}

func (r *ProductionRepository) StepBySortOrder(steps []domain.ProductionStepTemplate, sortOrder int64) *domain.ProductionStepTemplate {
	for i := range steps {
		if steps[i].SortOrder == sortOrder {
			return &steps[i]
		}
	}
	return nil
}

func (r *ProductionRepository) InflateUpdates(orderID int64, steps []domain.ProductionStepTemplate, persisted []domain.ProductionUpdate) []domain.ProductionUpdate {
	byStep := make(map[int64]domain.ProductionUpdate, len(persisted))
	for _, item := range persisted {
		byStep[item.StepID] = item
	}
	out := make([]domain.ProductionUpdate, 0, len(steps))
	for _, step := range steps {
		if item, ok := byStep[step.StepID]; ok {
			item.StepCode = step.StepCode
			item.StepNameTH = step.StepNameTH
			item.StepNameEN = step.StepNameEN
			item.SortOrder = step.SortOrder
			if item.ImageURLs == nil {
				item.ImageURLs = domain.StringArray{}
			}
			out = append(out, item)
			continue
		}
		out = append(out, domain.ProductionUpdate{
			UpdateID:    0,
			OrderID:     orderID,
			StepID:      step.StepID,
			StepCode:    step.StepCode,
			StepNameTH:  step.StepNameTH,
			StepNameEN:  step.StepNameEN,
			SortOrder:   step.SortOrder,
			Status:      "PD",
			Description: "",
			ImageURLs:   domain.StringArray{},
		})
	}
	return out
}

func normalizeRole(role string) string {
	switch role {
	case "CT":
		return "CU"
	default:
		return role
	}
}

func (r *ProductionRepository) IsAdminRole(role string) bool {
	switch normalizeRole(role) {
	case "AD", "ADMIN":
		return true
	default:
		return false
	}
}

func (r *ProductionRepository) IsFactoryRole(role string) bool {
	return normalizeRole(role) == "FT"
}

func (r *ProductionRepository) IsCustomerRole(role string) bool {
	return normalizeRole(role) == "CU"
}

func (r *ProductionRepository) LoadAuthorizedOrder(orderID, userID int64) (*ProductionOrderContext, string, error) {
	role, err := r.GetUserRole(userID)
	if err != nil {
		return nil, "", err
	}
	order, err := r.GetOrderByID(orderID)
	if err != nil {
		return nil, "", err
	}
	return order, normalizeRole(role), nil
}

func (r *ProductionRepository) StepIndex(steps []domain.ProductionStepTemplate, stepID int64) (int, error) {
	for idx, step := range steps {
		if step.StepID == stepID {
			return idx, nil
		}
	}
	return -1, fmt.Errorf("step not found")
}

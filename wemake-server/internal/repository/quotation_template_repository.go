package repository

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type QuotationTemplateRepository struct {
	db *sqlx.DB
}

func NewQuotationTemplateRepository(db *sqlx.DB) *QuotationTemplateRepository {
	return &QuotationTemplateRepository{db: db}
}

func (r *QuotationTemplateRepository) ListByFactoryID(factoryID int64) ([]domain.QuotationTemplate, error) {
	var items []domain.QuotationTemplate
	err := r.db.Select(&items, `
		SELECT template_id, factory_id, template_name, price_per_piece, mold_cost,
		       lead_time_days, shipping_method_id, note, is_active, created_at, updated_at
		FROM quotation_templates
		WHERE factory_id = $1
		ORDER BY created_at DESC
	`, factoryID)
	return items, err
}

func (r *QuotationTemplateRepository) GetByID(templateID, factoryID int64) (*domain.QuotationTemplate, error) {
	var item domain.QuotationTemplate
	err := r.db.Get(&item, `
		SELECT template_id, factory_id, template_name, price_per_piece, mold_cost,
		       lead_time_days, shipping_method_id, note, is_active, created_at, updated_at
		FROM quotation_templates
		WHERE template_id = $1 AND factory_id = $2
	`, templateID, factoryID)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *QuotationTemplateRepository) Create(t *domain.QuotationTemplate) error {
	return r.db.QueryRow(`
		INSERT INTO quotation_templates
		    (factory_id, template_name, price_per_piece, mold_cost, lead_time_days, shipping_method_id, note, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING template_id, created_at, updated_at
	`, t.FactoryID, t.TemplateName, t.PricePerPiece, t.MoldCost,
		t.LeadTimeDays, t.ShippingMethodID, t.Note, t.IsActive).
		Scan(&t.TemplateID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *QuotationTemplateRepository) Update(t *domain.QuotationTemplate) error {
	res, err := r.db.Exec(`
		UPDATE quotation_templates
		SET template_name = $1,
		    price_per_piece = $2,
		    mold_cost = $3,
		    lead_time_days = $4,
		    shipping_method_id = $5,
		    note = $6,
		    is_active = $7,
		    updated_at = NOW()
		WHERE template_id = $8 AND factory_id = $9
	`, t.TemplateName, t.PricePerPiece, t.MoldCost, t.LeadTimeDays,
		t.ShippingMethodID, t.Note, t.IsActive, t.TemplateID, t.FactoryID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *QuotationTemplateRepository) Delete(templateID, factoryID int64) error {
	res, err := r.db.Exec(`
		DELETE FROM quotation_templates WHERE template_id = $1 AND factory_id = $2
	`, templateID, factoryID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

package repository

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/domain"
)

type RFQRepository struct {
	db *sqlx.DB
}

func NewRFQRepository(db *sqlx.DB) *RFQRepository {
	return &RFQRepository{db: db}
}

func (r *RFQRepository) Create(rfq *domain.RFQ) error {
	query := `
		INSERT INTO rfqs (
			user_id, category_id, sub_category_id, title, quantity, unit_id, budget_per_piece, details,
			address_id, shipping_method_id, status, deadline_date, uploaded_at, created_at, updated_at, image_urls,
			material_grade, tolerance, color_finish, dimension_spec, weight_target_g, packaging_spec,
			target_unit_price, target_lead_time_days, required_delivery_date, incoterms, payment_terms, delivery_address_id,
			certifications_required, sample_required, sample_qty, inspection_type,
			tech_drawing_url, reference_images, spec_sheet_url
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22,
			$23, $24, $25, $26, $27, $28,
			$29, $30, $31, $32,
			$33, $34, $35
		)
		RETURNING rfq_id
	`
	return r.db.QueryRow(
		query,
		rfq.UserID,
		rfq.CategoryID,
		nullableInt64Value(rfq.SubCategoryID),
		rfq.Title,
		rfq.Quantity,
		rfq.UnitID,
		rfq.BudgetPerPiece,
		rfq.Details,
		rfq.AddressID,
		nullableInt64Value(rfq.ShippingMethodID),
		rfq.Status,
		nullableTimeValue(rfq.DeadlineDate),
		nullableTimeValue(rfq.UploadedAt),
		rfq.CreatedAt,
		rfq.UpdatedAt,
		rfq.ImageURLs,
		nullableStringPtr(rfq.MaterialGrade),
		nullableStringPtr(rfq.Tolerance),
		nullableStringPtr(rfq.ColorFinish),
		rfq.DimensionSpec,
		nullableFloat64(rfq.WeightTargetG),
		nullableStringPtr(rfq.PackagingSpec),
		nullableFloat64(rfq.TargetUnitPrice),
		nullableIntValue(rfq.TargetLeadTimeDays),
		nullableTimeValue(rfq.RequiredDeliveryDate),
		nullableStringPtr(rfq.Incoterms),
		nullableStringPtr(rfq.PaymentTerms),
		nullableInt64Value(rfq.DeliveryAddressID),
		rfq.CertificationsRequired,
		rfq.SampleRequired,
		nullableIntValue(rfq.SampleQty),
		nullableStringPtr(rfq.InspectionType),
		nullableStringPtr(rfq.TechDrawingURL),
		rfq.ReferenceImages,
		nullableStringPtr(rfq.SpecSheetURL),
	).Scan(&rfq.RFQID)
}

func (r *RFQRepository) ListByUserID(userID int64, status string) ([]domain.RFQ, error) {
	var rfqs []domain.RFQ
	query := `
		SELECT rfq_id, user_id, category_id, sub_category_id, title, quantity, unit_id, budget_per_piece, details, address_id,
		       shipping_method_id, status, deadline_date, uploaded_at, created_at, updated_at, image_urls,
		       material_grade, tolerance, color_finish, dimension_spec, weight_target_g, packaging_spec,
		       target_unit_price, target_lead_time_days, required_delivery_date, incoterms, payment_terms, delivery_address_id,
		       certifications_required, sample_required, sample_qty, inspection_type,
		       tech_drawing_url, reference_images, spec_sheet_url
		FROM rfqs
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	err := r.db.Select(&rfqs, query, args...)
	return rfqs, err
}

func (r *RFQRepository) GetByID(userID, rfqID int64) (*domain.RFQ, error) {
	var rfq domain.RFQ
	query := `
		SELECT rfq_id, user_id, category_id, sub_category_id, title, quantity, unit_id, budget_per_piece, details, address_id,
		       shipping_method_id, status, deadline_date, uploaded_at, created_at, updated_at, image_urls,
		       material_grade, tolerance, color_finish, dimension_spec, weight_target_g, packaging_spec,
		       target_unit_price, target_lead_time_days, required_delivery_date, incoterms, payment_terms, delivery_address_id,
		       certifications_required, sample_required, sample_qty, inspection_type,
		       tech_drawing_url, reference_images, spec_sheet_url
		FROM rfqs
		WHERE user_id = $1 AND rfq_id = $2
	`
	if err := r.db.Get(&rfq, query, userID, rfqID); err != nil {
		return nil, err
	}
	return &rfq, nil
}

func (r *RFQRepository) Cancel(userID, rfqID int64) error {
	query := "UPDATE rfqs SET status = 'CC', updated_at = NOW() WHERE user_id = $1 AND rfq_id = $2"
	_, err := r.db.Exec(query, userID, rfqID)
	return err
}

// CloseOpenRFQForUserTx sets RFQ status from OP to CL when the customer awards an order (same transaction as order create).
func (r *RFQRepository) CloseOpenRFQForUserTx(tx *sqlx.Tx, rfqID, userID int64) error {
	_, err := tx.Exec(`
		UPDATE rfqs
		SET status = 'CL', updated_at = NOW()
		WHERE rfq_id = $1 AND user_id = $2 AND status = 'OP'
	`, rfqID, userID)
	return err
}

func (r *RFQRepository) SubCategoryBelongsToCategory(subCategoryID, categoryID int64) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM lbi_sub_categories
			WHERE sub_category_id = $1
				AND category_id = $2
				AND status = '1'
		)
	`
	err := r.db.Get(&exists, query, subCategoryID, categoryID)
	return exists, err
}

func (r *RFQRepository) ShippingMethodExists(shippingMethodID int64) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM lbi_shipping_methods
			WHERE shipping_method_id = $1
				AND status = '1'
		)
	`
	err := r.db.Get(&exists, query, shippingMethodID)
	return exists, err
}

// GetByIDAny loads RFQ by id without customer ownership check.
func (r *RFQRepository) GetByIDAny(rfqID int64) (*domain.RFQ, error) {
	var rfq domain.RFQ
	query := `
		SELECT rfq_id, user_id, category_id, sub_category_id, title, quantity, unit_id, budget_per_piece, details, address_id,
		       shipping_method_id, status, deadline_date, uploaded_at, created_at, updated_at, image_urls,
		       material_grade, tolerance, color_finish, dimension_spec, weight_target_g, packaging_spec,
		       target_unit_price, target_lead_time_days, required_delivery_date, incoterms, payment_terms, delivery_address_id,
		       certifications_required, sample_required, sample_qty, inspection_type,
		       tech_drawing_url, reference_images, spec_sheet_url
		FROM rfqs
		WHERE rfq_id = $1
	`
	if err := r.db.Get(&rfq, query, rfqID); err != nil {
		return nil, err
	}
	return &rfq, nil
}

// ListMatchingForFactory returns open RFQs whose category (and optional sub-category) match the factory maps.
func (r *RFQRepository) ListMatchingForFactory(factoryID int64, status string) ([]domain.RFQ, error) {
	st := status
	if st == "" {
		st = "OP"
	}
	var rfqs []domain.RFQ
	query := `
		SELECT DISTINCT r.rfq_id, r.user_id, r.category_id, r.sub_category_id, r.title, r.quantity, r.unit_id, r.budget_per_piece, r.details, r.address_id,
		       r.shipping_method_id, r.status, r.deadline_date, r.uploaded_at, r.created_at, r.updated_at, r.image_urls,
		       r.material_grade, r.tolerance, r.color_finish, r.dimension_spec, r.weight_target_g, r.packaging_spec,
		       r.target_unit_price, r.target_lead_time_days, r.required_delivery_date, r.incoterms, r.payment_terms, r.delivery_address_id,
		       r.certifications_required, r.sample_required, r.sample_qty, r.inspection_type,
		       r.tech_drawing_url, r.reference_images, r.spec_sheet_url
		FROM rfqs r
		INNER JOIN map_factory_categories mfc ON mfc.category_id = r.category_id AND mfc.factory_id = $1
		WHERE r.status = $2
		  AND (
			r.sub_category_id IS NULL
			OR EXISTS (
				SELECT 1 FROM map_factory_sub_categories ms
				WHERE ms.factory_id = $1 AND ms.sub_category_id = r.sub_category_id
			)
		  )
		ORDER BY r.created_at DESC
	`
	err := r.db.Select(&rfqs, query, factoryID, st)
	return rfqs, err
}

// FactoryHasMatchingCategory returns true if factory accepts RFQ's category and sub-category rules.
func (r *RFQRepository) FactoryHasMatchingCategory(factoryID int64, rfq *domain.RFQ) (bool, error) {
	var ok bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM map_factory_categories mfc
			WHERE mfc.factory_id = $1 AND mfc.category_id = $2
		)
		AND (
			$3::bigint IS NULL
			OR EXISTS (
				SELECT 1 FROM map_factory_sub_categories ms
				WHERE ms.factory_id = $1 AND ms.sub_category_id = $3
			)
		)
	`
	err := r.db.Get(&ok, query, factoryID, rfq.CategoryID, nullableInt64Value(rfq.SubCategoryID))
	return ok, err
}

func (r *RFQRepository) FactoryHasQuotationOnRFQ(factoryID, rfqID int64) (bool, error) {
	var ok bool
	err := r.db.Get(&ok, `
		SELECT EXISTS (SELECT 1 FROM quotations WHERE factory_id = $1 AND rfq_id = $2)
	`, factoryID, rfqID)
	return ok, err
}

func nullableInt64Value(value *int64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTimeValue(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func nullableIntValue(value *int) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func (r *RFQRepository) Patch(userID, rfqID int64, rfq *domain.RFQ) error {
	_, err := r.db.NamedExec(`
		UPDATE rfqs
		SET category_id = :category_id,
		    sub_category_id = :sub_category_id,
		    title = :title,
		    quantity = :quantity,
		    unit_id = :unit_id,
		    budget_per_piece = :budget_per_piece,
		    details = :details,
		    address_id = :address_id,
		    shipping_method_id = :shipping_method_id,
		    deadline_date = :deadline_date,
		    image_urls = :image_urls,
		    material_grade = :material_grade,
		    tolerance = :tolerance,
		    color_finish = :color_finish,
		    dimension_spec = :dimension_spec,
		    weight_target_g = :weight_target_g,
		    packaging_spec = :packaging_spec,
		    target_unit_price = :target_unit_price,
		    target_lead_time_days = :target_lead_time_days,
		    required_delivery_date = :required_delivery_date,
		    incoterms = :incoterms,
		    payment_terms = :payment_terms,
		    delivery_address_id = :delivery_address_id,
		    certifications_required = :certifications_required,
		    sample_required = :sample_required,
		    sample_qty = :sample_qty,
		    inspection_type = :inspection_type,
		    tech_drawing_url = :tech_drawing_url,
		    reference_images = :reference_images,
		    spec_sheet_url = :spec_sheet_url,
		    updated_at = NOW()
		WHERE rfq_id = :rfq_id AND user_id = :user_id AND status = 'OP'
	`, rfq)
	return err
}

package repository

import (
	"database/sql"
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
	return r.createWithExecutor(r.db, rfq)
}

func (r *RFQRepository) CreateTx(tx *sqlx.Tx, rfq *domain.RFQ) error {
	return r.createWithExecutor(tx, rfq)
}

type rfqQueryRowExecutor interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

func (r *RFQRepository) createWithExecutor(exec rfqQueryRowExecutor, rfq *domain.RFQ) error {
	query := `
		INSERT INTO rfqs (
			user_id, category_id, sub_category_id, title, quantity, details,
			address_id, shipping_method_id, status, uploaded_at, created_at, updated_at,
			material_grade, target_unit_price, target_lead_time_days, required_delivery_date, delivery_address_id,
			certifications_required, sample_required, sample_qty, inspection_type,
			reference_images, rfq_type, initiated_by, factory_user_id, source_showcase_id, source_conv_id,
			boq_currency, boq_subtotal, boq_discount_amount, boq_vat_percent, boq_vat_amount, boq_grand_total,
			boq_moq, boq_lead_time_days, boq_payment_terms, boq_validity_days, boq_note,
			boq_sent_at, boq_responded_at, boq_response, boq_decline_reason
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17,
			$18, $19, $20, $21,
			$22, $23, $24, $25, $26, $27,
			$28, $29, $30, $31, $32, $33,
			$34, $35, $36, $37, $38,
			$39, $40, $41, $42
		)
		RETURNING rfq_id
	`
	return exec.QueryRow(
		query,
		rfq.UserID,
		nullableZeroInt64(rfq.CategoryID),
		nullableInt64Value(rfq.SubCategoryID),
		rfq.Title,
		rfq.Quantity,
		rfq.Details,
		nullableZeroInt64(rfq.AddressID),
		nullableInt64Value(rfq.ShippingMethodID),
		rfq.Status,
		nullableTimeValue(rfq.UploadedAt),
		rfq.CreatedAt,
		rfq.UpdatedAt,
		nullableStringPtr(rfq.MaterialGrade),
		nullableFloat64(rfq.TargetUnitPrice),
		nullableIntValue(rfq.TargetLeadTimeDays),
		nullableTimeValue(rfq.RequiredDeliveryDate),
		nullableInt64Value(rfq.DeliveryAddressID),
		rfq.CertificationsRequired,
		rfq.SampleRequired,
		nullableIntValue(rfq.SampleQty),
		nullableStringPtr(rfq.InspectionType),
		rfq.ReferenceImages,
		nullableRFQType(rfq.RFQType),
		nullableInitiatedBy(rfq.InitiatedBy),
		nullableInt64Value(rfq.FactoryUserID),
		nullableInt64Value(rfq.SourceShowcaseID),
		nullableInt64Value(rfq.SourceConvID),
		nullableStringPtr(rfq.BOQCurrency),
		nullableFloat64(rfq.BOQSubtotal),
		nullableFloat64(rfq.BOQDiscountAmount),
		nullableFloat64(rfq.BOQVatPercent),
		nullableFloat64(rfq.BOQVatAmount),
		nullableFloat64(rfq.BOQGrandTotal),
		nullableIntValue(rfq.BOQMOQ),
		nullableIntValue(rfq.BOQLeadTimeDays),
		nullableStringPtr(rfq.BOQPaymentTerms),
		nullableIntValue(rfq.BOQValidityDays),
		nullableStringPtr(rfq.BOQNote),
		nullableTimeValue(rfq.BOQSentAt),
		nullableTimeValue(rfq.BOQRespondedAt),
		nullableStringPtr(rfq.BOQResponse),
		nullableStringPtr(rfq.BOQDeclineReason),
	).Scan(&rfq.RFQID)
}

func (r *RFQRepository) ListByUserID(userID int64, status string) ([]domain.RFQ, error) {
	var rfqs []domain.RFQ
	query := `
		SELECT rfq_id, user_id, COALESCE(category_id, 0) AS category_id, sub_category_id, title, quantity, details, COALESCE(address_id, 0) AS address_id,
		       shipping_method_id, status, uploaded_at, created_at, updated_at,
		       material_grade, target_unit_price, target_lead_time_days, required_delivery_date, delivery_address_id,
		       certifications_required, sample_required, sample_qty, inspection_type,
		       reference_images, COALESCE(rfq_type, 'RFQ') AS rfq_type, COALESCE(initiated_by, 'buyer') AS initiated_by,
		       factory_user_id, source_showcase_id, source_conv_id,
		       boq_currency, boq_subtotal, boq_discount_amount, boq_vat_percent, boq_vat_amount, boq_grand_total,
		       boq_moq, boq_lead_time_days, boq_payment_terms, boq_validity_days, boq_note,
		       boq_sent_at, boq_responded_at, boq_response, boq_decline_reason
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
	if err != nil {
		return rfqs, err
	}
	for i := range rfqs {
		if err := r.enrichRFQLookups(&rfqs[i]); err != nil {
			return rfqs, err
		}
		domain.EnrichRFQBudgetFields(&rfqs[i])
	}
	return rfqs, nil
}

func (r *RFQRepository) GetByID(userID, rfqID int64) (*domain.RFQ, error) {
	var rfq domain.RFQ
	query := `
		SELECT rfq_id, user_id, COALESCE(category_id, 0) AS category_id, sub_category_id, title, quantity, details, COALESCE(address_id, 0) AS address_id,
		       shipping_method_id, status, uploaded_at, created_at, updated_at,
		       material_grade, target_unit_price, target_lead_time_days, required_delivery_date, delivery_address_id,
		       certifications_required, sample_required, sample_qty, inspection_type,
		       reference_images, COALESCE(rfq_type, 'RFQ') AS rfq_type, COALESCE(initiated_by, 'buyer') AS initiated_by,
		       factory_user_id, source_showcase_id, source_conv_id,
		       boq_currency, boq_subtotal, boq_discount_amount, boq_vat_percent, boq_vat_amount, boq_grand_total,
		       boq_moq, boq_lead_time_days, boq_payment_terms, boq_validity_days, boq_note,
		       boq_sent_at, boq_responded_at, boq_response, boq_decline_reason
		FROM rfqs
		WHERE user_id = $1 AND rfq_id = $2
	`
	if err := r.db.Get(&rfq, query, userID, rfqID); err != nil {
		return nil, err
	}
	if rfq.AddressID > 0 {
		addr, err := r.getAddressByID(rfq.AddressID)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if err == nil {
			rfq.Address = addr
		}
	}
	if err := r.enrichRFQLookups(&rfq); err != nil {
		return nil, err
	}
	domain.EnrichRFQBudgetFields(&rfq)
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
		SELECT rfq_id, user_id, COALESCE(category_id, 0) AS category_id, sub_category_id, title, quantity, details, COALESCE(address_id, 0) AS address_id,
		       shipping_method_id, status, uploaded_at, created_at, updated_at,
		       material_grade, target_unit_price, target_lead_time_days, required_delivery_date, delivery_address_id,
		       certifications_required, sample_required, sample_qty, inspection_type,
		       reference_images, COALESCE(rfq_type, 'RFQ') AS rfq_type, COALESCE(initiated_by, 'buyer') AS initiated_by,
		       factory_user_id, source_showcase_id, source_conv_id,
		       boq_currency, boq_subtotal, boq_discount_amount, boq_vat_percent, boq_vat_amount, boq_grand_total,
		       boq_moq, boq_lead_time_days, boq_payment_terms, boq_validity_days, boq_note,
		       boq_sent_at, boq_responded_at, boq_response, boq_decline_reason
		FROM rfqs
		WHERE rfq_id = $1
	`
	if err := r.db.Get(&rfq, query, rfqID); err != nil {
		return nil, err
	}
	if rfq.AddressID > 0 {
		addr, err := r.getAddressByID(rfq.AddressID)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if err == nil {
			rfq.Address = addr
		}
	}
	if err := r.enrichRFQLookups(&rfq); err != nil {
		return nil, err
	}
	domain.EnrichRFQBudgetFields(&rfq)
	return &rfq, nil
}

func (r *RFQRepository) getAddressByID(addressID int64) (*domain.Address, error) {
	var address domain.Address
	query := `
		SELECT address_id, user_id, address_type, address_detail, sub_district_id, district_id, province_id, zip_code, is_default
		FROM addresses
		WHERE address_id = $1
	`
	if err := r.db.Get(&address, query, addressID); err != nil {
		return nil, err
	}
	return &address, nil
}

// ListMatchingForFactory returns open RFQs whose category (and optional sub-category) match the factory maps.
func (r *RFQRepository) ListMatchingForFactory(factoryID int64, status string) ([]domain.RFQ, error) {
	st := status
	if st == "" {
		st = "OP"
	}
	var rfqs []domain.RFQ
	query := `
		SELECT DISTINCT r.rfq_id, r.user_id, COALESCE(r.category_id, 0) AS category_id, r.sub_category_id, r.title, r.quantity, r.details, COALESCE(r.address_id, 0) AS address_id,
		       r.shipping_method_id, r.status, r.uploaded_at, r.created_at, r.updated_at,
		       r.material_grade, r.target_unit_price, r.target_lead_time_days, r.required_delivery_date, r.delivery_address_id,
		       r.certifications_required, r.sample_required, r.sample_qty, r.inspection_type,
		       r.reference_images, COALESCE(r.rfq_type, 'RFQ') AS rfq_type, COALESCE(r.initiated_by, 'buyer') AS initiated_by,
		       r.factory_user_id, r.source_showcase_id, r.source_conv_id,
		       r.boq_currency, r.boq_subtotal, r.boq_discount_amount, r.boq_vat_percent, r.boq_vat_amount, r.boq_grand_total,
		       r.boq_moq, r.boq_lead_time_days, r.boq_payment_terms, r.boq_validity_days, r.boq_note,
		       r.boq_sent_at, r.boq_responded_at, r.boq_response, r.boq_decline_reason
		FROM rfqs r
		INNER JOIN map_factory_categories mfc ON mfc.category_id = r.category_id AND mfc.factory_id = $1
		LEFT JOIN lbi_sub_categories sc ON sc.sub_category_id = r.sub_category_id
		WHERE r.status = $2
		  AND (
			r.sub_category_id IS NULL
			OR COALESCE(sc.sort_order, 0) = 99
			OR EXISTS (
				SELECT 1 FROM map_factory_sub_categories ms
				WHERE ms.factory_id = $1 AND ms.sub_category_id = r.sub_category_id
			)
		  )
		ORDER BY r.created_at DESC
	`
	err := r.db.Select(&rfqs, query, factoryID, st)
	if err != nil {
		return rfqs, err
	}
	for i := range rfqs {
		if err := r.enrichRFQLookups(&rfqs[i]); err != nil {
			return rfqs, err
		}
		domain.EnrichRFQBudgetFields(&rfqs[i])
	}
	return rfqs, nil
}

func (r *RFQRepository) enrichRFQLookups(rfq *domain.RFQ) error {
	if rfq == nil {
		return nil
	}

	var categoryName sql.NullString
	if err := r.db.Get(&categoryName, `SELECT name FROM categories WHERE category_id = $1`, rfq.CategoryID); err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	} else if categoryName.Valid {
		rfq.CategoryName = &categoryName.String
	}

	if rfq.SubCategoryID != nil {
		var subCategoryName sql.NullString
		if err := r.db.Get(&subCategoryName, `SELECT name FROM lbi_sub_categories WHERE sub_category_id = $1`, *rfq.SubCategoryID); err != nil {
			if err != sql.ErrNoRows {
				return err
			}
		} else if subCategoryName.Valid {
			rfq.SubCategoryName = &subCategoryName.String
		}
	}

	if rfq.ShippingMethodID != nil {
		var shippingMethodName sql.NullString
		if err := r.db.Get(&shippingMethodName, `SELECT method_name FROM lbi_shipping_methods WHERE shipping_method_id = $1`, *rfq.ShippingMethodID); err != nil {
			if err != sql.ErrNoRows {
				return err
			}
		} else if shippingMethodName.Valid {
			rfq.ShippingMethodName = &shippingMethodName.String
		}
	}

	var addressSummary sql.NullString
	if rfq.AddressID <= 0 {
		return nil
	}
	if err := r.db.Get(&addressSummary, `
		SELECT TRIM(BOTH ' ' FROM CONCAT_WS(' / ',
			NULLIF(a.address_detail, ''),
			NULLIF(sd.name_th, ''),
			NULLIF(d.name_th, ''),
			NULLIF(p.name_th, ''),
			NULLIF(a.zip_code, '')
		))
		FROM addresses a
		LEFT JOIN lbi_sub_districts sd ON sd.row_id = a.sub_district_id
		LEFT JOIN lbi_districts d ON d.row_id = a.district_id
		LEFT JOIN lbi_provinces p ON p.row_id = a.province_id
		WHERE a.address_id = $1
	`, rfq.AddressID); err != nil {
		if err != sql.ErrNoRows {
			return err
		}
	} else if addressSummary.Valid {
		rfq.AddressSummary = &addressSummary.String
	}

	return nil
}

func nullableZeroInt64(v int64) interface{} {
	if v <= 0 {
		return nil
	}
	return v
}

func nullableRFQType(v string) interface{} {
	if v == "" {
		return "RFQ"
	}
	return v
}

func nullableInitiatedBy(v string) interface{} {
	if v == "" {
		return "buyer"
	}
	return v
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
				SELECT 1
				FROM lbi_sub_categories sc
				WHERE sc.sub_category_id = $3
				  AND COALESCE(sc.sort_order, 0) = 99
			)
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
		    details = :details,
		    address_id = :address_id,
		    shipping_method_id = :shipping_method_id,
		    material_grade = :material_grade,
		    target_unit_price = :target_unit_price,
		    target_lead_time_days = :target_lead_time_days,
		    required_delivery_date = :required_delivery_date,
		    delivery_address_id = :delivery_address_id,
		    certifications_required = :certifications_required,
		    sample_required = :sample_required,
		    sample_qty = :sample_qty,
		    inspection_type = :inspection_type,
		    reference_images = :reference_images,
		    updated_at = NOW()
		WHERE rfq_id = :rfq_id AND user_id = :user_id AND status = 'OP'
	`, rfq)
	return err
}

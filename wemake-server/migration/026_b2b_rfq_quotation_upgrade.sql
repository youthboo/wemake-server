BEGIN;

CREATE TABLE IF NOT EXISTS platform_config (
    config_id BIGSERIAL PRIMARY KEY,
    default_commission_rate DECIMAL(5,2) NOT NULL DEFAULT 5.00,
    promo_commission_rate DECIMAL(5,2),
    promo_start_at TIMESTAMP,
    promo_end_at TIMESTAMP,
    promo_label VARCHAR(120),
    vat_rate DECIMAL(5,2) NOT NULL DEFAULT 7.00,
    currency_code CHAR(3) NOT NULL DEFAULT 'THB',
    effective_from TIMESTAMP NOT NULL DEFAULT NOW(),
    effective_to TIMESTAMP,
    created_by BIGINT REFERENCES users(user_id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_platform_default_commission CHECK (default_commission_rate BETWEEN 0 AND 100),
    CONSTRAINT chk_platform_promo_commission CHECK (promo_commission_rate IS NULL OR promo_commission_rate BETWEEN 0 AND 100),
    CONSTRAINT chk_platform_vat_rate CHECK (vat_rate BETWEEN 0 AND 100),
    CONSTRAINT chk_platform_promo_window CHECK (promo_start_at IS NULL OR promo_end_at IS NULL OR promo_end_at > promo_start_at)
);

CREATE INDEX IF NOT EXISTS idx_platform_config_active
    ON platform_config(effective_from DESC)
    WHERE effective_to IS NULL;

INSERT INTO platform_config (default_commission_rate, vat_rate, currency_code)
SELECT 5.00, 7.00, 'THB'
WHERE NOT EXISTS (SELECT 1 FROM platform_config);

ALTER TABLE rfqs
    ADD COLUMN IF NOT EXISTS material_grade VARCHAR(120),
    ADD COLUMN IF NOT EXISTS tolerance VARCHAR(60),
    ADD COLUMN IF NOT EXISTS color_finish VARCHAR(120),
    ADD COLUMN IF NOT EXISTS dimension_spec JSONB,
    ADD COLUMN IF NOT EXISTS weight_target_g NUMERIC(10,2),
    ADD COLUMN IF NOT EXISTS packaging_spec TEXT,
    ADD COLUMN IF NOT EXISTS target_unit_price DECIMAL(15,2),
    ADD COLUMN IF NOT EXISTS target_lead_time_days INT,
    ADD COLUMN IF NOT EXISTS required_delivery_date DATE,
    ADD COLUMN IF NOT EXISTS incoterms VARCHAR(10),
    ADD COLUMN IF NOT EXISTS payment_terms VARCHAR(20),
    ADD COLUMN IF NOT EXISTS delivery_address_id BIGINT REFERENCES addresses(address_id),
    ADD COLUMN IF NOT EXISTS certifications_required TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    ADD COLUMN IF NOT EXISTS sample_required BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS sample_qty INT,
    ADD COLUMN IF NOT EXISTS inspection_type VARCHAR(20),
    ADD COLUMN IF NOT EXISTS tech_drawing_url TEXT,
    ADD COLUMN IF NOT EXISTS reference_images TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    ADD COLUMN IF NOT EXISTS spec_sheet_url TEXT;

ALTER TABLE rfqs DROP CONSTRAINT IF EXISTS chk_rfq_incoterms;
ALTER TABLE rfqs ADD CONSTRAINT chk_rfq_incoterms
    CHECK (incoterms IS NULL OR incoterms IN ('EXW','FOB','CIF','DDP'));
ALTER TABLE rfqs DROP CONSTRAINT IF EXISTS chk_rfq_payment_terms;
ALTER TABLE rfqs ADD CONSTRAINT chk_rfq_payment_terms
    CHECK (payment_terms IS NULL OR payment_terms IN ('50_50','30_70','net_30','lc_at_sight'));
ALTER TABLE rfqs DROP CONSTRAINT IF EXISTS chk_rfq_inspection_type;
ALTER TABLE rfqs ADD CONSTRAINT chk_rfq_inspection_type
    CHECK (inspection_type IS NULL OR inspection_type IN ('self','third_party','buyer_onsite'));

ALTER TABLE quotations
    ADD COLUMN IF NOT EXISTS subtotal DECIMAL(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS discount_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS shipping_cost DECIMAL(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS shipping_method VARCHAR(120),
    ADD COLUMN IF NOT EXISTS packaging_cost DECIMAL(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS tooling_mold_cost DECIMAL(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS vat_rate DECIMAL(5,2) NOT NULL DEFAULT 7.00,
    ADD COLUMN IF NOT EXISTS vat_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS platform_commission_rate DECIMAL(5,2) NOT NULL DEFAULT 5.00,
    ADD COLUMN IF NOT EXISTS platform_commission_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS platform_config_id BIGINT REFERENCES platform_config(config_id),
    ADD COLUMN IF NOT EXISTS grand_total DECIMAL(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS factory_net_receivable DECIMAL(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS production_start_date DATE,
    ADD COLUMN IF NOT EXISTS delivery_date DATE,
    ADD COLUMN IF NOT EXISTS incoterms VARCHAR(10),
    ADD COLUMN IF NOT EXISTS payment_terms VARCHAR(20),
    ADD COLUMN IF NOT EXISTS validity_days INT NOT NULL DEFAULT 30,
    ADD COLUMN IF NOT EXISTS valid_until DATE,
    ADD COLUMN IF NOT EXISTS warranty_period_months INT,
    ADD COLUMN IF NOT EXISTS revision_no INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS parent_quotation_id BIGINT REFERENCES quotations(quote_id);

ALTER TABLE quotations DROP CONSTRAINT IF EXISTS chk_quot_incoterms;
ALTER TABLE quotations ADD CONSTRAINT chk_quot_incoterms
    CHECK (incoterms IS NULL OR incoterms IN ('EXW','FOB','CIF','DDP'));
ALTER TABLE quotations DROP CONSTRAINT IF EXISTS chk_quot_payment_terms;
ALTER TABLE quotations ADD CONSTRAINT chk_quot_payment_terms
    CHECK (payment_terms IS NULL OR payment_terms IN ('50_50','30_70','net_30','lc_at_sight'));
ALTER TABLE quotations DROP CONSTRAINT IF EXISTS quotations_status_check;
ALTER TABLE quotations ADD CONSTRAINT quotations_status_check
    CHECK (status IN ('PD', 'AC', 'RJ', 'RV'));

CREATE INDEX IF NOT EXISTS idx_quotations_parent
    ON quotations(parent_quotation_id);

CREATE TABLE IF NOT EXISTS quotation_items (
    item_id BIGSERIAL PRIMARY KEY,
    quotation_id BIGINT NOT NULL REFERENCES quotations(quote_id) ON DELETE CASCADE,
    item_no INT NOT NULL,
    description TEXT NOT NULL,
    qty DECIMAL(15,3) NOT NULL,
    unit VARCHAR(30),
    unit_price DECIMAL(15,2) NOT NULL,
    discount_pct DECIMAL(5,2) NOT NULL DEFAULT 0,
    line_total DECIMAL(15,2) NOT NULL,
    note TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (quotation_id, item_no),
    CONSTRAINT chk_quotation_items_qty CHECK (qty > 0),
    CONSTRAINT chk_quotation_items_price CHECK (unit_price >= 0),
    CONSTRAINT chk_quotation_items_discount CHECK (discount_pct BETWEEN 0 AND 100)
);

CREATE INDEX IF NOT EXISTS idx_quotation_items_qid
    ON quotation_items(quotation_id);

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS platform_commission_amount DECIMAL(15,2) NOT NULL DEFAULT 0;

COMMIT;

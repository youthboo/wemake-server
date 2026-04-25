BEGIN;

DROP INDEX IF EXISTS idx_rfqs_status_deadline;

ALTER TABLE rfqs DROP CONSTRAINT IF EXISTS chk_rfq_incoterms;
ALTER TABLE rfqs DROP CONSTRAINT IF EXISTS chk_rfq_payment_terms;

ALTER TABLE rfqs
    DROP COLUMN IF EXISTS unit_id,
    DROP COLUMN IF EXISTS budget_per_piece,
    DROP COLUMN IF EXISTS deadline_date,
    DROP COLUMN IF EXISTS image_urls,
    DROP COLUMN IF EXISTS tolerance,
    DROP COLUMN IF EXISTS color_finish,
    DROP COLUMN IF EXISTS dimension_spec,
    DROP COLUMN IF EXISTS weight_target_g,
    DROP COLUMN IF EXISTS packaging_spec,
    DROP COLUMN IF EXISTS incoterms,
    DROP COLUMN IF EXISTS payment_terms,
    DROP COLUMN IF EXISTS tech_drawing_url,
    DROP COLUMN IF EXISTS spec_sheet_url;

COMMIT;

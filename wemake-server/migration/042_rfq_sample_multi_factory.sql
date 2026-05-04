-- Migration 042: RFQ sample multi-factory + factory USP

ALTER TABLE rfqs
    ADD COLUMN IF NOT EXISTS request_kind CHAR(2) NOT NULL DEFAULT 'PR';

ALTER TABLE rfqs
    DROP CONSTRAINT IF EXISTS rfqs_request_kind_check;
ALTER TABLE rfqs
    ADD CONSTRAINT rfqs_request_kind_check
    CHECK (request_kind IN ('PR', 'PS', 'MS'));

ALTER TABLE quotations
    ADD COLUMN IF NOT EXISTS factory_highlight VARCHAR(200) NULL;

ALTER TABLE quotations
    DROP CONSTRAINT IF EXISTS quotations_status_check;
ALTER TABLE quotations
    ADD CONSTRAINT quotations_status_check
    CHECK (status IN ('PD', 'AC', 'RJ', 'EX', 'RV'));

ALTER TABLE rfqs
    DROP CONSTRAINT IF EXISTS rfqs_status_check;
ALTER TABLE rfqs
    ADD CONSTRAINT rfqs_status_check
    CHECK (status IN ('OP', 'IR', 'CL', 'CC'));

CREATE INDEX IF NOT EXISTS idx_rfqs_request_kind
    ON rfqs(request_kind);

CREATE INDEX IF NOT EXISTS idx_rfqs_kind_status_created
    ON rfqs(request_kind, status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_quotations_rfq_status
    ON quotations(rfq_id, status);

CREATE INDEX IF NOT EXISTS idx_factory_showcases_mt_lookup
    ON factory_showcases(content_type, sub_category_id, category_id)
    WHERE content_type = 'MT' AND status = 'AC';

CREATE UNIQUE INDEX IF NOT EXISTS uq_quotations_active_per_factory_rfq
    ON quotations(rfq_id, factory_id)
    WHERE status IN ('PD', 'AC');

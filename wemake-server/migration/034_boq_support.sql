BEGIN;

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS source_showcase_id BIGINT REFERENCES factory_showcases(showcase_id),
    ADD COLUMN IF NOT EXISTS conv_type VARCHAR(20) NOT NULL DEFAULT 'general';

ALTER TABLE conversations
    DROP CONSTRAINT IF EXISTS chk_conv_type;
ALTER TABLE conversations
    ADD CONSTRAINT chk_conv_type
    CHECK (conv_type IN ('general', 'showcase_inquiry', 'rfq_followup', 'boq'));

CREATE INDEX IF NOT EXISTS idx_conversations_showcase
    ON conversations(source_showcase_id)
    WHERE source_showcase_id IS NOT NULL;

WITH ranked AS (
    SELECT conv_id,
           customer_id,
           factory_id,
           MIN(conv_id) OVER (PARTITION BY customer_id, factory_id) AS keep_conv_id,
           ROW_NUMBER() OVER (PARTITION BY customer_id, factory_id ORDER BY conv_id) AS rn
    FROM conversations
),
dupes AS (
    SELECT conv_id, keep_conv_id
    FROM ranked
    WHERE rn > 1
)
UPDATE messages m
SET conv_id = d.keep_conv_id
FROM dupes d
WHERE m.conv_id = d.conv_id;

DELETE FROM conversations c
USING (
    SELECT conv_id
    FROM (
        SELECT conv_id,
               ROW_NUMBER() OVER (PARTITION BY customer_id, factory_id ORDER BY conv_id) AS rn
        FROM conversations
    ) x
    WHERE x.rn > 1
) d
WHERE c.conv_id = d.conv_id;

ALTER TABLE conversations
    DROP CONSTRAINT IF EXISTS uq_conv_customer_factory;
ALTER TABLE conversations
    ADD CONSTRAINT uq_conv_customer_factory
    UNIQUE (customer_id, factory_id);

ALTER TABLE rfqs
    ADD COLUMN IF NOT EXISTS rfq_type VARCHAR(3) NOT NULL DEFAULT 'RFQ',
    ADD COLUMN IF NOT EXISTS initiated_by VARCHAR(10) NOT NULL DEFAULT 'buyer',
    ADD COLUMN IF NOT EXISTS factory_user_id BIGINT REFERENCES users(user_id),
    ADD COLUMN IF NOT EXISTS source_showcase_id BIGINT REFERENCES factory_showcases(showcase_id),
    ADD COLUMN IF NOT EXISTS source_conv_id BIGINT REFERENCES conversations(conv_id),
    ADD COLUMN IF NOT EXISTS boq_currency CHAR(3) DEFAULT 'THB',
    ADD COLUMN IF NOT EXISTS boq_subtotal DECIMAL(15,2),
    ADD COLUMN IF NOT EXISTS boq_discount_amount DECIMAL(15,2) DEFAULT 0,
    ADD COLUMN IF NOT EXISTS boq_vat_percent DECIMAL(5,2) DEFAULT 7.00,
    ADD COLUMN IF NOT EXISTS boq_vat_amount DECIMAL(15,2),
    ADD COLUMN IF NOT EXISTS boq_grand_total DECIMAL(15,2),
    ADD COLUMN IF NOT EXISTS boq_moq INT,
    ADD COLUMN IF NOT EXISTS boq_lead_time_days INT,
    ADD COLUMN IF NOT EXISTS boq_payment_terms TEXT,
    ADD COLUMN IF NOT EXISTS boq_validity_days INT DEFAULT 14,
    ADD COLUMN IF NOT EXISTS boq_note TEXT,
    ADD COLUMN IF NOT EXISTS boq_sent_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS boq_responded_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS boq_response VARCHAR(10),
    ADD COLUMN IF NOT EXISTS boq_decline_reason TEXT;

ALTER TABLE rfqs
    DROP CONSTRAINT IF EXISTS chk_rfq_type;
ALTER TABLE rfqs
    ADD CONSTRAINT chk_rfq_type
    CHECK (rfq_type IN ('RFQ', 'BQ'));

ALTER TABLE rfqs
    DROP CONSTRAINT IF EXISTS chk_rfq_initiated_by;
ALTER TABLE rfqs
    ADD CONSTRAINT chk_rfq_initiated_by
    CHECK (initiated_by IN ('buyer', 'factory'));

ALTER TABLE rfqs
    DROP CONSTRAINT IF EXISTS chk_boq_response;
ALTER TABLE rfqs
    ADD CONSTRAINT chk_boq_response
    CHECK (boq_response IS NULL OR boq_response IN ('accepted', 'declined'));

ALTER TABLE rfqs ALTER COLUMN address_id DROP NOT NULL;
ALTER TABLE rfqs
    DROP CONSTRAINT IF EXISTS chk_rfq_address_required;
ALTER TABLE rfqs
    ADD CONSTRAINT chk_rfq_address_required
    CHECK (initiated_by = 'factory' OR address_id IS NOT NULL);

ALTER TABLE rfqs ALTER COLUMN category_id DROP NOT NULL;
ALTER TABLE rfqs
    DROP CONSTRAINT IF EXISTS chk_rfq_category_required;
ALTER TABLE rfqs
    ADD CONSTRAINT chk_rfq_category_required
    CHECK (initiated_by = 'factory' OR category_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_rfqs_type_initiated
    ON rfqs(rfq_type, initiated_by);
CREATE INDEX IF NOT EXISTS idx_rfqs_source_conv
    ON rfqs(source_conv_id)
    WHERE source_conv_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rfqs_factory_user
    ON rfqs(factory_user_id)
    WHERE factory_user_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS rfq_items (
    item_id BIGSERIAL PRIMARY KEY,
    rfq_id BIGINT NOT NULL REFERENCES rfqs(rfq_id) ON DELETE CASCADE,
    item_no INT NOT NULL,
    description TEXT NOT NULL,
    specification TEXT,
    qty DECIMAL(15,3) NOT NULL,
    unit VARCHAR(30),
    unit_price DECIMAL(15,2) NOT NULL,
    discount_pct DECIMAL(5,2) NOT NULL DEFAULT 0,
    line_total DECIMAL(15,2) NOT NULL,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (rfq_id, item_no),
    CONSTRAINT chk_rfq_items_qty CHECK (qty > 0),
    CONSTRAINT chk_rfq_items_price CHECK (unit_price >= 0),
    CONSTRAINT chk_rfq_items_discount CHECK (discount_pct BETWEEN 0 AND 100)
);

CREATE INDEX IF NOT EXISTS idx_rfq_items_rfq_id
    ON rfq_items(rfq_id);

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS boq_rfq_id BIGINT REFERENCES rfqs(rfq_id);

CREATE INDEX IF NOT EXISTS idx_messages_boq_rfq
    ON messages(boq_rfq_id)
    WHERE boq_rfq_id IS NOT NULL;

COMMIT;

-- Factory ops completion:
-- - shipping baseline fields on orders
-- - basic showcase analytics counters

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS tracking_no VARCHAR(120),
    ADD COLUMN IF NOT EXISTS courier VARCHAR(120),
    ADD COLUMN IF NOT EXISTS shipped_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_orders_factory_status
    ON orders(factory_id, status);

ALTER TABLE factory_showcases
    ADD COLUMN IF NOT EXISTS view_count BIGINT NOT NULL DEFAULT 0;

BEGIN;

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP NULL;

CREATE INDEX IF NOT EXISTS idx_orders_status_shipped_at
    ON orders(status, shipped_at);

CREATE INDEX IF NOT EXISTS idx_disputes_order_status
    ON disputes(order_id, status);

COMMIT;

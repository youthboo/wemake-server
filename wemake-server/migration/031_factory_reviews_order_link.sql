BEGIN;

ALTER TABLE factory_reviews
    ADD COLUMN IF NOT EXISTS order_id BIGINT NULL REFERENCES orders(order_id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS factory_reply TEXT NULL,
    ADD COLUMN IF NOT EXISTS factory_reply_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS factory_reply_by BIGINT NULL REFERENCES users(user_id);

UPDATE factory_reviews
SET updated_at = created_at
WHERE updated_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_factory_reviews_factory_created
    ON factory_reviews(factory_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_factory_reviews_user_created
    ON factory_reviews(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_factory_reviews_order_id
    ON factory_reviews(order_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_factory_reviews_order_user
    ON factory_reviews(order_id, user_id)
    WHERE order_id IS NOT NULL;

COMMIT;

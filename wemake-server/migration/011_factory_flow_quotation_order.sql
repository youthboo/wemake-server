-- Quotation versioning + audit (Phase 1.5)
ALTER TABLE quotations
    ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS is_locked BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS last_edited_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS last_edited_by BIGINT REFERENCES users(user_id);

CREATE TABLE IF NOT EXISTS quotation_history (
    history_id BIGSERIAL PRIMARY KEY,
    quote_id BIGINT NOT NULL REFERENCES quotations(quote_id) ON DELETE CASCADE,
    event_type VARCHAR(8) NOT NULL,
    version_after INT NOT NULL,
    price_per_piece DECIMAL(12, 2),
    mold_cost DECIMAL(12, 2),
    lead_time_days INT,
    shipping_method_id BIGINT,
    status CHAR(2),
    reason TEXT,
    edited_by BIGINT REFERENCES users(user_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quotation_history_quote ON quotation_history(quote_id, created_at DESC);

CREATE TABLE IF NOT EXISTS order_activity_log (
    activity_id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(order_id) ON DELETE CASCADE,
    actor_user_id BIGINT REFERENCES users(user_id),
    event_code VARCHAR(32) NOT NULL,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_order_activity_order ON order_activity_log(order_id, created_at DESC);

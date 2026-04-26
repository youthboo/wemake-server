BEGIN;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar_url TEXT,
    ADD COLUMN IF NOT EXISTS bio VARCHAR(300);

ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS address_line1 VARCHAR(255),
    ADD COLUMN IF NOT EXISTS sub_district VARCHAR(100),
    ADD COLUMN IF NOT EXISTS district VARCHAR(100),
    ADD COLUMN IF NOT EXISTS province VARCHAR(100),
    ADD COLUMN IF NOT EXISTS postal_code VARCHAR(10);

ALTER TABLE notifications
    ALTER COLUMN type TYPE VARCHAR(50);

ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS data JSONB,
    ADD COLUMN IF NOT EXISTS read_at TIMESTAMP,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

UPDATE notifications
SET read_at = created_at
WHERE is_read = TRUE AND read_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_notifications_user_unread
    ON notifications(user_id, created_at DESC)
    WHERE is_read = FALSE AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_notifications_user_active
    ON notifications(user_id, created_at DESC)
    WHERE deleted_at IS NULL;

ALTER TABLE factory_reviews
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP;

CREATE TABLE IF NOT EXISTS user_notification_preferences (
    user_id BIGINT PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    order_updates BOOLEAN NOT NULL DEFAULT TRUE,
    rfq_updates BOOLEAN NOT NULL DEFAULT TRUE,
    chat_messages BOOLEAN NOT NULL DEFAULT TRUE,
    promotions BOOLEAN NOT NULL DEFAULT FALSE,
    email_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    push_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMIT;

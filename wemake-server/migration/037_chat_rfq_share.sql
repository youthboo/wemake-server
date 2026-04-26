BEGIN;

ALTER TABLE rfqs
    ADD COLUMN IF NOT EXISTS conversation_id BIGINT REFERENCES conversations(conv_id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_rfqs_conversation_id
    ON rfqs(conversation_id)
    WHERE conversation_id IS NOT NULL;

ALTER TABLE messages
    ALTER COLUMN message_type TYPE VARCHAR(20) USING TRIM(message_type::text);

COMMIT;

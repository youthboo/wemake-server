BEGIN;

ALTER TABLE messages
    DROP CONSTRAINT IF EXISTS messages_reference_type_check;

ALTER TABLE messages
    ALTER COLUMN reference_type DROP NOT NULL,
    ALTER COLUMN reference_id DROP NOT NULL;

ALTER TABLE messages
    ADD CONSTRAINT messages_reference_type_check
    CHECK (
        reference_type IS NULL
        OR reference_type IN ('RQ', 'OD', 'PD', 'PM', 'ID')
    );

DROP INDEX IF EXISTS idx_messages_reference;

CREATE INDEX IF NOT EXISTS idx_messages_reference
    ON messages (reference_type, reference_id)
    WHERE reference_type IS NOT NULL;

COMMIT;

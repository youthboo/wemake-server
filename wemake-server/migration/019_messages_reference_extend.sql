BEGIN;

ALTER TABLE messages
    DROP CONSTRAINT IF EXISTS messages_reference_type_check;

ALTER TABLE messages
    ALTER COLUMN reference_type DROP NOT NULL,
    ALTER COLUMN reference_id DROP NOT NULL;

-- Normalize legacy / malformed reference_type values before re-adding the check.
UPDATE messages
SET reference_type = NULL
WHERE reference_type IS NOT NULL
  AND BTRIM(reference_type::text) = '';

UPDATE messages
SET reference_type = 'RQ'
WHERE UPPER(BTRIM(reference_type::text)) IN ('RFQ', 'RQ', 'RF');

UPDATE messages
SET reference_type = 'OD'
WHERE UPPER(BTRIM(reference_type::text)) IN ('ORDER', 'OD', 'OR');

-- Any unsupported reference type is preserved as a plain message by nulling the pair.
UPDATE messages
SET reference_type = NULL,
    reference_id = NULL
WHERE reference_type IS NOT NULL
  AND UPPER(BTRIM(reference_type::text)) NOT IN ('RQ', 'OD', 'PD', 'PM', 'ID');

-- Keep reference_type / reference_id aligned as an optional pair.
UPDATE messages
SET reference_id = NULL
WHERE reference_type IS NULL;

UPDATE messages
SET reference_type = NULL
WHERE reference_id IS NULL;

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

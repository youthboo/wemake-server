BEGIN;

ALTER TABLE messages
    ALTER COLUMN created_at TYPE TIMESTAMPTZ
    USING created_at AT TIME ZONE 'Asia/Bangkok';

ALTER TABLE messages
    ALTER COLUMN created_at SET DEFAULT NOW();

ALTER TABLE conversations
    ALTER COLUMN updated_at TYPE TIMESTAMPTZ
    USING updated_at AT TIME ZONE 'Asia/Bangkok';

ALTER TABLE conversations
    ALTER COLUMN updated_at SET DEFAULT NOW();

COMMIT;

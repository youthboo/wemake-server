BEGIN;

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS direction CHAR(1),
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(160),
    ADD COLUMN IF NOT EXISTS settlement_group_id UUID;

UPDATE transactions
SET direction = CASE
    WHEN amount < 0 THEN 'D'
    ELSE 'C'
END
WHERE direction IS NULL;

ALTER TABLE transactions
    DROP CONSTRAINT IF EXISTS transactions_direction_check;

ALTER TABLE transactions
    ADD CONSTRAINT transactions_direction_check
    CHECK (direction IS NULL OR direction IN ('D', 'C'));

CREATE UNIQUE INDEX IF NOT EXISTS ux_transactions_idempotency_key
    ON transactions (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transactions_settlement_group_id
    ON transactions (settlement_group_id)
    WHERE settlement_group_id IS NOT NULL;

COMMIT;

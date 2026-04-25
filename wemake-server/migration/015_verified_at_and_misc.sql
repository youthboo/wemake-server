-- Migration 015: verified_at on factory_profiles + misc

-- 1) factory_profiles.verified_at — when the factory was verified by admin
ALTER TABLE factory_profiles ADD COLUMN IF NOT EXISTS verified_at TIMESTAMP;

-- 2) Index for expiration jobs
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'rfqs'
          AND column_name = 'deadline_date'
    ) THEN
        CREATE INDEX IF NOT EXISTS idx_rfqs_status_deadline
            ON rfqs(status, deadline_date)
            WHERE status = 'OP';
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_quotations_status_create_time
    ON quotations(status, create_time)
    WHERE status = 'PD';

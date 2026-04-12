-- Migration 015: verified_at on factory_profiles + misc

-- 1) factory_profiles.verified_at — when the factory was verified by admin
ALTER TABLE factory_profiles ADD COLUMN IF NOT EXISTS verified_at TIMESTAMP;

-- 2) Index for expiration jobs
CREATE INDEX IF NOT EXISTS idx_rfqs_status_deadline
    ON rfqs(status, deadline_date)
    WHERE status = 'OP';

CREATE INDEX IF NOT EXISTS idx_quotations_status_create_time
    ON quotations(status, create_time)
    WHERE status = 'PD';

BEGIN;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;

ALTER TABLE users
    ADD CONSTRAINT users_role_check
    CHECK (role IN ('CT', 'FT', 'AM', 'AD', 'SA'));

ALTER TABLE factory_profiles
    ADD COLUMN IF NOT EXISTS approval_status CHAR(2) NOT NULL DEFAULT 'PE',
    ADD COLUMN IF NOT EXISTS verified_by BIGINT NULL REFERENCES users(user_id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS rejection_reason TEXT NULL,
    ADD COLUMN IF NOT EXISTS submitted_at TIMESTAMP NULL;

ALTER TABLE factory_profiles DROP CONSTRAINT IF EXISTS chk_factory_profiles_approval_status;
ALTER TABLE factory_profiles
    ADD CONSTRAINT chk_factory_profiles_approval_status
    CHECK (approval_status IN ('PE', 'AP', 'RJ', 'SU'));

UPDATE factory_profiles
SET approval_status = 'AP'
WHERE COALESCE(is_verified, FALSE) = TRUE
  AND approval_status = 'PE';

CREATE INDEX IF NOT EXISTS idx_factory_profiles_approval_status
    ON factory_profiles(approval_status);

CREATE INDEX IF NOT EXISTS idx_factory_profiles_submitted_at
    ON factory_profiles(submitted_at)
    WHERE submitted_at IS NOT NULL;

CREATE TABLE IF NOT EXISTS commission_rules (
    rule_id BIGSERIAL PRIMARY KEY,
    factory_id BIGINT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    rate_percent DECIMAL(5,2) NOT NULL,
    effective_from TIMESTAMP NOT NULL DEFAULT NOW(),
    effective_to TIMESTAMP NULL,
    note TEXT NULL,
    created_by BIGINT NOT NULL REFERENCES users(user_id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_commission_rate CHECK (rate_percent BETWEEN 0 AND 100),
    CONSTRAINT chk_commission_window CHECK (effective_to IS NULL OR effective_to > effective_from)
);

CREATE INDEX IF NOT EXISTS idx_commission_rules_factory_id
    ON commission_rules(factory_id)
    WHERE factory_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_commission_rules_active
    ON commission_rules(factory_id, effective_from DESC)
    WHERE effective_to IS NULL;

CREATE TABLE IF NOT EXISTS factory_commission_exemptions (
    exemption_id BIGSERIAL PRIMARY KEY,
    factory_id BIGINT NOT NULL UNIQUE REFERENCES users(user_id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    expires_at TIMESTAMP NULL,
    created_by BIGINT NOT NULL REFERENCES users(user_id),
    revoked_by BIGINT NULL REFERENCES users(user_id),
    revoked_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_exemptions_factory_id
    ON factory_commission_exemptions(factory_id);

CREATE INDEX IF NOT EXISTS idx_exemptions_active
    ON factory_commission_exemptions(factory_id)
    WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS admin_audit_log (
    log_id BIGSERIAL PRIMARY KEY,
    actor_id BIGINT NOT NULL REFERENCES users(user_id),
    action VARCHAR(80) NOT NULL,
    target_type VARCHAR(40) NOT NULL,
    target_id TEXT NOT NULL,
    payload JSONB NULL,
    ip_address VARCHAR(45) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_log_actor_id
    ON admin_audit_log(actor_id);

CREATE INDEX IF NOT EXISTS idx_admin_audit_log_created_at
    ON admin_audit_log(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_admin_audit_log_action
    ON admin_audit_log(action);

CREATE TABLE IF NOT EXISTS admin_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
    display_name VARCHAR(150) NOT NULL,
    department VARCHAR(100) NULL,
    created_by BIGINT NULL REFERENCES users(user_id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMIT;

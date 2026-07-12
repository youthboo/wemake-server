-- 027: Payment config (direct-pay vs escrow) + recreate withdrawal_requests
--
-- config_payment = '1'  → flow เดิม: ลูกค้าโอนตรงบัญชีโรงงาน, โรงงาน verify slip,
--                          commission invoice cron ทำงาน
-- config_payment = '0'  → escrow: ลูกค้าโอนเข้าบัญชี Tryly, superadmin verify slip,
--                          เงิน hold (tx PT) จน order CP แล้ว settle เข้า wallet โรงงาน,
--                          โรงงานถอนผ่าน withdrawal_requests

-- 1. Seed global payment-flow flag (default = current direct-pay flow)
INSERT INTO tconfig (key, value) VALUES ('config_payment', '1')
ON CONFLICT (key) DO NOTHING;

-- 2. Recreate withdrawal_requests (dropped in 001; service/repo code still targets it).
--    Columns match internal/repository/wallet/withdrawal_repository.go
--    + slip_url / processed_by for the superadmin manual-transfer flow.
CREATE TABLE IF NOT EXISTS withdrawal_requests (
    request_id      BIGSERIAL PRIMARY KEY,
    wallet_id       BIGINT NOT NULL REFERENCES wallets(wallet_id),
    factory_id      BIGINT NOT NULL,
    amount          NUMERIC(10,2) NOT NULL CHECK (amount > 0),
    bank_account_no VARCHAR(50) NOT NULL,
    bank_name       VARCHAR(100) NOT NULL,
    account_name    VARCHAR(150) NOT NULL,
    status          CHAR(2) NOT NULL DEFAULT 'PE', -- PE=Pending, AP=Approved, RJ=Rejected, CP=Complete(โอนแล้ว+แนบสลิป)
    note            TEXT,
    slip_url        TEXT,                          -- สลิปโอนเงินจาก superadmin (required เมื่อ CP)
    processed_by    BIGINT REFERENCES users(user_id),
    processed_at    TIMESTAMP,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_withdrawal_requests_status ON withdrawal_requests(status);
CREATE INDEX IF NOT EXISTS idx_withdrawal_requests_factory ON withdrawal_requests(factory_id);

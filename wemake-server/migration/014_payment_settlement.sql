-- Migration 014: Payment, Settlement, Topup, Withdrawal, Dispute, Quotation Templates

-- 1) orders.payment_type (DP=deposit, FP=full payment)
ALTER TABLE orders ADD COLUMN IF NOT EXISTS payment_type VARCHAR(10);

-- 2) payment_schedules — installment payment schedule per order
CREATE TABLE IF NOT EXISTS payment_schedules (
    schedule_id   BIGSERIAL PRIMARY KEY,
    order_id      BIGINT       NOT NULL REFERENCES orders(order_id) ON DELETE CASCADE,
    installment_no INT         NOT NULL,
    due_date      DATE         NOT NULL,
    amount        NUMERIC(15,2) NOT NULL,
    status        VARCHAR(10)  NOT NULL DEFAULT 'PE', -- PE=pending, PD=paid, OD=overdue
    paid_at       TIMESTAMP,
    created_at    TIMESTAMP    NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_payment_schedules_order_id ON payment_schedules(order_id);

-- 3) settlements — factory payout records
CREATE TABLE IF NOT EXISTS settlements (
    settlement_id BIGSERIAL PRIMARY KEY,
    factory_id    BIGINT       NOT NULL REFERENCES users(user_id),
    order_id      BIGINT       REFERENCES orders(order_id),
    amount        NUMERIC(15,2) NOT NULL,
    status        VARCHAR(10)  NOT NULL DEFAULT 'PE', -- PE, PR(processing), CP(completed), FL(failed)
    settled_at    TIMESTAMP,
    note          TEXT,
    created_at    TIMESTAMP    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP    NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_settlements_factory_id ON settlements(factory_id);
CREATE INDEX IF NOT EXISTS idx_settlements_order_id   ON settlements(order_id);

-- 4) topup_intents — PromptPay QR top-up requests
CREATE TABLE IF NOT EXISTS topup_intents (
    intent_id    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id    BIGINT       NOT NULL REFERENCES wallets(wallet_id),
    amount       NUMERIC(15,2) NOT NULL,
    qr_payload   TEXT,
    status       VARCHAR(10)  NOT NULL DEFAULT 'PE', -- PE, CP(completed), EX(expired), FL(failed)
    expires_at   TIMESTAMP,
    confirmed_at TIMESTAMP,
    created_at   TIMESTAMP    NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_topup_intents_wallet_id ON topup_intents(wallet_id);

-- 5) withdrawal_requests — factory withdrawal requests
CREATE TABLE IF NOT EXISTS withdrawal_requests (
    request_id      BIGSERIAL PRIMARY KEY,
    wallet_id       BIGINT        NOT NULL REFERENCES wallets(wallet_id),
    factory_id      BIGINT        NOT NULL REFERENCES users(user_id),
    amount          NUMERIC(15,2) NOT NULL,
    bank_account_no VARCHAR(20)   NOT NULL,
    bank_name       VARCHAR(100)  NOT NULL,
    account_name    VARCHAR(150)  NOT NULL,
    status          VARCHAR(10)   NOT NULL DEFAULT 'PE', -- PE, AP(approved), RJ(rejected), CP(completed)
    processed_at    TIMESTAMP,
    note            TEXT,
    created_at      TIMESTAMP     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP     NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_withdrawal_requests_factory_id ON withdrawal_requests(factory_id);
CREATE INDEX IF NOT EXISTS idx_withdrawal_requests_wallet_id  ON withdrawal_requests(wallet_id);

-- 6) disputes — order disputes
CREATE TABLE IF NOT EXISTS disputes (
    dispute_id  BIGSERIAL PRIMARY KEY,
    order_id    BIGINT      NOT NULL REFERENCES orders(order_id),
    opened_by   BIGINT      NOT NULL REFERENCES users(user_id),
    reason      TEXT        NOT NULL,
    status      VARCHAR(10) NOT NULL DEFAULT 'OP', -- OP(open), RS(resolved), CL(closed)
    resolution  TEXT,
    resolved_at TIMESTAMP,
    created_at  TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP   NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_disputes_order_id ON disputes(order_id);

-- 7) quotation_templates — factory reusable quotation templates
CREATE TABLE IF NOT EXISTS quotation_templates (
    template_id        BIGSERIAL PRIMARY KEY,
    factory_id         BIGINT        NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    template_name      VARCHAR(150)  NOT NULL,
    price_per_piece    NUMERIC(15,2),
    mold_cost          NUMERIC(15,2),
    lead_time_days     INT,
    shipping_method_id BIGINT        REFERENCES lbi_shipping_methods(shipping_method_id),
    note               TEXT,
    is_active          BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMP     NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP     NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_quotation_templates_factory_id ON quotation_templates(factory_id);

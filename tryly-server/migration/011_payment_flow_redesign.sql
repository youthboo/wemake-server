-- 011_payment_flow_redesign.sql
-- Switch from wallet-based to direct bank transfer payment flow

-- 1. Factory bank accounts
CREATE TABLE IF NOT EXISTS factory_bank_accounts (
    account_id     BIGSERIAL PRIMARY KEY,
    factory_id     BIGINT NOT NULL REFERENCES users(user_id),
    bank_name      VARCHAR(100) NOT NULL,
    account_number VARCHAR(20)  NOT NULL,
    account_name   VARCHAR(200) NOT NULL,
    is_default     BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_factory_bank_accounts_factory_id
    ON factory_bank_accounts(factory_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_factory_bank_accounts_default
    ON factory_bank_accounts(factory_id) WHERE is_default = true;

-- 2. Extend transactions for slip
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS slip_url      TEXT,
    ADD COLUMN IF NOT EXISTS slip_note     TEXT,
    ADD COLUMN IF NOT EXISTS verified_by   BIGINT REFERENCES users(user_id),
    ADD COLUMN IF NOT EXISTS verified_at   TIMESTAMP,
    ADD COLUMN IF NOT EXISTS email_sent_at TIMESTAMP;

-- 3. Add slip_status to orders
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS slip_status CHAR(2) DEFAULT 'PE';
-- PE = Pending (waiting slip)
-- ST = Submitted (customer attached slip)
-- AP = Approved (factory verified)
-- RJ = Rejected (factory rejected slip)

-- Backfill existing orders
UPDATE orders SET slip_status = 'AP' WHERE status IN ('PD','PR','QC','SH','CP','DL','AC');
UPDATE orders SET slip_status = 'PE' WHERE slip_status IS NULL OR slip_status = 'PE';

-- 4. Commission invoices
CREATE TABLE IF NOT EXISTS commission_invoices (
    invoice_id        BIGSERIAL PRIMARY KEY,
    factory_id        BIGINT NOT NULL REFERENCES users(user_id),
    period_month      INT NOT NULL,
    period_year       INT NOT NULL,
    total_orders      INT NOT NULL DEFAULT 0,
    total_amount      NUMERIC(15,2) NOT NULL DEFAULT 0,
    commission_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    vat_amount        NUMERIC(15,2) NOT NULL DEFAULT 0,
    grand_total       NUMERIC(15,2) NOT NULL DEFAULT 0,
    status            CHAR(2) NOT NULL DEFAULT 'DR',
    -- DR=Draft, ST=Sent, PA=Paid(slip attached), VR=Verified
    slip_url          TEXT,
    slip_note         TEXT,
    verified_by       BIGINT REFERENCES users(user_id),
    verified_at       TIMESTAMP,
    email_sent_at     TIMESTAMP,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_commission_invoices_factory_id
    ON commission_invoices(factory_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_commission_invoices_period
    ON commission_invoices(factory_id, period_month, period_year);

-- 5. Commission invoice line items
CREATE TABLE IF NOT EXISTS commission_invoice_items (
    item_id           BIGSERIAL PRIMARY KEY,
    invoice_id        BIGINT NOT NULL REFERENCES commission_invoices(invoice_id),
    order_id          BIGINT NOT NULL REFERENCES orders(order_id),
    order_amount      NUMERIC(15,2) NOT NULL,
    commission_rate   NUMERIC(5,2)  NOT NULL,
    commission_amount NUMERIC(15,2) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_commission_invoice_items_invoice
    ON commission_invoice_items(invoice_id);

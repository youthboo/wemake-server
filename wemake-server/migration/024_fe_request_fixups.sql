BEGIN;

-- FE sends explicit billing/shipping address codes. Keep legacy C/M values
-- so existing rows and older clients remain valid.
ALTER TABLE addresses
    DROP CONSTRAINT IF EXISTS addresses_address_type_check;

ALTER TABLE addresses
    ADD CONSTRAINT addresses_address_type_check
    CHECK (address_type IN ('B', 'S', 'C', 'M'));

-- Order creation now starts at PP (pending deposit). Keep legacy statuses too
-- because older flows still reference them in cancellation/shipping paths.
ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS orders_status_check;

ALTER TABLE orders
    ADD CONSTRAINT orders_status_check
    CHECK (status IN ('PP', 'PE', 'PD', 'PR', 'QC', 'SH', 'CP', 'CN', 'CC', 'WF', 'DL', 'AC'));

COMMIT;

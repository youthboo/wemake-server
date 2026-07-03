-- Allow factories to attach multiple commission-payment slip images (JPG/PNG)
-- to a commission_invoices row instead of a single TEXT url.
ALTER TABLE commission_invoices
    ADD COLUMN IF NOT EXISTS slip_urls JSONB DEFAULT '[]'::jsonb NOT NULL;

UPDATE commission_invoices
SET slip_urls = jsonb_build_array(slip_url)
WHERE slip_url IS NOT NULL AND slip_url <> '' AND slip_urls = '[]'::jsonb;

ALTER TABLE commission_invoices
    DROP COLUMN IF EXISTS slip_url;

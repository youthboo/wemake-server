-- 017_quotation_factory_qty.sql
-- Allow factory to propose their own qty + unit in a quotation.
-- NULL = accept RFQ qty as-is; non-NULL = factory counter-proposes.

ALTER TABLE quotations
  ADD COLUMN IF NOT EXISTS factory_qty  INTEGER,
  ADD COLUMN IF NOT EXISTS factory_unit_id INTEGER REFERENCES lbi_units(unit_id);

-- Propagate to orders: store the effective qty/unit that was agreed upon.
ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS quantity  INTEGER,
  ADD COLUMN IF NOT EXISTS unit_id   INTEGER REFERENCES lbi_units(unit_id);

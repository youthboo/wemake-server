-- 015_showcase_unit_id.sql
-- Add unit_id to factory_showcases so MOQ can reference a specific unit from lbi_units

ALTER TABLE factory_showcases ADD COLUMN IF NOT EXISTS unit_id INTEGER;

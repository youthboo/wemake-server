-- 014_factory_type_enum.sql
-- Replace factory_type_id (FK to lbi_factory_types) with factory_type VARCHAR ('PD','MT','all')

-- 1. Add new column
ALTER TABLE factory_profiles ADD COLUMN IF NOT EXISTS factory_type VARCHAR(5);

-- 2. Migrate existing data: map old lbi_factory_types rows to PD/MT/all
--    Default to 'all' for any existing rows
UPDATE factory_profiles SET factory_type = 'all' WHERE factory_type IS NULL;

-- 3. Set NOT NULL default
ALTER TABLE factory_profiles ALTER COLUMN factory_type SET DEFAULT 'all';
ALTER TABLE factory_profiles ALTER COLUMN factory_type SET NOT NULL;

-- 4. Drop old FK column (keep lbi_factory_types table for now — can drop later)
ALTER TABLE factory_profiles DROP COLUMN IF EXISTS factory_type_id;

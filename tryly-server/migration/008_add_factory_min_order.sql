-- ============================================================
-- Migration 008: factory_profiles.min_order
-- ============================================================
-- Adds min_order for frontend factory list/detail queries (fp.min_order).
-- Safe on DBs where 001 was applied before this column existed.

ALTER TABLE factory_profiles
  ADD COLUMN IF NOT EXISTS min_order INTEGER;

-- ============================================================
-- Rollback:
--   ALTER TABLE factory_profiles DROP COLUMN IF EXISTS min_order;
-- ============================================================

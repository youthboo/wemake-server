-- ============================================================
-- 035 — Hub cover image for /factory-ideas-hub cards
-- ============================================================

BEGIN;

ALTER TABLE lbi_hub
  ADD COLUMN IF NOT EXISTS img TEXT;

COMMENT ON COLUMN lbi_hub.img IS 'Hub cover image URL (admin upload)';

COMMIT;

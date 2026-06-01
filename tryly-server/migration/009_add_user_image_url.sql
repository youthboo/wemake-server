-- ============================================================
-- Migration 009: users.image_url (profile avatar)
-- ============================================================
-- Stores Cloudinary URL from POST /api/v1/profile/avatar.
-- API continues to expose this as avatar_url in JSON responses.

ALTER TABLE users
  ADD COLUMN IF NOT EXISTS image_url TEXT;

-- ============================================================
-- Rollback:
--   ALTER TABLE users DROP COLUMN IF EXISTS image_url;
-- ============================================================

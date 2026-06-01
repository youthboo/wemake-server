-- ============================================================
-- Migration 010: remove users.image_url (avatar handled on FE)
-- ============================================================
-- Run if 009_add_user_image_url was previously applied.

ALTER TABLE users
  DROP COLUMN IF EXISTS image_url;

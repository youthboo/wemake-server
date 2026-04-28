BEGIN;

ALTER TABLE factory_profiles
    ADD COLUMN IF NOT EXISTS background_image_url TEXT;

COMMIT;

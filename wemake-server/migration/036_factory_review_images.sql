BEGIN;

ALTER TABLE factory_reviews
    ADD COLUMN IF NOT EXISTS image_urls JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE factory_reviews
    DROP CONSTRAINT IF EXISTS chk_factory_reviews_image_urls_array;
ALTER TABLE factory_reviews
    ADD CONSTRAINT chk_factory_reviews_image_urls_array
    CHECK (jsonb_typeof(image_urls) = 'array');

ALTER TABLE factory_reviews
    DROP CONSTRAINT IF EXISTS chk_factory_reviews_image_urls_max_5;
ALTER TABLE factory_reviews
    ADD CONSTRAINT chk_factory_reviews_image_urls_max_5
    CHECK (jsonb_array_length(image_urls) <= 5);

COMMIT;

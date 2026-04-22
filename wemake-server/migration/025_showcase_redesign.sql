BEGIN;

ALTER TABLE factory_showcases
    ADD COLUMN IF NOT EXISTS moq INT,
    ADD COLUMN IF NOT EXISTS production_capacity INT,
    ADD COLUMN IF NOT EXISTS base_price NUMERIC(12, 2),
    ADD COLUMN IF NOT EXISTS promo_price NUMERIC(12, 2),
    ADD COLUMN IF NOT EXISTS start_date DATE,
    ADD COLUMN IF NOT EXISTS end_date DATE,
    ADD COLUMN IF NOT EXISTS sample_available BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS content TEXT,
    ADD COLUMN IF NOT EXISTS images JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS linked_showcases JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN IF NOT EXISTS published_at TIMESTAMP;

UPDATE factory_showcases
SET moq = COALESCE(moq, min_order),
    content = COALESCE(content, description),
    images = CASE
        WHEN images = '[]'::jsonb AND COALESCE(image_url, '') <> ''
            THEN jsonb_build_array(image_url)
        ELSE images
    END,
    published_at = CASE
        WHEN status = 'AC' AND published_at IS NULL THEN created_at
        ELSE published_at
    END;

ALTER TABLE factory_showcases
    DROP CONSTRAINT IF EXISTS factory_showcases_content_type_check,
    DROP CONSTRAINT IF EXISTS factory_showcases_status_check;

ALTER TABLE factory_showcases
    ADD CONSTRAINT factory_showcases_content_type_check CHECK (content_type IN ('PD', 'PM', 'ID')),
    ADD CONSTRAINT factory_showcases_status_check CHECK (status IN ('DR', 'AC', 'HI', 'AR'));

CREATE INDEX IF NOT EXISTS idx_factory_showcases_type_status
    ON factory_showcases(content_type, status);

CREATE INDEX IF NOT EXISTS idx_factory_showcases_category
    ON factory_showcases(category_id, sub_category_id);

COMMIT;

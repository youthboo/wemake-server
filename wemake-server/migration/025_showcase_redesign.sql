BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'factory_showcases'
          AND column_name = 'images'
          AND data_type <> 'jsonb'
    ) THEN
        ALTER TABLE factory_showcases
            ALTER COLUMN images DROP DEFAULT,
            ALTER COLUMN images TYPE JSONB
            USING CASE
                WHEN images IS NULL OR BTRIM(images::text, '"') = '' THEN '[]'::jsonb
                WHEN LEFT(BTRIM(images::text), 1) IN ('[', '{', '"')
                    THEN images::jsonb
                ELSE jsonb_build_array(images::text)
            END,
            ALTER COLUMN images SET DEFAULT '[]'::jsonb;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'factory_showcases'
          AND column_name = 'linked_showcases'
          AND data_type <> 'jsonb'
    ) THEN
        ALTER TABLE factory_showcases
            ALTER COLUMN linked_showcases DROP DEFAULT,
            ALTER COLUMN linked_showcases TYPE JSONB
            USING CASE
                WHEN linked_showcases IS NULL OR BTRIM(linked_showcases::text, '"') = '' THEN '[]'::jsonb
                WHEN LEFT(BTRIM(linked_showcases::text), 1) IN ('[', '{', '"')
                    THEN linked_showcases::jsonb
                ELSE jsonb_build_array(linked_showcases::text)
            END,
            ALTER COLUMN linked_showcases SET DEFAULT '[]'::jsonb;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'factory_showcases'
          AND column_name = 'type'
    ) THEN
        ALTER TABLE factory_showcases ADD COLUMN "type" CHAR(2);
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'factory_showcases'
          AND column_name = 'content_type'
    ) THEN
        UPDATE factory_showcases
           SET "type" = COALESCE("type", content_type, 'PD');
    ELSE
        UPDATE factory_showcases
           SET "type" = COALESCE("type", 'PD');
    END IF;

    ALTER TABLE factory_showcases ALTER COLUMN "type" SET DEFAULT 'PD';
    ALTER TABLE factory_showcases ALTER COLUMN "type" SET NOT NULL;
END $$;

ALTER TABLE factory_showcases
    ADD COLUMN IF NOT EXISTS moq INT,
    ADD COLUMN IF NOT EXISTS production_capacity INT,
    ADD COLUMN IF NOT EXISTS lead_time_days INT,
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

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'factory_showcases'
          AND column_name = 'min_order'
    ) THEN
        UPDATE factory_showcases SET moq = COALESCE(moq, min_order);
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'factory_showcases'
          AND column_name = 'description'
    ) THEN
        UPDATE factory_showcases SET content = COALESCE(content, description);
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'factory_showcases'
          AND column_name = 'image_url'
    ) THEN
        UPDATE factory_showcases
           SET images = CASE
               WHEN COALESCE(images, '[]'::jsonb) = '[]'::jsonb AND COALESCE(image_url, '') <> ''
                   THEN jsonb_build_array(image_url)
               ELSE images
           END;
    END IF;
END $$;

UPDATE factory_showcases
SET published_at = CASE
    WHEN status = 'AC' AND published_at IS NULL THEN created_at
    ELSE published_at
END;

ALTER TABLE factory_showcases
    DROP CONSTRAINT IF EXISTS factory_showcases_content_type_check,
    DROP CONSTRAINT IF EXISTS factory_showcases_type_check,
    DROP CONSTRAINT IF EXISTS factory_showcases_status_check;

ALTER TABLE factory_showcases
    ADD CONSTRAINT factory_showcases_type_check CHECK ("type" IN ('PD', 'PM', 'ID')),
    ADD CONSTRAINT factory_showcases_status_check CHECK (status IN ('DR', 'AC', 'HI', 'AR'));

ALTER TABLE factory_showcases
    DROP CONSTRAINT IF EXISTS factory_showcases_images_max_5,
    DROP CONSTRAINT IF EXISTS factory_showcases_linked_showcases_max_5,
    DROP CONSTRAINT IF EXISTS factory_showcases_promo_date_order,
    DROP CONSTRAINT IF EXISTS factory_showcases_promo_price_order,
    DROP CONSTRAINT IF EXISTS factory_showcases_non_negative_values;

ALTER TABLE factory_showcases
    ADD CONSTRAINT factory_showcases_images_max_5
        CHECK (jsonb_typeof(images) = 'array' AND jsonb_array_length(images) <= 5),
    ADD CONSTRAINT factory_showcases_linked_showcases_max_5
        CHECK (jsonb_typeof(linked_showcases) = 'array' AND jsonb_array_length(linked_showcases) <= 5),
    ADD CONSTRAINT factory_showcases_promo_date_order
        CHECK (start_date IS NULL OR end_date IS NULL OR end_date >= start_date),
    ADD CONSTRAINT factory_showcases_promo_price_order
        CHECK (base_price IS NULL OR promo_price IS NULL OR promo_price <= base_price),
    ADD CONSTRAINT factory_showcases_non_negative_values
        CHECK (
            (moq IS NULL OR moq >= 0)
            AND (production_capacity IS NULL OR production_capacity >= 0)
            AND (lead_time_days IS NULL OR lead_time_days >= 0)
            AND (base_price IS NULL OR base_price >= 0)
            AND (promo_price IS NULL OR promo_price >= 0)
        );

CREATE INDEX IF NOT EXISTS idx_factory_showcases_type_status
    ON factory_showcases("type", status);

CREATE INDEX IF NOT EXISTS idx_factory_showcases_category
    ON factory_showcases(category_id, sub_category_id);

COMMIT;

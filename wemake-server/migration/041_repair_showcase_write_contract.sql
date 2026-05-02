BEGIN;

-- Repair the write contract used by PATCH /api/v1/showcases/:showcase_id.
-- Some existing databases were created before migration history existed, so
-- older startup code may have marked pre-040 migrations as applied without
-- replaying every showcase schema change. Keep this migration additive and
-- non-destructive.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'factory_showcases'
          AND column_name = 'type'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'factory_showcases'
          AND column_name = 'content_type'
    ) THEN
        ALTER TABLE factory_showcases RENAME COLUMN "type" TO content_type;
    END IF;
END $$;

ALTER TABLE factory_showcases
    ADD COLUMN IF NOT EXISTS content_type CHAR(2),
    ADD COLUMN IF NOT EXISTS content TEXT,
    ADD COLUMN IF NOT EXISTS linked_showcases JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS image_url TEXT,
    ADD COLUMN IF NOT EXISTS excerpt TEXT,
    ADD COLUMN IF NOT EXISTS sub_category_id BIGINT,
    ADD COLUMN IF NOT EXISTS moq INT,
    ADD COLUMN IF NOT EXISTS base_price DECIMAL(15, 2),
    ADD COLUMN IF NOT EXISTS promo_price DECIMAL(15, 2),
    ADD COLUMN IF NOT EXISTS start_date DATE,
    ADD COLUMN IF NOT EXISTS end_date DATE,
    ADD COLUMN IF NOT EXISTS likes_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS view_count BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS status CHAR(2) NOT NULL DEFAULT 'AC',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN IF NOT EXISTS published_at TIMESTAMPTZ;

DO $$
BEGIN
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

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'factory_showcases'
          AND column_name = 'tags'
          AND data_type <> 'jsonb'
    ) THEN
        ALTER TABLE factory_showcases
            ALTER COLUMN tags DROP DEFAULT,
            ALTER COLUMN tags TYPE JSONB
            USING CASE
                WHEN tags IS NULL OR BTRIM(tags::text, '"') = '' THEN '[]'::jsonb
                WHEN LEFT(BTRIM(tags::text), 1) IN ('[', '{', '"')
                    THEN tags::jsonb
                ELSE jsonb_build_array(tags::text)
            END,
            ALTER COLUMN tags SET DEFAULT '[]'::jsonb;
    END IF;
END $$;

UPDATE factory_showcases
SET content_type = COALESCE(content_type, 'PD')
WHERE content_type IS NULL;

UPDATE factory_showcases
SET content_type = 'PD'
WHERE content_type NOT IN ('PD', 'PM', 'ID');

UPDATE factory_showcases
SET status = 'AC'
WHERE status NOT IN ('DR', 'AC', 'HI', 'AR', 'CL');

UPDATE factory_showcases
SET published_at = CASE
    WHEN status = 'AC' AND published_at IS NULL THEN created_at
    ELSE published_at
END;

ALTER TABLE factory_showcases
    ALTER COLUMN content_type SET DEFAULT 'PD',
    ALTER COLUMN content_type SET NOT NULL;

ALTER TABLE factory_showcases
    DROP CONSTRAINT IF EXISTS factory_showcases_content_type_check,
    DROP CONSTRAINT IF EXISTS factory_showcases_type_check,
    DROP CONSTRAINT IF EXISTS factory_showcases_status_check,
    DROP CONSTRAINT IF EXISTS chk_linked_showcases_max5,
    DROP CONSTRAINT IF EXISTS chk_moq_positive,
    DROP CONSTRAINT IF EXISTS chk_base_price_positive,
    DROP CONSTRAINT IF EXISTS chk_promo_price_lte_base,
    DROP CONSTRAINT IF EXISTS chk_pm_date_order;

ALTER TABLE factory_showcases
    ADD CONSTRAINT factory_showcases_content_type_check
        CHECK (content_type IN ('PD', 'PM', 'ID')),
    ADD CONSTRAINT factory_showcases_status_check
        CHECK (status IN ('DR', 'AC', 'HI', 'AR', 'CL')),
    ADD CONSTRAINT chk_linked_showcases_max5
        CHECK (jsonb_typeof(linked_showcases) = 'array' AND jsonb_array_length(linked_showcases) <= 5),
    ADD CONSTRAINT chk_moq_positive
        CHECK (moq IS NULL OR moq >= 0),
    ADD CONSTRAINT chk_base_price_positive
        CHECK (base_price IS NULL OR base_price >= 0),
    ADD CONSTRAINT chk_promo_price_lte_base
        CHECK (promo_price IS NULL OR base_price IS NULL OR promo_price <= base_price),
    ADD CONSTRAINT chk_pm_date_order
        CHECK (start_date IS NULL OR end_date IS NULL OR end_date >= start_date);

CREATE INDEX IF NOT EXISTS idx_showcases_content_type_status
    ON factory_showcases(content_type, status);

CREATE INDEX IF NOT EXISTS idx_showcases_category_id
    ON factory_showcases(category_id);

CREATE INDEX IF NOT EXISTS idx_showcases_sub_category_id
    ON factory_showcases(sub_category_id)
    WHERE sub_category_id IS NOT NULL;

COMMIT;

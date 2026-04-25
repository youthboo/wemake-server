-- RFQ images: store URLs as JSONB array on rfqs; drop legacy rfq_images table.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'rfqs'
          AND column_name = 'image_urls'
          AND data_type <> 'jsonb'
    ) THEN
        ALTER TABLE rfqs
            ALTER COLUMN image_urls DROP DEFAULT,
            ALTER COLUMN image_urls TYPE JSONB
            USING CASE
                WHEN image_urls IS NULL OR BTRIM(image_urls::text, '"') = '' THEN '[]'::jsonb
                WHEN LEFT(BTRIM(image_urls::text), 1) IN ('[', '{', '"')
                    THEN image_urls::jsonb
                ELSE jsonb_build_array(image_urls::text)
            END,
            ALTER COLUMN image_urls SET DEFAULT '[]'::jsonb;
    END IF;
END $$;

ALTER TABLE rfqs
    ADD COLUMN IF NOT EXISTS image_urls JSONB NOT NULL DEFAULT '[]'::jsonb;

DO $$
BEGIN
    IF to_regclass('public.rfq_images') IS NOT NULL THEN
        UPDATE rfqs r
        SET image_urls = sub.j
        FROM (
            SELECT rfq_id, jsonb_agg(image_url ORDER BY image_id) AS j
            FROM rfq_images
            GROUP BY rfq_id
        ) sub
        WHERE r.rfq_id = sub.rfq_id;

        DROP TABLE rfq_images;
    END IF;
END $$;

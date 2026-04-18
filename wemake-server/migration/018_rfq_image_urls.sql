-- RFQ images: store URLs as JSONB array on rfqs; drop legacy rfq_images table.

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

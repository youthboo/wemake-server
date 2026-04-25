BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'production_updates'
          AND column_name = 'image_urls'
          AND data_type <> 'jsonb'
    ) THEN
        ALTER TABLE production_updates
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

CREATE TABLE IF NOT EXISTS domain_events (
    event_id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP NULL
);

ALTER TABLE lbi_production
    ADD COLUMN IF NOT EXISTS step_code VARCHAR(40),
    ADD COLUMN IF NOT EXISTS step_name_th VARCHAR(150),
    ADD COLUMN IF NOT EXISTS step_name_en VARCHAR(150),
    ADD COLUMN IF NOT EXISTS sort_order INT,
    ADD COLUMN IF NOT EXISTS requires_evidence BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS min_photos SMALLINT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS is_payment_trigger BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS icon_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE lbi_production SET is_active = FALSE;

UPDATE lbi_production
SET factory_type_id = COALESCE(factory_type_id, 1),
    step_name = 'จัดเตรียมวัตถุดิบ',
    sequence = 1,
    status = '1',
    step_code = 'MATERIAL_PREP',
    step_name_th = 'จัดเตรียมวัตถุดิบ',
    step_name_en = 'Material Preparation',
    sort_order = 1,
    requires_evidence = TRUE,
    min_photos = 1,
    is_payment_trigger = FALSE,
    icon_name = 'package',
    description = 'ตรวจรับวัตถุดิบและเตรียมความพร้อมก่อนเริ่มงาน',
    is_active = TRUE
WHERE step_id = 1;

UPDATE lbi_production
SET factory_type_id = COALESCE(factory_type_id, 1),
    step_name = 'ผลิต / ประกอบ',
    sequence = 2,
    status = '1',
    step_code = 'PRODUCTION',
    step_name_th = 'ผลิต / ประกอบ',
    step_name_en = 'Production',
    sort_order = 2,
    requires_evidence = TRUE,
    min_photos = 2,
    is_payment_trigger = TRUE,
    icon_name = 'settings-2',
    description = 'แสดงหลักฐานระหว่างการผลิตหรือประกอบสินค้า',
    is_active = TRUE
WHERE step_id = 2;

UPDATE lbi_production
SET factory_type_id = COALESCE(factory_type_id, 1),
    step_name = 'ตรวจสอบคุณภาพ (QC)',
    sequence = 3,
    status = '1',
    step_code = 'QUALITY_CONTROL',
    step_name_th = 'ตรวจสอบคุณภาพ (QC)',
    step_name_en = 'Quality Control',
    sort_order = 3,
    requires_evidence = TRUE,
    min_photos = 2,
    is_payment_trigger = FALSE,
    icon_name = 'badge-check',
    description = 'บันทึกผลการตรวจสอบคุณภาพก่อนแพ็กสินค้า',
    is_active = TRUE
WHERE step_id = 3;

UPDATE lbi_production
SET factory_type_id = COALESCE(factory_type_id, 1),
    step_name = 'บรรจุภัณฑ์',
    sequence = 4,
    status = '1',
    step_code = 'PACKAGING',
    step_name_th = 'บรรจุภัณฑ์',
    step_name_en = 'Packaging',
    sort_order = 4,
    requires_evidence = TRUE,
    min_photos = 1,
    is_payment_trigger = FALSE,
    icon_name = 'box',
    description = 'แสดงภาพสินค้าหลังบรรจุภัณฑ์เรียบร้อย',
    is_active = TRUE
WHERE step_id = 4;

INSERT INTO lbi_production (
    step_id, factory_type_id, step_name, sequence, status,
    step_code, step_name_th, step_name_en, sort_order,
    requires_evidence, min_photos, is_payment_trigger, icon_name, description, is_active
)
VALUES
    (5, 1, 'พร้อมจัดส่ง', 5, '1', 'READY_TO_SHIP', 'พร้อมจัดส่ง', 'Ready to Ship', 5, TRUE, 1, TRUE, 'truck', 'ยืนยันความพร้อมก่อนจัดส่ง', TRUE),
    (6, 1, 'จัดส่งแล้ว', 6, '1', 'SHIPPED', 'จัดส่งแล้ว', 'Shipped', 6, TRUE, 1, FALSE, 'package-check', 'บันทึกหลักฐานการจัดส่งสินค้า', TRUE)
ON CONFLICT (step_id) DO UPDATE SET
    factory_type_id = EXCLUDED.factory_type_id,
    step_name = EXCLUDED.step_name,
    sequence = EXCLUDED.sequence,
    status = EXCLUDED.status,
    step_code = EXCLUDED.step_code,
    step_name_th = EXCLUDED.step_name_th,
    step_name_en = EXCLUDED.step_name_en,
    sort_order = EXCLUDED.sort_order,
    requires_evidence = EXCLUDED.requires_evidence,
    min_photos = EXCLUDED.min_photos,
    is_payment_trigger = EXCLUDED.is_payment_trigger,
    icon_name = EXCLUDED.icon_name,
    description = EXCLUDED.description,
    is_active = EXCLUDED.is_active;

CREATE UNIQUE INDEX IF NOT EXISTS idx_lbi_production_step_code ON lbi_production(step_code);
CREATE UNIQUE INDEX IF NOT EXISTS idx_lbi_production_sort_order ON lbi_production(sort_order) WHERE is_active = TRUE;

ALTER TABLE production_updates
    ADD COLUMN IF NOT EXISTS image_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP NULL,
    ADD COLUMN IF NOT EXISTS rejected_reason TEXT NULL,
    ADD COLUMN IF NOT EXISTS updated_by_user_id BIGINT NULL REFERENCES users(user_id),
    ADD COLUMN IF NOT EXISTS last_updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE production_updates
SET image_urls = CASE
    WHEN COALESCE(image_url, '') <> '' THEN jsonb_build_array(image_url)
    ELSE '[]'::jsonb
END
WHERE COALESCE(image_urls, '[]'::jsonb) = '[]'::jsonb;

UPDATE production_updates
SET status = CASE status
    WHEN 'CR' THEN 'PD'
    ELSE status
END;

ALTER TABLE production_updates DROP CONSTRAINT IF EXISTS production_updates_status_check;
ALTER TABLE production_updates
    ADD CONSTRAINT production_updates_status_check
    CHECK (status IN ('PD', 'IP', 'CD', 'RJ'));

UPDATE production_updates
SET completed_at = COALESCE(completed_at, update_date, created_at)
WHERE status = 'CD' AND completed_at IS NULL;

ALTER TABLE production_updates DROP CONSTRAINT IF EXISTS production_updates_status_completed_check;
ALTER TABLE production_updates
    ADD CONSTRAINT production_updates_status_completed_check
    CHECK ((status = 'CD' AND completed_at IS NOT NULL) OR status <> 'CD');

CREATE UNIQUE INDEX IF NOT EXISTS idx_production_updates_order_step ON production_updates(order_id, step_id);

ALTER TABLE production_updates DROP CONSTRAINT IF EXISTS production_updates_step_id_fkey;
ALTER TABLE production_updates
    ADD CONSTRAINT production_updates_step_id_fkey
    FOREIGN KEY (step_id) REFERENCES lbi_production(step_id) ON DELETE RESTRICT;

CREATE OR REPLACE FUNCTION trg_production_updates_touch()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_production_updates_touch ON production_updates;
CREATE TRIGGER trg_production_updates_touch
BEFORE UPDATE ON production_updates
FOR EACH ROW
EXECUTE FUNCTION trg_production_updates_touch();

COMMIT;

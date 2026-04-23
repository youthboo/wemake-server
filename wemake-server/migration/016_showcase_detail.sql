-- Migration: 016_showcase_detail.sql
-- Adds showcase detail support: description/price_range/status columns,
-- gallery images table, sections + section_items tables, and default seed data.

-- ============================================================
-- 1. ALTER factory_showcases — add description, price_range, status
-- ============================================================
DO $$
BEGIN
  IF to_regclass('public.factory_showcases') IS NOT NULL
     AND NOT EXISTS (
       SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'factory_showcases'
          AND column_name = 'content_type'
     ) THEN
    ALTER TABLE factory_showcases ADD COLUMN content_type CHAR(2);

    IF EXISTS (
      SELECT 1 FROM information_schema.columns
       WHERE table_schema = 'public'
         AND table_name = 'factory_showcases'
         AND column_name = 'type'
    ) THEN
      EXECUTE 'UPDATE factory_showcases SET content_type = COALESCE("type", ''PD'')';
    ELSE
      UPDATE factory_showcases SET content_type = 'PD';
    END IF;

    ALTER TABLE factory_showcases ALTER COLUMN content_type SET DEFAULT 'PD';
    ALTER TABLE factory_showcases ALTER COLUMN content_type SET NOT NULL;
  END IF;
END $$;

ALTER TABLE factory_showcases
  ADD COLUMN IF NOT EXISTS description TEXT,
  ADD COLUMN IF NOT EXISTS price_range VARCHAR(100),
  ADD COLUMN IF NOT EXISTS status      CHAR(2) NOT NULL DEFAULT 'AC';

-- ============================================================
-- 2. CREATE showcase_images
-- ============================================================
CREATE TABLE IF NOT EXISTS showcase_images (
  image_id    BIGSERIAL PRIMARY KEY,
  showcase_id BIGINT NOT NULL
                REFERENCES factory_showcases(showcase_id) ON DELETE CASCADE,
  image_url   TEXT NOT NULL,
  sort_order  INTEGER DEFAULT 0,
  caption     VARCHAR(200),
  created_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_showcase_images_showcase
  ON showcase_images(showcase_id, sort_order);

-- ============================================================
-- 3. CREATE showcase_sections + showcase_section_items
-- ============================================================
CREATE TABLE IF NOT EXISTS showcase_sections (
  section_id    BIGSERIAL PRIMARY KEY,
  showcase_id   BIGINT NOT NULL
                  REFERENCES factory_showcases(showcase_id) ON DELETE CASCADE,
  section_type  VARCHAR(10) NOT NULL,
  section_title VARCHAR(200) NOT NULL,
  sort_order    INTEGER DEFAULT 0,
  created_at    TIMESTAMP DEFAULT NOW(),
  CONSTRAINT chk_section_type CHECK (section_type IN ('highlight', 'checklist'))
);

CREATE INDEX IF NOT EXISTS idx_showcase_sections_showcase
  ON showcase_sections(showcase_id, sort_order);

CREATE TABLE IF NOT EXISTS showcase_section_items (
  item_id     BIGSERIAL PRIMARY KEY,
  section_id  BIGINT NOT NULL
                REFERENCES showcase_sections(section_id) ON DELETE CASCADE,
  title       VARCHAR(200),
  description TEXT NOT NULL,
  icon_name   VARCHAR(50),
  sort_order  INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_section_items_section
  ON showcase_section_items(section_id, sort_order);

-- ============================================================
-- 4. Seed default sections for existing showcases (run once)
-- ============================================================

-- content_type = 'ID' (Idea)
DO $$
DECLARE
  sc           RECORD;
  highlight_id BIGINT;
  checklist_id BIGINT;
BEGIN
  FOR sc IN
    SELECT fs.showcase_id FROM factory_showcases fs
    WHERE fs.content_type = 'ID'
      AND NOT EXISTS (SELECT 1 FROM showcase_sections ss WHERE ss.showcase_id = fs.showcase_id)
  LOOP
    INSERT INTO showcase_sections (showcase_id, section_type, section_title, sort_order)
      VALUES (sc.showcase_id, 'highlight', 'วิธีนำไอเดียไปต่อยอด', 0)
      RETURNING section_id INTO highlight_id;
    INSERT INTO showcase_sections (showcase_id, section_type, section_title, sort_order)
      VALUES (sc.showcase_id, 'checklist', 'สิ่งที่ควรเตรียมก่อนเริ่มผลิต', 1)
      RETURNING section_id INTO checklist_id;

    INSERT INTO showcase_section_items (section_id, title, description, icon_name, sort_order) VALUES
      (highlight_id, 'วางแผน Requirement', 'สรุปสเปกสินค้า วัตถุดิบ และงบประมาณที่ต้องการก่อนเริ่มคุยกับโรงงาน เพื่อให้ได้ราคาที่แม่นยำ', 'ListChecks', 0),
      (highlight_id, 'ทดสอบตลาดด้วย MOQ เล็ก', 'เริ่มจากล็อตเล็กเพื่อวัดตลาด ก่อนขยาย production เพื่อลดความเสี่ยงในการถือสต็อก', 'TrendingUp', 1),
      (highlight_id, 'ปรับ Positioning และแพ็กเกจ', 'ออกแบบแพ็กเกจและจุดขายให้ตรงกลุ่มเป้าหมายเพื่อเพิ่มอัตราการซื้อซ้ำและ margin', 'Lightbulb', 2);

    INSERT INTO showcase_section_items (section_id, title, description, icon_name, sort_order) VALUES
      (checklist_id, NULL, 'กลุ่มเป้าหมายและจุดขายหลักของสินค้า', NULL, 0),
      (checklist_id, NULL, 'ขนาดบรรจุ / วัสดุ / สเปกที่ต้องการ', NULL, 1),
      (checklist_id, NULL, 'งบประมาณต่อรอบผลิตและเวลาเปิดตัว', NULL, 2),
      (checklist_id, NULL, 'เอกสารที่ต้องใช้ เช่น อย., HALAL', NULL, 3);
  END LOOP;
END $$;

-- content_type = 'PD' (Product)
DO $$
DECLARE
  sc           RECORD;
  highlight_id BIGINT;
  checklist_id BIGINT;
BEGIN
  FOR sc IN
    SELECT fs.showcase_id FROM factory_showcases fs
    WHERE fs.content_type = 'PD'
      AND NOT EXISTS (SELECT 1 FROM showcase_sections ss WHERE ss.showcase_id = fs.showcase_id)
  LOOP
    INSERT INTO showcase_sections (showcase_id, section_type, section_title, sort_order)
      VALUES (sc.showcase_id, 'highlight', 'จุดเด่นที่เหมาะกับแบรนด์', 0)
      RETURNING section_id INTO highlight_id;
    INSERT INTO showcase_sections (showcase_id, section_type, section_title, sort_order)
      VALUES (sc.showcase_id, 'checklist', 'ข้อมูลที่ควรแจ้งโรงงานก่อนเริ่มผลิต', 1)
      RETURNING section_id INTO checklist_id;

    INSERT INTO showcase_section_items (section_id, title, description, icon_name, sort_order) VALUES
      (highlight_id, 'รองรับการเริ่มต้น', 'เหมาะกับการทดลองตลาดและปรับสูตร/สเปกได้ตามกลุ่มลูกค้า', 'Package', 0),
      (highlight_id, 'Lead time ชัดเจน', 'ระยะเวลาผลิตโดยเฉลี่ย {lead_time} ช่วยวางแผนเปิดตัวได้ง่าย', 'Clock', 1),
      (highlight_id, 'ยกระดับ Perceived Value', 'เพิ่มความน่าเชื่อถือและมูลค่าแบรนด์ด้วยรายละเอียดและมาตรฐานที่ครบ', 'Star', 2);

    INSERT INTO showcase_section_items (section_id, title, description, icon_name, sort_order) VALUES
      (checklist_id, NULL, 'กลุ่มเป้าหมายและจุดขายหลักของสินค้า', NULL, 0),
      (checklist_id, NULL, 'ขนาดบรรจุ/วัสดุ/รสชาติหรือสเปกที่ต้องการ', NULL, 1),
      (checklist_id, NULL, 'งบประมาณต่อรอบผลิต และช่วงเวลาที่ต้องการเปิดขาย', NULL, 2),
      (checklist_id, NULL, 'เอกสารที่ต้องใช้ เช่น อย., HALAL, หรือมาตรฐานเฉพาะแบรนด์', NULL, 3);
  END LOOP;
END $$;

-- content_type = 'PM' (Promotion)
DO $$
DECLARE
  sc           RECORD;
  highlight_id BIGINT;
  checklist_id BIGINT;
BEGIN
  FOR sc IN
    SELECT fs.showcase_id FROM factory_showcases fs
    WHERE fs.content_type = 'PM'
      AND NOT EXISTS (SELECT 1 FROM showcase_sections ss WHERE ss.showcase_id = fs.showcase_id)
  LOOP
    INSERT INTO showcase_sections (showcase_id, section_type, section_title, sort_order)
      VALUES (sc.showcase_id, 'highlight', 'เงื่อนไขโปรโมชัน', 0)
      RETURNING section_id INTO highlight_id;
    INSERT INTO showcase_sections (showcase_id, section_type, section_title, sort_order)
      VALUES (sc.showcase_id, 'checklist', 'รายละเอียดที่จำเป็นก่อนรับโปรโมชัน', 1)
      RETURNING section_id INTO checklist_id;

    INSERT INTO showcase_section_items (section_id, title, description, icon_name, sort_order) VALUES
      (highlight_id, 'คำสั่งซื้อใหม่', 'สำหรับคำสั่งซื้อใหม่ที่เริ่มผลิตภายในช่วงแคมเปญเท่านั้น', 'CirclePercent', 0),
      (highlight_id, 'ระยะเวลาผลิต', 'ระยะเวลาผลิตโดยเฉลี่ย {lead_time} ขึ้นอยู่กับสเปกและปริมาณจริง', 'CalendarClock', 1),
      (highlight_id, 'MOQ {min_order}', 'ขั้นต่ำการสั่งผลิตที่กำหนด สามารถเจรจาเพิ่มเติมได้ผ่านแชท', 'TicketPercent', 2);

    INSERT INTO showcase_section_items (section_id, title, description, icon_name, sort_order) VALUES
      (checklist_id, NULL, 'ช่วงเวลาที่ต้องการเริ่มผลิตและวันเปิดตัวสินค้า', NULL, 0),
      (checklist_id, NULL, 'จำนวนผลิตที่คาดการณ์ในรอบแรกและรอบถัดไป', NULL, 1),
      (checklist_id, NULL, 'รูปแบบแพ็กเกจหรือฉลากที่ต้องการให้รวมในโปรฯ', NULL, 2),
      (checklist_id, NULL, 'เงื่อนไขการชำระเงินและเอกสารที่แบรนด์ต้องใช้', NULL, 3);
  END LOOP;
END $$;

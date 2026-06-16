-- ============================================================
-- 019 — Hub & Category Refactor
-- Platform: Tryly
-- สร้าง lbi_hub และย้าย scope จาก lbi_categories → lbi_hub
-- lbi_categories ใช้ hub_id FK แทน scope column เดิม
-- lbi_sub_categories — ไม่เปลี่ยน schema
-- ============================================================

BEGIN;

-- ═══════════════════════════════════════════════════════════════
-- 1. สร้างตาราง lbi_hub
-- ═══════════════════════════════════════════════════════════════
CREATE TABLE lbi_hub (
    hub_id  SMALLSERIAL  NOT NULL,
    name    VARCHAR(150) NOT NULL,
    scope   BPCHAR(2)    NOT NULL, -- 'PD' = Product/OEM, 'MT' = Material
    CONSTRAINT lbi_hub_pkey PRIMARY KEY (hub_id)
);

-- ═══════════════════════════════════════════════════════════════
-- 2. Seed hubs (ตรงกับ UI spec /factory-ideas hub)
--    PD tab: สัตว์เลี้ยง, บรรจุภัณฑ์สำเร็จรูป
--    MT tab: เกษตรและธรรมชาติ, เคมีภัณฑ์และสารเติมแต่ง, วัตถุดิบบรรจุภัณฑ์
-- ═══════════════════════════════════════════════════════════════
INSERT INTO lbi_hub (name, scope) VALUES
    ('สัตว์เลี้ยง',              'PD'),  -- hub_id = 1
    ('บรรจุภัณฑ์สำเร็จรูป',     'PD'),  -- hub_id = 2
    ('เกษตรและธรรมชาติ',         'MT'),  -- hub_id = 3
    ('เคมีภัณฑ์และสารเติมแต่ง',  'MT'),  -- hub_id = 4
    ('วัตถุดิบบรรจุภัณฑ์',       'MT');  -- hub_id = 5

-- ═══════════════════════════════════════════════════════════════
-- 3. เพิ่ม hub_id ใน lbi_categories (ยังไม่ NOT NULL — map ก่อน)
-- ═══════════════════════════════════════════════════════════════
ALTER TABLE lbi_categories ADD COLUMN hub_id SMALLINT;

-- ─── Map จาก scope + category_id เดิม ────────────────────────

-- hub 1: สัตว์เลี้ยง (PD) — cat 1–6
UPDATE lbi_categories SET hub_id = 1 WHERE category_id IN (1, 2, 3, 4, 5, 6);

-- hub 2: บรรจุภัณฑ์สำเร็จรูป (PD) — cat 7
UPDATE lbi_categories SET hub_id = 2 WHERE category_id = 7;

-- hub 3: เกษตรและธรรมชาติ (MT) — โปรตีน, ธัญพืช, น้ำมัน, วัตถุดิบอื่นๆ
UPDATE lbi_categories SET hub_id = 3 WHERE category_id IN (9, 10, 14, 16);

-- hub 4: เคมีภัณฑ์และสารเติมแต่ง (MT) — สารเติมแต่งอาหาร
UPDATE lbi_categories SET hub_id = 4 WHERE category_id = 12;

-- hub 5: วัตถุดิบบรรจุภัณฑ์ (MT) — วัตถุดิบบรรจุภัณฑ์
UPDATE lbi_categories SET hub_id = 5 WHERE category_id = 13;

-- ─── เพิ่ม NOT NULL + FK หลัง map เสร็จ ─────────────────────
ALTER TABLE lbi_categories ALTER COLUMN hub_id SET NOT NULL;
ALTER TABLE lbi_categories
    ADD CONSTRAINT fk_lbi_categories_hub_id
    FOREIGN KEY (hub_id) REFERENCES lbi_hub(hub_id);

-- ═══════════════════════════════════════════════════════════════
-- 4. ลบ scope column ออกจาก lbi_categories
--    (scope ย้ายไปอยู่ที่ lbi_hub แล้ว)
-- ═══════════════════════════════════════════════════════════════
ALTER TABLE lbi_categories DROP COLUMN scope;

-- ═══════════════════════════════════════════════════════════════
-- 5. ปรับชื่อ lbi_categories ให้ตรงกับ 002 + hub ใหม่
-- ═══════════════════════════════════════════════════════════════

-- hub 1: สัตว์เลี้ยง (PD)
UPDATE lbi_categories SET name = 'อาหารสัตว์เลี้ยง'                WHERE category_id = 1;
UPDATE lbi_categories SET name = 'ขนมและของว่างสัตว์เลี้ยง'        WHERE category_id = 2;
UPDATE lbi_categories SET name = 'ยาและผลิตภัณฑ์รักษาโรค'          WHERE category_id = 3;
UPDATE lbi_categories SET name = 'อุปกรณ์และของใช้สัตว์เลี้ยง'     WHERE category_id = 4;
UPDATE lbi_categories SET name = 'ผลิตภัณฑ์ดูแลสุขภาพ'             WHERE category_id = 5;
UPDATE lbi_categories SET name = 'ผลิตภัณฑ์กำจัดปรสิตและถ่ายพยาธิ' WHERE category_id = 6;

-- hub 2: บรรจุภัณฑ์สำเร็จรูป (PD)
UPDATE lbi_categories SET name = 'บรรจุภัณฑ์สำเร็จรูป'             WHERE category_id = 7;

-- hub 3: เกษตรและธรรมชาติ (MT)
UPDATE lbi_categories SET name = 'วัตถุดิบโปรตีน'                  WHERE category_id = 9;
UPDATE lbi_categories SET name = 'วัตถุดิบธัญพืชและเส้นใย'         WHERE category_id = 10;
UPDATE lbi_categories SET name = 'น้ำมันและไขมัน'                  WHERE category_id = 14;
UPDATE lbi_categories SET name = 'วัตถุดิบอื่นๆ'                   WHERE category_id = 16;

-- hub 4: เคมีภัณฑ์และสารเติมแต่ง (MT)
UPDATE lbi_categories SET name = 'สารเติมแต่งอาหาร'                WHERE category_id = 12;

-- hub 5: วัตถุดิบบรรจุภัณฑ์ (MT)
UPDATE lbi_categories SET name = 'วัตถุดิบบรรจุภัณฑ์'              WHERE category_id = 13;

-- ═══════════════════════════════════════════════════════════════
-- 6. ลบ category 8, 11, 15 (ทำใน 002 แล้ว — guard ด้วย WHERE EXISTS)
--    ป้องกัน error หากรัน migration นี้บน DB ที่ยัง clean
-- ═══════════════════════════════════════════════════════════════
UPDATE rfqs              SET category_id = NULL WHERE category_id IN (8, 11, 15);
UPDATE factory_showcases SET category_id = NULL WHERE category_id IN (8, 11, 15);
DELETE FROM map_factory_categories WHERE category_id IN (8, 11, 15);
DELETE FROM lbi_sub_categories     WHERE category_id IN (8, 11, 15);
DELETE FROM lbi_categories         WHERE category_id IN (8, 11, 15);

-- ═══════════════════════════════════════════════════════════════
-- 7. Rebuild lbi_sub_categories (คัดลอกจาก 002 — ล้างแล้ว insert ใหม่)
-- ═══════════════════════════════════════════════════════════════
DELETE FROM lbi_sub_categories;
SELECT setval(pg_get_serial_sequence('lbi_sub_categories', 'sub_category_id'), 1, false);

INSERT INTO lbi_sub_categories (category_id, name, status, sort_order) VALUES

-- ── cat 1: อาหารสัตว์เลี้ยง (hub: สัตว์เลี้ยง) ───────────────
(1, 'อาหารชนิดเม็ด (Dry Food)',         '1', 1),
(1, 'อาหารชนิดเปียก (Wet Food)',        '1', 2),
(1, 'อาหารแช่แข็ง / Frozen',           '1', 3),
(1, 'อาหารเม็ดนิ่ม (Semi-moist)',       '1', 4),
(1, 'อาหารว่าง / Snack Food',           '1', 5),
(1, 'นมและผลิตภัณฑ์นมสัตว์เลี้ยง',    '1', 6),
(1, 'อาหารฟรีซดราย (Freeze-Dried)',     '1', 7),
(1, 'ท้อปเปอร์ / โรยหน้า',             '1', 8),
(1, 'สูตรพิเศษ (Prescription Diet)',    '1', 9),
(1, 'ทั้งหมด',                          '1', 99),

-- ── cat 2: ขนมและของว่างสัตว์เลี้ยง (hub: สัตว์เลี้ยง) ───────
(2, 'ขนมขบเคี้ยวต่างๆ',                '1', 1),
(2, 'ขนมขัดฟัน (Dental Chew)',          '1', 2),
(2, 'กระดูกและหนังสัตว์',              '1', 3),
(2, 'ขนมอบแห้ง / เนื้อสัตว์',          '1', 4),
(2, 'ขนมเลีย (Lickable Treat)',         '1', 5),
(2, 'Jerky / เนื้อแผ่น',               '1', 6),
(2, 'Biscuit / คุกกี้สัตว์เลี้ยง',     '1', 7),
(2, 'ทั้งหมด',                          '1', 99),

-- ── cat 3: ยาและผลิตภัณฑ์รักษาโรค (hub: สัตว์เลี้ยง) ─────────
(3, 'ชนิดเม็ด',                         '1', 1),
(3, 'ชนิดเปียก / Gel',                  '1', 2),
(3, 'ชนิดกิน / น้ำ',                   '1', 3),
(3, 'ชนิดหยด (Spot-on)',                '1', 4),
(3, 'ชนิดฉีด',                          '1', 5),
(3, 'ยาสมุนไพรสัตว์',                   '1', 6),
(3, 'ทั้งหมด',                          '1', 99),

-- ── cat 4: อุปกรณ์และของใช้สัตว์เลี้ยง (hub: สัตว์เลี้ยง) ────
(4, 'ผลิตภัณฑ์ทำความสะอาดสัตว์เลี้ยง', '1', 1),
(4, 'ผลิตภัณฑ์ของใช้สำหรับขับถ่าย',    '1', 2),
(4, 'ของเล่นสัตว์เลี้ยง',              '1', 3),
(4, 'อุปกรณ์ให้น้ำ-อาหาร',             '1', 4),
(4, 'กรง – คอกสัตว์เลี้ยง',            '1', 5),
(4, 'ผลิตภัณฑ์ทำความสะอาด-ดับกลิ่น',  '1', 6),
(4, 'ปลอกคอ-สายจูง',                    '1', 7),
(4, 'กระเป๋าและอุปกรณ์เดินทางต่างๆ',   '1', 8),
(4, 'รถเข็นสัตว์เลี้ยง',               '1', 9),
(4, 'เสื้อผ้าและเครื่องแต่งกายสัตว์',  '1', 10),
(4, 'รองเท้าสัตว์เลี้ยง',              '1', 11),
(4, 'ที่ข่วนเล็บ / บ้านแมว',           '1', 12),
(4, 'ทั้งหมด',                          '1', 99),

-- ── cat 5: ผลิตภัณฑ์ดูแลสุขภาพ (hub: สัตว์เลี้ยง) ───────────
(5, 'ผลิตภัณฑ์ดูแล-ช่องหู',            '1', 1),
(5, 'ผลิตภัณฑ์ดูแล-ช่องตา',            '1', 2),
(5, 'ผลิตภัณฑ์ดูแล-ช่องปาก',           '1', 3),
(5, 'ผลิตภัณฑ์ป้องกันเห็บ หมัด แมลง',  '1', 4),
(5, 'วิตามิน-อาหารเสริม',              '1', 5),
(5, 'ผลิตภัณฑ์ดูแล-รักษาผิวหนัง',     '1', 6),
(5, 'แชมพูและครีมนวดสัตว์',             '1', 7),
(5, 'Probiotic / Prebiotic',            '1', 8),
(5, 'Omega-3 / น้ำมันปลา (สำเร็จรูป)', '1', 9),
(5, 'Collagen / Joint Support',         '1', 10),
(5, 'อาหารเสริมบำรุงขน',              '1', 11),
(5, 'ทั้งหมด',                          '1', 99),

-- ── cat 6: ผลิตภัณฑ์กำจัดปรสิตและถ่ายพยาธิ (hub: สัตว์เลี้ยง)
(6, 'ชนิดเม็ด',                         '1', 1),
(6, 'ชนิดกิน / น้ำ',                   '1', 2),
(6, 'ชนิดหยด (Spot-on)',                '1', 3),
(6, 'ชนิดสเปรย์',                       '1', 4),
(6, 'ชนิดปลอกคอ',                       '1', 5),
(6, 'ทั้งหมด',                          '1', 99),

-- ── cat 7: บรรจุภัณฑ์สำเร็จรูป (hub: บรรจุภัณฑ์สำเร็จรูป) ───
(7, 'ถุง Stand-up Pouch',               '1', 1),
(7, 'กระป๋องอลูมิเนียม',               '1', 2),
(7, 'ซองฟอยล์ / Sachet',               '1', 3),
(7, 'กล่องกระดาษ / Box',               '1', 4),
(7, 'ขวดพลาสติก / PET',                '1', 5),
(7, 'ถุง Kraft Paper',                  '1', 6),
(7, 'กล่องกระดาษลูกฟูก (Carton)',      '1', 7),
(7, 'ทั้งหมด',                          '1', 99),

-- ── cat 9: วัตถุดิบโปรตีน (hub: เกษตรและธรรมชาติ) ────────────
(9,  'Chicken Meal / เนื้อไก่',         '1', 1),
(9,  'Fish Meal / ปลาป่น',              '1', 2),
(9,  'เนื้อวัว (Beef)',                  '1', 3),
(9,  'เนื้อหมู (Pork)',                  '1', 4),
(9,  'ไข่และผลิตภัณฑ์ไข่',              '1', 5),
(9,  'กุ้ง / ซีฟู้ด',                   '1', 6),
(9,  'เนื้อเป็ด (Duck)',                 '1', 7),
(9,  'แมลง (Insect Protein)',            '1', 8),
(9,  'Whey Protein',                     '1', 9),
(9,  'สารสกัดโปรตีนพืช (Pea/Soy)',     '1', 10),
(9,  'Premix วิตามิน-แร่ธาตุ',          '1', 11),
(9,  'ทั้งหมด',                          '1', 99),

-- ── cat 10: วัตถุดิบธัญพืชและเส้นใย (hub: เกษตรและธรรมชาติ) ──
(10, 'ข้าว / Rice Bran',                '1', 1),
(10, 'โอ๊ต (Oat)',                       '1', 2),
(10, 'ข้าวโพด (Corn)',                   '1', 3),
(10, 'ข้าวสาลี (Wheat)',                 '1', 4),
(10, 'มันสำปะหลัง',                     '1', 5),
(10, 'Beet Pulp (เส้นใยบีทรูท)',        '1', 6),
(10, 'ถั่วลันเตา (Pea)',                 '1', 7),
(10, 'ผักและผลไม้อบแห้ง',               '1', 8),
(10, 'ทั้งหมด',                          '1', 99),

-- ── cat 12: สารเติมแต่งอาหาร (hub: เคมีภัณฑ์และสารเติมแต่ง) ──
(12, 'สารกันเสีย (Preservatives)',       '1', 1),
(12, 'สารแต่งสี (Color Additives)',      '1', 2),
(12, 'สารให้กลิ่น (Flavoring)',         '1', 3),
(12, 'สารเพิ่มความข้น (Thickener)',     '1', 4),
(12, 'สารต้านอนุมูลอิสระ',              '1', 5),
(12, 'Enzyme (เอนไซม์)',                '1', 6),
(12, 'สารเพิ่มความชุ่มชื้น',            '1', 7),
(12, 'ทั้งหมด',                          '1', 99),

-- ── cat 13: วัตถุดิบบรรจุภัณฑ์ (hub: วัตถุดิบบรรจุภัณฑ์) ─────
(13, 'ฟิล์มพลาสติก (Film)',             '1', 1),
(13, 'ถุง Kraft Paper',                  '1', 2),
(13, 'กล่องกระดาษลูกฟูก',               '1', 3),
(13, 'ฉลากและสติกเกอร์',                '1', 4),
(13, 'ซิปล็อค / Zip Lock',              '1', 5),
(13, 'กระป๋อง / Can',                   '1', 6),
(13, 'ทั้งหมด',                          '1', 99),

-- ── cat 14: น้ำมันและไขมัน (hub: เกษตรและธรรมชาติ) ───────────
(14, 'น้ำมันปลา (Fish Oil)',             '1', 1),
(14, 'น้ำมันไก่ (Chicken Fat)',          '1', 2),
(14, 'น้ำมันพืช (Vegetable Oil)',        '1', 3),
(14, 'น้ำมันมะพร้าว (Coconut Oil)',      '1', 4),
(14, 'น้ำมันตับปลา (Cod Liver Oil)',     '1', 5),
(14, 'ทั้งหมด',                          '1', 99);

SELECT setval(
    pg_get_serial_sequence('lbi_sub_categories', 'sub_category_id'),
    GREATEST((SELECT MAX(sub_category_id) FROM lbi_sub_categories), 1)
);

-- ═══════════════════════════════════════════════════════════════
-- 8. เพิ่ม hub_id ใน factory_showcases และ rfqs
--    FE จะมี dropdown เลือก hub_id ตอน create/edit showcase และ create RFQ
--    nullable เพื่อรองรับ record เก่า + backfill จาก category_id เดิม
-- ═══════════════════════════════════════════════════════════════

-- ─── factory_showcases ─────────────────────────────────────────
ALTER TABLE factory_showcases ADD COLUMN hub_id SMALLINT;

UPDATE factory_showcases fs
SET hub_id = c.hub_id
FROM lbi_categories c
WHERE c.category_id = fs.category_id
  AND fs.category_id IS NOT NULL;

ALTER TABLE factory_showcases
    ADD CONSTRAINT fk_factory_showcases_hub_id
    FOREIGN KEY (hub_id) REFERENCES lbi_hub(hub_id);

CREATE INDEX IF NOT EXISTS idx_factory_showcases_hub_id ON factory_showcases(hub_id);

-- ─── rfqs ──────────────────────────────────────────────────────
ALTER TABLE rfqs ADD COLUMN hub_id SMALLINT;

UPDATE rfqs r
SET hub_id = c.hub_id
FROM lbi_categories c
WHERE c.category_id = r.category_id
  AND r.category_id IS NOT NULL;

ALTER TABLE rfqs
    ADD CONSTRAINT fk_rfqs_hub_id
    FOREIGN KEY (hub_id) REFERENCES lbi_hub(hub_id);

CREATE INDEX IF NOT EXISTS idx_rfqs_hub_id ON rfqs(hub_id);

COMMIT;

-- ════════════════════════════════════════════════════════════════
-- ตรวจหลังรัน:
-- SELECT hub_id, name, scope FROM lbi_hub ORDER BY hub_id;
-- -- 1 | สัตว์เลี้ยง             | PD
-- -- 2 | บรรจุภัณฑ์สำเร็จรูป    | PD
-- -- 3 | เกษตรและธรรมชาติ        | MT
-- -- 4 | เคมีภัณฑ์และสารเติมแต่ง | MT
-- -- 5 | วัตถุดิบบรรจุภัณฑ์      | MT
--
-- SELECT h.name AS hub, c.category_id, c.name
-- FROM lbi_hub h
-- JOIN lbi_categories c ON c.hub_id = h.hub_id
-- ORDER BY h.hub_id, c.category_id;
-- -- hub สัตว์เลี้ยง:          cat 1,2,3,4,5,6
-- -- hub บรรจุภัณฑ์สำเร็จรูป: cat 7
-- -- hub เกษตรและธรรมชาติ:     cat 9,10,14,16
-- -- hub เคมีภัณฑ์:            cat 12
-- -- hub วัตถุดิบบรรจุภัณฑ์:   cat 13
--
-- SELECT COUNT(*) FROM lbi_categories;   -- 13
-- SELECT COUNT(*) FROM lbi_sub_categories; -- ~118
-- ════════════════════════════════════════════════════════════════

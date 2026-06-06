-- ============================================================
-- SEED: Expanded LBI Master Data  v2
-- Platform: Tryly (Pet Food OEM B2B)
-- SA: 77 จังหวัด, ~925 อำเภอ, broad category + sub-category
--     province_code (รหัสจังหวัดทางการ) ใช้เป็น FK-safe lookup
--     ไม่เพิ่มตารางใหม่ — ใช้เฉพาะ ALTER COLUMN + DML
-- ============================================================
BEGIN;


-- ═══════════════════════════════════════════════════════════════
-- 4. lbi_categories — ปรับชื่อให้กว้างขึ้น
--    ลบ category 8 (ของสมนาคุณ), 11 (วิตามิน), 15 (สารสกัด)
--    ออกจากโครงสร้างหลัก เพราะไม่ match กับ scope โรงงาน OEM
-- ═══════════════════════════════════════════════════════════════
UPDATE lbi_categories SET name='อาหารสัตว์เลี้ยง',                  scope='PD' WHERE category_id=1;
UPDATE lbi_categories SET name='ขนมและของว่างสัตว์เลี้ยง',          scope='PD' WHERE category_id=2;
UPDATE lbi_categories SET name='ยาและผลิตภัณฑ์รักษาโรค',            scope='PD' WHERE category_id=3;
UPDATE lbi_categories SET name='อุปกรณ์และของใช้สัตว์เลี้ยง',       scope='PD' WHERE category_id=4;
UPDATE lbi_categories SET name='ผลิตภัณฑ์ดูแลสุขภาพ',               scope='PD' WHERE category_id=5;
UPDATE lbi_categories SET name='ผลิตภัณฑ์กำจัดปรสิตและถ่ายพยาธิ',   scope='PD' WHERE category_id=6;
UPDATE lbi_categories SET name='บรรจุภัณฑ์สำเร็จรูป',               scope='PD' WHERE category_id=7;
-- category_id 8  → DELETE (ของสมนาคุณ — ไม่ใช่ scope โรงงาน)
UPDATE lbi_categories SET name='วัตถุดิบโปรตีน',                    scope='MT' WHERE category_id=9;
UPDATE lbi_categories SET name='วัตถุดิบธัญพืชและเส้นใย',           scope='MT' WHERE category_id=10;
-- category_id 11 → DELETE (วิตามิน — รวมไว้ใน sub-cat ของ cat 5 แล้ว)
UPDATE lbi_categories SET name='สารเติมแต่งอาหาร',                  scope='MT' WHERE category_id=12;
UPDATE lbi_categories SET name='วัตถุดิบบรรจุภัณฑ์',                scope='MT' WHERE category_id=13;
UPDATE lbi_categories SET name='น้ำมันและไขมัน',                    scope='MT' WHERE category_id=14;
-- category_id 15 → DELETE (สารสกัด/พรีมิกซ์ — รวมใน sub-cat 9 แล้ว)
UPDATE lbi_categories SET name='วัตถุดิบอื่นๆ',                     scope='MT' WHERE category_id=16;

-- ─── ล้าง FK ก่อน DELETE lbi_categories 8,11,15 ─────────────
UPDATE rfqs            SET category_id = NULL WHERE category_id IN (8,11,15);
UPDATE factory_showcases SET category_id = NULL WHERE category_id IN (8,11,15);
DELETE FROM map_factory_categories  WHERE category_id IN (8,11,15);
DELETE FROM lbi_sub_categories      WHERE category_id IN (8,11,15);
DELETE FROM lbi_categories          WHERE category_id IN (8,11,15);

-- ═══════════════════════════════════════════════════════════════
-- 5. lbi_sub_categories — ล้างทั้งหมดและ insert ใหม่
--    ครอบคลุม category 1–7 (PD) และ 9,10,12,13,14,16 (MT)
-- ═══════════════════════════════════════════════════════════════
DELETE FROM lbi_sub_categories;
SELECT setval(pg_get_serial_sequence('lbi_sub_categories','sub_category_id'), 1, false);

INSERT INTO lbi_sub_categories (category_id, name, status, sort_order) VALUES

-- ── cat 1: อาหารสัตว์เลี้ยง ──────────────────────────────────
(1,'อาหารชนิดเม็ด (Dry Food)',         '1',1),
(1,'อาหารชนิดเปียก (Wet Food)',        '1',2),
(1,'อาหารแช่แข็ง / Frozen',           '1',3),
(1,'อาหารเม็ดนิ่ม (Semi-moist)',       '1',4),
(1,'อาหารว่าง / Snack Food',           '1',5),
(1,'นมและผลิตภัณฑ์นมสัตว์เลี้ยง',    '1',6),
(1,'อาหารฟรีซดราย (Freeze-Dried)',     '1',7),
(1,'ท้อปเปอร์ / โรยหน้า',             '1',8),
(1,'สูตรพิเศษ (Prescription Diet)',    '1',9),
(1,'อื่นๆ',                            '1',99),

-- ── cat 2: ขนมและของว่างสัตว์เลี้ยง ─────────────────────────
(2,'ขนมขบเคี้ยวต่างๆ',                '1',1),
(2,'ขนมขัดฟัน (Dental Chew)',          '1',2),
(2,'กระดูกและหนังสัตว์',              '1',3),
(2,'ขนมอบแห้ง / เนื้อสัตว์',          '1',4),
(2,'ขนมเลีย (Lickable Treat)',         '1',5),
(2,'Jerky / เนื้อแผ่น',               '1',6),
(2,'Biscuit / คุกกี้สัตว์เลี้ยง',     '1',7),
(2,'อื่นๆ',                            '1',99),

-- ── cat 3: ยาและผลิตภัณฑ์รักษาโรค ───────────────────────────
(3,'ชนิดเม็ด',                         '1',1),
(3,'ชนิดเปียก / Gel',                  '1',2),
(3,'ชนิดกิน / น้ำ',                   '1',3),
(3,'ชนิดหยด (Spot-on)',                '1',4),
(3,'ชนิดฉีด',                          '1',5),
(3,'ยาสมุนไพรสัตว์',                   '1',6),
(3,'อื่นๆ',                            '1',99),

-- ── cat 4: อุปกรณ์และของใช้สัตว์เลี้ยง ──────────────────────
(4,'ผลิตภัณฑ์ทำความสะอาดสัตว์เลี้ยง', '1',1),
(4,'ผลิตภัณฑ์ของใช้สำหรับขับถ่าย',    '1',2),
(4,'ของเล่นสัตว์เลี้ยง',              '1',3),
(4,'อุปกรณ์ให้น้ำ-อาหาร',             '1',4),
(4,'กรง – คอกสัตว์เลี้ยง',            '1',5),
(4,'ผลิตภัณฑ์ทำความสะอาด-ดับกลิ่น',  '1',6),
(4,'ปลอกคอ-สายจูง',                    '1',7),
(4,'กระเป๋าและอุปกรณ์เดินทางต่างๆ',   '1',8),
(4,'รถเข็นสัตว์เลี้ยง',               '1',9),
(4,'เสื้อผ้าและเครื่องแต่งกายสัตว์',  '1',10),
(4,'รองเท้าสัตว์เลี้ยง',              '1',11),
(4,'ที่ข่วนเล็บ / บ้านแมว',           '1',12),
(4,'อื่นๆ',                            '1',99),

-- ── cat 5: ผลิตภัณฑ์ดูแลสุขภาพ ──────────────────────────────
(5,'ผลิตภัณฑ์ดูแล-ช่องหู',            '1',1),
(5,'ผลิตภัณฑ์ดูแล-ช่องตา',            '1',2),
(5,'ผลิตภัณฑ์ดูแล-ช่องปาก',           '1',3),
(5,'ผลิตภัณฑ์ป้องกันเห็บ หมัด แมลง',  '1',4),
(5,'วิตามิน-อาหารเสริม',              '1',5),
(5,'ผลิตภัณฑ์ดูแล-รักษาผิวหนัง',     '1',6),
(5,'แชมพูและครีมนวดสัตว์',             '1',7),
(5,'Probiotic / Prebiotic',           '1',8),
(5,'Omega-3 / น้ำมันปลา (สำเร็จรูป)','1',9),
(5,'Collagen / Joint Support',        '1',10),
(5,'อาหารเสริมบำรุงขน',              '1',11),
(5,'อื่นๆ',                           '1',99),

-- ── cat 6: ผลิตภัณฑ์กำจัดปรสิตและถ่ายพยาธิ ─────────────────
(6,'ชนิดเม็ด',                         '1',1),
(6,'ชนิดกิน / น้ำ',                   '1',2),
(6,'ชนิดหยด (Spot-on)',                '1',3),
(6,'ชนิดสเปรย์',                       '1',4),
(6,'ชนิดปลอกคอ',                       '1',5),
(6,'อื่นๆ',                            '1',99),

-- ── cat 7: บรรจุภัณฑ์สำเร็จรูป ──────────────────────────────
(7,'ถุง Stand-up Pouch',              '1',1),
(7,'กระป๋องอลูมิเนียม',               '1',2),
(7,'ซองฟอยล์ / Sachet',               '1',3),
(7,'กล่องกระดาษ / Box',               '1',4),
(7,'ขวดพลาสติก / PET',                '1',5),
(7,'ถุง Kraft Paper',                  '1',6),
(7,'กล่องกระดาษลูกฟูก (Carton)',      '1',7),
(7,'อื่นๆ',                            '1',99),

-- ── cat 9: วัตถุดิบโปรตีน (MT) ──────────────────────────────
(9,'Chicken Meal / เนื้อไก่',         '1',1),
(9,'Fish Meal / ปลาป่น',              '1',2),
(9,'เนื้อวัว (Beef)',                  '1',3),
(9,'เนื้อหมู (Pork)',                  '1',4),
(9,'ไข่และผลิตภัณฑ์ไข่',              '1',5),
(9,'กุ้ง / ซีฟู้ด',                   '1',6),
(9,'เนื้อเป็ด (Duck)',                 '1',7),
(9,'แมลง (Insect Protein)',            '1',8),
(9,'Whey Protein',                     '1',9),
(9,'สารสกัดโปรตีนพืช (Pea/Soy)',     '1',10),
(9,'Premix วิตามิน-แร่ธาตุ',          '1',11),
(9,'อื่นๆ',                           '1',99),

-- ── cat 10: วัตถุดิบธัญพืชและเส้นใย (MT) ────────────────────
(10,'ข้าว / Rice Bran',               '1',1),
(10,'โอ๊ต (Oat)',                      '1',2),
(10,'ข้าวโพด (Corn)',                  '1',3),
(10,'ข้าวสาลี (Wheat)',                '1',4),
(10,'มันสำปะหลัง',                    '1',5),
(10,'Beet Pulp (เส้นใยบีทรูท)',       '1',6),
(10,'ถั่วลันเตา (Pea)',                '1',7),
(10,'ผักและผลไม้อบแห้ง',              '1',8),
(10,'อื่นๆ',                          '1',99),

-- ── cat 12: สารเติมแต่งอาหาร (MT) ───────────────────────────
(12,'สารกันเสีย (Preservatives)',      '1',1),
(12,'สารแต่งสี (Color Additives)',     '1',2),
(12,'สารให้กลิ่น (Flavoring)',        '1',3),
(12,'สารเพิ่มความข้น (Thickener)',    '1',4),
(12,'สารต้านอนุมูลอิสระ',             '1',5),
(12,'Enzyme (เอนไซม์)',               '1',6),
(12,'สารเพิ่มความชุ่มชื้น',           '1',7),
(12,'อื่นๆ',                          '1',99),

-- ── cat 13: วัตถุดิบบรรจุภัณฑ์ (MT) ─────────────────────────
(13,'ฟิล์มพลาสติก (Film)',            '1',1),
(13,'ถุง Kraft Paper',                 '1',2),
(13,'กล่องกระดาษลูกฟูก',              '1',3),
(13,'ฉลากและสติกเกอร์',               '1',4),
(13,'ซิปล็อค / Zip Lock',             '1',5),
(13,'กระป๋อง / Can',                  '1',6),
(13,'อื่นๆ',                          '1',99),

-- ── cat 14: น้ำมันและไขมัน (MT) ─────────────────────────────
(14,'น้ำมันปลา (Fish Oil)',            '1',1),
(14,'น้ำมันไก่ (Chicken Fat)',         '1',2),
(14,'น้ำมันพืช (Vegetable Oil)',       '1',3),
(14,'น้ำมันมะพร้าว (Coconut Oil)',     '1',4),
(14,'น้ำมันตับปลา (Cod Liver Oil)',    '1',5),
(14,'อื่นๆ',                           '1',99),

-- ── cat 16: วัตถุดิบอื่นๆ (MT) ──────────────────────────────
(16,'วัตถุดิบอื่นๆ',                  '1',1);

SELECT setval(pg_get_serial_sequence('lbi_sub_categories','sub_category_id'),
  GREATEST((SELECT MAX(sub_category_id) FROM lbi_sub_categories), 1));

-- ═══════════════════════════════════════════════════════════════
-- 6. lbi_factory_types — ปรับให้กว้างและ map กับ category ที่เหลือ
-- ═══════════════════════════════════════════════════════════════
UPDATE lbi_factory_types SET type_name='โรงงานผลิตอาหารสัตว์เลี้ยง'              WHERE factory_type_id=1;  -- cat 1,2
UPDATE lbi_factory_types SET type_name='โรงงานผลิตยาและผลิตภัณฑ์สุขภาพสัตว์'    WHERE factory_type_id=2;  -- cat 3,5,6
UPDATE lbi_factory_types SET type_name='โรงงานผลิตอุปกรณ์และของใช้สัตว์เลี้ยง'  WHERE factory_type_id=3;  -- cat 4
UPDATE lbi_factory_types SET type_name='โรงงานบรรจุภัณฑ์สัตว์เลี้ยง'            WHERE factory_type_id=4;  -- cat 7
UPDATE lbi_factory_types SET type_name='โรงงานวัตถุดิบโปรตีนสัตว์และพืช'        WHERE factory_type_id=5;  -- cat 9
UPDATE lbi_factory_types SET type_name='โรงงานวัตถุดิบธัญพืช เส้นใย และน้ำมัน'  WHERE factory_type_id=6;  -- cat 10,14
UPDATE lbi_factory_types SET type_name='โรงงานสารเติมแต่งและวัตถุเจือปนอาหาร'   WHERE factory_type_id=7;  -- cat 12
UPDATE lbi_factory_types SET type_name='โรงงานผลิตวัตถุดิบบรรจุภัณฑ์'           WHERE factory_type_id=8;  -- cat 13
UPDATE lbi_factory_types SET type_name='โรงงานแปรรูปและผลิตวัตถุดิบอื่นๆ'       WHERE factory_type_id=9;  -- cat 16


COMMIT;

-- ════════════════════════════════════════════════════════════════
-- ตรวจหลังรัน:
-- SELECT COUNT(*) FROM lbi_provinces;         -- 77
-- SELECT COUNT(*) FROM lbi_districts;         -- ~925 (48 BKK + ต่างจังหวัด)
-- SELECT COUNT(*) FROM lbi_categories;        -- 13  (ลบ 8,11,15 ออก)
-- SELECT COUNT(*) FROM lbi_sub_categories;    -- ~118
-- SELECT COUNT(*) FROM lbi_factory_types;     -- 10
-- SELECT province_code, name_th FROM lbi_provinces ORDER BY province_code;
-- ════════════════════════════════════════════════════════════════

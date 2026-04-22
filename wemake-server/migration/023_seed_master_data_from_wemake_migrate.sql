-- Seed data imported from /Users/poon/Downloads/wemake_migrate.sql.
-- Keep this migration data-only and idempotent: the source file is a
-- consolidated psql script with CREATE DATABASE and \c, so it must not be
-- executed directly by the app migration runner.

BEGIN;

INSERT INTO lbi_provinces (row_id, name_th, name_en, status) VALUES
    (1, 'กรุงเทพมหานคร', 'Bangkok', '1')
ON CONFLICT (row_id) DO UPDATE SET
    name_th = EXCLUDED.name_th,
    name_en = EXCLUDED.name_en,
    status = EXCLUDED.status;

INSERT INTO lbi_districts (row_id, province_id, name_th, name_en, status) VALUES
    (1001, 1, 'เขตพระนคร', 'Phra Nakhon', '1')
ON CONFLICT (row_id) DO UPDATE SET
    province_id = EXCLUDED.province_id,
    name_th = EXCLUDED.name_th,
    name_en = EXCLUDED.name_en,
    status = EXCLUDED.status;

INSERT INTO lbi_sub_districts (row_id, district_id, name_th, name_en, zip_code, status) VALUES
    (100101, 1001, 'พระบรมมหาราชวัง', 'Phra Borom Maha Ratchawang', '10200', '1')
ON CONFLICT (row_id) DO UPDATE SET
    district_id = EXCLUDED.district_id,
    name_th = EXCLUDED.name_th,
    name_en = EXCLUDED.name_en,
    zip_code = EXCLUDED.zip_code,
    status = EXCLUDED.status;

INSERT INTO lbi_factory_types (factory_type_id, type_name, status) VALUES
    (1, 'โรงพิมพ์บรรจุภัณฑ์', '1'),
    (2, 'โรงงานอาหารสัตว์', '1')
ON CONFLICT (factory_type_id) DO UPDATE SET
    type_name = EXCLUDED.type_name,
    status = EXCLUDED.status;

INSERT INTO lbi_units (unit_id, unit_name_th, unit_name_en, status) VALUES
    (1, 'ชิ้น', 'Piece', '1'),
    (2, 'กล่อง', 'Box', '1')
ON CONFLICT (unit_id) DO UPDATE SET
    unit_name_th = EXCLUDED.unit_name_th,
    unit_name_en = EXCLUDED.unit_name_en,
    status = EXCLUDED.status;

INSERT INTO lbi_shipping_methods (shipping_method_id, method_name, status) VALUES
    (1, 'ลูกค้ารับเองที่โรงงาน', '1'),
    (2, 'ขนส่งเอกชน', '1')
ON CONFLICT (shipping_method_id) DO UPDATE SET
    method_name = EXCLUDED.method_name,
    status = EXCLUDED.status;

INSERT INTO lbi_certificates (cert_id, cert_name, description, status) VALUES
    (1, 'ISO 9001', 'ระบบบริหารงานคุณภาพ', '1'),
    (2, 'GMP', NULL, '1'),
    (3, 'HACCP', NULL, '1')
ON CONFLICT (cert_id) DO UPDATE SET
    cert_name = EXCLUDED.cert_name,
    description = EXCLUDED.description,
    status = EXCLUDED.status;

INSERT INTO categories (category_id, name) VALUES
    (1, 'อาหารสัตว์'),
    (2, 'อาหารเสริม'),
    (3, 'ของเล่นสัตว์เลี้ยง'),
    (4, 'เสื้อผ้าสัตว์เลี้ยง'),
    (5, 'อุปกรณ์สัตว์เลี้ยง'),
    (6, 'บรรจุภัณฑ์'),
    (7, 'เครื่องสำอาง (สัตว์เลี้ยง)'),
    (8, 'เสื้อผ้า/สิ่งทอ'),
    (9, 'เฟอร์นิเจอร์'),
    (10, 'พลาสติก'),
    (11, 'ขนมสัตว์เลี้ยง')
ON CONFLICT (category_id) DO UPDATE SET
    name = EXCLUDED.name;

INSERT INTO lbi_sub_categories (category_id, name, sort_order) VALUES
    (1, 'อาหารสุนัข', 1),
    (1, 'อาหารแมว', 2),
    (1, 'อาหารนก/สัตว์เล็ก', 3),
    (1, 'อาหารสัตว์ทุกชนิด', 99),
    (2, 'อาหารเสริมสุนัข', 1),
    (2, 'อาหารเสริมแมว', 2),
    (2, 'อาหารเสริมสัตว์ทุกชนิด', 99),
    (3, 'ของเล่นสุนัข', 1),
    (3, 'ของเล่นแมว', 2),
    (3, 'ของเล่นสัตว์ทุกชนิด', 99),
    (4, 'เสื้อผ้าสุนัข', 1),
    (4, 'เสื้อผ้าแมว', 2),
    (4, 'เสื้อผ้าสัตว์ทุกชนิด', 99),
    (5, 'อุปกรณ์สุนัข', 1),
    (5, 'อุปกรณ์แมว', 2),
    (5, 'อุปกรณ์สัตว์ทุกชนิด', 99),
    (6, 'ถุง/Pouch', 1),
    (6, 'กล่องกระดาษ', 2),
    (6, 'ขวด/กระป๋อง', 3),
    (6, 'ฉลาก/สติกเกอร์', 4),
    (6, 'บรรจุภัณฑ์อื่นๆ', 99),
    (7, 'แชมพู/ครีมนวด', 1),
    (7, 'สบู่/โฟมอาบน้ำ', 2),
    (7, 'สเปรย์/โลชั่น', 3),
    (7, 'ผลิตภัณฑ์ดูแลทุกชนิด', 99),
    (8, 'ผ้าทอ/ผ้าถัก', 1),
    (8, 'ผ้าสำเร็จรูป', 2),
    (8, 'สิ่งทอทุกชนิด', 99),
    (9, 'คอนโดแมว/ที่ลับเล็บ', 1),
    (9, 'บ้าน/กรงสัตว์เลี้ยง', 2),
    (9, 'เฟอร์นิเจอร์สัตว์ทุกชนิด', 99),
    (10, 'ชิ้นงานพลาสติกฉีด', 1),
    (10, 'ชิ้นงานพลาสติกเป่า', 2),
    (10, 'งานพลาสติกทุกชนิด', 99),
    (11, 'ขนมสุนัข', 1),
    (11, 'ขนมแมว', 2),
    (11, 'ขนมสัตว์ทุกชนิด', 99)
ON CONFLICT (category_id, name) DO UPDATE SET
    sort_order = EXCLUDED.sort_order,
    status = '1';

INSERT INTO production_steps (name, sort_order) VALUES
    ('deposit_confirmed', 1),
    ('raw_material', 2),
    ('production', 3),
    ('qc', 4),
    ('shipping', 5),
    ('completed', 6)
ON CONFLICT (name) DO UPDATE SET
    sort_order = EXCLUDED.sort_order;

SELECT setval(pg_get_serial_sequence('lbi_provinces', 'row_id'), COALESCE((SELECT MAX(row_id) FROM lbi_provinces), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_districts', 'row_id'), COALESCE((SELECT MAX(row_id) FROM lbi_districts), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_sub_districts', 'row_id'), COALESCE((SELECT MAX(row_id) FROM lbi_sub_districts), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_factory_types', 'factory_type_id'), COALESCE((SELECT MAX(factory_type_id) FROM lbi_factory_types), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_units', 'unit_id'), COALESCE((SELECT MAX(unit_id) FROM lbi_units), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_shipping_methods', 'shipping_method_id'), COALESCE((SELECT MAX(shipping_method_id) FROM lbi_shipping_methods), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_certificates', 'cert_id'), COALESCE((SELECT MAX(cert_id) FROM lbi_certificates), 1), true);
SELECT setval(pg_get_serial_sequence('categories', 'category_id'), COALESCE((SELECT MAX(category_id) FROM categories), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_sub_categories', 'sub_category_id'), COALESCE((SELECT MAX(sub_category_id) FROM lbi_sub_categories), 1), true);
SELECT setval(pg_get_serial_sequence('production_steps', 'step_id'), COALESCE((SELECT MAX(step_id) FROM production_steps), 1), true);

COMMIT;

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
ON CONFLICT (category_id) DO NOTHING;

SELECT setval(
    pg_get_serial_sequence('categories', 'category_id'),
    COALESCE((SELECT MAX(category_id) FROM categories), 1),
    true
);

CREATE TABLE IF NOT EXISTS lbi_sub_categories (
    sub_category_id BIGSERIAL PRIMARY KEY,
    category_id BIGINT NOT NULL REFERENCES categories(category_id),
    name VARCHAR(100) NOT NULL,
    status CHAR(1) NOT NULL DEFAULT '1' CHECK (status IN ('1', '0')),
    sort_order INT NOT NULL DEFAULT 0,
    UNIQUE(category_id, name)
);

CREATE INDEX IF NOT EXISTS idx_lbi_sub_categories_cat ON lbi_sub_categories(category_id);

CREATE TABLE IF NOT EXISTS map_factory_sub_categories (
    factory_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    sub_category_id BIGINT NOT NULL REFERENCES lbi_sub_categories(sub_category_id) ON DELETE CASCADE,
    PRIMARY KEY (factory_id, sub_category_id)
);

ALTER TABLE rfqs
    ADD COLUMN IF NOT EXISTS sub_category_id BIGINT REFERENCES lbi_sub_categories(sub_category_id),
    ADD COLUMN IF NOT EXISTS shipping_method_id BIGINT REFERENCES lbi_shipping_methods(shipping_method_id);

CREATE INDEX IF NOT EXISTS idx_rfqs_sub_category ON rfqs(sub_category_id);

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
ON CONFLICT (category_id, name) DO NOTHING;

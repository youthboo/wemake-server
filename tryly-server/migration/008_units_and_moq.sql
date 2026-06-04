-- 008_units_and_moq.sql
-- Feature 1: Units of Measurement master data + unit_id in rfqs/quotations
-- Feature 2: MOQ per factory category/sub-category for RFQ matching

-- ═══════════════════════════════════════════════════════════════════
-- Feature 1: lbi_units master table
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS lbi_units (
    unit_id    SERIAL PRIMARY KEY,
    code       VARCHAR(10)  NOT NULL UNIQUE,
    name_th    VARCHAR(40)  NOT NULL,
    name_en    VARCHAR(40)  NOT NULL,
    group_th   VARCHAR(30)  NOT NULL,
    group_en   VARCHAR(30)  NOT NULL,
    sort_order INT          NOT NULL DEFAULT 0
);

-- Seed unit data (25 rows across 6 groups)
INSERT INTO lbi_units (code, name_th, name_en, group_th, group_en, sort_order) VALUES
    -- นับจำนวน (Counting) — sort 100-199
    ('pc',     'ชิ้น',         'Piece',      'นับจำนวน',    'Counting',  100),
    ('set',    'ชุด',          'Set',        'นับจำนวน',    'Counting',  101),
    ('dz',     'โหล',          'Dozen',      'นับจำนวน',    'Counting',  102),
    ('pair',   'คู่',          'Pair',       'นับจำนวน',    'Counting',  103),
    ('unit',   'หน่วย',        'Unit',       'นับจำนวน',    'Counting',  104),

    -- น้ำหนัก (Weight) — sort 200-299
    ('g',      'กรัม',         'Gram',       'น้ำหนัก',     'Weight',    200),
    ('kg',     'กิโลกรัม',     'Kilogram',   'น้ำหนัก',     'Weight',    201),
    ('ton',    'ตัน',          'Ton',        'น้ำหนัก',     'Weight',    202),

    -- ปริมาตร (Volume) — sort 300-399
    ('ml',     'มิลลิลิตร',    'Milliliter', 'ปริมาตร',     'Volume',    300),
    ('l',      'ลิตร',         'Liter',      'ปริมาตร',     'Volume',    301),
    ('gal',    'แกลลอน',       'Gallon',     'ปริมาตร',     'Volume',    302),

    -- ความยาว (Length) — sort 400-499
    ('cm',     'เซนติเมตร',    'Centimeter', 'ความยาว',     'Length',    400),
    ('m',      'เมตร',         'Meter',      'ความยาว',     'Length',    401),
    ('yd',     'หลา',          'Yard',       'ความยาว',     'Length',    402),
    ('roll',   'ม้วน',         'Roll',       'ความยาว',     'Length',    403),

    -- พื้นที่ (Area) — sort 500-599
    ('sqm',    'ตารางเมตร',    'Sq. Meter',  'พื้นที่',     'Area',      500),
    ('sqft',   'ตารางฟุต',     'Sq. Foot',   'พื้นที่',     'Area',      501),
    ('sheet',  'แผ่น',         'Sheet',      'พื้นที่',     'Area',      502),

    -- บรรจุภัณฑ์ (Packaging) — sort 600-699
    ('box',    'กล่อง',        'Box',        'บรรจุภัณฑ์',  'Packaging', 600),
    ('bag',    'ถุง',          'Bag',        'บรรจุภัณฑ์',  'Packaging', 601),
    ('pack',   'แพ็ค',         'Pack',       'บรรจุภัณฑ์',  'Packaging', 602),
    ('pallet', 'พาเลท',       'Pallet',     'บรรจุภัณฑ์',  'Packaging', 603),
    ('carton', 'ลัง',          'Carton',     'บรรจุภัณฑ์',  'Packaging', 604),
    ('drum',   'ถัง',          'Drum',       'บรรจุภัณฑ์',  'Packaging', 605),
    ('bottle', 'ขวด',          'Bottle',     'บรรจุภัณฑ์',  'Packaging', 606),
    ('tube',   'หลอด',         'Tube',       'บรรจุภัณฑ์',  'Packaging', 607)
ON CONFLICT (code) DO NOTHING;

-- Add unit_id FK to rfqs
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='rfqs' AND column_name='unit_id') THEN
        ALTER TABLE rfqs ADD COLUMN unit_id BIGINT REFERENCES lbi_units(unit_id);
    END IF;
END $$;

-- Add unit_id FK to quotations (nullable — inherits from RFQ if null)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='quotations' AND column_name='unit_id') THEN
        ALTER TABLE quotations ADD COLUMN unit_id BIGINT REFERENCES lbi_units(unit_id);
    END IF;
END $$;

-- Default existing rfqs to 'pc' (ชิ้น)
UPDATE rfqs SET unit_id = (SELECT unit_id FROM lbi_units WHERE code = 'pc' LIMIT 1) WHERE unit_id IS NULL;

-- ═══════════════════════════════════════════════════════════════════
-- Feature 2: MOQ per factory category / sub-category
-- ═══════════════════════════════════════════════════════════════════

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='map_factory_categories' AND column_name='moq') THEN
        ALTER TABLE map_factory_categories ADD COLUMN moq INTEGER NOT NULL DEFAULT 0;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='map_factory_sub_categories' AND column_name='moq') THEN
        ALTER TABLE map_factory_sub_categories ADD COLUMN moq INTEGER NOT NULL DEFAULT 0;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS transactions (
    tx_id VARCHAR(50) PRIMARY KEY,
    wallet_id BIGINT NOT NULL REFERENCES wallets(wallet_id) ON DELETE CASCADE,
    order_id BIGINT NULL REFERENCES orders(order_id) ON DELETE SET NULL,
    type CHAR(2) NOT NULL CHECK (type IN ('DP', 'WD', 'BU', 'SC', 'RF')),
    amount DECIMAL(10, 2) NOT NULL,
    status CHAR(2) NOT NULL CHECK (status IN ('ST', 'PT', 'RJ')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX IF NOT EXISTS idx_transactions_order_id ON transactions(order_id);
CREATE INDEX IF NOT EXISTS idx_transactions_type_status ON transactions(type, status);

CREATE TABLE IF NOT EXISTS lbi_provinces (
    row_id BIGSERIAL PRIMARY KEY,
    name_th VARCHAR(150) NOT NULL,
    name_en VARCHAR(150) NOT NULL,
    status CHAR(1) NOT NULL DEFAULT '1' CHECK (status IN ('1', '0'))
);

CREATE TABLE IF NOT EXISTS lbi_districts (
    row_id BIGSERIAL PRIMARY KEY,
    province_id BIGINT NOT NULL REFERENCES lbi_provinces(row_id),
    name_th VARCHAR(150) NOT NULL,
    name_en VARCHAR(150) NOT NULL,
    status CHAR(1) NOT NULL DEFAULT '1' CHECK (status IN ('1', '0'))
);

CREATE TABLE IF NOT EXISTS lbi_sub_districts (
    row_id BIGSERIAL PRIMARY KEY,
    district_id BIGINT NOT NULL REFERENCES lbi_districts(row_id),
    name_th VARCHAR(150) NOT NULL,
    name_en VARCHAR(150) NOT NULL,
    zip_code VARCHAR(10) NOT NULL,
    status CHAR(1) NOT NULL DEFAULT '1' CHECK (status IN ('1', '0'))
);

CREATE TABLE IF NOT EXISTS lbi_factory_types (
    factory_type_id BIGSERIAL PRIMARY KEY,
    type_name VARCHAR(150) NOT NULL UNIQUE,
    status CHAR(1) NOT NULL DEFAULT '1' CHECK (status IN ('1', '0'))
);

CREATE TABLE IF NOT EXISTS lbi_product_categories (
    category_id BIGSERIAL PRIMARY KEY,
    parent_category_id BIGINT NULL REFERENCES lbi_product_categories(category_id),
    category_name VARCHAR(150) NOT NULL,
    status CHAR(1) NOT NULL DEFAULT '1' CHECK (status IN ('1', '0'))
);

CREATE TABLE IF NOT EXISTS lbi_production (
    step_id BIGSERIAL PRIMARY KEY,
    factory_type_id BIGINT NOT NULL REFERENCES lbi_factory_types(factory_type_id),
    step_name VARCHAR(150) NOT NULL,
    sequence INT NOT NULL,
    status CHAR(1) NOT NULL DEFAULT '1' CHECK (status IN ('1', '0')),
    UNIQUE(factory_type_id, sequence)
);

CREATE TABLE IF NOT EXISTS lbi_units (
    unit_id BIGSERIAL PRIMARY KEY,
    unit_name_th VARCHAR(50) NOT NULL,
    unit_name_en VARCHAR(50) NOT NULL,
    status CHAR(1) NOT NULL DEFAULT '1' CHECK (status IN ('1', '0'))
);

CREATE TABLE IF NOT EXISTS lbi_shipping_methods (
    shipping_method_id BIGSERIAL PRIMARY KEY,
    method_name VARCHAR(100) NOT NULL UNIQUE,
    status CHAR(1) NOT NULL DEFAULT '1' CHECK (status IN ('1', '0'))
);

CREATE INDEX IF NOT EXISTS idx_lbi_districts_province_id ON lbi_districts(province_id);
CREATE INDEX IF NOT EXISTS idx_lbi_sub_districts_district_id ON lbi_sub_districts(district_id);
CREATE INDEX IF NOT EXISTS idx_lbi_product_categories_parent ON lbi_product_categories(parent_category_id);
CREATE INDEX IF NOT EXISTS idx_lbi_production_factory_type ON lbi_production(factory_type_id);

INSERT INTO lbi_provinces (row_id, name_th, name_en, status) VALUES
    (1, 'กรุงเทพมหานคร', 'Bangkok', '1')
ON CONFLICT (row_id) DO NOTHING;

INSERT INTO lbi_districts (row_id, province_id, name_th, name_en, status) VALUES
    (1001, 1, 'เขตพระนคร', 'Phra Nakhon', '1')
ON CONFLICT (row_id) DO NOTHING;

INSERT INTO lbi_sub_districts (row_id, district_id, name_th, name_en, zip_code, status) VALUES
    (100101, 1001, 'พระบรมมหาราชวัง', 'Phra Borom Maha Ratchawang', '10200', '1')
ON CONFLICT (row_id) DO NOTHING;

INSERT INTO lbi_factory_types (factory_type_id, type_name, status) VALUES
    (1, 'โรงพิมพ์บรรจุภัณฑ์', '1'),
    (2, 'โรงงานอาหารสัตว์', '1')
ON CONFLICT (factory_type_id) DO NOTHING;

INSERT INTO lbi_product_categories (category_id, parent_category_id, category_name, status) VALUES
    (1, NULL, 'กล่องกระดาษลูกฟูก', '1'),
    (2, NULL, 'อาหารสัตว์เลี้ยง', '1')
ON CONFLICT (category_id) DO NOTHING;

INSERT INTO lbi_production (step_id, factory_type_id, step_name, sequence, status) VALUES
    (1, 1, 'ยืนยันคำสั่งซื้อ', 1, '1'),
    (2, 1, 'พิมพ์ลายและปั้มไดคัท', 2, '1'),
    (3, 1, 'QC', 3, '1'),
    (4, 1, 'จัดส่ง', 4, '1')
ON CONFLICT (step_id) DO NOTHING;

INSERT INTO lbi_units (unit_id, unit_name_th, unit_name_en, status) VALUES
    (1, 'ชิ้น', 'Piece', '1'),
    (2, 'กล่อง', 'Box', '1')
ON CONFLICT (unit_id) DO NOTHING;

INSERT INTO lbi_shipping_methods (shipping_method_id, method_name, status) VALUES
    (1, 'ลูกค้ารับเองที่โรงงาน', '1'),
    (2, 'ขนส่งเอกชน', '1')
ON CONFLICT (shipping_method_id) DO NOTHING;

SELECT setval(pg_get_serial_sequence('lbi_provinces', 'row_id'), COALESCE((SELECT MAX(row_id) FROM lbi_provinces), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_districts', 'row_id'), COALESCE((SELECT MAX(row_id) FROM lbi_districts), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_sub_districts', 'row_id'), COALESCE((SELECT MAX(row_id) FROM lbi_sub_districts), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_factory_types', 'factory_type_id'), COALESCE((SELECT MAX(factory_type_id) FROM lbi_factory_types), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_product_categories', 'category_id'), COALESCE((SELECT MAX(category_id) FROM lbi_product_categories), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_production', 'step_id'), COALESCE((SELECT MAX(step_id) FROM lbi_production), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_units', 'unit_id'), COALESCE((SELECT MAX(unit_id) FROM lbi_units), 1), true);
SELECT setval(pg_get_serial_sequence('lbi_shipping_methods', 'shipping_method_id'), COALESCE((SELECT MAX(shipping_method_id) FROM lbi_shipping_methods), 1), true);

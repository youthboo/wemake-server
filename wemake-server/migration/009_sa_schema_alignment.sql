-- SA schema alignment: master tables, column additions, message reference codes, drop legacy tables.

-- 1) Certificate & tag masters (referenced by map_* tables)
CREATE TABLE IF NOT EXISTS lbi_certificates (
    cert_id BIGSERIAL PRIMARY KEY,
    cert_name VARCHAR(150) NOT NULL,
    description TEXT,
    status CHAR(1) NOT NULL DEFAULT '1' CHECK (status IN ('1', '0'))
);

CREATE TABLE IF NOT EXISTS lbi_tags (
    tag_id BIGSERIAL PRIMARY KEY,
    tag_name VARCHAR(100) NOT NULL UNIQUE,
    status CHAR(1) NOT NULL DEFAULT '1' CHECK (status IN ('1', '0'))
);

INSERT INTO lbi_certificates (cert_id, cert_name, description, status) VALUES
    (1, 'ISO 9001', 'ระบบบริหารงานคุณภาพ', '1'),
    (2, 'GMP', NULL, '1'),
    (3, 'HACCP', NULL, '1')
ON CONFLICT (cert_id) DO NOTHING;

SELECT setval(
    pg_get_serial_sequence('lbi_certificates', 'cert_id'),
    COALESCE((SELECT MAX(cert_id) FROM lbi_certificates), 1),
    true
);

INSERT INTO lbi_tags (tag_id, tag_name, status) VALUES
    (1, 'OEM', '1'),
    (2, 'ODM', '1'),
    (3, 'Green packaging', '1')
ON CONFLICT (tag_id) DO NOTHING;

SELECT setval(
    pg_get_serial_sequence('lbi_tags', 'tag_id'),
    COALESCE((SELECT MAX(tag_id) FROM lbi_tags), 1),
    true
);

DELETE FROM map_showcase_tags WHERE tag_id NOT IN (SELECT tag_id FROM lbi_tags);
DELETE FROM map_factory_tags WHERE tag_id NOT IN (SELECT tag_id FROM lbi_tags);
DELETE FROM map_factory_certificates WHERE cert_id NOT IN (SELECT cert_id FROM lbi_certificates);

ALTER TABLE map_factory_certificates
    DROP CONSTRAINT IF EXISTS map_factory_certificates_cert_id_fkey;
ALTER TABLE map_factory_certificates
    ADD CONSTRAINT map_factory_certificates_cert_id_fkey
    FOREIGN KEY (cert_id) REFERENCES lbi_certificates(cert_id);

ALTER TABLE map_factory_tags
    DROP CONSTRAINT IF EXISTS map_factory_tags_tag_id_fkey;
ALTER TABLE map_factory_tags
    ADD CONSTRAINT map_factory_tags_tag_id_fkey
    FOREIGN KEY (tag_id) REFERENCES lbi_tags(tag_id);

ALTER TABLE map_showcase_tags
    DROP CONSTRAINT IF EXISTS map_showcase_tags_tag_id_fkey;
ALTER TABLE map_showcase_tags
    ADD CONSTRAINT map_showcase_tags_tag_id_fkey
    FOREIGN KEY (tag_id) REFERENCES lbi_tags(tag_id);

-- 2) factory_profiles — SA fields + FK to factory type master
ALTER TABLE factory_profiles
    ADD COLUMN IF NOT EXISTS location VARCHAR(100),
    ADD COLUMN IF NOT EXISTS rating DECIMAL(3, 2),
    ADD COLUMN IF NOT EXISTS review_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS specialization VARCHAR(200),
    ADD COLUMN IF NOT EXISTS min_order INT,
    ADD COLUMN IF NOT EXISTS lead_time_desc VARCHAR(50),
    ADD COLUMN IF NOT EXISTS is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS completed_orders INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS image_url TEXT,
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS price_range VARCHAR(50);

UPDATE factory_profiles
SET is_verified = (COALESCE(tax_id, '') <> '')
WHERE is_verified IS DISTINCT FROM (COALESCE(tax_id, '') <> '');

ALTER TABLE factory_profiles
    DROP CONSTRAINT IF EXISTS factory_profiles_factory_type_id_fkey;
ALTER TABLE factory_profiles
    ADD CONSTRAINT factory_profiles_factory_type_id_fkey
    FOREIGN KEY (factory_type_id) REFERENCES lbi_factory_types(factory_type_id);

-- 3) orders — estimated delivery (SA)
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS estimated_delivery DATE;

-- 4) rfqs — deadline & uploaded_at (SA)
ALTER TABLE rfqs
    ADD COLUMN IF NOT EXISTS deadline_date DATE,
    ADD COLUMN IF NOT EXISTS uploaded_at TIMESTAMP;

UPDATE rfqs SET uploaded_at = created_at WHERE uploaded_at IS NULL;

-- 5) production_updates — step status & update timestamp (SA)
ALTER TABLE production_updates
    ADD COLUMN IF NOT EXISTS status CHAR(2) NOT NULL DEFAULT 'CR'
        CHECK (status IN ('CD', 'CR', 'PD')),
    ADD COLUMN IF NOT EXISTS update_date TIMESTAMP;

UPDATE production_updates SET update_date = created_at WHERE update_date IS NULL;
ALTER TABLE production_updates ALTER COLUMN update_date SET NOT NULL;
ALTER TABLE production_updates ALTER COLUMN update_date SET DEFAULT CURRENT_TIMESTAMP;

-- 6) units — English name (SA / future i18n)
ALTER TABLE units
    ADD COLUMN IF NOT EXISTS unit_name_en VARCHAR(50);

UPDATE units SET unit_name_en = CASE name
    WHEN 'ชิ้น' THEN 'Piece'
    WHEN 'กล่อง' THEN 'Box'
    WHEN 'กิโลกรัม' THEN 'Kilogram'
    WHEN 'แพ็ค' THEN 'Pack'
    ELSE COALESCE(unit_name_en, 'Unit')
END
WHERE unit_name_en IS NULL;

ALTER TABLE units ALTER COLUMN unit_name_en SET NOT NULL;
ALTER TABLE units ALTER COLUMN unit_name_en SET DEFAULT 'Piece';

-- 7) users — remove redundant log column (SA)
ALTER TABLE users DROP COLUMN IF EXISTS log_timestamp;

-- 8) categories — name length (SA)
ALTER TABLE categories ALTER COLUMN name TYPE VARCHAR(150);

-- 9) transactions — uploaded_at (SA)
ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS uploaded_at TIMESTAMP;

UPDATE transactions SET uploaded_at = created_at WHERE uploaded_at IS NULL;
ALTER TABLE transactions ALTER COLUMN uploaded_at SET NOT NULL;
ALTER TABLE transactions ALTER COLUMN uploaded_at SET DEFAULT CURRENT_TIMESTAMP;

-- 10) messages — reference_type RQ/OD, reference_id BIGINT (SA)
ALTER TABLE messages DROP CONSTRAINT IF EXISTS messages_reference_type_check;

UPDATE messages SET reference_type = 'RQ' WHERE UPPER(TRIM(reference_type::text)) IN ('RFQ', 'RQ');
UPDATE messages SET reference_type = 'OD' WHERE UPPER(TRIM(reference_type::text)) IN ('ORDER', 'OD');

ALTER TABLE messages ADD COLUMN IF NOT EXISTS reference_id_num BIGINT;

UPDATE messages
SET reference_id_num = NULLIF(TRIM(reference_id::text), '')::bigint
WHERE reference_id ~ '^[[:space:]]*[0-9]+[[:space:]]*$';

UPDATE messages SET reference_id_num = 0 WHERE reference_id_num IS NULL;

ALTER TABLE messages DROP COLUMN reference_id;
ALTER TABLE messages RENAME COLUMN reference_id_num TO reference_id;

ALTER TABLE messages
    ALTER COLUMN reference_type TYPE VARCHAR(2) USING UPPER(SUBSTRING(TRIM(reference_type::text) FROM 1 FOR 2));

ALTER TABLE messages
    ADD CONSTRAINT messages_reference_type_check CHECK (reference_type IN ('RQ', 'OD'));

-- 11) Drop deprecated / duplicate tables (SA)
DROP TABLE IF EXISTS lbi_product_categories CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS promotions CASCADE;
DROP TABLE IF EXISTS promo_codes CASCADE;
DROP TABLE IF EXISTS connections CASCADE;
DROP TABLE IF EXISTS factories CASCADE;
DROP TABLE IF EXISTS entrepreneurs CASCADE;

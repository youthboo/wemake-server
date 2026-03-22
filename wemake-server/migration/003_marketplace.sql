CREATE TABLE IF NOT EXISTS categories (
    category_id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS units (
    unit_id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS addresses (
    address_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    address_type VARCHAR(1) NOT NULL CHECK (address_type IN ('C', 'M')),
    address_detail VARCHAR(255) NOT NULL,
    sub_district_id BIGINT NOT NULL,
    district_id BIGINT NOT NULL,
    province_id BIGINT NOT NULL,
    zip_code VARCHAR(10) NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS wallets (
    wallet_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(user_id) ON DELETE CASCADE,
    good_fund DECIMAL(10, 2) NOT NULL DEFAULT 0,
    pending_fund DECIMAL(10, 2) NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS rfqs (
    rfq_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(category_id),
    title VARCHAR(100) NOT NULL,
    quantity BIGINT NOT NULL,
    unit_id BIGINT NOT NULL REFERENCES units(unit_id),
    budget_per_piece DECIMAL(10, 2) NOT NULL DEFAULT 0,
    details TEXT,
    address_id BIGINT NOT NULL REFERENCES addresses(address_id),
    status CHAR(2) NOT NULL DEFAULT 'OP' CHECK (status IN ('OP', 'CL', 'CC')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rfq_images (
    image_id VARCHAR(64) PRIMARY KEY,
    rfq_id BIGINT NOT NULL REFERENCES rfqs(rfq_id) ON DELETE CASCADE,
    image_url TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_addresses_user_id ON addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_rfqs_user_id ON rfqs(user_id);
CREATE INDEX IF NOT EXISTS idx_rfqs_status ON rfqs(status);
CREATE INDEX IF NOT EXISTS idx_rfq_images_rfq_id ON rfq_images(rfq_id);

INSERT INTO categories (name)
VALUES ('อาหารสัตว์'), ('อาหารเสริม'), ('ของเล่นสัตว์เลี้ยง'), ('เสื้อผ้าสัตว์เลี้ยง'), ('อุปกรณ์สัตว์เลี้ยง')
ON CONFLICT (name) DO NOTHING;

INSERT INTO units (name)
VALUES ('ชิ้น'), ('กล่อง'), ('กิโลกรัม'), ('แพ็ค')
ON CONFLICT (name) DO NOTHING;

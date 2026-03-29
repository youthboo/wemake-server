CREATE TABLE IF NOT EXISTS products (
    id VARCHAR(50) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    price VARCHAR(100) NOT NULL,
    image_url VARCHAR(500) NOT NULL,
    discount VARCHAR(50),
    factory_id VARCHAR(50),
    category_id VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS promotions (
    id VARCHAR(50) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    price VARCHAR(100) NOT NULL,
    image_url VARCHAR(500) NOT NULL,
    tag VARCHAR(100) NOT NULL,
    factory_id VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS promo_codes (
    id VARCHAR(50) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    subtitle TEXT NOT NULL,
    code VARCHAR(100) NOT NULL UNIQUE,
    valid_until DATE
);

CREATE TABLE IF NOT EXISTS factory_reviews (
    review_id SERIAL PRIMARY KEY,
    factory_id INT NOT NULL,
    user_id INT NOT NULL,
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS conversations (
    conv_id SERIAL PRIMARY KEY,
    customer_id INT NOT NULL,
    factory_id INT NOT NULL,
    last_message TEXT,
    unread_customer INT DEFAULT 0,
    unread_factory INT DEFAULT 0,
    has_quote BOOLEAN DEFAULT FALSE,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS notifications (
    noti_id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    type CHAR(2) NOT NULL,
    title VARCHAR(150) NOT NULL,
    message TEXT,
    link_to VARCHAR(255),
    is_read BOOLEAN DEFAULT FALSE,
    reference_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS factory_showcases (
    showcase_id SERIAL PRIMARY KEY,
    factory_id INT NOT NULL,
    content_type CHAR(2) NOT NULL,
    title VARCHAR(200) NOT NULL,
    excerpt TEXT,
    image_url TEXT,
    category_id INT,
    min_order INT,
    lead_time_days INT,
    likes_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS promo_slides (
    slide_id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    subtitle VARCHAR(255),
    code VARCHAR(50),
    image_url TEXT,
    status CHAR(1) DEFAULT '1'
);

CREATE TABLE IF NOT EXISTS map_factory_tags (
    map_id SERIAL PRIMARY KEY,
    factory_id INT NOT NULL,
    tag_id INT NOT NULL
);

CREATE TABLE IF NOT EXISTS map_showcase_tags (
    map_id SERIAL PRIMARY KEY,
    showcase_id INT NOT NULL,
    tag_id INT NOT NULL
);

CREATE TABLE IF NOT EXISTS favorites (
    fav_id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    showcase_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS map_factory_certificates (
    map_id SERIAL PRIMARY KEY,
    factory_id INT NOT NULL,
    cert_id INT NOT NULL,
    document_url TEXT NOT NULL,
    expire_date DATE,
    cert_number VARCHAR(100),
    verify_status CHAR(2) DEFAULT 'PD',
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE messages
ADD COLUMN IF NOT EXISTS conv_id INT,
ADD COLUMN IF NOT EXISTS message_type CHAR(2) DEFAULT 'TX',
ADD COLUMN IF NOT EXISTS quote_data JSONB,
ADD COLUMN IF NOT EXISTS is_read BOOLEAN DEFAULT FALSE;

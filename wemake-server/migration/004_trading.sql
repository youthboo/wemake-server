CREATE TABLE IF NOT EXISTS shipping_methods (
    shipping_method_id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS quotations (
    quote_id BIGSERIAL PRIMARY KEY,
    rfq_id BIGINT NOT NULL REFERENCES rfqs(rfq_id) ON DELETE CASCADE,
    factory_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    price_per_piece DECIMAL(10, 2) NOT NULL,
    mold_cost DECIMAL(10, 2) NOT NULL DEFAULT 0,
    lead_time_days INT NOT NULL,
    shipping_method_id BIGINT NOT NULL REFERENCES shipping_methods(shipping_method_id),
    status CHAR(2) NOT NULL DEFAULT 'PD' CHECK (status IN ('PD', 'AC', 'RJ')),
    create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    log_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS orders (
    order_id BIGSERIAL PRIMARY KEY,
    quote_id BIGINT NOT NULL UNIQUE REFERENCES quotations(quote_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    factory_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    total_amount DECIMAL(10, 2) NOT NULL,
    deposit_amount DECIMAL(10, 2) NOT NULL,
    status CHAR(2) NOT NULL DEFAULT 'PR' CHECK (status IN ('PR', 'QC', 'SH', 'CP')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS production_steps (
    step_id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    sort_order INT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS production_updates (
    update_id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(order_id) ON DELETE CASCADE,
    step_id BIGINT NOT NULL REFERENCES production_steps(step_id),
    description TEXT,
    image_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    message_id VARCHAR(50) PRIMARY KEY,
    reference_type VARCHAR(10) NOT NULL CHECK (reference_type IN ('RFQ', 'ORDER')),
    reference_id VARCHAR(50) NOT NULL,
    sender_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    receiver_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    attachment_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quotations_rfq_id ON quotations(rfq_id);
CREATE INDEX IF NOT EXISTS idx_quotations_factory_id ON quotations(factory_id);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_factory_id ON orders(factory_id);
CREATE INDEX IF NOT EXISTS idx_production_updates_order_id ON production_updates(order_id);
CREATE INDEX IF NOT EXISTS idx_messages_reference ON messages(reference_type, reference_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_receiver ON messages(sender_id, receiver_id);

INSERT INTO shipping_methods (name)
VALUES ('pickup'), ('courier'), ('freight')
ON CONFLICT (name) DO NOTHING;

INSERT INTO production_steps (name, sort_order)
VALUES
    ('deposit_confirmed', 1),
    ('raw_material', 2),
    ('production', 3),
    ('qc', 4),
    ('shipping', 5),
    ('completed', 6)
ON CONFLICT (name) DO NOTHING;

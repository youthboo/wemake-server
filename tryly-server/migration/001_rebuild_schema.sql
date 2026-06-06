-- Rebuilt from /Users/poon/Downloads/DB Prototype (1).xlsx, sheet Final_db!!!.
-- This migration intentionally replaces the legacy incremental migration chain.
-- WARNING: it drops managed application tables and recreates the canonical schema.
--
-- Migration chain (fresh prod):
--   001 — schema + reference seed (factory types, certs, production steps, category stubs for 002)
--   002 — category master data (lbi_categories / lbi_sub_categories / lbi_factory_types names)
--   003 — address master data (lbi_provinces / lbi_districts / lbi_sub_districts)
--   004+ — feature schema/data that is not already represented in 001

BEGIN;

DROP TABLE IF EXISTS
    addresses,
    admin_audit_log,
    admin_profiles,
    categories,
    commission_rules,
    connections,
    conversations,
    customers,
    disputes,
    domain_events,
    entrepreneurs,
    factories,
    factory_commission_exemptions,
    factory_profiles,
    factory_reviews,
    factory_rfq_dismissals,
    factory_showcases,
    favorites,
    lbi_categories,
    lbi_certificates,
    lbi_districts,
    lbi_factory_types,
    lbi_product_categories,
    lbi_production,
    lbi_provinces,
    lbi_shipping_methods,
    lbi_sub_categories,
    lbi_sub_districts,
    lbi_tags,
    lbi_units,
    map_factory_categories,
    map_factory_certificates,
    map_factory_sub_categories,
    map_factory_tags,
    map_showcase_categories,
    map_showcase_tags,
    messages,
    notifications,
    order_activity_log,
    orders,
    password_reset_tokens,
    payment_schedules,
    platform_config,
    platform_configs,
    production_steps,
    production_updates,
    products,
    promo_codes,
    promo_slides,
    promotions,
    quotation_history,
    quotation_items,
    quotation_templates,
    quotations,
    rfq_images,
    rfq_items,
    rfqs,
    settlements,
    shipping_methods,
    showcase_images,
    showcase_section_items,
    showcase_sections,
    showcase_specs,
    tconfig,
    topup_intents,
    transactions,
    units,
    user_notification_preferences,
    users,
    wallets,
    withdrawal_requests CASCADE;

CREATE TABLE IF NOT EXISTS addresses (
    address_id BIGSERIAL NOT NULL,
    user_id BIGINT NOT NULL,
    address_type VARCHAR(1) NOT NULL,
    address_detail VARCHAR(255) NOT NULL,
    sub_district_id BIGINT NOT NULL,
    district_id BIGINT NOT NULL,
    province_id BIGINT NOT NULL,
    zip_code VARCHAR(10) NOT NULL,
    is_default BOOLEAN DEFAULT FALSE NOT NULL,
    CONSTRAINT addresses_pkey PRIMARY KEY (address_id)
);

CREATE TABLE IF NOT EXISTS admin_audit_log (
    log_id BIGSERIAL NOT NULL,
    actor_id BIGINT NOT NULL,
    action VARCHAR(80) NOT NULL,
    target_type VARCHAR(40) NOT NULL,
    target_id TEXT NOT NULL,
    payload JSONB,
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT now() NOT NULL,
    CONSTRAINT admin_audit_log_pkey PRIMARY KEY (log_id)
);

CREATE TABLE IF NOT EXISTS admin_profiles (
    user_id BIGINT NOT NULL,
    display_name VARCHAR(150) NOT NULL,
    department VARCHAR(100),
    created_by BIGINT,
    created_at TIMESTAMP NOT NULL,
    CONSTRAINT admin_profiles_pkey PRIMARY KEY (user_id)
);

CREATE TABLE IF NOT EXISTS lbi_categories (
    category_id BIGSERIAL NOT NULL,
    name VARCHAR(150) NOT NULL,
    scope CHAR(2) NOT NULL,
    CONSTRAINT lbi_categories_pkey PRIMARY KEY (category_id)
);

CREATE TABLE IF NOT EXISTS lbi_sub_categories (
    sub_category_id BIGSERIAL NOT NULL,
    category_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    status BPCHAR(1) DEFAULT '1' NOT NULL,
    sort_order INTEGER DEFAULT 0 NOT NULL,
    CONSTRAINT lbi_sub_categories_pkey PRIMARY KEY (sub_category_id)
);

CREATE TABLE IF NOT EXISTS conversations (
    conv_id BIGSERIAL NOT NULL,
    customer_id INTEGER NOT NULL,
    factory_id INTEGER NOT NULL,
    last_message TEXT,
    unread_customer INTEGER DEFAULT 0,
    unread_factory INTEGER DEFAULT 0,
    updated_at TIMESTAMPTZ DEFAULT now(),
    CONSTRAINT conversations_pkey PRIMARY KEY (conv_id)
);

CREATE TABLE IF NOT EXISTS customers (
    user_id BIGINT NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    CONSTRAINT customers_pkey PRIMARY KEY (user_id)
);

CREATE TABLE IF NOT EXISTS factory_commission_exemptions (
    exemption_id BIGSERIAL NOT NULL,
    factory_id BIGINT NOT NULL,
    reason TEXT NOT NULL,
    expires_at TIMESTAMP,
    created_by BIGINT NOT NULL,
    revoked_by BIGINT,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT now() NOT NULL,
    CONSTRAINT factory_commission_exemptions_pkey PRIMARY KEY (exemption_id)
);

CREATE TABLE IF NOT EXISTS factory_rfq_dismissals (
    factory_id BIGINT NOT NULL,
    rfq_id     BIGINT NOT NULL,
    dismissed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT factory_rfq_dismissals_pkey PRIMARY KEY (factory_id, rfq_id)
);

CREATE TABLE IF NOT EXISTS factory_profiles (
    user_id BIGINT NOT NULL,
    approval_status CHAR(2) NOT NULL,
    config_id BIGINT,
    factory_name VARCHAR(150) NOT NULL,
    description TEXT,
    tax_id VARCHAR(20),
    lead_time_desc VARCHAR(50),
    province_id BIGINT,
    image_url TEXT,
    background_image_url TEXT,
    rating NUMERIC(3,2),
    review_count INTEGER NOT NULL,
    completed_orders INTEGER NOT NULL,
    submitted_at TIMESTAMP,
    verified_at TIMESTAMP,
    verified_by BIGINT,
    rejection_reason TEXT,
    CONSTRAINT factory_profiles_pkey PRIMARY KEY (user_id)
);

CREATE TABLE IF NOT EXISTS factory_reviews (
    review_id BIGSERIAL NOT NULL,
    factory_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    order_id BIGINT,
    rating NUMERIC(5,2) NOT NULL,
    comment TEXT,
    image_urls JSONB DEFAULT '[]'::jsonb NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    factory_reply TEXT,
    factory_reply_at TIMESTAMP,
    factory_reply_by BIGINT,
    deleted_at TIMESTAMP,
    CONSTRAINT factory_reviews_pkey PRIMARY KEY (review_id)
);

CREATE TABLE IF NOT EXISTS factory_showcases (
    showcase_id BIGSERIAL NOT NULL,
    factory_id INTEGER NOT NULL,
    category_id INTEGER,
    sub_category_id BIGINT,
    content_type CHAR(2) NOT NULL,
    title VARCHAR(200) NOT NULL,
    content TEXT,
    linked_showcases JSONB DEFAULT '[]'::jsonb NOT NULL,
    moq INTEGER,
    unit_id INTEGER,
    lead_time_days INTEGER,
    base_price NUMERIC(12,2),
    promo_price NUMERIC(12,2),
    start_date DATE,
    end_date DATE,
    status CHAR(2) DEFAULT 'AC' NOT NULL,
    likes_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    published_at TIMESTAMP,
    CONSTRAINT factory_showcases_pkey PRIMARY KEY (showcase_id)
);

CREATE TABLE IF NOT EXISTS favorites (
    fav_id BIGSERIAL NOT NULL,
    user_id INTEGER NOT NULL,
    showcase_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT favorites_pkey PRIMARY KEY (fav_id)
);

CREATE TABLE IF NOT EXISTS lbi_certificates (
    cert_id BIGSERIAL NOT NULL,
    cert_name VARCHAR(150) NOT NULL,
    description TEXT,
    status CHAR(1) DEFAULT '1' NOT NULL,
    CONSTRAINT lbi_certificates_pkey PRIMARY KEY (cert_id)
);

CREATE TABLE IF NOT EXISTS lbi_districts (
    row_id BIGSERIAL NOT NULL,
    province_id BIGINT NOT NULL,
    name_th VARCHAR(150) NOT NULL,
    name_en VARCHAR(150) NOT NULL,
    status CHAR(1) DEFAULT '1' NOT NULL,
    CONSTRAINT lbi_districts_pkey PRIMARY KEY (row_id)
);

CREATE TABLE IF NOT EXISTS lbi_sub_districts (
    row_id BIGSERIAL NOT NULL,
    district_id BIGINT NOT NULL,
    name_th VARCHAR(150) NOT NULL,
    name_en VARCHAR(150) NOT NULL,
    zip_code VARCHAR(10) NOT NULL,
    status BPCHAR(1) DEFAULT '1' NOT NULL,
    CONSTRAINT lbi_sub_districts_pkey PRIMARY KEY (row_id)
);

CREATE TABLE IF NOT EXISTS lbi_factory_types (
    factory_type_id BIGSERIAL NOT NULL,
    type_name VARCHAR(150) NOT NULL,
    status CHAR(1) DEFAULT '1' NOT NULL,
    CONSTRAINT lbi_factory_types_pkey PRIMARY KEY (factory_type_id)
);

CREATE TABLE IF NOT EXISTS lbi_production (
    step_id BIGSERIAL NOT NULL,
    step_name VARCHAR(150) NOT NULL,
    step_name_th VARCHAR(150),
    description TEXT,
    sort_order INTEGER,
    CONSTRAINT lbi_production_pkey PRIMARY KEY (step_id)
);

CREATE TABLE IF NOT EXISTS lbi_provinces (
    row_id BIGSERIAL NOT NULL,
    name_th VARCHAR(150) NOT NULL,
    name_en VARCHAR(150) NOT NULL,
    status CHAR(1) DEFAULT '1' NOT NULL,
    CONSTRAINT lbi_provinces_pkey PRIMARY KEY (row_id)
);

CREATE TABLE IF NOT EXISTS lbi_shipping_methods (
    shipping_method_id BIGSERIAL NOT NULL,
    method_name VARCHAR(100) NOT NULL,
    status CHAR(1) DEFAULT '1' NOT NULL,
    CONSTRAINT lbi_shipping_methods_pkey PRIMARY KEY (shipping_method_id)
);

CREATE TABLE IF NOT EXISTS map_factory_categories (
    map_id BIGSERIAL NOT NULL,
    factory_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    CONSTRAINT map_factory_categories_pkey PRIMARY KEY (map_id)
);

CREATE TABLE IF NOT EXISTS map_factory_certificates (
    map_id BIGSERIAL NOT NULL,
    factory_id INTEGER NOT NULL,
    cert_id INTEGER NOT NULL,
    document_url TEXT NOT NULL,
    expire_date DATE,
    cert_number VARCHAR(100),
    verify_status CHAR(2),
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT map_factory_certificates_pkey PRIMARY KEY (map_id)
);

CREATE TABLE IF NOT EXISTS map_factory_sub_categories (
    factory_id BIGINT NOT NULL,
    sub_category_id BIGINT NOT NULL,
    CONSTRAINT map_factory_sub_categories_pkey PRIMARY KEY (factory_id, sub_category_id)
);

CREATE TABLE IF NOT EXISTS messages (
    message_id BIGSERIAL NOT NULL,
    conv_id INTEGER,
    sender_id BIGINT NOT NULL,
    receiver_id BIGINT NOT NULL,
    message_type VARCHAR(20) DEFAULT 'TX',
    reference_id BIGINT,
    content TEXT NOT NULL,
    quote_data JSONB,
    attachment_url TEXT,
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    CONSTRAINT messages_pkey PRIMARY KEY (message_id)
);

CREATE TABLE IF NOT EXISTS notifications (
    noti_id BIGSERIAL NOT NULL,
    user_id INTEGER NOT NULL,
    type VARCHAR(50) NOT NULL,
    title VARCHAR(150) NOT NULL,
    message TEXT,
    link_to VARCHAR(255),
    is_read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    CONSTRAINT notifications_pkey PRIMARY KEY (noti_id)
);

CREATE TABLE IF NOT EXISTS order_activity_log (
    activity_id BIGSERIAL NOT NULL,
    order_id BIGINT NOT NULL,
    actor_user_id BIGINT,
    event_code VARCHAR(32) NOT NULL,
    payload JSONB,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    CONSTRAINT order_activity_log_pkey PRIMARY KEY (activity_id)
);

CREATE TABLE IF NOT EXISTS orders (
    order_id BIGSERIAL NOT NULL,
    quote_id BIGINT NOT NULL,
    customer_id BIGINT NOT NULL,
    factory_id BIGINT NOT NULL,
    status CHAR(2) DEFAULT 'PR' NOT NULL,
    total_amount NUMERIC(10,2) NOT NULL,
    deposit_amount NUMERIC(10,2) NOT NULL,
    payment_type VARCHAR(10),
    estimated_delivery DATE,
    tracking_no VARCHAR(120),
    courier VARCHAR(120),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT orders_pkey PRIMARY KEY (order_id)
);

CREATE TABLE IF NOT EXISTS payment_schedules (
    schedule_id BIGSERIAL NOT NULL,
    order_id BIGINT NOT NULL,
    installment_no INTEGER NOT NULL,
    due_date DATE NOT NULL,
    amount NUMERIC(15,2) NOT NULL,
    status VARCHAR(10) DEFAULT 'PD' NOT NULL,
    paid_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT now() NOT NULL,
    CONSTRAINT payment_schedules_pkey PRIMARY KEY (schedule_id)
);

CREATE TABLE IF NOT EXISTS platform_config (
    config_id BIGSERIAL NOT NULL,
    label VARCHAR(100),
    default_commission_rate NUMERIC(5,2) NOT NULL,
    vat_rate NUMERIC(5,2) NOT NULL,
    effective_from TIMESTAMP DEFAULT now() NOT NULL,
    effective_to TIMESTAMP,
    created_by BIGINT,
    created_at TIMESTAMP DEFAULT now() NOT NULL,
    CONSTRAINT platform_config_pkey PRIMARY KEY (config_id)
);

CREATE TABLE IF NOT EXISTS tconfig (
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    CONSTRAINT tconfig_pkey PRIMARY KEY (key)
);

INSERT INTO tconfig (key, value)
VALUES
    ('shipping_days', '7'),
    ('rfq_expired', '30')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

CREATE TABLE IF NOT EXISTS production_updates (
    update_id BIGSERIAL NOT NULL,
    order_id BIGINT NOT NULL,
    step_id BIGINT NOT NULL,
    status CHAR(2) DEFAULT 'CR' NOT NULL,
    description TEXT,
    image_urls JSONB DEFAULT '[]'::jsonb NOT NULL,
    updated_by_user_id BIGINT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    last_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT production_updates_pkey PRIMARY KEY (update_id),
    CONSTRAINT uq_production_updates_order_step UNIQUE (order_id, step_id)
);

CREATE TABLE IF NOT EXISTS quotation_history (
    history_id BIGSERIAL NOT NULL,
    quote_id BIGINT NOT NULL,
    event_type VARCHAR(8) NOT NULL,
    status CHAR(2),
    version_after INTEGER NOT NULL,
    grand_total NUMERIC(15,2) DEFAULT 0 NOT NULL,
    price_per_piece NUMERIC(12,2),
    mold_cost NUMERIC(12,2),
    lead_time_days INTEGER,
    shipping_method_id BIGINT,
    reason TEXT,
    edited_by BIGINT,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    CONSTRAINT quotation_history_pkey PRIMARY KEY (history_id)
);

CREATE TABLE IF NOT EXISTS quotations (
    quote_id BIGSERIAL NOT NULL,
    rfq_id BIGINT NOT NULL,
    factory_id BIGINT NOT NULL,
    status CHAR(2) DEFAULT 'PD' NOT NULL,
    factory_highlight VARCHAR(200),
    image_urls JSONB DEFAULT '[]'::jsonb NOT NULL,
    subtotal NUMERIC(15,2) DEFAULT 0 NOT NULL,
    price_per_piece NUMERIC(10,2) NOT NULL,
    mold_cost NUMERIC(10,2) DEFAULT 0 NOT NULL,
    packaging_cost NUMERIC(15,2) DEFAULT 0 NOT NULL,
    shipping_cost NUMERIC(15,2) DEFAULT 0 NOT NULL,
    vat_rate NUMERIC(5,2) DEFAULT 7 NOT NULL,
    vat_amount NUMERIC(15,2) DEFAULT 0 NOT NULL,
    platform_commission_rate NUMERIC(5,2) DEFAULT 5 NOT NULL,
    platform_commission_amount NUMERIC(15,2) DEFAULT 0 NOT NULL,
    discount_amount NUMERIC(15,2) DEFAULT 0 NOT NULL,
    platform_config_id BIGINT,
    grand_total NUMERIC(15,2) DEFAULT 0 NOT NULL,
    factory_net_receivable NUMERIC(15,2) DEFAULT 0 NOT NULL,
    lead_time_days INTEGER NOT NULL,
    shipping_method_id BIGINT NOT NULL,
    payment_terms VARCHAR(20),
    validity_days INTEGER DEFAULT 30 NOT NULL,
    valid_until DATE,
    version INTEGER DEFAULT 1 NOT NULL,
    is_locked BOOLEAN DEFAULT FALSE NOT NULL,
    factory_note TEXT,
    create_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    log_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT quotations_pkey PRIMARY KEY (quote_id)
);

CREATE TABLE IF NOT EXISTS rfqs (
    rfq_id BIGSERIAL NOT NULL,
    user_id BIGINT NOT NULL,
    category_id BIGINT,
    sub_category_id BIGINT,
    request_kind CHAR(2) DEFAULT 'PR' NOT NULL,
    status CHAR(2) DEFAULT 'OP' NOT NULL,
    title VARCHAR(100) NOT NULL,
    details TEXT,
    quantity BIGINT NOT NULL,
    target_price NUMERIC(15,2),
    reference_images TEXT[] DEFAULT ARRAY[]::TEXT[] NOT NULL,
    material_grade VARCHAR(120),
    certifications_required TEXT[] DEFAULT ARRAY[]::TEXT[] NOT NULL,
    target_lead_time_days INTEGER,
    delivery_address_id BIGINT,
    shipping_method_id BIGINT,
    expired_date TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT rfqs_pkey PRIMARY KEY (rfq_id)
);

CREATE TABLE IF NOT EXISTS schema_migrations (
    filename TEXT NOT NULL,
    applied_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    CONSTRAINT schema_migrations_pkey PRIMARY KEY (filename)
);

CREATE TABLE IF NOT EXISTS domain_events (
    event_id BIGSERIAL NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT domain_events_pkey PRIMARY KEY (event_id)
);

CREATE INDEX IF NOT EXISTS idx_domain_events_type ON domain_events(event_type);
CREATE INDEX IF NOT EXISTS idx_domain_events_created_at ON domain_events(created_at);

CREATE TABLE IF NOT EXISTS transactions (
    tx_id BIGSERIAL NOT NULL,
    wallet_id BIGINT NOT NULL,
    order_id BIGINT,
    type CHAR(2) NOT NULL,
    status CHAR(2) NOT NULL,
    amount NUMERIC(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT transactions_pkey PRIMARY KEY (tx_id)
);

CREATE TABLE IF NOT EXISTS users (
    user_id BIGSERIAL NOT NULL,
    role CHAR(2) NOT NULL,
    email VARCHAR(100) NOT NULL,
    phone VARCHAR(20),
    password_hash TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT users_pkey PRIMARY KEY (user_id)
);

CREATE TABLE IF NOT EXISTS wallets (
    wallet_id BIGSERIAL NOT NULL,
    user_id BIGINT NOT NULL,
    good_fund NUMERIC(10,2) DEFAULT 0 NOT NULL,
    pending_fund NUMERIC(10,2) DEFAULT 0 NOT NULL,
    CONSTRAINT wallets_pkey PRIMARY KEY (wallet_id)
);

ALTER TABLE addresses
    ADD CONSTRAINT fk_addresses_user_id_users_user_id
    FOREIGN KEY (user_id) REFERENCES users(user_id);

ALTER TABLE admin_audit_log
    ADD CONSTRAINT fk_admin_audit_log_actor_id_users_user_id
    FOREIGN KEY (actor_id) REFERENCES users(user_id);

ALTER TABLE admin_profiles
    ADD CONSTRAINT fk_admin_profiles_user_id_users_user_id
    FOREIGN KEY (user_id) REFERENCES users(user_id);

ALTER TABLE admin_profiles
    ADD CONSTRAINT fk_admin_profiles_created_by_users_user_id
    FOREIGN KEY (created_by) REFERENCES users(user_id);

ALTER TABLE customers
    ADD CONSTRAINT fk_customers_user_id_users_user_id
    FOREIGN KEY (user_id) REFERENCES users(user_id);

ALTER TABLE factory_commission_exemptions
    ADD CONSTRAINT fk_factory_commission_exemptions_factory_id_users_user_id
    FOREIGN KEY (factory_id) REFERENCES users(user_id);

ALTER TABLE factory_commission_exemptions
    ADD CONSTRAINT fk_factory_commission_exemptions_created_by_users_user_id
    FOREIGN KEY (created_by) REFERENCES users(user_id);

ALTER TABLE factory_commission_exemptions
    ADD CONSTRAINT fk_factory_commission_exemptions_revoked_by_users_user_id
    FOREIGN KEY (revoked_by) REFERENCES users(user_id);

ALTER TABLE factory_profiles
    ADD CONSTRAINT fk_factory_profiles_user_id_users_user_id
    FOREIGN KEY (user_id) REFERENCES users(user_id);

ALTER TABLE factory_profiles
    ADD CONSTRAINT fk_factory_profiles_config_id_platform_config_config_id
    FOREIGN KEY (config_id) REFERENCES platform_config(config_id);

ALTER TABLE factory_profiles
    ADD CONSTRAINT fk_factory_profiles_province_id_lbi_provinces_row_id
    FOREIGN KEY (province_id) REFERENCES lbi_provinces(row_id);

ALTER TABLE factory_profiles
    ADD CONSTRAINT fk_factory_profiles_verified_by_users_user_id
    FOREIGN KEY (verified_by) REFERENCES users(user_id);

ALTER TABLE factory_reviews
    ADD CONSTRAINT fk_factory_reviews_order_id_orders_order_id
    FOREIGN KEY (order_id) REFERENCES orders(order_id);

ALTER TABLE factory_reviews
    ADD CONSTRAINT fk_factory_reviews_factory_reply_by_users_user_id
    FOREIGN KEY (factory_reply_by) REFERENCES users(user_id);

ALTER TABLE lbi_districts
    ADD CONSTRAINT fk_lbi_districts_province_id_lbi_provinces_row_id
    FOREIGN KEY (province_id) REFERENCES lbi_provinces(row_id);

ALTER TABLE lbi_sub_categories
    ADD CONSTRAINT fk_lbi_sub_categories_category_id_lbi_categories_category_id
    FOREIGN KEY (category_id) REFERENCES lbi_categories(category_id);

ALTER TABLE lbi_sub_districts
    ADD CONSTRAINT fk_lbi_sub_districts_district_id_lbi_districts_row_id
    FOREIGN KEY (district_id) REFERENCES lbi_districts(row_id);

ALTER TABLE map_factory_categories
    ADD CONSTRAINT fk_map_factory_categories_factory_id_users_user_id
    FOREIGN KEY (factory_id) REFERENCES users(user_id);

ALTER TABLE map_factory_categories
    ADD CONSTRAINT fk_map_factory_categories_category_id_lbi_categories_category_id
    FOREIGN KEY (category_id) REFERENCES lbi_categories(category_id);

ALTER TABLE map_factory_certificates
    ADD CONSTRAINT fk_map_factory_certificates_cert_id_lbi_certificates_cert_id
    FOREIGN KEY (cert_id) REFERENCES lbi_certificates(cert_id);

ALTER TABLE map_factory_sub_categories
    ADD CONSTRAINT fk_map_factory_sub_categories_factory_id_users_user_id
    FOREIGN KEY (factory_id) REFERENCES users(user_id);

ALTER TABLE messages
    ADD CONSTRAINT fk_messages_sender_id_users_user_id
    FOREIGN KEY (sender_id) REFERENCES users(user_id);

ALTER TABLE messages
    ADD CONSTRAINT fk_messages_receiver_id_users_user_id
    FOREIGN KEY (receiver_id) REFERENCES users(user_id);

ALTER TABLE order_activity_log
    ADD CONSTRAINT fk_order_activity_log_order_id_orders_order_id
    FOREIGN KEY (order_id) REFERENCES orders(order_id);

ALTER TABLE order_activity_log
    ADD CONSTRAINT fk_order_activity_log_actor_user_id_users_user_id
    FOREIGN KEY (actor_user_id) REFERENCES users(user_id);

ALTER TABLE orders
    ADD CONSTRAINT fk_orders_quote_id_quotations_quote_id
    FOREIGN KEY (quote_id) REFERENCES quotations(quote_id);

ALTER TABLE orders
    ADD CONSTRAINT fk_orders_customer_id_users_user_id
    FOREIGN KEY (customer_id) REFERENCES users(user_id);

ALTER TABLE orders
    ADD CONSTRAINT fk_orders_factory_id_users_user_id
    FOREIGN KEY (factory_id) REFERENCES users(user_id);

ALTER TABLE payment_schedules
    ADD CONSTRAINT fk_payment_schedules_order_id_orders_order_id
    FOREIGN KEY (order_id) REFERENCES orders(order_id);

ALTER TABLE platform_config
    ADD CONSTRAINT fk_platform_config_created_by_users_user_id
    FOREIGN KEY (created_by) REFERENCES users(user_id);

ALTER TABLE production_updates
    ADD CONSTRAINT fk_production_updates_order_id_orders_order_id
    FOREIGN KEY (order_id) REFERENCES orders(order_id);

ALTER TABLE production_updates
    ADD CONSTRAINT fk_production_updates_step_id_lbi_production_step_id
    FOREIGN KEY (step_id) REFERENCES lbi_production(step_id);

ALTER TABLE production_updates
    ADD CONSTRAINT fk_production_updates_updated_by_user_id_users_user_id
    FOREIGN KEY (updated_by_user_id) REFERENCES users(user_id);

ALTER TABLE quotation_history
    ADD CONSTRAINT fk_quotation_history_quote_id_quotations_quote_id
    FOREIGN KEY (quote_id) REFERENCES quotations(quote_id);

ALTER TABLE quotation_history
    ADD CONSTRAINT fk_quotation_history_edited_by_users_user_id
    FOREIGN KEY (edited_by) REFERENCES users(user_id);

ALTER TABLE quotations
    ADD CONSTRAINT fk_quotations_rfq_id_rfqs_rfq_id
    FOREIGN KEY (rfq_id) REFERENCES rfqs(rfq_id);

ALTER TABLE quotations
    ADD CONSTRAINT fk_quotations_factory_id_users_user_id
    FOREIGN KEY (factory_id) REFERENCES users(user_id);

ALTER TABLE quotations
    ADD CONSTRAINT fk_quotations_platform_config_id_platform_config_config_id
    FOREIGN KEY (platform_config_id) REFERENCES platform_config(config_id);

ALTER TABLE quotations
    ADD CONSTRAINT fk_quotations_shipping_method_id_lbi_shipping_methods_shipping_method_id
    FOREIGN KEY (shipping_method_id) REFERENCES lbi_shipping_methods(shipping_method_id);

ALTER TABLE rfqs
    ADD CONSTRAINT fk_rfqs_user_id_users_user_id
    FOREIGN KEY (user_id) REFERENCES users(user_id);

ALTER TABLE rfqs
    ADD CONSTRAINT fk_rfqs_category_id_lbi_categories_category_id
    FOREIGN KEY (category_id) REFERENCES lbi_categories(category_id);

ALTER TABLE rfqs
    ADD CONSTRAINT fk_rfqs_delivery_address_id_addresses_address_id
    FOREIGN KEY (delivery_address_id) REFERENCES addresses(address_id);

ALTER TABLE rfqs
    ADD CONSTRAINT fk_rfqs_shipping_method_id_lbi_shipping_methods_shipping_method_id
    FOREIGN KEY (shipping_method_id) REFERENCES lbi_shipping_methods(shipping_method_id);

ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_wallet_id_wallets_wallet_id
    FOREIGN KEY (wallet_id) REFERENCES wallets(wallet_id);

ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_order_id_orders_order_id
    FOREIGN KEY (order_id) REFERENCES orders(order_id);

ALTER TABLE wallets
    ADD CONSTRAINT fk_wallets_user_id_users_user_id
    FOREIGN KEY (user_id) REFERENCES users(user_id);

CREATE INDEX IF NOT EXISTS idx_addresses_user_id ON addresses(user_id);
CREATE INDEX IF NOT EXISTS idx_admin_audit_log_actor_id ON admin_audit_log(actor_id);
CREATE INDEX IF NOT EXISTS idx_admin_profiles_user_id ON admin_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_admin_profiles_created_by ON admin_profiles(created_by);
CREATE INDEX IF NOT EXISTS idx_customers_user_id ON customers(user_id);
CREATE INDEX IF NOT EXISTS idx_factory_commission_exemptions_factory_id ON factory_commission_exemptions(factory_id);
CREATE INDEX IF NOT EXISTS idx_factory_commission_exemptions_created_by ON factory_commission_exemptions(created_by);
CREATE INDEX IF NOT EXISTS idx_factory_commission_exemptions_revoked_by ON factory_commission_exemptions(revoked_by);
CREATE INDEX IF NOT EXISTS idx_factory_profiles_user_id ON factory_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_factory_profiles_config_id ON factory_profiles(config_id);
CREATE INDEX IF NOT EXISTS idx_factory_profiles_province_id ON factory_profiles(province_id);
CREATE INDEX IF NOT EXISTS idx_factory_profiles_verified_by ON factory_profiles(verified_by);
CREATE INDEX IF NOT EXISTS idx_factory_reviews_order_id ON factory_reviews(order_id);
CREATE INDEX IF NOT EXISTS idx_factory_reviews_factory_reply_by ON factory_reviews(factory_reply_by);
CREATE INDEX IF NOT EXISTS idx_lbi_districts_province_id ON lbi_districts(province_id);
CREATE INDEX IF NOT EXISTS idx_lbi_sub_categories_category_id ON lbi_sub_categories(category_id);
CREATE INDEX IF NOT EXISTS idx_lbi_sub_districts_district_id ON lbi_sub_districts(district_id);
CREATE INDEX IF NOT EXISTS idx_map_factory_categories_factory_id ON map_factory_categories(factory_id);
CREATE INDEX IF NOT EXISTS idx_map_factory_categories_category_id ON map_factory_categories(category_id);
CREATE INDEX IF NOT EXISTS idx_map_factory_certificates_cert_id ON map_factory_certificates(cert_id);
CREATE INDEX IF NOT EXISTS idx_map_factory_sub_categories_factory_id ON map_factory_sub_categories(factory_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);
CREATE INDEX IF NOT EXISTS idx_messages_receiver_id ON messages(receiver_id);
CREATE INDEX IF NOT EXISTS idx_order_activity_log_order_id ON order_activity_log(order_id);
CREATE INDEX IF NOT EXISTS idx_order_activity_log_actor_user_id ON order_activity_log(actor_user_id);
CREATE INDEX IF NOT EXISTS idx_orders_quote_id ON orders(quote_id);
CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_factory_id ON orders(factory_id);
CREATE INDEX IF NOT EXISTS idx_payment_schedules_order_id ON payment_schedules(order_id);
CREATE INDEX IF NOT EXISTS idx_platform_config_created_by ON platform_config(created_by);
CREATE INDEX IF NOT EXISTS idx_production_updates_order_id ON production_updates(order_id);
CREATE INDEX IF NOT EXISTS idx_production_updates_step_id ON production_updates(step_id);
CREATE INDEX IF NOT EXISTS idx_production_updates_updated_by_user_id ON production_updates(updated_by_user_id);
CREATE INDEX IF NOT EXISTS idx_quotation_history_quote_id ON quotation_history(quote_id);
CREATE INDEX IF NOT EXISTS idx_quotation_history_edited_by ON quotation_history(edited_by);
CREATE INDEX IF NOT EXISTS idx_quotations_rfq_id ON quotations(rfq_id);
CREATE INDEX IF NOT EXISTS idx_quotations_factory_id ON quotations(factory_id);
CREATE INDEX IF NOT EXISTS idx_quotations_platform_config_id ON quotations(platform_config_id);
CREATE INDEX IF NOT EXISTS idx_quotations_shipping_method_id ON quotations(shipping_method_id);
CREATE INDEX IF NOT EXISTS idx_rfqs_user_id ON rfqs(user_id);
CREATE INDEX IF NOT EXISTS idx_rfqs_category_id ON rfqs(category_id);
CREATE INDEX IF NOT EXISTS idx_rfqs_delivery_address_id ON rfqs(delivery_address_id);
CREATE INDEX IF NOT EXISTS idx_rfqs_shipping_method_id ON rfqs(shipping_method_id);
CREATE INDEX IF NOT EXISTS idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX IF NOT EXISTS idx_transactions_order_id ON transactions(order_id);
CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email);
CREATE INDEX IF NOT EXISTS idx_rfqs_user_status ON rfqs(user_id, status);
CREATE INDEX IF NOT EXISTS idx_quotations_rfq_factory ON quotations(rfq_id, factory_id);
CREATE INDEX IF NOT EXISTS idx_orders_customer_status ON orders(customer_id, status);
CREATE INDEX IF NOT EXISTS idx_orders_factory_status ON orders(factory_id, status);
CREATE INDEX IF NOT EXISTS idx_messages_conv_created ON messages(conv_id, created_at);
CREATE INDEX IF NOT EXISTS idx_factory_showcases_factory ON factory_showcases(factory_id);

-- FK targets omitted because the target tables are not present in Final_db!!!:
-- - factory_showcases.sub_category_id -> public.lbi_sub_categories(sub_category_id)
-- - map_factory_sub_categories.sub_category_id -> public.lbi_sub_categories(sub_category_id)
-- - rfqs.sub_category_id -> public.lbi_sub_categories(sub_category_id)

-- ═══════════════════════════════════════════════════════════════
-- Reference seed: required before 002_category_masterdata.
-- Address + sub-categories are loaded by 002 / 003 — not seeded here.
-- ═══════════════════════════════════════════════════════════════

INSERT INTO lbi_factory_types (type_name, status) VALUES
  ('โรงงานอาหารสัตว์เลี้ยง',         '1'),
  ('โรงงานขนมสัตว์เลี้ยง',           '1'),
  ('โรงงานอาหารเสริมสัตว์เลี้ยง',    '1'),
  ('โรงงานบรรจุภัณฑ์อาหารสัตว์',     '1'),
  ('โรงงานผลิตของเล่นสัตว์เลี้ยง',   '1'),
  ('โรงงานผลิตเสื้อผ้าสัตว์เลี้ยง',  '1'),
  ('โรงงานผลิตวัคซีน/ยาสัตว์',       '1'),
  ('โรงงานผลิตอุปกรณ์สัตว์เลี้ยง',   '1'),
  ('โรงงานแปรรูปวัตถุดิบ',           '1'),
  ('อื่นๆ',                           '1');

INSERT INTO lbi_certificates (cert_name, description, status) VALUES
  ('GMP',        'Good Manufacturing Practice — มาตรฐานการผลิตที่ดี',                                        '1'),
  ('HACCP',      'Hazard Analysis Critical Control Points — ระบบวิเคราะห์อันตรายและจุดวิกฤต',               '1'),
  ('ISO 9001',   'ระบบบริหารคุณภาพ',                                                                         '1'),
  ('ISO 22000',  'ระบบการจัดการความปลอดภัยของอาหาร',                                                         '1'),
  ('Halal',      'ใบรับรองฮาลาล — มาตรฐานอาหารสำหรับผู้นับถือศาสนาอิสลาม',                                  '1'),
  ('อย.',        'สำนักงานคณะกรรมการอาหารและยา (FDA Thailand)',                                              '1'),
  ('มกษ.',       'มาตรฐานสินค้าเกษตร — กรมวิชาการเกษตร',                                                     '1'),
  ('FAMI-QS',    'Feed Additives & Premixtures Quality System',                                              '1'),
  ('BRC',        'British Retail Consortium Global Standard for Food Safety',                                '1'),
  ('GHP',        'Good Hygiene Practice — หลักปฏิบัติด้านสุขลักษณะที่ดี',                                    '1');

INSERT INTO lbi_shipping_methods (method_name, status) VALUES
  ('ลูกค้ารับเองที่โรงงาน',  '1'),
  ('จัดส่งเดลิเวอร์รี่',     '1');

INSERT INTO lbi_production (step_id, step_name, step_name_th, description, sort_order) VALUES
  (0, 'Accept Order',          'ยืนยันรับงาน',         'โรงงานยืนยันรับงานและเริ่มกระบวนการผลิต',                           0)
ON CONFLICT (step_id) DO NOTHING;

INSERT INTO lbi_production (step_name, step_name_th, description, sort_order) VALUES
  ('Material Preparation',    'จัดเตรียมวัตถุดิบ',    'ตรวจรับและเตรียมวัตถุดิบก่อนเริ่มการผลิต',                          1),
  ('Processing / Production', 'ขั้นตอนการผลิต',       'กระบวนการผลิตหลัก เช่น อัดเม็ด อบ ฟรีซดราย ฯลฯ',                  2),
  ('Quality Control (QC)',    'ตรวจสอบคุณภาพ',         'ตรวจสอบคุณภาพสินค้าก่อนบรรจุ ทดสอบค่าโภชนาการและความปลอดภัย',    3),
  ('Ship Order',              'จัดส่งแล้ว',           'จัดส่งสินค้าไปยังลูกค้า',                                            4),
  ('Delivery Confirmed',      'จัดส่งสำเร็จ',          'รอลูกค้ายืนยันรับสินค้า หรือระบบปิดอัตโนมัติหลัง 14 วัน',           5);

-- Stubs for 002_category_masterdata (IDs 1–16; 002 renames and replaces sub-categories).
INSERT INTO lbi_categories (name, scope) VALUES
  ('อาหารเม็ดสัตว์เลี้ยง',          'PD'),
  ('อาหารเปียก / Wet Food',         'PD'),
  ('ขนมสัตว์เลี้ยง',                'PD'),
  ('อาหารฟรีซดราย',                 'PD'),
  ('ท้อปเปอร์ / โรยหน้า',           'PD'),
  ('อาหารเสริมและวิตามิน',          'PD'),
  ('บรรจุภัณฑ์สำเร็จรูป',          'PD'),
  ('ของเล่นสัตว์เลี้ยง',           'PD'),
  ('เสื้อผ้าและเครื่องแต่งกาย',    'PD'),
  ('อุปกรณ์และของใช้',              'PD'),
  ('วัตถุดิบโปรตีนสัตว์',           'MT'),
  ('วัตถุดิบโปรตีนพืช',             'MT'),
  ('วิตามินและแร่ธาตุ',             'MT'),
  ('วัตถุเจือปนอาหารและสารกันเสีย', 'MT'),
  ('วัตถุดิบเส้นใย / ธัญพืช',       'MT'),
  ('วัตถุดิบบรรจุภัณฑ์',            'MT');

SELECT setval(pg_get_serial_sequence('lbi_categories', 'category_id'), GREATEST((SELECT MAX(category_id) FROM lbi_categories), 16));
SELECT setval(pg_get_serial_sequence('lbi_factory_types', 'factory_type_id'), GREATEST((SELECT MAX(factory_type_id) FROM lbi_factory_types), 10));

DELETE FROM schema_migrations;

COMMIT;

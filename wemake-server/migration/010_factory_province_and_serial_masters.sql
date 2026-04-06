-- factory_profiles: store province as lbi_provinces.row_id (drop legacy free-text location)
ALTER TABLE factory_profiles DROP COLUMN IF EXISTS location;

ALTER TABLE factory_profiles
    ADD COLUMN IF NOT EXISTS province_id BIGINT NULL REFERENCES lbi_provinces(row_id);

-- Legacy DBs: master tables had BIGINT PK without DEFAULT — align with BIGSERIAL behavior (idempotent)
DO $$
BEGIN
  IF to_regclass('public.lbi_provinces') IS NOT NULL AND pg_get_serial_sequence('lbi_provinces', 'row_id') IS NULL THEN
    CREATE SEQUENCE lbi_provinces_row_id_seq;
    PERFORM setval('lbi_provinces_row_id_seq', COALESCE((SELECT MAX(row_id) FROM lbi_provinces), 1), true);
    ALTER TABLE lbi_provinces ALTER COLUMN row_id SET DEFAULT nextval('lbi_provinces_row_id_seq');
    ALTER SEQUENCE lbi_provinces_row_id_seq OWNED BY lbi_provinces.row_id;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('public.lbi_districts') IS NOT NULL AND pg_get_serial_sequence('lbi_districts', 'row_id') IS NULL THEN
    CREATE SEQUENCE lbi_districts_row_id_seq;
    PERFORM setval('lbi_districts_row_id_seq', COALESCE((SELECT MAX(row_id) FROM lbi_districts), 1), true);
    ALTER TABLE lbi_districts ALTER COLUMN row_id SET DEFAULT nextval('lbi_districts_row_id_seq');
    ALTER SEQUENCE lbi_districts_row_id_seq OWNED BY lbi_districts.row_id;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('public.lbi_sub_districts') IS NOT NULL AND pg_get_serial_sequence('lbi_sub_districts', 'row_id') IS NULL THEN
    CREATE SEQUENCE lbi_sub_districts_row_id_seq;
    PERFORM setval('lbi_sub_districts_row_id_seq', COALESCE((SELECT MAX(row_id) FROM lbi_sub_districts), 1), true);
    ALTER TABLE lbi_sub_districts ALTER COLUMN row_id SET DEFAULT nextval('lbi_sub_districts_row_id_seq');
    ALTER SEQUENCE lbi_sub_districts_row_id_seq OWNED BY lbi_sub_districts.row_id;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('public.lbi_factory_types') IS NOT NULL AND pg_get_serial_sequence('lbi_factory_types', 'factory_type_id') IS NULL THEN
    CREATE SEQUENCE lbi_factory_types_factory_type_id_seq;
    PERFORM setval('lbi_factory_types_factory_type_id_seq', COALESCE((SELECT MAX(factory_type_id) FROM lbi_factory_types), 1), true);
    ALTER TABLE lbi_factory_types ALTER COLUMN factory_type_id SET DEFAULT nextval('lbi_factory_types_factory_type_id_seq');
    ALTER SEQUENCE lbi_factory_types_factory_type_id_seq OWNED BY lbi_factory_types.factory_type_id;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('public.lbi_product_categories') IS NOT NULL AND pg_get_serial_sequence('lbi_product_categories', 'category_id') IS NULL THEN
    CREATE SEQUENCE lbi_product_categories_category_id_seq;
    PERFORM setval('lbi_product_categories_category_id_seq', COALESCE((SELECT MAX(category_id) FROM lbi_product_categories), 1), true);
    ALTER TABLE lbi_product_categories ALTER COLUMN category_id SET DEFAULT nextval('lbi_product_categories_category_id_seq');
    ALTER SEQUENCE lbi_product_categories_category_id_seq OWNED BY lbi_product_categories.category_id;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('public.lbi_production') IS NOT NULL AND pg_get_serial_sequence('lbi_production', 'step_id') IS NULL THEN
    CREATE SEQUENCE lbi_production_step_id_seq;
    PERFORM setval('lbi_production_step_id_seq', COALESCE((SELECT MAX(step_id) FROM lbi_production), 1), true);
    ALTER TABLE lbi_production ALTER COLUMN step_id SET DEFAULT nextval('lbi_production_step_id_seq');
    ALTER SEQUENCE lbi_production_step_id_seq OWNED BY lbi_production.step_id;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('public.lbi_units') IS NOT NULL AND pg_get_serial_sequence('lbi_units', 'unit_id') IS NULL THEN
    CREATE SEQUENCE lbi_units_unit_id_seq;
    PERFORM setval('lbi_units_unit_id_seq', COALESCE((SELECT MAX(unit_id) FROM lbi_units), 1), true);
    ALTER TABLE lbi_units ALTER COLUMN unit_id SET DEFAULT nextval('lbi_units_unit_id_seq');
    ALTER SEQUENCE lbi_units_unit_id_seq OWNED BY lbi_units.unit_id;
  END IF;
END $$;

DO $$
BEGIN
  IF to_regclass('public.lbi_shipping_methods') IS NOT NULL AND pg_get_serial_sequence('lbi_shipping_methods', 'shipping_method_id') IS NULL THEN
    CREATE SEQUENCE lbi_shipping_methods_shipping_method_id_seq;
    PERFORM setval('lbi_shipping_methods_shipping_method_id_seq', COALESCE((SELECT MAX(shipping_method_id) FROM lbi_shipping_methods), 1), true);
    ALTER TABLE lbi_shipping_methods ALTER COLUMN shipping_method_id SET DEFAULT nextval('lbi_shipping_methods_shipping_method_id_seq');
    ALTER SEQUENCE lbi_shipping_methods_shipping_method_id_seq OWNED BY lbi_shipping_methods.shipping_method_id;
  END IF;
END $$;

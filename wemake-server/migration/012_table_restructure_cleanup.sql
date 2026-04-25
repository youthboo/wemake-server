-- Backend migration 003 cleanup:
-- - replace legacy tag maps with category maps
-- - switch RFQ / quotation FKs to lbi master tables
-- - add factory_showcases.sub_category_id
-- - drop duplicate legacy tables

-- 1) factory_showcases.sub_category_id
ALTER TABLE factory_showcases
    ADD COLUMN IF NOT EXISTS sub_category_id BIGINT;

DO $$
BEGIN
    IF to_regclass('public.factory_showcases') IS NOT NULL
       AND to_regclass('public.lbi_sub_categories') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1
           FROM pg_constraint
           WHERE conname = 'factory_showcases_sub_category_id_fkey'
             AND conrelid = 'factory_showcases'::regclass
       ) THEN
        ALTER TABLE factory_showcases
            ADD CONSTRAINT factory_showcases_sub_category_id_fkey
            FOREIGN KEY (sub_category_id) REFERENCES lbi_sub_categories(sub_category_id);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_factory_showcases_sub_category_id
    ON factory_showcases(sub_category_id);

-- 2) category mapping tables
CREATE TABLE IF NOT EXISTS map_factory_categories (
    map_id BIGSERIAL PRIMARY KEY,
    factory_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS map_showcase_categories (
    map_id BIGSERIAL PRIMARY KEY,
    showcase_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL
);

DO $$
BEGIN
    IF to_regclass('public.map_factory_categories') IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'map_factory_categories_factory_category_key'
              AND conrelid = 'map_factory_categories'::regclass
        ) THEN
            ALTER TABLE map_factory_categories
                ADD CONSTRAINT map_factory_categories_factory_category_key
                UNIQUE (factory_id, category_id);
        END IF;

        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'map_factory_categories_factory_id_fkey'
              AND conrelid = 'map_factory_categories'::regclass
        ) THEN
            ALTER TABLE map_factory_categories
                ADD CONSTRAINT map_factory_categories_factory_id_fkey
                FOREIGN KEY (factory_id) REFERENCES users(user_id) ON DELETE CASCADE;
        END IF;

        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'map_factory_categories_category_id_fkey'
              AND conrelid = 'map_factory_categories'::regclass
        ) THEN
            ALTER TABLE map_factory_categories
                ADD CONSTRAINT map_factory_categories_category_id_fkey
                FOREIGN KEY (category_id) REFERENCES categories(category_id) ON DELETE CASCADE;
        END IF;
    END IF;

    IF to_regclass('public.map_showcase_categories') IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'map_showcase_categories_showcase_category_key'
              AND conrelid = 'map_showcase_categories'::regclass
        ) THEN
            ALTER TABLE map_showcase_categories
                ADD CONSTRAINT map_showcase_categories_showcase_category_key
                UNIQUE (showcase_id, category_id);
        END IF;

        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'map_showcase_categories_showcase_id_fkey'
              AND conrelid = 'map_showcase_categories'::regclass
        ) THEN
            ALTER TABLE map_showcase_categories
                ADD CONSTRAINT map_showcase_categories_showcase_id_fkey
                FOREIGN KEY (showcase_id) REFERENCES factory_showcases(showcase_id) ON DELETE CASCADE;
        END IF;

        IF NOT EXISTS (
            SELECT 1
            FROM pg_constraint
            WHERE conname = 'map_showcase_categories_category_id_fkey'
              AND conrelid = 'map_showcase_categories'::regclass
        ) THEN
            ALTER TABLE map_showcase_categories
                ADD CONSTRAINT map_showcase_categories_category_id_fkey
                FOREIGN KEY (category_id) REFERENCES categories(category_id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_map_factory_categories_factory_id
    ON map_factory_categories(factory_id);
CREATE INDEX IF NOT EXISTS idx_map_factory_categories_category_id
    ON map_factory_categories(category_id);
CREATE INDEX IF NOT EXISTS idx_map_showcase_categories_showcase_id
    ON map_showcase_categories(showcase_id);
CREATE INDEX IF NOT EXISTS idx_map_showcase_categories_category_id
    ON map_showcase_categories(category_id);

-- 3) migrate legacy tag mappings only when a category name matches an old tag name.
DO $$
BEGIN
    IF to_regclass('public.map_factory_tags') IS NOT NULL
       AND to_regclass('public.lbi_tags') IS NOT NULL THEN
        INSERT INTO map_factory_categories (factory_id, category_id)
        SELECT DISTINCT mft.factory_id::bigint, c.category_id
        FROM map_factory_tags mft
        INNER JOIN lbi_tags t ON t.tag_id = mft.tag_id
        INNER JOIN categories c ON lower(trim(c.name)) = lower(trim(t.tag_name))
        ON CONFLICT ON CONSTRAINT map_factory_categories_factory_category_key DO NOTHING;
    END IF;

    IF to_regclass('public.map_showcase_tags') IS NOT NULL
       AND to_regclass('public.lbi_tags') IS NOT NULL THEN
        INSERT INTO map_showcase_categories (showcase_id, category_id)
        SELECT DISTINCT mst.showcase_id::bigint, c.category_id
        FROM map_showcase_tags mst
        INNER JOIN lbi_tags t ON t.tag_id = mst.tag_id
        INNER JOIN categories c ON lower(trim(c.name)) = lower(trim(t.tag_name))
        ON CONFLICT ON CONSTRAINT map_showcase_categories_showcase_category_key DO NOTHING;
    END IF;
END $$;

-- 4) rfqs.unit_id -> lbi_units.unit_id
DO $$
DECLARE
    rec RECORD;
    has_unit_id BOOLEAN := FALSE;
BEGIN
    IF to_regclass('public.rfqs') IS NOT NULL THEN
        SELECT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'rfqs'
              AND column_name = 'unit_id'
        )
        INTO has_unit_id;
    END IF;

    IF to_regclass('public.rfqs') IS NOT NULL
       AND has_unit_id
       AND to_regclass('public.units') IS NOT NULL
       AND to_regclass('public.lbi_units') IS NOT NULL THEN
        UPDATE rfqs r
        SET unit_id = lu.unit_id
        FROM units u
        INNER JOIN lbi_units lu ON lu.unit_name_th = u.name
        WHERE r.unit_id = u.unit_id
          AND r.unit_id IS DISTINCT FROM lu.unit_id;
    END IF;

    IF to_regclass('public.rfqs') IS NOT NULL AND has_unit_id THEN
        FOR rec IN
            SELECT con.conname
            FROM pg_constraint con
            INNER JOIN pg_attribute att
                ON att.attrelid = con.conrelid
               AND att.attnum = ANY (con.conkey)
            WHERE con.conrelid = 'rfqs'::regclass
              AND con.contype = 'f'
              AND att.attname = 'unit_id'
        LOOP
            EXECUTE format('ALTER TABLE rfqs DROP CONSTRAINT IF EXISTS %I', rec.conname);
        END LOOP;

        IF to_regclass('public.lbi_units') IS NOT NULL
           AND NOT EXISTS (
               SELECT 1
               FROM pg_constraint
               WHERE conname = 'rfqs_unit_id_fkey'
                 AND conrelid = 'rfqs'::regclass
           ) THEN
            ALTER TABLE rfqs
                ADD CONSTRAINT rfqs_unit_id_fkey
                FOREIGN KEY (unit_id) REFERENCES lbi_units(unit_id);
        END IF;
    END IF;
END $$;

-- 5) quotations.shipping_method_id -> lbi_shipping_methods.shipping_method_id
DO $$
DECLARE
    rec RECORD;
BEGIN
    IF to_regclass('public.quotations') IS NOT NULL
       AND to_regclass('public.shipping_methods') IS NOT NULL
       AND to_regclass('public.lbi_shipping_methods') IS NOT NULL THEN
        UPDATE quotations q
        SET shipping_method_id = CASE sm.name
            WHEN 'pickup' THEN 1
            WHEN 'courier' THEN 2
            WHEN 'freight' THEN 4
            ELSE q.shipping_method_id
        END
        FROM shipping_methods sm
        WHERE q.shipping_method_id = sm.shipping_method_id;
    END IF;

    IF to_regclass('public.quotations') IS NOT NULL THEN
        FOR rec IN
            SELECT con.conname
            FROM pg_constraint con
            INNER JOIN pg_attribute att
                ON att.attrelid = con.conrelid
               AND att.attnum = ANY (con.conkey)
            WHERE con.conrelid = 'quotations'::regclass
              AND con.contype = 'f'
              AND att.attname = 'shipping_method_id'
        LOOP
            EXECUTE format('ALTER TABLE quotations DROP CONSTRAINT IF EXISTS %I', rec.conname);
        END LOOP;

        IF to_regclass('public.lbi_shipping_methods') IS NOT NULL
           AND NOT EXISTS (
               SELECT 1
               FROM pg_constraint
               WHERE conname = 'quotations_shipping_method_id_fkey'
                 AND conrelid = 'quotations'::regclass
           ) THEN
            ALTER TABLE quotations
                ADD CONSTRAINT quotations_shipping_method_id_fkey
                FOREIGN KEY (shipping_method_id) REFERENCES lbi_shipping_methods(shipping_method_id);
        END IF;
    END IF;
END $$;

-- 6) drop deprecated legacy tables after migration
DROP TABLE IF EXISTS map_showcase_tags;
DROP TABLE IF EXISTS map_factory_tags;
DROP TABLE IF EXISTS lbi_tags;
DROP TABLE IF EXISTS shipping_methods;
DROP TABLE IF EXISTS units;

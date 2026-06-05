-- 016_drop_factory_type.sql
-- factory_type is redundant — category_scopes (derived from map_factory_categories) is used instead
-- min_order is unused — MOQ is now per-showcase (factory_showcases.moq) not per-factory
ALTER TABLE factory_profiles DROP COLUMN IF EXISTS factory_type;
ALTER TABLE factory_profiles DROP COLUMN IF EXISTS min_order;

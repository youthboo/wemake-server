-- Migration: 017_showcase_specs.sql
-- Adds showcase_specs table for structured product specifications (PD type).

CREATE TABLE IF NOT EXISTS showcase_specs (
  spec_id     BIGSERIAL PRIMARY KEY,
  showcase_id BIGINT NOT NULL
                REFERENCES factory_showcases(showcase_id) ON DELETE CASCADE,
  spec_key    VARCHAR(100) NOT NULL,
  spec_value  VARCHAR(500) NOT NULL,
  sort_order  INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_showcase_specs_showcase
  ON showcase_specs(showcase_id, sort_order);

BEGIN;

-- 1) platform_config - add display label for config packages.
ALTER TABLE platform_config
    ADD COLUMN IF NOT EXISTS label VARCHAR(100) NULL;

UPDATE platform_config
SET label = 'มาตรฐาน (Default)'
WHERE label IS NULL
  AND config_id = (SELECT MIN(config_id) FROM platform_config);

-- 2) factory_profiles - assign factories to platform config packages.
ALTER TABLE factory_profiles
    ADD COLUMN IF NOT EXISTS config_id BIGINT
        REFERENCES platform_config(config_id) ON DELETE SET NULL;

UPDATE factory_profiles
SET config_id = (
    SELECT config_id FROM platform_config
    ORDER BY config_id ASC LIMIT 1
)
WHERE config_id IS NULL;

-- 3) factory_showcases - allow CL status.
ALTER TABLE factory_showcases
    DROP CONSTRAINT IF EXISTS factory_showcases_status_check;

ALTER TABLE factory_showcases
    ADD CONSTRAINT factory_showcases_status_check
    CHECK (status IN ('DR', 'AC', 'HI', 'AR', 'CL'));

COMMIT;

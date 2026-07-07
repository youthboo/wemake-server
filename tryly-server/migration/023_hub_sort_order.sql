-- Migration 023: Add sort_order to lbi_hub for custom ordering in UI

ALTER TABLE lbi_hub ADD COLUMN IF NOT EXISTS sort_order smallint NOT NULL DEFAULT 99;

UPDATE lbi_hub SET sort_order = CASE hub_id
  WHEN 7 THEN 1   -- อาหารและขนม (PD)
  WHEN 1 THEN 2   -- สัตว์เลี้ยง (PD)
  WHEN 2 THEN 3   -- บรรจุภัณฑ์สำเร็จรูป (PD)
  WHEN 3 THEN 11  -- เกษตรและธรรมชาติ (MT)
  WHEN 4 THEN 12  -- เคมีภัณฑ์และสารเติมแต่ง (MT)
  WHEN 5 THEN 13  -- วัตถุดิบบรรจุภัณฑ์ (MT)
  WHEN 6 THEN 14  -- วัตถุดิบอื่นๆ (MT)
  ELSE 99
END;

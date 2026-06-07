-- 018_seed_platform_config_defaults.sql
-- Init/seed: platform_config + tconfig defaults
-- กู้คืนค่า config คงที่กรณีฐานข้อมูลถูกล้าง (ค่าอ้างอิงจาก constant ที่เคย hardcode ไว้ใน FE/BE)
-- หมายเหตุ: tmail_templates ถูก DROP แล้ว (ดู 008_drop_legacy_mail_templates.sql)
--           mail template ปัจจุบันอยู่ใน internal/mailer/mail_templates.go
--           และค่า SMTP อ่านจาก environment variables ไม่ใช่จาก DB — จึงไม่ seed กลับ

-- ─── 1. platform_config (label = 'default_comm') ────────────
INSERT INTO platform_config (label, default_commission_rate, vat_rate, effective_from)
SELECT 'default_comm', 5.00, 7.00, NOW()
WHERE NOT EXISTS (SELECT 1 FROM platform_config WHERE label = 'default_comm');

-- ─── 2. tconfig key-value defaults ──────────────────────────
INSERT INTO tconfig (key, value) VALUES
  ('platform_name',       'Tryly'),
  ('contact_email',       'noreply.tryly@gmail.com'),
  ('support_phone',       ''),
  ('shipping_days',       '7'),
  ('rfq_expired',         '30'),
  ('public_web_url',      'https://tryly-web.vercel.app'),
  ('cron_commission_hour','9'),
  ('verify_requirements', '[
    {"id":"dbd","label":"ต้องมีเอกสาร DBD","enabled":true},
    {"id":"photo","label":"ต้องมีรูปถ่ายโรงงาน","enabled":true},
    {"id":"email","label":"ต้องยืนยัน email","enabled":true},
    {"id":"iso","label":"ต้องมี ISO certificate","enabled":false},
    {"id":"address","label":"ต้องมีที่อยู่จดทะเบียน","enabled":true}
  ]')
ON CONFLICT (key) DO NOTHING;

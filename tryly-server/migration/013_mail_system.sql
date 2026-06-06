-- 013_mail_system.sql
-- Email notification system: templates and send logs

-- ─── 1. Mail-related app config in tconfig ───────────────────────────────────
-- SMTP settings are read from environment variables in the app, not from DB.
INSERT INTO tconfig (key, value) VALUES
  ('public_web_url', 'https://tryly-web.vercel.app'),
  ('cron_commission_hour', '9')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- ─── 2. Mail send log ───────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS mail_logs (
  log_id         BIGSERIAL    PRIMARY KEY,
  template_code  VARCHAR(50)  NOT NULL,
  recipient      TEXT         NOT NULL,
  subject        TEXT         NOT NULL,
  body           TEXT         NOT NULL,
  status         VARCHAR(10)  NOT NULL DEFAULT 'OK',   -- OK, FAIL
  error_message  TEXT,
  ref_type       VARCHAR(30),                           -- order, invoice
  ref_id         BIGINT,                                -- order_id or invoice_id
  created_at     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mail_logs_ref ON mail_logs (ref_type, ref_id);
CREATE INDEX IF NOT EXISTS idx_mail_logs_created ON mail_logs (created_at DESC);

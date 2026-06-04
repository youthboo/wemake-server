-- 013_mail_system.sql
-- Email notification system: SMTP config, templates, and send logs

-- ─── 1. SMTP config in tconfig ──────────────────────────────────────────────
INSERT INTO tconfig (key, value) VALUES
  ('smtp_host',     'smtp.gmail.com'),
  ('smtp_port',     '587'),
  ('smtp_user',     'noreply.tryly@gmail.com'),
  ('smtp_password', 'fwbt pnpp ocgk xntk'),
  ('smtp_from',     'Tryly <noreply.tryly@gmail.com>'),
  ('public_web_url', 'https://tryly-web.vercel.app'),
  ('cron_commission_hour', '9')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- ─── 2. Email template table ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS tmail_templates (
  template_code  VARCHAR(50)  PRIMARY KEY,
  subject        TEXT         NOT NULL,
  body           TEXT         NOT NULL,
  description    TEXT,
  updated_at     TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Placeholders: {{.OrderID}}, {{.FactoryName}}, {{.Amount}}, {{.Link}}, etc.
INSERT INTO tmail_templates (template_code, subject, body, description) VALUES
(
  'SLIP_ATTACHED',
  'ลูกค้าแนบสลีปชำระเงิน — Order #{{.OrderID}}',
  'สวัสดีครับ/ค่ะ {{.FactoryName}}

ลูกค้าได้แนบสลีปการชำระเงินสำหรับ Order #{{.OrderID}} แล้ว

รายละเอียด:
- หมายเลขคำสั่งซื้อ: #{{.OrderID}}
- ยอดรวม: {{.Amount}} บาท

กรุณาตรวจสอบสลีปและ Approve/Reject ที่ระบบ Tryly:
{{.Link}}

ขอบคุณครับ/ค่ะ
ทีม Tryly',
  'ส่งให้โรงงานเมื่อลูกค้าแนบสลีป'
),
(
  'SLIP_APPROVED',
  'โรงงานยืนยันรับชำระเงินแล้ว — Order #{{.OrderID}}',
  'สวัสดีครับ/ค่ะ

โรงงาน {{.FactoryName}} ได้ยืนยันรับชำระเงินสำหรับ Order #{{.OrderID}} เรียบร้อยแล้ว
ขั้นตอนการผลิตจะเริ่มต้นในลำดับถัดไป

ติดตามสถานะได้ที่:
{{.Link}}

ขอบคุณครับ/ค่ะ
ทีม Tryly',
  'ส่งให้ลูกค้าเมื่อโรงงาน approve สลีป'
),
(
  'COMMISSION_INVOICE',
  'ใบแจ้งค่า Commission เดือน {{.PeriodMonth}}/{{.PeriodYear}} — Tryly',
  'สวัสดีครับ/ค่ะ {{.FactoryName}}

แพลตฟอร์ม Tryly ขอแจ้งค่า Service Charge ประจำเดือน {{.PeriodMonth}}/{{.PeriodYear}}

สรุป:
- จำนวน Order: {{.TotalOrders}} รายการ
- ยอดค่า Commission: {{.CommissionAmount}} บาท
- VAT: {{.VatAmount}} บาท
- รวมทั้งสิ้น: {{.GrandTotal}} บาท

กรุณาชำระเงินและแนบสลีปที่ระบบ:
{{.Link}}

ขอบคุณครับ/ค่ะ
ทีม Tryly',
  'ส่งให้โรงงานเมื่อ SA กด Send Invoice สิ้นเดือน'
),
(
  'COMM_SLIP_ATTACHED',
  'โรงงาน {{.FactoryName}} แนบสลีปค่า Commission แล้ว',
  'สวัสดีครับ/ค่ะ Admin

โรงงาน {{.FactoryName}} ได้แนบสลีปชำระค่า Commission สำหรับ Invoice #{{.InvoiceID}} แล้ว

รายละเอียด:
- Invoice ID: #{{.InvoiceID}}
- เดือน: {{.PeriodMonth}}/{{.PeriodYear}}
- ยอดรวม: {{.GrandTotal}} บาท

กรุณาตรวจสอบที่:
{{.Link}}

ขอบคุณครับ/ค่ะ
ทีม Tryly',
  'ส่งให้ SA เมื่อโรงงานแนบสลีปค่า Commission'
)
ON CONFLICT (template_code) DO UPDATE
  SET subject = EXCLUDED.subject, body = EXCLUDED.body, description = EXCLUDED.description, updated_at = NOW();

-- ─── 3. Mail send log ───────────────────────────────────────────────────────
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

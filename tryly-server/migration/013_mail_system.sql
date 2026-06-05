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
-- Body is HTML. Use inline styles for maximum email-client compatibility.
INSERT INTO tmail_templates (template_code, subject, body, description) VALUES
(
  'SLIP_ATTACHED',
  'ลูกค้าแนบสลีปชำระเงิน — Order #{{.OrderID}}',
  '<!DOCTYPE html>
<html lang="th">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:''Sarabun'',''Helvetica Neue'',Arial,sans-serif">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f5f5f5;padding:32px 0">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:12px;overflow:hidden;max-width:600px;width:100%">

        <!-- Logo -->
        <tr>
          <td align="center" style="padding:40px 40px 24px">
            <img src="https://tryly-web.vercel.app/assets/tryly-logo.png" alt="Tryly" width="140" style="border-radius:16px;display:block">
          </td>
        </tr>

        <!-- Body -->
        <tr>
          <td style="padding:0 40px 32px;color:#222222;font-size:16px;line-height:1.7">
            <p style="margin:0 0 16px">เรียน <strong>{{.FactoryName}}</strong></p>
            <p style="margin:0 0 16px">ลูกค้าได้แนบสลีปการชำระเงินสำหรับ Order <strong>#{{.OrderID}}</strong> เรียบร้อยแล้ว</p>

            <!-- Details box -->
            <table width="100%" cellpadding="0" cellspacing="0" style="background:#f9f9f9;border-radius:8px;margin:0 0 24px">
              <tr><td style="padding:16px 20px;font-size:15px;color:#444444">
                <p style="margin:0 0 8px">📦 หมายเลขคำสั่งซื้อ: <strong>#{{.OrderID}}</strong></p>
                <p style="margin:0">💰 ยอดรวม: <strong>{{.Amount}} บาท</strong></p>
              </td></tr>
            </table>

            <p style="margin:0 0 20px"><strong>กรุณาตรวจสอบสลีปและ Approve / Reject ที่ระบบ Tryly:</strong></p>
            <p style="margin:0 0 32px;text-align:center">
              <a href="{{.Link}}" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;font-size:15px;font-weight:bold;padding:12px 32px;border-radius:8px">ไปที่คำสั่งซื้อ</a>
            </p>

            <p style="margin:0 0 4px">หากมีข้อสงสัย กรุณาติดต่อเราที่แอปพลิเคชัน Tryly: เมนู → ศูนย์ช่วยเหลือ</p>
          </td>
        </tr>

        <!-- Sign-off -->
        <tr>
          <td style="padding:0 40px 40px;color:#222222;font-size:16px;line-height:1.7">
            <p style="margin:0 0 4px">ขอบคุณที่ใช้งาน <strong>Tryly</strong></p>
            <p style="margin:0 0 4px">ขอแสดงความนับถืออย่างสูง</p>
            <p style="margin:0"><strong>ทีม Tryly</strong></p>
          </td>
        </tr>

        <!-- Footer -->
        <tr>
          <td style="background:#1e293b;padding:20px 40px;text-align:center">
            <p style="margin:0 0 8px;color:#94a3b8;font-size:13px">
              <a href="https://tryly-web.vercel.app" style="color:#94a3b8;text-decoration:none">ศูนย์ช่วยเหลือ</a>
              &nbsp;&nbsp;|&nbsp;&nbsp;
              <a href="https://tryly-web.vercel.app" style="color:#94a3b8;text-decoration:none">ติดต่อเรา</a>
            </p>
            <p style="margin:0;color:#64748b;font-size:12px">© 2025 Tryly. All rights reserved.</p>
          </td>
        </tr>

      </table>
    </td></tr>
  </table>
</body>
</html>',
  'ส่งให้โรงงานเมื่อลูกค้าแนบสลีป'
),
(
  'SLIP_APPROVED',
  'โรงงานยืนยันรับชำระเงินแล้ว — Order #{{.OrderID}}',
  '<!DOCTYPE html>
<html lang="th">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:''Sarabun'',''Helvetica Neue'',Arial,sans-serif">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f5f5f5;padding:32px 0">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:12px;overflow:hidden;max-width:600px;width:100%">

        <!-- Logo -->
        <tr>
          <td align="center" style="padding:40px 40px 24px">
            <img src="https://tryly-web.vercel.app/assets/tryly-logo.png" alt="Tryly" width="140" style="border-radius:16px;display:block">
          </td>
        </tr>

        <!-- Body -->
        <tr>
          <td style="padding:0 40px 32px;color:#222222;font-size:16px;line-height:1.7">
            <p style="margin:0 0 16px">เรียน ผู้ใช้งาน <strong>Tryly</strong></p>
            <p style="margin:0 0 16px">โรงงาน <strong>{{.FactoryName}}</strong> ได้ยืนยันรับชำระเงินสำหรับ Order <strong>#{{.OrderID}}</strong> เรียบร้อยแล้ว</p>
            <p style="margin:0 0 24px">ขั้นตอนการผลิตจะเริ่มต้นในลำดับถัดไป</p>

            <p style="margin:0 0 20px"><strong>ติดตามสถานะคำสั่งซื้อของท่านได้ที่ระบบ Tryly:</strong></p>
            <p style="margin:0 0 32px;text-align:center">
              <a href="{{.Link}}" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;font-size:15px;font-weight:bold;padding:12px 32px;border-radius:8px">ติดตามสถานะ</a>
            </p>

            <p style="margin:0 0 4px">หากมีข้อสงสัย กรุณาติดต่อเราที่แอปพลิเคชัน Tryly: เมนู → ศูนย์ช่วยเหลือ</p>
          </td>
        </tr>

        <!-- Sign-off -->
        <tr>
          <td style="padding:0 40px 40px;color:#222222;font-size:16px;line-height:1.7">
            <p style="margin:0 0 4px">ขอบคุณที่ใช้งาน <strong>Tryly</strong></p>
            <p style="margin:0 0 4px">ขอแสดงความนับถืออย่างสูง</p>
            <p style="margin:0"><strong>ทีม Tryly</strong></p>
          </td>
        </tr>

        <!-- Footer -->
        <tr>
          <td style="background:#1e293b;padding:20px 40px;text-align:center">
            <p style="margin:0 0 8px;color:#94a3b8;font-size:13px">
              <a href="https://tryly-web.vercel.app" style="color:#94a3b8;text-decoration:none">ศูนย์ช่วยเหลือ</a>
              &nbsp;&nbsp;|&nbsp;&nbsp;
              <a href="https://tryly-web.vercel.app" style="color:#94a3b8;text-decoration:none">ติดต่อเรา</a>
            </p>
            <p style="margin:0;color:#64748b;font-size:12px">© 2025 Tryly. All rights reserved.</p>
          </td>
        </tr>

      </table>
    </td></tr>
  </table>
</body>
</html>',
  'ส่งให้ลูกค้าเมื่อโรงงาน approve สลีป'
),
(
  'COMMISSION_INVOICE',
  'ใบแจ้งค่า Commission เดือน {{.PeriodMonth}}/{{.PeriodYear}} — Tryly',
  '<!DOCTYPE html>
<html lang="th">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:''Sarabun'',''Helvetica Neue'',Arial,sans-serif">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f5f5f5;padding:32px 0">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:12px;overflow:hidden;max-width:600px;width:100%">

        <!-- Logo -->
        <tr>
          <td align="center" style="padding:40px 40px 24px">
            <img src="https://tryly-web.vercel.app/assets/tryly-logo.png" alt="Tryly" width="140" style="border-radius:16px;display:block">
          </td>
        </tr>

        <!-- Body -->
        <tr>
          <td style="padding:0 40px 32px;color:#222222;font-size:16px;line-height:1.7">
            <p style="margin:0 0 16px">เรียน <strong>{{.FactoryName}}</strong></p>
            <p style="margin:0 0 16px">แพลตฟอร์ม <strong>Tryly</strong> ขอแจ้งค่า Service Charge ประจำเดือน <strong>{{.PeriodMonth}}/{{.PeriodYear}}</strong></p>

            <!-- Summary box -->
            <table width="100%" cellpadding="0" cellspacing="0" style="background:#f9f9f9;border-radius:8px;margin:0 0 24px">
              <tr><td style="padding:16px 20px;font-size:15px;color:#444444">
                <p style="margin:0 0 8px">📋 จำนวน Order: <strong>{{.TotalOrders}} รายการ</strong></p>
                <p style="margin:0 0 8px">💼 ยอดค่า Commission: <strong>{{.CommissionAmount}} บาท</strong></p>
                <p style="margin:0 0 8px">🧾 VAT: <strong>{{.VatAmount}} บาท</strong></p>
                <p style="margin:0;font-size:16px;color:#222222">💰 รวมทั้งสิ้น: <strong>{{.GrandTotal}} บาท</strong></p>
              </td></tr>
            </table>

            <p style="margin:0 0 20px"><strong>กรุณาชำระเงินและแนบสลีปที่ระบบ Tryly:</strong></p>
            <p style="margin:0 0 32px;text-align:center">
              <a href="{{.Link}}" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;font-size:15px;font-weight:bold;padding:12px 32px;border-radius:8px">ชำระค่า Commission</a>
            </p>

            <p style="margin:0 0 4px">หากมีข้อสงสัย กรุณาติดต่อเราที่แอปพลิเคชัน Tryly: เมนู → ศูนย์ช่วยเหลือ</p>
          </td>
        </tr>

        <!-- Sign-off -->
        <tr>
          <td style="padding:0 40px 40px;color:#222222;font-size:16px;line-height:1.7">
            <p style="margin:0 0 4px">ขอบคุณที่ใช้งาน <strong>Tryly</strong></p>
            <p style="margin:0 0 4px">ขอแสดงความนับถืออย่างสูง</p>
            <p style="margin:0"><strong>ทีม Tryly</strong></p>
          </td>
        </tr>

        <!-- Footer -->
        <tr>
          <td style="background:#1e293b;padding:20px 40px;text-align:center">
            <p style="margin:0 0 8px;color:#94a3b8;font-size:13px">
              <a href="https://tryly-web.vercel.app" style="color:#94a3b8;text-decoration:none">ศูนย์ช่วยเหลือ</a>
              &nbsp;&nbsp;|&nbsp;&nbsp;
              <a href="https://tryly-web.vercel.app" style="color:#94a3b8;text-decoration:none">การกำหนดราคา</a>
              &nbsp;&nbsp;|&nbsp;&nbsp;
              <a href="https://tryly-web.vercel.app" style="color:#94a3b8;text-decoration:none">ติดต่อเรา</a>
            </p>
            <p style="margin:0;color:#64748b;font-size:12px">© 2025 Tryly. All rights reserved.</p>
          </td>
        </tr>

      </table>
    </td></tr>
  </table>
</body>
</html>',
  'ส่งให้โรงงานเมื่อ SA กด Send Invoice สิ้นเดือน'
),
(
  'COMM_SLIP_ATTACHED',
  'โรงงาน {{.FactoryName}} แนบสลีปค่า Commission แล้ว',
  '<!DOCTYPE html>
<html lang="th">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:''Sarabun'',''Helvetica Neue'',Arial,sans-serif">
  <table width="100%" cellpadding="0" cellspacing="0" style="background:#f5f5f5;padding:32px 0">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:12px;overflow:hidden;max-width:600px;width:100%">

        <!-- Logo -->
        <tr>
          <td align="center" style="padding:40px 40px 24px">
            <img src="https://tryly-web.vercel.app/assets/tryly-logo.png" alt="Tryly" width="140" style="border-radius:16px;display:block">
          </td>
        </tr>

        <!-- Body -->
        <tr>
          <td style="padding:0 40px 32px;color:#222222;font-size:16px;line-height:1.7">
            <p style="margin:0 0 16px">เรียน <strong>Admin</strong></p>
            <p style="margin:0 0 16px">โรงงาน <strong>{{.FactoryName}}</strong> ได้แนบสลีปชำระค่า Commission สำหรับ Invoice <strong>#{{.InvoiceID}}</strong> เรียบร้อยแล้ว</p>

            <!-- Details box -->
            <table width="100%" cellpadding="0" cellspacing="0" style="background:#f9f9f9;border-radius:8px;margin:0 0 24px">
              <tr><td style="padding:16px 20px;font-size:15px;color:#444444">
                <p style="margin:0 0 8px">🧾 Invoice ID: <strong>#{{.InvoiceID}}</strong></p>
                <p style="margin:0 0 8px">📅 เดือน: <strong>{{.PeriodMonth}}/{{.PeriodYear}}</strong></p>
                <p style="margin:0">💰 ยอดรวม: <strong>{{.GrandTotal}} บาท</strong></p>
              </td></tr>
            </table>

            <p style="margin:0 0 20px"><strong>กรุณาตรวจสอบและ Approve / Reject สลีปที่ระบบ Tryly:</strong></p>
            <p style="margin:0 0 32px;text-align:center">
              <a href="{{.Link}}" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;font-size:15px;font-weight:bold;padding:12px 32px;border-radius:8px">ไปที่ Invoice</a>
            </p>

            <p style="margin:0 0 4px">หากมีข้อสงสัย กรุณาติดต่อเราที่แอปพลิเคชัน Tryly: เมนู → ศูนย์ช่วยเหลือ</p>
          </td>
        </tr>

        <!-- Sign-off -->
        <tr>
          <td style="padding:0 40px 40px;color:#222222;font-size:16px;line-height:1.7">
            <p style="margin:0 0 4px">ขอบคุณที่ใช้งาน <strong>Tryly</strong></p>
            <p style="margin:0 0 4px">ขอแสดงความนับถืออย่างสูง</p>
            <p style="margin:0"><strong>ทีม Tryly</strong></p>
          </td>
        </tr>

        <!-- Footer -->
        <tr>
          <td style="background:#1e293b;padding:20px 40px;text-align:center">
            <p style="margin:0 0 8px;color:#94a3b8;font-size:13px">
              <a href="https://tryly-web.vercel.app" style="color:#94a3b8;text-decoration:none">ศูนย์ช่วยเหลือ</a>
              &nbsp;&nbsp;|&nbsp;&nbsp;
              <a href="https://tryly-web.vercel.app" style="color:#94a3b8;text-decoration:none">ติดต่อเรา</a>
            </p>
            <p style="margin:0;color:#64748b;font-size:12px">© 2025 Tryly. All rights reserved.</p>
          </td>
        </tr>

      </table>
    </td></tr>
  </table>
</body>
</html>',
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

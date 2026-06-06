package mailer

var mailTemplates = map[string]Template{
	"SLIP_ATTACHED": {
		Subject: `ลูกค้าแนบสลีปชำระเงิน — Order #{{.OrderID}}`,
		Body: `<!DOCTYPE html>
<html lang="th">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:'Sarabun','Helvetica Neue',Arial,sans-serif">
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
</html>`,
	},
	"SLIP_APPROVED": {
		Subject: `โรงงานยืนยันรับชำระเงินแล้ว — Order #{{.OrderID}}`,
		Body: `<!DOCTYPE html>
<html lang="th">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:'Sarabun','Helvetica Neue',Arial,sans-serif">
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
</html>`,
	},
	"COMMISSION_INVOICE": {
		Subject: `ใบแจ้งค่า Commission เดือน {{.PeriodMonth}}/{{.PeriodYear}} — Tryly`,
		Body: `<!DOCTYPE html>
<html lang="th">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:'Sarabun','Helvetica Neue',Arial,sans-serif">
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
</html>`,
	},
	"COMM_SLIP_ATTACHED": {
		Subject: `โรงงาน {{.FactoryName}} แนบสลีปค่า Commission แล้ว`,
		Body: `<!DOCTYPE html>
<html lang="th">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f5f5f5;font-family:'Sarabun','Helvetica Neue',Arial,sans-serif">
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
</html>`,
	},
}

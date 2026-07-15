-- 030: Slip auto-verification (SlipOK) support — low DB effort
--
-- ตรวจสลิปอัตโนมัติผ่าน SlipOK แล้วเก็บผลไว้บน transaction เดิม (type 'BU')
-- แทนการสร้างตารางใหม่ ลด effort DB:
--   bank_ref        = transRef จริงจากธนาคาร (กัน replay)
--   verify_status   = pending / verified / failed
--   verify_response = raw JSON เต็มจาก SlipOK (audit/debug ย้อนหลัง)
--   transferred_at  = เวลาที่โอนจริงตามสลิป (กันเอาสลิปเก่ามาเคลม)

ALTER TABLE transactions
    ADD COLUMN IF NOT EXISTS bank_ref        VARCHAR(40),
    ADD COLUMN IF NOT EXISTS verify_status   VARCHAR(10),
    ADD COLUMN IF NOT EXISTS verify_response JSONB,
    ADD COLUMN IF NOT EXISTS transferred_at  TIMESTAMP;

-- Replay protection: เลขอ้างอิงจริงห้ามซ้ำ — แต่บังคับเฉพาะสลิปที่ verified แล้วเท่านั้น
-- (สลิปที่ fail เก็บ ref แบบ surrogate ได้ ไม่บล็อกการแนบ ref เดิมซ้ำในออเดอร์ที่ถูกต้องภายหลัง)
CREATE UNIQUE INDEX IF NOT EXISTS uq_transactions_bank_ref_verified
    ON transactions (bank_ref)
    WHERE verify_status = 'verified' AND bank_ref IS NOT NULL;

-- ดัชนีช่วย query สลิปที่รอตรวจ (admin fallback)
CREATE INDEX IF NOT EXISTS idx_transactions_verify_status
    ON transactions (verify_status)
    WHERE verify_status IS NOT NULL;

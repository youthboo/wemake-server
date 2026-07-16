-- 032: เก็บข้อมูลบัญชีปลายทาง + ช่องทางติดต่อกลับ สำหรับการคืนเงิน
--
-- ลูกค้าจ่ายผ่านโอนธนาคาร (ไม่มี wallet) → superadmin ต้องรู้บัญชี/พร้อมเพย์
-- ปลายทาง + ชื่อบัญชี เพื่อโอนเงินคืน และช่องทางติดต่อกลับ

ALTER TABLE disputes
    ADD COLUMN IF NOT EXISTS refund_account      VARCHAR(50),  -- เลขบัญชี หรือ พร้อมเพย์
    ADD COLUMN IF NOT EXISTS refund_account_name VARCHAR(150), -- ชื่อ-นามสกุล บัญชีปลายทาง
    ADD COLUMN IF NOT EXISTS contact_email       VARCHAR(150),
    ADD COLUMN IF NOT EXISTS contact_phone       VARCHAR(30);

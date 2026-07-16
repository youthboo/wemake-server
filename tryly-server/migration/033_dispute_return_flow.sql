-- 033: RMA return flow for disputes (ส่งสินค้าคืน + ตรวจรับ + คืนเงินบางส่วน)
--
-- ขยาย state machine: OP → RT (รอลูกค้าส่งคืน) → RC (รอตรวจรับ) → RF/RJ
-- โดยเก็บหลักฐานการส่งคืน (tracking/ขนส่ง/บิล/รูป) และรองรับคืนเงินบางส่วน
-- (refund_amount มีอยู่แล้วจาก 031 — ตอนนี้ admin กรอกได้ ไม่บังคับเต็มจำนวน)

ALTER TABLE disputes
    ADD COLUMN IF NOT EXISTS return_tracking_no   VARCHAR(100),
    ADD COLUMN IF NOT EXISTS return_courier       VARCHAR(60),
    ADD COLUMN IF NOT EXISTS return_note          TEXT,
    ADD COLUMN IF NOT EXISTS return_evidence_urls JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS return_requested_at  TIMESTAMP,  -- admin สั่งให้ส่งคืนเมื่อ
    ADD COLUMN IF NOT EXISTS return_submitted_at  TIMESTAMP;  -- ลูกค้าแนบหลักฐานเมื่อ

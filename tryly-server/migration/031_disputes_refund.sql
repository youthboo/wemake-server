-- 031: Customer complaint / refund tickets (disputes)
--
-- ลูกค้า (CT) เปิด ticket ร้องเรียนขอคืนเงินบน order ที่จ่ายเงินแล้ว เมื่อ
-- "ไม่ได้รับสินค้า" (NR) หรือ "สินค้าไม่ตรงปก" (ND) → superadmin ตรวจแล้ว
-- คืนเงินเต็มจำนวน (โอนคืน + แนบสลิป) หรือปฏิเสธคำร้อง
--
-- ตาราง disputes ถูก DROP ใน 001 แต่ไม่เคยสร้าง — สร้างที่นี่ให้ตรงกับ
-- internal/repository/wallet/dispute_repository.go + ส่วนขยาย refund

CREATE TABLE IF NOT EXISTS disputes (
    dispute_id      BIGSERIAL PRIMARY KEY,
    order_id        BIGINT NOT NULL REFERENCES orders(order_id),
    opened_by       BIGINT NOT NULL REFERENCES users(user_id),
    category        CHAR(2) NOT NULL DEFAULT 'NR',   -- NR=ไม่ได้รับสินค้า, ND=สินค้าไม่ตรงปก, OT=อื่นๆ
    reason          TEXT NOT NULL,                    -- คำอธิบายจากลูกค้า
    evidence_urls   JSONB NOT NULL DEFAULT '[]',      -- รูปหลักฐาน (สูงสุด 5)
    status          CHAR(2) NOT NULL DEFAULT 'OP',    -- OP=รอตรวจสอบ, RF=คืนเงินแล้ว, RJ=ปฏิเสธ
    resolution      TEXT,                             -- หมายเหตุจาก superadmin
    refund_amount   NUMERIC(10,2),                    -- ยอดคืน (เต็มจำนวน = total_amount)
    refund_slip_url TEXT,                             -- สลิปโอนคืนจาก superadmin
    resolved_by     BIGINT REFERENCES users(user_id),
    resolved_at     TIMESTAMP,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_disputes_order  ON disputes(order_id);
CREATE INDEX IF NOT EXISTS idx_disputes_status ON disputes(status);

-- กันเปิด ticket ที่ยังค้าง (OP) ซ้ำบน order เดียวกัน
CREATE UNIQUE INDEX IF NOT EXISTS uq_disputes_open_per_order
    ON disputes(order_id) WHERE status = 'OP';

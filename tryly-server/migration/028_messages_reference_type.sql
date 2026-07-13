-- 028: add messages.reference_type
--
-- ก่อนหน้านี้ messages ไม่มี reference_type — โค้ด derive จาก message_type
-- ทำให้ข้อความ TX ที่อ้างถึง Order แยกไม่ออกจากที่อ้าง RFQ (reference_id ชนกันได้)
-- และลิงก์ผิด (order → /rfqs/:id). เพิ่ม column เก็บ "สิ่งที่ข้อความอ้างถึง" ตรง ๆ
-- แยกจาก message_type ที่บอก "รูปแบบข้อความ".
--
-- RQ=RFQ, OD=Order, PD/PM/ID=Showcase (product/promotion/idea)

ALTER TABLE messages ADD COLUMN IF NOT EXISTS reference_type VARCHAR(2);

-- Backfill row เดิมจาก message_type (logic เดียวกับที่โค้ดเคย derive)
-- row ที่ derive ไม่ได้ (เช่น TX ที่อ้าง Order) จะเป็น NULL — read path มี fallback ให้
UPDATE messages SET reference_type = CASE message_type
    WHEN 'QT'             THEN 'RQ'
    WHEN 'rfq_card'       THEN 'RQ'
    WHEN 'quotation_card' THEN 'RQ'
    WHEN 'PD'             THEN 'PD'
    WHEN 'PM'             THEN 'PM'
    WHEN 'ID'             THEN 'ID'
    ELSE NULL
END
WHERE reference_type IS NULL;

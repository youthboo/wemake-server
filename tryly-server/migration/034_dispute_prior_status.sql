-- 034: เก็บสถานะ order ก่อนเปิด dispute เพื่อคืนค่าให้ถูกต้องตอนปฏิเสธ/คืนเงินบางส่วน
--
-- เดิม Reject/partial-Refund คืนค่า order กลับเป็น 'PD' เสมอ — ผิดถ้า order มา
-- จากสถานะอื่น (PR/WF/QC/SH/DL/CP) ตอนเปิด dispute เก็บ prior_order_status ไว้
-- แล้วคืนค่ากลับตามจริง

ALTER TABLE disputes
    ADD COLUMN IF NOT EXISTS prior_order_status CHAR(2);

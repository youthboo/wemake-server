-- 029: Platform (Tryly) system wallet
--
-- Escrow flow ที่ถูกต้อง: เงินลูกค้า (CT) เข้า wallet ของ platform (SA) ก่อน
-- แล้วตอน order เสร็จจึงกระจาย → net ให้โรงงาน (FT) + commission เก็บไว้ที่ SA
-- ใช้ user_id ที่กำหนดใน tconfig เป็นเจ้าของ platform wallet (เปลี่ยนได้ภายหลัง)

INSERT INTO tconfig (key, value) VALUES ('platform_user_id', '5')
ON CONFLICT (key) DO NOTHING;

-- สร้าง wallet ให้ platform user ถ้ายังไม่มี (wallets.user_id ไม่มี unique index จึงใช้ NOT EXISTS)
INSERT INTO wallets (user_id, good_fund, pending_fund)
SELECT p.uid, 0, 0
FROM (SELECT (value)::bigint AS uid FROM tconfig WHERE key = 'platform_user_id') p
WHERE NOT EXISTS (SELECT 1 FROM wallets w WHERE w.user_id = p.uid);

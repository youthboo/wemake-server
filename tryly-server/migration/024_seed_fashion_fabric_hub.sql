-- Migration 024: Add "ทั้งหมด" to hub_id=7 + seed hub 8 (แฟชั่น PD) + hub 9 (ผ้า MT)

-- ============================================================
-- SECTION 1: เพิ่ม "ทั้งหมด" (sort_order=99) ให้ hub_id=7
-- sub_category_id 209–215
-- ============================================================
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (209, 18, 'ทั้งหมด', 99, '1'),
  (210, 19, 'ทั้งหมด', 99, '1'),
  (211, 20, 'ทั้งหมด', 99, '1'),
  (212, 21, 'ทั้งหมด', 99, '1'),
  (213, 22, 'ทั้งหมด', 99, '1'),
  (214, 23, 'ทั้งหมด', 99, '1'),
  (215, 24, 'ทั้งหมด', 99, '1');

-- ============================================================
-- SECTION 2: HUB 8 — แฟชั่นและเสื้อผ้า (scope=PD, sort_order=4)
-- ============================================================
INSERT INTO lbi_hub (hub_id, name, scope, sort_order) VALUES
  (8, 'แฟชั่นและเสื้อผ้า', 'PD', 4);

-- categories hub_id=8
INSERT INTO lbi_categories (category_id, hub_id, name) VALUES
  (25, 8, 'เสื้อผ้าสำเร็จรูป'),
  (26, 8, 'ชุดกีฬาและ Activewear'),
  (27, 8, 'ชุดชั้นในและชุดนอน'),
  (28, 8, 'เสื้อผ้าเด็ก'),
  (29, 8, 'ชุดทำงานและยูนิฟอร์ม'),
  (30, 8, 'กระเป๋าและเครื่องหนัง'),
  (31, 8, 'หมวก รองเท้า และเครื่องประดับแฟชั่น');

-- sub-categories: cat 25 เสื้อผ้าสำเร็จรูป
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (220, 25, 'เสื้อยืดและโปโล', 1, '1'),
  (221, 25, 'เสื้อเชิ้ต / Shirt', 2, '1'),
  (222, 25, 'กางเกงขายาวและขาสั้น', 3, '1'),
  (223, 25, 'เดรสและกระโปรง', 4, '1'),
  (224, 25, 'แจ็กเก็ตและเสื้อกันหนาว', 5, '1'),
  (225, 25, 'ชุดเซ็ต / Co-ord Set', 6, '1'),
  (226, 25, 'Oversize / Streetwear', 7, '1'),
  (227, 25, 'Plus Size / Petite', 8, '1'),
  (228, 25, 'Casual / Everyday Wear', 9, '1'),
  (229, 25, 'ทั้งหมด', 99, '1');

-- sub-categories: cat 26 ชุดกีฬาและ Activewear
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (230, 26, 'ชุดออกกำลังกาย / Gym Wear', 1, '1'),
  (231, 26, 'ชุดว่ายน้ำ / Swimwear', 2, '1'),
  (232, 26, 'ชุดวิ่งและโยคะ', 3, '1'),
  (233, 26, 'ชุดกีฬาทีม / Team Jersey', 4, '1'),
  (234, 26, 'เสื้อผ้าไต่เขาและ Outdoor', 5, '1'),
  (235, 26, 'Compression Wear', 6, '1'),
  (236, 26, 'Athleisure / Daily Active', 7, '1'),
  (237, 26, 'ทั้งหมด', 99, '1');

-- sub-categories: cat 27 ชุดชั้นในและชุดนอน
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (240, 27, 'บราและชุดชั้นในหญิง', 1, '1'),
  (241, 27, 'บ็อกเซอร์และชุดชั้นในชาย', 2, '1'),
  (242, 27, 'ชุดนอนและ Loungewear', 3, '1'),
  (243, 27, 'ชุดคลุม / Robe', 4, '1'),
  (244, 27, 'ถุงเท้าและถุงน่อง', 5, '1'),
  (245, 27, 'ชุดชั้นในเด็ก', 6, '1'),
  (246, 27, 'ทั้งหมด', 99, '1');

-- sub-categories: cat 28 เสื้อผ้าเด็ก
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (250, 28, 'เสื้อผ้าทารก (0–2 ปี)', 1, '1'),
  (251, 28, 'เสื้อผ้าเด็กเล็ก (3–6 ปี)', 2, '1'),
  (252, 28, 'เสื้อผ้าเด็กโต (7–12 ปี)', 3, '1'),
  (253, 28, 'ชุดนักเรียน', 4, '1'),
  (254, 28, 'ชุดกีฬาเด็ก', 5, '1'),
  (255, 28, 'Costume / ชุดแฟนซีเด็ก', 6, '1'),
  (256, 28, 'ทั้งหมด', 99, '1');

-- sub-categories: cat 29 ชุดทำงานและยูนิฟอร์ม
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (260, 29, 'เสื้อโปโลองค์กร / Corporate Polo', 1, '1'),
  (261, 29, 'ชุดพนักงานร้านค้าและโรงแรม', 2, '1'),
  (262, 29, 'ยูนิฟอร์มโรงงานและช่าง', 3, '1'),
  (263, 29, 'ชุดบุคลากรทางการแพทย์', 4, '1'),
  (264, 29, 'เสื้อกีฬาองค์กร / Sport Day', 5, '1'),
  (265, 29, 'ชุดพนักงานขนส่ง / Driver Uniform', 6, '1'),
  (266, 29, 'Apron และผ้ากันเปื้อน', 7, '1'),
  (267, 29, 'ทั้งหมด', 99, '1');

-- sub-categories: cat 30 กระเป๋าและเครื่องหนัง
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (270, 30, 'กระเป๋าถือ / Handbag', 1, '1'),
  (271, 30, 'กระเป๋าเป้ / Backpack', 2, '1'),
  (272, 30, 'กระเป๋าสะพาย / Crossbody', 3, '1'),
  (273, 30, 'กระเป๋าเดินทาง / Luggage', 4, '1'),
  (274, 30, 'กระเป๋าผ้าและ Tote Bag', 5, '1'),
  (275, 30, 'เข็มขัดและกระเป๋าสตางค์', 6, '1'),
  (276, 30, 'ชิ้นงานหนังอื่นๆ (สาย ซอง ฯลฯ)', 7, '1'),
  (277, 30, 'ทั้งหมด', 99, '1');

-- sub-categories: cat 31 หมวก รองเท้า และเครื่องประดับแฟชั่น
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (280, 31, 'หมวกและ Cap OEM', 1, '1'),
  (281, 31, 'รองเท้าผ้าใบและ Casual Shoes', 2, '1'),
  (282, 31, 'รองเท้าแตะและ Sandal', 3, '1'),
  (283, 31, 'ผ้าพันคอและ Scarf', 4, '1'),
  (284, 31, 'เครื่องประดับสังเคราะห์ (Costume Jewelry)', 5, '1'),
  (285, 31, 'ถุงมือและอุปกรณ์แฟชั่นอื่นๆ', 6, '1'),
  (286, 31, 'ทั้งหมด', 99, '1');

-- ============================================================
-- SECTION 3: HUB 9 — ผ้าและอุปกรณ์ตัดเย็บ (scope=MT, sort_order=15)
-- ============================================================
INSERT INTO lbi_hub (hub_id, name, scope, sort_order) VALUES
  (9, 'ผ้าและอุปกรณ์ตัดเย็บ', 'MT', 15);

-- categories hub_id=9
INSERT INTO lbi_categories (category_id, hub_id, name) VALUES
  (32, 9, 'ผ้าทอและผ้าถัก'),
  (33, 9, 'ผ้า Technical และผ้าเฉพาะทาง'),
  (34, 9, 'ด้ายและด้ายปัก'),
  (35, 9, 'ซิป กระดุม และอุปกรณ์ประกอบ'),
  (36, 9, 'วัสดุซับในและฟองน้ำ');

-- sub-categories: cat 32 ผ้าทอและผ้าถัก
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (290, 32, 'ผ้าฝ้าย / Cotton Fabric', 1, '1'),
  (291, 32, 'ผ้าโพลีเอสเตอร์ / Polyester', 2, '1'),
  (292, 32, 'ผ้าลินิน / Linen', 3, '1'),
  (293, 32, 'ผ้าไนลอน / Nylon', 4, '1'),
  (294, 32, 'ผ้าสแปนเด็กซ์ / Spandex-Blend', 5, '1'),
  (295, 32, 'ผ้าวิสคอส / Viscose & Rayon', 6, '1'),
  (296, 32, 'ผ้าถัก / Knit Fabric', 7, '1'),
  (297, 32, 'ผ้าทอยกดอก / Jacquard', 8, '1'),
  (298, 32, 'ผ้าแคนวาส / Canvas', 9, '1'),
  (299, 32, 'ทั้งหมด', 99, '1');

-- sub-categories: cat 33 ผ้า Technical และผ้าเฉพาะทาง
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (300, 33, 'ผ้ากันน้ำ / Waterproof Fabric', 1, '1'),
  (301, 33, 'ผ้าระบายอากาศ / Moisture Wicking', 2, '1'),
  (302, 33, 'ผ้ายีนส์ / Denim', 3, '1'),
  (303, 33, 'ผ้าหนัง PU / PU Leather', 4, '1'),
  (304, 33, 'ผ้าไมโครไฟเบอร์ / Microfiber', 5, '1'),
  (305, 33, 'ผ้า Mesh และอีลาสติก', 6, '1'),
  (306, 33, 'ผ้าทนไฟ / Flame Retardant', 7, '1'),
  (307, 33, 'ทั้งหมด', 99, '1');

-- sub-categories: cat 34 ด้ายและด้ายปัก
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (310, 34, 'ด้ายเย็บ Polyester', 1, '1'),
  (311, 34, 'ด้ายเย็บ Cotton / Nylon', 2, '1'),
  (312, 34, 'ด้ายปักลาย (Embroidery Thread)', 3, '1'),
  (313, 34, 'ด้ายยืดหยุ่น / Elastic Thread', 4, '1'),
  (314, 34, 'ด้ายหลอดสำหรับจักรอุตสาหกรรม', 5, '1'),
  (315, 34, 'ทั้งหมด', 99, '1');

-- sub-categories: cat 35 ซิป กระดุม และอุปกรณ์ประกอบ
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (320, 35, 'ซิปโลหะและซิปพลาสติก', 1, '1'),
  (321, 35, 'กระดุมและด้ายกระดุม', 2, '1'),
  (322, 35, 'ตีนตุ๊กแก / Velcro', 3, '1'),
  (323, 35, 'หัวเข็มขัดและตัวล็อก', 4, '1'),
  (324, 35, 'ห่วงและตะขอเสื้อผ้า', 5, '1'),
  (325, 35, 'แถบยาง / Elastic Band', 6, '1'),
  (326, 35, 'ป้ายและ Label เสื้อผ้า', 7, '1'),
  (327, 35, 'ทั้งหมด', 99, '1');

-- sub-categories: cat 36 วัสดุซับในและฟองน้ำ
INSERT INTO lbi_sub_categories (sub_category_id, category_id, name, sort_order, status) VALUES
  (330, 36, 'ผ้าซับในทั่วไป / Lining Fabric', 1, '1'),
  (331, 36, 'Interlining และผ้ากาว', 2, '1'),
  (332, 36, 'ฟองน้ำและ Padding', 3, '1'),
  (333, 36, 'ใยโพลีเอสเตอร์ (Fiberfill)', 4, '1'),
  (334, 36, 'ผ้าร้อยรัดและเทป', 5, '1'),
  (335, 36, 'ทั้งหมด', 99, '1');

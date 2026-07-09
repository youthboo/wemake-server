-- Migration 025: Seed factory users + profiles + showcases for hub 8 (แฟชั่น PD) + hub 9 (ผ้า MT)

-- ============================================================
-- SECTION 1: USERS (role=FT)
-- ============================================================
INSERT INTO users (role, email, phone, password_hash, is_active) VALUES
  ('FT', 'factory.stylecraft@tryly.dev',   '0822221101', '$2a$10$examplehashStyleCraft01xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.uniformpro@tryly.dev',   '0822221102', '$2a$10$examplehashUniformPro02xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.activewear@tryly.dev',   '0822221103', '$2a$10$examplehashActiveWear03xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.baghouse@tryly.dev',     '0822221104', '$2a$10$examplehashBagHouse004xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.kidswear@tryly.dev',     '0822221105', '$2a$10$examplehashKidsWear005xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.fabricmill@tryly.dev',   '0822221106', '$2a$10$examplehashFabricMill06xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.threadworks@tryly.dev',  '0822221107', '$2a$10$examplehashThreadWork07xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true);

-- ============================================================
-- SECTION 2: FACTORY PROFILES
-- ============================================================
INSERT INTO factory_profiles (
  user_id, approval_status, factory_name, description,
  lead_time_desc, province_id, image_url,
  rating, review_count, completed_orders,
  submitted_at, verified_at
)
SELECT u.user_id, 'AP', p.factory_name, p.description,
  p.lead_time_desc, p.province_id, p.image_url,
  p.rating, p.review_count, p.completed_orders,
  NOW() - INTERVAL '60 days', NOW() - INTERVAL '30 days'
FROM users u
JOIN (VALUES
  ('factory.stylecraft@tryly.dev',
   'สไตล์คราฟท์ แอพพาเรล',
   'โรงงานตัดเย็บเสื้อผ้าแฟชั่นครบวงจร รับ OEM และ Private Label ทุกรูปแบบ ได้มาตรฐาน ISO 9001 มีทีมดีไซเนอร์ในบ้านพร้อมพัฒนาแพทเทิร์นให้ลูกค้า',
   '14–21 วัน', 4,
   'https://images.unsplash.com/photo-1558769132-cb1aea458c5e?w=800&q=80',
   4.75, 61, 88),

  ('factory.uniformpro@tryly.dev',
   'ยูนิฟอร์มโปร แมนูแฟกเจอริ่ง',
   'ผู้เชี่ยวชาญผลิตเครื่องแบบองค์กร ชุดพนักงาน ชุดกีฬาสีของบริษัท ด้วยเทคนิคสกรีน ปัก และซับลิเมชั่น รองรับออเดอร์ตั้งแต่ 50 ชิ้น',
   '7–14 วัน', 1,
   'https://images.unsplash.com/photo-1581091226825-a6a2a5aee158?w=800&q=80',
   4.60, 44, 72),

  ('factory.activewear@tryly.dev',
   'แอคทีฟเวียร์ เทค โปรดักชั่น',
   'โรงงานผลิตชุดกีฬาและ Activewear เฉพาะทาง ผ้าเทคนิคระบายอากาศ กันน้ำ และ Compression สำหรับ Brand กีฬาและ Fitness',
   '14–21 วัน', 11,
   'https://images.unsplash.com/photo-1517836357463-d25dfeac3438?w=800&q=80',
   4.80, 52, 94),

  ('factory.baghouse@tryly.dev',
   'แบ็กเฮ้าส์ ไทยแลนด์',
   'โรงงานผลิตกระเป๋าครบวงจร ทั้งหนัง PU ผ้า Canvas และไนลอน รับออกแบบและผลิต OEM ทุกสไตล์ ส่งออกยุโรปและอเมริกา',
   '21–30 วัน', 1,
   'https://images.unsplash.com/photo-1547949003-9792a18a2601?w=800&q=80',
   4.70, 38, 55),

  ('factory.kidswear@tryly.dev',
   'คิดส์เวียร์ เฟคทอรี่',
   'เชี่ยวชาญผลิตเสื้อผ้าเด็กและชุดนักเรียน ผ้าปลอดสารพิษ มาตรฐาน Oeko-Tex รับออกแบบ OEM และ Private Label สำหรับแบรนด์เด็ก',
   '14–21 วัน', 4,
   'https://images.unsplash.com/photo-1503454537195-1dcabb73ffb9?w=800&q=80',
   4.55, 29, 41),

  ('factory.fabricmill@tryly.dev',
   'แฟบริคมิลล์ ซัพพลาย',
   'ผู้ผลิตและจำหน่ายผ้าอุตสาหกรรมครบวงจร ทั้งผ้าทอ ผ้าถัก และผ้า Technical ส่งตรงจากโรงงาน MOQ ต่ำ เหมาะสำหรับสตาร์ทอัพแฟชั่น',
   '5–10 วัน', 73,
   'https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=800&q=80',
   4.65, 33, 47),

  ('factory.threadworks@tryly.dev',
   'เธรดเวิร์คส อุตสาหกรรม',
   'ผู้ผลิตด้ายและอุปกรณ์ตัดเย็บครบวงจร ด้ายเย็บ ด้ายปัก ซิป กระดุม และ Accessories สำหรับอุตสาหกรรมเสื้อผ้าและกระเป๋า',
   '3–7 วัน', 11,
   'https://images.unsplash.com/photo-1586495777744-4e6232bf2177?w=800&q=80',
   4.50, 22, 35)
) AS p(email, factory_name, description, lead_time_desc, province_id, image_url, rating, review_count, completed_orders)
ON u.email = p.email;

-- ============================================================
-- SECTION 3: MAP CATEGORIES
-- ============================================================
INSERT INTO map_factory_categories (factory_id, category_id)
SELECT u.user_id, m.category_id
FROM users u
JOIN (VALUES
  ('factory.stylecraft@tryly.dev',   25),
  ('factory.stylecraft@tryly.dev',   27),
  ('factory.uniformpro@tryly.dev',   29),
  ('factory.uniformpro@tryly.dev',   25),
  ('factory.activewear@tryly.dev',   26),
  ('factory.baghouse@tryly.dev',     30),
  ('factory.kidswear@tryly.dev',     28),
  ('factory.kidswear@tryly.dev',     27),
  ('factory.fabricmill@tryly.dev',   32),
  ('factory.fabricmill@tryly.dev',   33),
  ('factory.threadworks@tryly.dev',  34),
  ('factory.threadworks@tryly.dev',  35),
  ('factory.threadworks@tryly.dev',  36)
) AS m(email, category_id) ON u.email = m.email;

-- ============================================================
-- SECTION 4: SHOWCASES — hub_id=8 แฟชั่นและเสื้อผ้า (PD)
-- showcase_id เริ่มจาก 328
-- ============================================================
INSERT INTO factory_showcases (
  factory_id, category_id, sub_category_id, hub_id,
  content_type, title, content,
  moq, unit_id, lead_time_days, base_price, status,
  likes_count, published_at, linked_showcases
)
SELECT u.user_id, sc.category_id, sc.sub_category_id, 8,
  sc.content_type, sc.title, sc.content,
  sc.moq::int, sc.unit_id::int, sc.lead_time_days::int,
  sc.base_price::numeric, 'AC',
  sc.likes_count::int, NOW() - (sc.days_ago || ' days')::INTERVAL,
  sc.img::jsonb
FROM users u
JOIN (VALUES
  -- StyleCraft: เสื้อผ้าสำเร็จรูป
  ('factory.stylecraft@tryly.dev', 25, 220, 'PD',
   'เสื้อยืด Cotton 100% OEM ทุก Style',
   'รับผลิตเสื้อยืด Cotton combed 30s/40s ทุกรูปแบบ เนื้อผ้านุ่ม ไม่หดตัว พิมพ์ลาย DTF/DTG/สกรีนได้ MOQ 50 ตัวต่อสี',
   '50', '1', '14', '145.00', '67', '3',
   '["https://images.unsplash.com/photo-1521572163474-6864f9cf17ab?w=800&q=80"]'),

  ('factory.stylecraft@tryly.dev', 25, 225, 'PD',
   'ชุด Co-ord Set ผ้า Linen OEM Private Label',
   'รับผลิตชุดเซ็ต 2 ชิ้น ผ้าลินินผสม ระบายอากาศดี ออกแบบแพทเทิร์นฟรี รับผลิตตั้งแต่ 30 ชุดต่อ Design',
   '30', '2', '21', '420.00', '44', '8',
   '["https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?w=800&q=80"]'),

  ('factory.stylecraft@tryly.dev', 27, 242, 'PD',
   'ชุดนอนและ Loungewear ผ้า Modal OEM',
   'ผลิตชุดนอนผ้า Modal นุ่มพิเศษ ไม่ระคายเคืองผิว รับ OEM ทุก Design พร้อมแพ็กเกจจิ้งแบรนด์ลูกค้า MOQ 50 ชุด',
   '50', '2', '21', '380.00', '31', '12',
   '["https://images.unsplash.com/photo-1545291730-faff8ca1d4b0?w=800&q=80"]'),

  -- StyleCraft: ไอเดีย
  ('factory.stylecraft@tryly.dev', 25, NULL, 'ID',
   'เปิดแบรนด์เสื้อผ้าของตัวเอง: ต้องรู้อะไรบ้างก่อน OEM',
   'การเริ่ม Fashion Brand ด้วย OEM ไม่จำเป็นต้องมีโรงงานเป็นของตัวเอง\n\nสิ่งสำคัญ 3 อย่างที่ต้องเตรียม:\n1. Mood Board และ Reference ที่ชัดเจน\n2. Size Spec และ Technical Drawing\n3. งบประมาณต่อ SKU และ MOQ ที่รับได้\n\nโรงงานที่ดีจะช่วย develop แพทเทิร์นและทำ Sample ให้ก่อนผลิตจริง',
   NULL, NULL, NULL, NULL, '28', '15',
   '["https://images.unsplash.com/photo-1558618666-fcd25c85cd64?w=800&q=80"]'),

  -- UniformPro: ชุดทำงาน
  ('factory.uniformpro@tryly.dev', 29, 260, 'PD',
   'เสื้อโปโลองค์กร ปักโลโก้ OEM ตั้งแต่ 50 ตัว',
   'รับผลิตเสื้อโปโลองค์กร ผ้า TK Pique ระบายอากาศ ปักโลโก้หน้าอก/แขน ออกแบบสีตาม Pantone รับออเดอร์ตั้งแต่ 50 ตัว',
   '50', '1', '10', '189.00', '83', '2',
   '["https://images.unsplash.com/photo-1618354691373-d851c5c3a990?w=800&q=80"]'),

  ('factory.uniformpro@tryly.dev', 29, 263, 'PD',
   'ชุด Scrub บุคลากรทางการแพทย์ OEM',
   'ผลิตชุด Scrub ผ้า Cool Dry ระบายอากาศ ทนต่อการซัก ปักชื่อ/โลโก้ได้ มีให้เลือกทั้งแบบ Unisex และ Fit รับตั้งแต่ 50 ชุด',
   '50', '2', '14', '650.00', '55', '6',
   '["https://images.unsplash.com/photo-1584982751601-97ddc0e26e05?w=800&q=80"]'),

  ('factory.uniformpro@tryly.dev', 29, 266, 'PD',
   'Apron ผ้าแคนวาสและกันน้ำ OEM พิมพ์โลโก้',
   'รับผลิต Apron ผ้า Canvas 12oz และผ้ากันน้ำ Waxed รับพิมพ์โลโก้ ปักลาย หรือสกรีน สีตาม Pantone MOQ 30 ชิ้น',
   '30', '1', '7', '280.00', '39', '10',
   '["https://images.unsplash.com/photo-1607631568010-a87245c0daf8?w=800&q=80"]'),

  -- ActiveWear
  ('factory.activewear@tryly.dev', 26, 230, 'PD',
   'ชุด Gym Wear ผ้า Dry-Fit 4 Way Stretch OEM',
   'ผลิตชุดออกกำลังกาย เสื้อ+กางเกง ผ้า Polyester Spandex 4-way stretch ระบายเหงื่อ กันกลิ่น Anti-Odor MOQ 50 set',
   '50', '2', '21', '520.00', '71', '4',
   '["https://images.unsplash.com/photo-1538805060514-97d9cc17730c?w=800&q=80"]'),

  ('factory.activewear@tryly.dev', 26, 233, 'PD',
   'เสื้อกีฬาทีม Jersey Sublimation พิมพ์เต็มตัว',
   'รับผลิต Jersey กีฬา Sublimation พิมพ์ Full-color ทั้งตัว ผ้า Microfiber เบาสบาย ระบายอากาศดี MOQ เพียง 10 ตัว',
   '10', '1', '14', '350.00', '92', '1',
   '["https://images.unsplash.com/photo-1551698618-1dfe5d97d256?w=800&q=80"]'),

  ('factory.activewear@tryly.dev', 26, 231, 'PD',
   'ชุดว่ายน้ำ Swimwear ผ้า Chlorine Resistant OEM',
   'ผลิตชุดว่ายน้ำผ้า Lycra/Xtra Life ทนคลอรีน ยืดหยุ่นสูง รับออกแบบทุก Style ทั้ง One-piece และ Bikini Set',
   '30', '2', '21', '680.00', '46', '9',
   '["https://images.unsplash.com/photo-1570976447640-ac859083963f?w=800&q=80"]'),

  -- BagHouse
  ('factory.baghouse@tryly.dev', 30, 271, 'PD',
   'กระเป๋าเป้ Canvas OEM ทุก Design ส่งออก',
   'รับผลิตกระเป๋าเป้ผ้า Canvas และ Waxed Canvas ทุก Size ตั้งแต่ 14" ถึง 17" พร้อมออกแบบโครงสร้างภายในตามลูกค้า ส่งออกได้',
   '30', '1', '30', '890.00', '38', '11',
   '["https://images.unsplash.com/photo-1553062407-98eeb64c6a62?w=800&q=80"]'),

  ('factory.baghouse@tryly.dev', 30, 274, 'PD',
   'Tote Bag ผ้า Canvas พิมพ์ลาย OEM ของชำร่วย',
   'รับผลิต Tote Bag ผ้า Canvas 10oz/12oz พิมพ์สกรีน/DTG ใช้เป็นของชำร่วย ของที่ระลึก และ Corporate Gift MOQ 50 ใบ',
   '50', '1', '14', '185.00', '64', '5',
   '["https://images.unsplash.com/photo-1544816155-12df9643f363?w=800&q=80"]'),

  -- KidsWear
  ('factory.kidswear@tryly.dev', 28, 250, 'PD',
   'เสื้อผ้าทารก 0–2 ปี ผ้า Organic Cotton OEM',
   'รับผลิตเสื้อผ้าทารก Romper, Onesie, ชุด Set ผ้า Organic Cotton 100% ปลอดสารพิษ Oeko-Tex Standard 100 MOQ 30 ชิ้น',
   '30', '1', '21', '220.00', '27', '14',
   '["https://images.unsplash.com/photo-1522771930-78848d9293e8?w=800&q=80"]'),

  ('factory.kidswear@tryly.dev', 28, 253, 'PD',
   'ชุดนักเรียน OEM ครบชุด ผ้าคุณภาพสูง',
   'รับผลิตชุดนักเรียนชาย-หญิง เสื้อ กางเกง/กระโปรง ตามสเปคโรงเรียน ผ้า Polyester-Cotton ทนทาน รีดง่าย MOQ 100 ชุด',
   '100', '2', '21', '350.00', '33', '7',
   '["https://images.unsplash.com/photo-1576087503901-b21a04e5a143?w=800&q=80"]'),

  -- StyleCraft: หมวกและรองเท้า
  ('factory.stylecraft@tryly.dev', 31, 280, 'PD',
   'หมวก Cap 5 Panel / 6 Panel OEM ปักโลโก้',
   'รับผลิตหมวก Baseball Cap, Trucker Hat, Beanie ทุกแบบ ปักโลโก้ด้านหน้าและด้านข้าง ผ้า Cotton/Denim/Canvas MOQ 50 ใบ',
   '50', '1', '14', '195.00', '51', '6',
   '["https://images.unsplash.com/photo-1588850561407-ed78c282e89b?w=800&q=80"]')

) AS sc(email, category_id, sub_category_id, content_type, title, content, moq, unit_id, lead_time_days, base_price, likes_count, days_ago, img)
ON u.email = sc.email;

-- ============================================================
-- SECTION 5: SHOWCASES — hub_id=9 ผ้าและอุปกรณ์ตัดเย็บ (MT)
-- ============================================================
INSERT INTO factory_showcases (
  factory_id, category_id, sub_category_id, hub_id,
  content_type, title, content,
  moq, unit_id, lead_time_days, base_price, status,
  likes_count, published_at, linked_showcases
)
SELECT u.user_id, sc.category_id, sc.sub_category_id::bigint, sc.hub_id,
  sc.content_type, sc.title, sc.content,
  sc.moq::int, sc.unit_id::int, sc.lead_time_days::int,
  sc.base_price::numeric, 'AC',
  sc.likes_count::int, NOW() - (sc.days_ago || ' days')::INTERVAL,
  sc.img::jsonb
FROM users u
JOIN (VALUES
  -- FabricMill: ผ้าทอและผ้าถัก
  ('factory.fabricmill@tryly.dev', 32, NULL, 9, 'MT',
   'ผ้าฝ้าย Combed Cotton 30s/40s เกรด A ส่งตรงโรงงาน',
   'ผ้าฝ้าย Combed Cotton บริสุทธิ์ น้ำหนัก 160–200 gsm สีพื้น 100+ สี พร้อมส่งสต็อก MOQ 50 หลา Certificate: Oeko-Tex',
   '50', '10', '5', '85.00', '72', '3',
   '["https://images.unsplash.com/photo-1558769132-cb1aea458c5e?w=800&q=80"]'),

  ('factory.fabricmill@tryly.dev', 32, NULL, 9, 'MT',
   'ผ้าถัก Jersey Interlock Spandex สำหรับ Activewear',
   'ผ้าถัก Poly-Spandex 88/12 น้ำหนัก 200–240 gsm ยืดหยุ่น 4 ทาง เหมาะสำหรับชุดกีฬา ว่ายน้ำ และ Compression Wear',
   '30', '10', '5', '120.00', '58', '7',
   '["https://images.unsplash.com/photo-1617785408427-a9eed9f99d03?w=800&q=80"]'),

  ('factory.fabricmill@tryly.dev', 32, NULL, 9, 'MT',
   'ผ้า Linen เกรดส่งออก น้ำหนักเบา ระบายอากาศดี',
   'ผ้าลินินแท้ 100% น้ำหนัก 120–180 gsm เนื้อละเอียด ระบายอากาศดีเยี่ยม เหมาะสำหรับเสื้อผ้า Summer และชุด Resort',
   '20', '10', '7', '145.00', '44', '10',
   '["https://images.unsplash.com/photo-1558618047-3c8c76ca7d13?w=800&q=80"]'),

  -- FabricMill: ผ้า Technical
  ('factory.fabricmill@tryly.dev', 33, NULL, 9, 'MT',
   'ผ้ากันน้ำ Waterproof Membrane เกรด Outdoor',
   'ผ้า Nylon/Polyester กันน้ำ WR DWR Coating 10,000mm Waterproofing เหมาะสำหรับเสื้อกันฝน Jacket และเครื่องแต่งกาย Outdoor',
   '20', '10', '7', '280.00', '37', '5',
   '["https://images.unsplash.com/photo-1519823551278-64ac92734fb1?w=800&q=80"]'),

  ('factory.fabricmill@tryly.dev', 33, NULL, 9, 'MT',
   'ผ้ายีนส์ Denim 10oz–14oz ทุกน้ำหนัก',
   'ผ้า Denim Cotton 100% และ Stretch Denim (Cotton/Spandex) น้ำหนัก 10oz ถึง 14oz Indigo และ Color Denim MOQ 20 หลา',
   '20', '10', '7', '195.00', '49', '8',
   '["https://images.unsplash.com/photo-1582552938357-32b906df40cb?w=800&q=80"]'),

  -- FabricMill: ไอเดีย MT
  ('factory.fabricmill@tryly.dev', 32, NULL, 9, 'ID',
   'เลือกผ้าอย่างไรให้ตรงกับ Product ของคุณ',
   'การเลือกผ้าให้เหมาะกับสินค้าต้องพิจารณา 4 ปัจจัย:\n\n1. **น้ำหนัก (GSM)** — เสื้อยืดทั่วไป 160–180gsm, กีฬา 180–220gsm\n2. **ความยืดหยุ่น** — ต้องการ stretch ต้องใช้ผ้าที่มี Spandex/Elastane อย่างน้อย 5%\n3. **การดูแลรักษา** — ผ้า Polyester ทนทาน ซักง่าย ผ้าฝ้ายต้องระวังหด\n4. **ใบรับรอง** — สินค้าเด็กต้องการ Oeko-Tex, ส่งออก EU ต้องการ REACH Compliance',
   NULL, NULL, NULL, NULL, '35', '20',
   '["https://images.unsplash.com/photo-1615486364404-00b5e91815e3?w=800&q=80"]'),

  -- ThreadWorks: ด้ายและอุปกรณ์
  ('factory.threadworks@tryly.dev', 34, NULL, 9, 'MT',
   'ด้ายเย็บ Polyester Spun 40/2 ทุกสี MOQ ต่ำ',
   'ด้ายเย็บ Polyester Spun 40/2 และ 60/2 มีให้เลือกกว่า 200 สี พร้อมส่ง สำหรับจักรเย็บ Industrial และ Overlock MOQ 10 โคน',
   '10', '1', '3', '45.00', '63', '4',
   '["https://images.unsplash.com/photo-1596203517565-0c67a2d3c4a1?w=800&q=80"]'),

  ('factory.threadworks@tryly.dev', 34, NULL, 9, 'MT',
   'ด้ายปักลาย Embroidery Thread Rayon/Polyester ครบสี',
   'ด้ายปัก Rayon 40wt และ Polyester 40wt กว่า 500 สี เหมาะสำหรับจักรปัก Brother/Tajima/Barudan MOQ 5 โคนต่อสี',
   '5', '1', '3', '65.00', '48', '6',
   '["https://images.unsplash.com/photo-1615486364404-00b5e91815e3?w=800&q=80"]'),

  -- ThreadWorks: ซิปและอุปกรณ์
  ('factory.threadworks@tryly.dev', 35, NULL, 9, 'MT',
   'ซิป YKK และ Lampo ทุกขนาด ส่งตรงจาก Importer',
   'ซิปโลหะ Brass และ Aluminium ซิปพลาสติก Nylon Coil และ Vislon ทุกขนาด #3 #5 #8 #10 จาก YKK, Lampo, SBS พร้อมส่ง',
   '10', '5', '3', '8.50', '77', '2',
   '["https://images.unsplash.com/photo-1584464491033-06628f3a6b7b?w=800&q=80"]'),

  ('factory.threadworks@tryly.dev', 35, NULL, 9, 'MT',
   'แถบยาง Elastic Band ทุกความกว้าง สำหรับอุตสาหกรรมเสื้อผ้า',
   'แถบยาง Woven Elastic และ Knitted Elastic ความกว้าง 6mm–80mm ความยืดหยุ่น 50–100% เหมาะสำหรับกางเกง ชุดชั้นใน และเสื้อผ้ากีฬา',
   '10', '10', '3', '12.00', '41', '9',
   '["https://images.unsplash.com/photo-1586495777744-4e6232bf2177?w=800&q=80"]'),

  -- ThreadWorks: วัสดุซับใน
  ('factory.threadworks@tryly.dev', 36, NULL, 9, 'MT',
   'Interlining ผ้ากาว Woven/Non-woven ทุกน้ำหนัก',
   'Fusible Interlining แบบ Woven และ Non-woven น้ำหนัก 30–120gsm ใช้ติดกาวด้วยความร้อน เหมาะสำหรับคอเสื้อ ปกเสื้อ และกระเป๋า',
   '20', '10', '5', '38.00', '29', '13',
   '["https://images.unsplash.com/photo-1558618047-3c8c76ca7d13?w=800&q=80"]')

) AS sc(email, category_id, sub_category_id, hub_id, content_type, title, content, moq, unit_id, lead_time_days, base_price, likes_count, days_ago, img)
ON u.email = sc.email;

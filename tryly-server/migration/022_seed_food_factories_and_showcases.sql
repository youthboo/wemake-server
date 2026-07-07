-- Migration 022: Seed factory users + profiles + showcases for อาหารและขนม (hub_id=7)
-- Also seed raw-material (MT) showcases related to food ingredients

-- ============================================================
-- SECTION 1: USERS (role=FT)
-- password_hash = bcrypt of 'Tryly@1234' (placeholder hash)
-- ============================================================
INSERT INTO users (role, email, phone, password_hash, is_active) VALUES
  ('FT', 'factory.foodcraft@tryly.dev',   '0811111101', '$2a$10$examplehashFoodCraft001xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.snackhouse@tryly.dev',  '0811111102', '$2a$10$examplehashSnackHouse002xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.drinkpro@tryly.dev',    '0811111103', '$2a$10$examplehashDrinkPro003xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.sauceland@tryly.dev',   '0811111104', '$2a$10$examplehashSauceLand004xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.dairyplus@tryly.dev',   '0811111105', '$2a$10$examplehashDairyPlus005xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.rawingredient@tryly.dev','0811111106', '$2a$10$examplehashRawIngred006xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true),
  ('FT', 'factory.grainmill@tryly.dev',   '0811111107', '$2a$10$examplehashGrainMill007xxxxxxxxxxxxxxxxxxxxxxxxxxxxx', true);

-- ============================================================
-- SECTION 2: FACTORY PROFILES
-- ============================================================
INSERT INTO factory_profiles (
  user_id, approval_status, factory_name, description,
  lead_time_desc, province_id, rating, review_count, completed_orders,
  submitted_at, verified_at
)
SELECT
  u.user_id,
  'AP',
  p.factory_name,
  p.description,
  p.lead_time_desc,
  p.province_id,
  p.rating,
  p.review_count,
  p.completed_orders,
  NOW() - INTERVAL '60 days',
  NOW() - INTERVAL '30 days'
FROM users u
JOIN (VALUES
  ('factory.foodcraft@tryly.dev',    'ฟู้ดคราฟท์ โปรดักชั่น',       'โรงงานผลิตอาหารแปรรูปและพร้อมทานครบวงจร ได้มาตรฐาน GMP HACCP มีกำลังการผลิต 5 ตันต่อวัน รองรับ OEM และ ODM ทุกรูปแบบ',      '7–14 วัน',  1,  4.70, 38, 54),
  ('factory.snackhouse@tryly.dev',   'สแนคเฮ้าส์ แมนูแฟกเจอริ่ง',  'ผู้ผลิตขนมขบเคี้ยวชั้นนำ ครอบคลุมมันฝรั่ง ถั่ว สาหร่าย และข้าวเกรียบ มีห้องปลอดเชื้อ Clean Room ได้มาตรฐาน FDA',        '10–21 วัน', 13, 4.55, 29, 41),
  ('factory.drinkpro@tryly.dev',     'ดริ๊งค์โปร อินดัสทรี',         'โรงงานผลิตเครื่องดื่ม UHT น้ำผลไม้ และผงชงดื่มครบวงจร ระบบ Aseptic Filling ได้มาตรฐาน ISO 22000',                          '14–21 วัน', 20, 4.80, 52, 73),
  ('factory.sauceland@tryly.dev',    'ซอสแลนด์ โปรดักส์',           'เชี่ยวชาญการผลิตเครื่องปรุงรส ซอส น้ำพริก และพริกแกงสำเร็จรูป ใช้วัตถุดิบไทยแท้ ผ่าน GMP และ HACCP',                     '7–14 วัน',  26, 4.65, 44, 60),
  ('factory.dairyplus@tryly.dev',    'แดร์รี่พลัส โรงงาน',           'ผลิตผลิตภัณฑ์นม โยเกิร์ต และของหวานแช่เย็น ด้วยเทคโนโลยี Cold Chain ครบวงจร ได้มาตรฐาน อย.',                             '14–30 วัน', 1,  4.45, 21, 33),
  ('factory.rawingredient@tryly.dev','รอว์อิงกรีเดียนท์ ซัพพลาย',   'ผู้จำหน่ายและแปรรูปวัตถุดิบอาหาร ครอบคลุมโปรตีน ธัญพืช น้ำมัน และสารสกัดธรรมชาติสำหรับอุตสาหกรรมอาหาร',               '5–10 วัน',  11, 4.60, 16, 28),
  ('factory.grainmill@tryly.dev',    'เกรนมิลล์ อะกริโปรดักส์',     'โรงสีและแปรรูปธัญพืชครบวงจร ข้าว ข้าวโพด ถั่ว แป้งชนิดต่างๆ สำหรับอุตสาหกรรมอาหารและขนม ได้มาตรฐาน GMP',              '7–14 วัน',  16, 4.50, 18, 25)
) AS p(email, factory_name, description, lead_time_desc, province_id, rating, review_count, completed_orders)
ON u.email = p.email;

-- ============================================================
-- SECTION 3: MAP CATEGORIES (hub_id=7 PD categories)
-- ============================================================
INSERT INTO map_factory_categories (factory_id, category_id)
SELECT u.user_id, m.category_id
FROM users u
JOIN (VALUES
  ('factory.foodcraft@tryly.dev',    18),
  ('factory.foodcraft@tryly.dev',    19),
  ('factory.snackhouse@tryly.dev',   23),
  ('factory.snackhouse@tryly.dev',   19),
  ('factory.drinkpro@tryly.dev',     20),
  ('factory.sauceland@tryly.dev',    22),
  ('factory.dairyplus@tryly.dev',    24),
  ('factory.dairyplus@tryly.dev',    19),
  ('factory.rawingredient@tryly.dev',21),
  ('factory.grainmill@tryly.dev',    21)
) AS m(email, category_id) ON u.email = m.email;
-- NOTE: original had JOIN alias bug (c.category_id); fixed to m.category_id above

-- ============================================================
-- SECTION 4: SHOWCASES — PD (hub_id=7 อาหารและขนม)
-- content_type = 'PD' = สินค้า/ผลิตภัณฑ์
-- ============================================================
INSERT INTO factory_showcases (
  factory_id, category_id, sub_category_id, hub_id,
  content_type, title, content,
  moq, unit_id, lead_time_days, base_price, status,
  likes_count, published_at
)
SELECT u.user_id, sc.category_id, sc.sub_category_id, 7,
       sc.content_type, sc.title, sc.content,
       sc.moq, sc.unit_id, sc.lead_time_days, sc.base_price, 'AC',
       sc.likes_count, NOW() - (sc.days_ago || ' days')::INTERVAL
FROM users u
JOIN (VALUES
  -- FoodCraft: อาหารแปรรูป
  ('factory.foodcraft@tryly.dev',    18, 202, 'PD', 'อาหารพร้อมทาน (Ready to Eat) สูตรไทยแท้',
   'รับผลิต RTE ครบครัน ทั้งแกงไทย ต้มยำ ผัดกะเพรา บรรจุถุงรีทอร์ท อายุสินค้า 12 เดือน ได้มาตรฐาน GMP/HACCP',
   500, 2, 14, 85.00, 42, 5),
  ('factory.foodcraft@tryly.dev',    18, 206, 'PD', 'ไส้กรอกไก่สมุนไพร OEM ตราลูกค้า',
   'รับผลิตไส้กรอกไก่ผสมสมุนไพรไทย ปราศจากสารกันบูด บรรจุแวคคั่ม อายุสินค้า 30 วัน MOQ 500 กก.',
   500, 7, 10, 220.00, 38, 12),
  ('factory.foodcraft@tryly.dev',    19, 110, 'PD', 'คุกกี้เนยสด OEM ทุกรูปแบบ',
   'รับผลิตคุกกี้เนยสด ชาเขียว ช็อกโกแลต และสูตรพิเศษตามลูกค้า บรรจุถุง/กล่อง ได้ อย.',
   200, 7, 14, 180.00, 27, 8),

  -- SnackHouse: ขนมขบเคี้ยว
  ('factory.snackhouse@tryly.dev',   23, 151, 'PD', 'ถั่วอบปรุงรสหลากรส OEM',
   'รับผลิตถั่วอบ ถั่วลิสง ถั่วเม็ดมะม่วงหิมพานต์ และส่วนผสมธัญพืช ปรุงรสได้ตามต้องการ FDA อย.',
   300, 7, 21, 320.00, 55, 3),
  ('factory.snackhouse@tryly.dev',   23, 152, 'PD', 'สาหร่ายทอดกรอบรสออริจินัลและวาซาบิ',
   'ผลิตสาหร่ายทอดคุณภาพส่งออก ควบคุมน้ำมันด้วยระบบ Vacuum Frying ลด Trans Fat รับ OEM ขั้นต่ำ 200 กก.',
   200, 7, 14, 480.00, 61, 7),
  ('factory.snackhouse@tryly.dev',   19, 112, 'PD', 'ช็อกโกแลตและลูกกวาดปรุงรสพิเศษ',
   'รับผลิตช็อกโกแลตแท่ง ช็อกโกแลตเคลือบ และลูกกวาดเจลาตินรูปสัตว์ รับออกแบบ packaging ฟรี',
   500, 1, 21, 95.00, 33, 15),

  -- DrinkPro: เครื่องดื่ม
  ('factory.drinkpro@tryly.dev',     20, 120, 'PD', 'น้ำผลไม้ UHT 100% OEM ทุกสูตร',
   'รับผลิตน้ำผลไม้ UHT ส้ม มะม่วง ฝรั่ง มะพร้าว บรรจุกล่อง Tetra Pak หรือขวด PET ตั้งแต่ 200 ml ถึง 1 ลิตร',
   1000, 1, 21, 18.50, 74, 2),
  ('factory.drinkpro@tryly.dev',     20, 123, 'PD', 'ผงชงดื่ม 3-in-1 กาแฟ ชา โกโก้ OEM',
   'รับผลิตผงชงดื่มสูตรลูกค้า กาแฟ ชานม โกโก้ และสมุนไพร ด้วยระบบ Spray Dry คุณภาพสูง',
   500, 7, 14, 290.00, 49, 9),
  ('factory.drinkpro@tryly.dev',     20, 121, 'PD', 'เครื่องดื่มชูกำลังฟังก์ชันนัล OEM',
   'รับผลิต Functional Drink ผสมวิตามิน คอลลาเจน อิเล็กโทรไลต์ บรรจุขวด PET และ Aluminium Can',
   2000, 1, 30, 25.00, 41, 18),

  -- SauceLand: เครื่องปรุง
  ('factory.sauceland@tryly.dev',    22, 142, 'PD', 'พริกแกงสำเร็จรูปครบ 10 สูตร OEM',
   'รับผลิตพริกแกงไทยครบทุกสูตร แกงเขียวหวาน แดง มัสมั่น พะแนง บรรจุถุงซีล หรือบรรจุภัณฑ์ลูกค้า',
   100, 7, 7, 350.00, 36, 4),
  ('factory.sauceland@tryly.dev',    22, 143, 'PD', 'น้ำพริกเผาและซอสพริกสูตรต้นตำรับ',
   'ผลิตน้ำพริกเผา น้ำพริกนรก ซอสพริกศรีราชา และซอสแซ่บ สูตรไทยแท้ ไม่ใส่สีสังเคราะห์',
   200, 7, 7, 195.00, 28, 6),
  ('factory.sauceland@tryly.dev',    22, 147, 'PD', 'ผงปรุงรสและซุปก้อน OEM',
   'รับผลิตผงปรุงรส ผงซุปไก่ ผงซุปหมู และผงกะหรี่สำเร็จรูป บรรจุซอง ขวด หรือถุงใหญ่สำหรับ B2B',
   500, 7, 14, 120.00, 19, 20),

  -- DairyPlus: นมและไข่
  ('factory.dairyplus@tryly.dev',    24, 161, 'PD', 'โยเกิร์ตพร้อมดื่มและโยเกิร์ตถ้วย OEM',
   'รับผลิตโยเกิร์ตครีมมี่ โยเกิร์ตไขมันต่ำ และโยเกิร์ตผลไม้ ระบบ Cold Chain ตลอดสาย อย. ไทย',
   300, 1, 21, 28.00, 22, 11),
  ('factory.dairyplus@tryly.dev',    24, 165, 'PD', 'นมถั่วเหลืองและนมพืช Plant-based OEM',
   'ผลิตนมถั่วเหลือง นมอัลมอนด์ นมข้าว และนมโอ๊ต บรรจุกล่อง UHT และขวด PET สำหรับ private label',
   500, 1, 21, 32.00, 17, 14),

  -- FoodCraft: ไอเดีย (ID)
  ('factory.foodcraft@tryly.dev',    18, NULL, 'ID', 'เทรนด์อาหารพร้อมทานปี 2025 โอกาสสำหรับผู้ผลิต',
   'ตลาด Ready-to-Eat ไทยมูลค่ากว่า 3 หมื่นล้านบาทต่อปี กำลังเติบโตอย่างต่อเนื่องหลัง COVID เปิดโอกาสให้แบรนด์ใหม่เข้าตลาดด้วยการ OEM ต้นทุนต่ำ\n\nสิ่งสำคัญที่ต้องรู้: การเลือกโรงงานที่มี GMP/HACCP จะช่วยให้ขึ้นทะเบียน อย. ได้เร็วขึ้น 50% และเพิ่มความน่าเชื่อถือกับ Modern Trade',
   NULL, NULL, NULL, NULL, 18, 10),

  -- SnackHouse: ไอเดีย (ID)
  ('factory.snackhouse@tryly.dev',   23, NULL, 'ID', 'ขนมขบเคี้ยวสุขภาพ: Functional Snack คือโอกาสที่รอคุณอยู่',
   'ผู้บริโภคไทยหันมาสนใจขนมที่ดีต่อสุขภาพมากขึ้น ทั้ง High Protein ไขมันต่ำ เพิ่มไฟเบอร์ และ Keto\n\nโรงงานที่มีระบบ Clean Room และควบคุมสูตรได้ยืดหยุ่นจะตอบโจทย์แบรนด์ Functional Snack ได้ดีที่สุด',
   NULL, NULL, NULL, NULL, 12, 20),

  -- DrinkPro: วัตถุดิบ (MT)
  ('factory.drinkpro@tryly.dev',     20, 120, 'MT', 'น้ำผลไม้เข้มข้น (Juice Concentrate) สำหรับอุตสาหกรรม',
   'จำหน่าย Juice Concentrate คุณภาพส่งออก มะม่วง ส้ม ฝรั่ง และเสาวรส ค่า Brix 65–70 บรรจุถังอาหาร 200 ลิตร',
   200, 10, 7, 680.00, 31, 22)
) AS sc(email, category_id, sub_category_id, content_type, title, content, moq, unit_id, lead_time_days, base_price, likes_count, days_ago)
ON u.email = sc.email;

-- ============================================================
-- SECTION 5: SHOWCASES — MT วัตถุดิบเกี่ยวกับอาหารคน
-- ใช้ hub_id 3 (เกษตรและธรรมชาติ) และ hub_id 4 (เคมีภัณฑ์และสารเติมแต่ง)
-- factory: rawingredient + grainmill
-- ============================================================
INSERT INTO factory_showcases (
  factory_id, category_id, sub_category_id, hub_id,
  content_type, title, content,
  moq, unit_id, lead_time_days, base_price, status,
  likes_count, published_at
)
SELECT u.user_id, sc.category_id, sc.sub_category_id, sc.hub_id,
       'MT', sc.title, sc.content,
       sc.moq, sc.unit_id, sc.lead_time_days, sc.base_price, 'AC',
       sc.likes_count, NOW() - (sc.days_ago || ' days')::INTERVAL
FROM users u
JOIN (VALUES
  -- RawIngredient: วัตถุดิบโปรตีน (cat 9 hub 3)
  ('factory.rawingredient@tryly.dev', 9, NULL, 3, 'โปรตีนถั่วเหลือง (Soy Protein Isolate) 90%',
   'โปรตีนถั่วเหลือง Isolate บริสุทธิ์ 90% ไม่มี GMO เหมาะสำหรับผสมในอาหารเสริม ขนมโปรตีน และ Functional Food',
   50, 7, 5, 380.00, 44, 6),
  ('factory.rawingredient@tryly.dev', 9, NULL, 3, 'Whey Protein Concentrate (WPC) เกรดอาหาร',
   'WPC 80% นำเข้าจากนิวซีแลนด์ เหมาะสำหรับผลิต Protein Bar เครื่องดื่ม และอาหารเสริมกีฬา',
   25, 7, 7, 520.00, 39, 3),

  -- RawIngredient: สารเติมแต่งอาหาร (cat 12 hub 4)
  ('factory.rawingredient@tryly.dev', 12, NULL, 4, 'สารให้ความหวาน Stevia Extract (95% Rebaudioside-A)',
   'สารให้ความหวานธรรมชาติจากหญ้าหวาน หวานกว่าน้ำตาล 300 เท่า ไม่มีแคลอรี เหมาะสำหรับผลิตภัณฑ์ Sugar-Free',
   10, 7, 7, 950.00, 57, 8),
  ('factory.rawingredient@tryly.dev', 12, NULL, 4, 'Lecithin ถั่วเหลือง (Emulsifier E322) เกรดอาหาร',
   'Soy Lecithin ผง ใช้เป็นอิมัลซิไฟเออร์ในเบเกอรี่ ช็อกโกแลต และนมพืช ได้มาตรฐาน Kosher/Halal',
   20, 7, 5, 280.00, 28, 12),
  ('factory.rawingredient@tryly.dev', 12, NULL, 4, 'Citric Acid (กรดมะนาว) เกรดอาหาร',
   'กรดซิตริกบริสุทธิ์ 99.5% ใช้ปรับ pH ถนอมอาหาร และแต่งรสเปรี้ยว ในเครื่องดื่ม แยม และลูกอม',
   50, 7, 5, 95.00, 22, 17),

  -- GrainMill: วัตถุดิบธัญพืช (cat 10 hub 3)
  ('factory.grainmill@tryly.dev', 10, NULL, 3, 'แป้งข้าวเจ้าบดละเอียด เกรด A สำหรับอุตสาหกรรมอาหาร',
   'แป้งข้าวเจ้า 100% บดละเอียด mesh 100 เหมาะสำหรับขนมไทย เส้นก๋วยเตี๋ยว และผลิตภัณฑ์ Gluten-Free',
   100, 7, 7, 38.00, 63, 4),
  ('factory.grainmill@tryly.dev', 10, NULL, 3, 'แป้งข้าวโพดดัดแปลง (Modified Corn Starch)',
   'Modified Starch คุณสมบัติทนความร้อนสูง ใช้ทำซอส สตาร์ชสำหรับทอด และสารเพิ่มความข้น',
   100, 7, 5, 52.00, 35, 9),
  ('factory.grainmill@tryly.dev', 10, NULL, 3, 'ข้าวโพดบดหยาบ (Corn Grits) สำหรับผลิต Snack Extruded',
   'Corn Grits ขนาดเมล็ด 0.5–1.5 mm เหมาะสำหรับเครื่อง Extruder ผลิตข้าวโพดอบกรอบและขนมขบเคี้ยว',
   500, 7, 7, 28.00, 41, 14),

  -- GrainMill: น้ำมันและไขมัน (cat 14 hub 3)
  ('factory.grainmill@tryly.dev', 14, NULL, 3, 'น้ำมันรำข้าว Cold Press เกรดพิเศษ',
   'น้ำมันรำข้าวบีบเย็น สูง Gamma-Oryzanol จุดควันสูง 250°C เหมาะสำหรับผลิตขนม ทอด และ Salad Dressing',
   50, 10, 5, 85.00, 48, 11),
  ('factory.grainmill@tryly.dev', 14, NULL, 3, 'Palm Olein เกรดอาหาร สำหรับอุตสาหกรรมขนม',
   'น้ำมันปาล์มโอเลอิน บริสุทธิ์ ค่า FFA ต่ำกว่า 0.1% เหมาะสำหรับทอดขนมอุตสาหกรรม มีใบรับรอง RSPO',
   1000, 10, 7, 45.00, 29, 16),

  -- RawIngredient: ไอเดีย MT (ID)
  ('factory.rawingredient@tryly.dev', 12, NULL, 4, 'วัตถุดิบอาหารเสริม: เลือกอย่างไรให้ได้คุณภาพและผ่าน อย.',
   'การเลือกวัตถุดิบสำหรับ Functional Food ต้องคำนึงถึง 3 ปัจจัยหลัก: ความบริสุทธิ์ (Purity) ใบรับรองมาตรฐาน (COA/Halal/Kosher) และ Bioavailability\n\nวัตถุดิบที่มีเอกสารครบจะช่วยให้ขึ้น อย. ได้รวดเร็วและลดความเสี่ยงด้านกฎหมาย',
   NULL, NULL, NULL, NULL, 23, 25)
) AS sc(email, category_id, sub_category_id, hub_id, title, content, moq, unit_id, lead_time_days, base_price, likes_count, days_ago)
ON u.email = sc.email;

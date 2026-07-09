-- Migration 026: Add img column to lbi_categories (relative web path)
-- Path format: /assets/category_img/category_{id}.png
-- Files live in /public/assets/category_img/ — served as static assets by Vite/web server

ALTER TABLE lbi_categories ADD COLUMN IF NOT EXISTS img text;

UPDATE lbi_categories SET img = '/assets/category_img/category_' || category_id || '.png'
WHERE category_id IN (
  SELECT category_id FROM lbi_categories
  WHERE category_id IN (1,2,3,4,5,6,7,9,10,12,13,14,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33)
);

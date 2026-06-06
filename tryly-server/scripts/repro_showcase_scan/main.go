package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/yourusername/wemake/internal/domain"
)

const getDetailSQL = `
		SELECT
			fs.showcase_id, fs.factory_id, fs.content_type,
			fs.title, NULL::text AS excerpt, NULLIF(fs.linked_showcases->>0, '') AS image_url,
			fs.category_id, fs.sub_category_id,
			fs.moq, fs.lead_time_days,
			fs.base_price, fs.promo_price, fs.start_date, fs.end_date,
			fs.content, fs.linked_showcases, '[]'::jsonb AS tags,
			fs.likes_count, 0::bigint AS view_count, fs.status, fs.created_at,
			fs.updated_at, fs.published_at,
			fp.factory_name,
			fp.image_url         AS factory_image_url,
			fp.rating::float8    AS factory_rating,
			(fp.approval_status = 'AP') AS factory_verified,
				fp.description       AS factory_specialization,
			fp.review_count      AS factory_review_count,
			p.name_th            AS province_name,
			c.name               AS category_name,
			sc.name              AS sub_category_name
		FROM factory_showcases fs
		INNER JOIN factory_profiles fp       ON fs.factory_id      = fp.user_id
			LEFT JOIN lbi_categories c              ON fs.category_id     = c.category_id
		LEFT JOIN  lbi_sub_categories sc     ON fs.sub_category_id = sc.sub_category_id
		LEFT JOIN  lbi_provinces p           ON fp.province_id     = p.row_id
		WHERE fs.showcase_id = $1
	`

const exploreSubsetSQL = `
		SELECT
			fs.showcase_id, fs.factory_id, fs.content_type,
			fs.title, NULL::text AS excerpt, NULLIF(fs.linked_showcases->>0, '') AS image_url,
			fs.category_id, fs.sub_category_id,
			fs.moq, fs.lead_time_days,
			fs.base_price, fs.promo_price, fs.start_date, fs.end_date,
			fs.linked_showcases, '[]'::jsonb AS tags,
			fs.likes_count, 0::bigint AS view_count, fs.status, fs.created_at,
			fs.updated_at, fs.published_at,
			fp.factory_name,
			fp.image_url AS factory_image_url,
			fp.rating::float8 AS factory_rating,
			(fp.approval_status = 'AP') AS factory_verified,
			p.name_th AS province_name,
			c.name AS category_name,
			sc.name AS sub_category_name
		FROM factory_showcases fs
		INNER JOIN factory_profiles fp ON fs.factory_id = fp.user_id
		LEFT JOIN lbi_provinces p ON fp.province_id = p.row_id
		LEFT JOIN lbi_categories c ON fs.category_id = c.category_id
		LEFT JOIN lbi_sub_categories sc ON fs.sub_category_id = sc.sub_category_id
		WHERE fs.showcase_id = $1
	`

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	const showcaseID = int64(1)

	fmt.Println("=== Scan into domain.ShowcaseDetail (GetDetail SQL) ===")
	var detail domain.ShowcaseDetail
	err = db.Get(&detail, getDetailSQL, showcaseID)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		fmt.Printf("ERROR type: %T\n", err)
	} else {
		fmt.Printf("OK: showcase_id=%d title=%q content_type=%q\n", detail.ShowcaseID, detail.Title, detail.ContentType)
	}

	fmt.Println()
	fmt.Println("=== Scan into domain.ShowcaseExploreItem (explore-like columns) ===")
	var explore domain.ShowcaseExploreItem
	err = db.Get(&explore, exploreSubsetSQL, showcaseID)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		fmt.Printf("ERROR type: %T\n", err)
	} else {
		fmt.Printf("OK: showcase_id=%d title=%q content_type=%q\n", explore.ShowcaseID, explore.Title, explore.ContentType)
	}
}

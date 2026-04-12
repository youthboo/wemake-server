// Package jobs contains background goroutines that run on a timer.
// Started from main.go via jobs.Start(db).
package jobs

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// Start launches all background jobs. Call once from main.go after DB is ready.
// Each job runs in its own goroutine and loops forever until the process exits.
func Start(db *sqlx.DB) {
	go runExpiration(db)
	go runMatchingNotifications(db)
}

// --------------------------------------------------------------------------
// Expiration job — runs every hour
// --------------------------------------------------------------------------

// runExpiration auto-closes overdue RFQs and quotations.
func runExpiration(db *sqlx.DB) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Run once immediately on start, then every hour.
	expireRFQs(db)
	expireQuotations(db)

	for range ticker.C {
		expireRFQs(db)
		expireQuotations(db)
	}
}

// expireRFQs sets status = 'EX' for open RFQs whose deadline_date < today.
func expireRFQs(db *sqlx.DB) {
	res, err := db.Exec(`
		UPDATE rfqs
		SET status = 'EX', updated_at = NOW()
		WHERE status = 'OP'
		  AND deadline_date IS NOT NULL
		  AND deadline_date < CURRENT_DATE
	`)
	if err != nil {
		log.Printf("[jobs/expiration] expireRFQs error: %v", err)
		return
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		log.Printf("[jobs/expiration] expired %d RFQ(s)", n)
	}
}

// expireQuotations sets status = 'EX' for pending (PD) quotations older than 7 days
// with no response from the customer.
func expireQuotations(db *sqlx.DB) {
	res, err := db.Exec(`
		UPDATE quotations
		SET status = 'EX', log_timestamp = NOW()
		WHERE status = 'PD'
		  AND COALESCE(is_locked, false) = false
		  AND create_time < NOW() - INTERVAL '7 days'
	`)
	if err != nil {
		log.Printf("[jobs/expiration] expireQuotations error: %v", err)
		return
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		log.Printf("[jobs/expiration] expired %d quotation(s)", n)
	}
}

// --------------------------------------------------------------------------
// Auto-matching notification job — runs every 5 minutes
// --------------------------------------------------------------------------

// runMatchingNotifications checks for new open RFQs and sends a notification
// to each factory whose category mapping matches the RFQ.
func runMatchingNotifications(db *sqlx.DB) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sendMatchingNotifications(db)
	}
}

// sendMatchingNotifications finds RFQs created in the last 6 minutes (slightly
// longer than the ticker interval to tolerate drift) and notifies matching factories
// that have not yet received a notification for that RFQ.
func sendMatchingNotifications(db *sqlx.DB) {
	type rfqRow struct {
		RFQID         int64  `db:"rfq_id"`
		Title         string `db:"title"`
		SubCategoryID *int64 `db:"sub_category_id"`
		CategoryID    int64  `db:"category_id"`
	}

	var newRFQs []rfqRow
	err := db.Select(&newRFQs, `
		SELECT rfq_id, title, sub_category_id, category_id
		FROM rfqs
		WHERE status = 'OP'
		  AND created_at >= NOW() - INTERVAL '6 minutes'
	`)
	if err != nil {
		log.Printf("[jobs/matching] query new RFQs error: %v", err)
		return
	}
	if len(newRFQs) == 0 {
		return
	}

	for _, rfq := range newRFQs {
		notifyMatchingFactories(db, rfq.RFQID, rfq.CategoryID, rfq.SubCategoryID, rfq.Title)
	}
}

func notifyMatchingFactories(db *sqlx.DB, rfqID, categoryID int64, subCategoryID *int64, rfqTitle string) {
	// Find factories whose category (and optional sub-category) matches this RFQ
	// and have not already been notified for this RFQ.
	type factoryRow struct {
		UserID int64 `db:"user_id"`
	}
	var factories []factoryRow

	var subCatArg interface{}
	if subCategoryID != nil {
		subCatArg = *subCategoryID
	}

	err := db.Select(&factories, `
		SELECT DISTINCT mfc.factory_id AS user_id
		FROM map_factory_categories mfc
		INNER JOIN users u ON u.user_id = mfc.factory_id AND u.role = 'FT' AND u.is_active = TRUE
		WHERE mfc.category_id = $1
		  AND (
			$2::bigint IS NULL
			OR EXISTS (
				SELECT 1 FROM map_factory_sub_categories ms
				WHERE ms.factory_id = mfc.factory_id
				  AND ms.sub_category_id = $2
			)
		  )
		  AND NOT EXISTS (
			SELECT 1 FROM notifications n
			WHERE n.user_id = mfc.factory_id
			  AND n.reference_type = 'RFQ'
			  AND n.reference_id = $3
		  )
	`, categoryID, subCatArg, rfqID)
	if err != nil {
		log.Printf("[jobs/matching] query matching factories error: %v", err)
		return
	}
	if len(factories) == 0 {
		return
	}

	title := "มี RFQ ใหม่ตรงกับหมวดของคุณ"
	body := "มีคำขอ RFQ ใหม่: " + rfqTitle + " กดเพื่อดูรายละเอียด"

	for _, f := range factories {
		_, err := db.Exec(`
			INSERT INTO notifications (user_id, title, body, reference_type, reference_id, is_read)
			VALUES ($1, $2, $3, 'RFQ', $4, FALSE)
		`, f.UserID, title, body, rfqID)
		if err != nil {
			log.Printf("[jobs/matching] insert notification error (factory %d, rfq %d): %v", f.UserID, rfqID, err)
		}
	}
	log.Printf("[jobs/matching] sent notifications for RFQ %d to %d factory/factories", rfqID, len(factories))
}

// Package jobs contains background goroutines that run on a timer.
// Started from main.go via jobs.Start(db).
package jobs

import (
	"encoding/json"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/repository"
	"github.com/yourusername/wemake/internal/service"
)

// Start launches all background jobs. Call once from main.go after DB is ready.
// Each job runs in its own goroutine and loops forever until the process exits.
func Start(db *sqlx.DB) {
	orderService := service.NewOrderService(
		db,
		repository.NewOrderRepository(db),
		nil,
		repository.NewWalletRepository(db),
		repository.NewTransactionRepository(db),
		nil,
		nil,
		repository.NewReviewRepository(db),
		nil,
		nil,
	)
	go runExpiration(db)
	go runOrderAutoClose(orderService)
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
	expirePendingDeposits(db)

	for range ticker.C {
		expireRFQs(db)
		expireQuotations(db)
		expirePendingDeposits(db)
	}
}

func runOrderAutoClose(orderService *service.OrderService) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	if n, err := orderService.AutoCloseShippedOrders(); err != nil {
		log.Printf("[jobs/order-auto-close] error: %v", err)
	} else if n > 0 {
		log.Printf("[jobs/order-auto-close] auto-closed %d order(s)", n)
	}

	for range ticker.C {
		if n, err := orderService.AutoCloseShippedOrders(); err != nil {
			log.Printf("[jobs/order-auto-close] error: %v", err)
		} else if n > 0 {
			log.Printf("[jobs/order-auto-close] auto-closed %d order(s)", n)
		}
	}
}

func expireRFQs(db *sqlx.DB) {
	// RFQ expiry by deadline_date is disabled because the legacy deadline column was removed.
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

func expirePendingDeposits(db *sqlx.DB) {
	type orderRow struct {
		OrderID int64 `db:"order_id"`
	}

	var rows []orderRow
	err := db.Select(&rows, `
		WITH expired_orders AS (
			SELECT o.order_id
			FROM orders o
			WHERE o.status = 'PP'
			  AND COALESCE(
				(
					SELECT ps.due_date::timestamp + TIME '23:59:59'
					FROM payment_schedules ps
					WHERE ps.order_id = o.order_id
					ORDER BY ps.installment_no ASC, ps.schedule_id ASC
					LIMIT 1
				),
				o.created_at + INTERVAL '3 days'
			  ) < NOW()
		)
		UPDATE orders o
		SET status = 'PE',
		    updated_at = NOW()
		FROM expired_orders e
		WHERE o.order_id = e.order_id
		RETURNING o.order_id
	`)
	if err != nil {
		log.Printf("[jobs/expiration] expirePendingDeposits error: %v", err)
		return
	}
	for _, row := range rows {
		payload, _ := json.Marshal(map[string]interface{}{"order_id": row.OrderID})
		if _, err := db.Exec(`INSERT INTO domain_events (event_type, payload) VALUES ($1, $2)`, "order.deposit_expired", payload); err != nil {
			log.Printf("[jobs/expiration] deposit_expired event error (order %d): %v", row.OrderID, err)
		}
	}
	if len(rows) > 0 {
		log.Printf("[jobs/expiration] expired %d pending deposit order(s)", len(rows))
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

// Package jobs contains background goroutines that run on a timer.
// Started from main.go via jobs.Start(db).
package jobs

import (
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/logger"
	orderrepo "github.com/yourusername/wemake/internal/repository/order"
	userrepo "github.com/yourusername/wemake/internal/repository/user"
	walletrepo "github.com/yourusername/wemake/internal/repository/wallet"
	orderservice "github.com/yourusername/wemake/internal/service/order"
)

// Start launches all background jobs. Call once from main.go after DB is ready.
// Each job runs in its own goroutine and loops forever until the process exits.
func Start(db *sqlx.DB) {
	orderService := orderservice.NewOrderService(
		db,
		orderrepo.NewOrderRepository(db),
		nil,
		walletrepo.NewWalletRepository(db),
		walletrepo.NewTransactionRepository(db),
		nil,
		nil,
		userrepo.NewReviewRepository(db),
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
// RFQ expiration runs once per day at midnight (Bangkok time).
// Quotation and deposit expiration still runs every hour.
func runExpiration(db *sqlx.DB) {
	hourlyTicker := time.NewTicker(1 * time.Hour)
	defer hourlyTicker.Stop()

	// Run once immediately on start.
	expireRFQs(db)
	expireQuotations(db)
	expirePendingDeposits(db)

	bangkok, _ := time.LoadLocation("Asia/Bangkok")
	if bangkok == nil {
		bangkok = time.FixedZone("ICT", 7*3600)
	}

	nextMidnight := func() time.Time {
		now := time.Now().In(bangkok)
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, bangkok)
		return next
	}

	midnightTimer := time.NewTimer(time.Until(nextMidnight()))
	defer midnightTimer.Stop()

	for {
		select {
		case <-hourlyTicker.C:
			expireQuotations(db)
			expirePendingDeposits(db)
		case <-midnightTimer.C:
			expireRFQs(db)
			midnightTimer.Reset(time.Until(nextMidnight()))
		}
	}
}

func runOrderAutoClose(orderService *orderservice.OrderService) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	if n, err := orderService.AutoCloseShippedOrders(); err != nil {
		logger.Error("order auto-close job failed", "err", err)
	} else if n > 0 {
		logger.Info("order auto-close job completed", "closed_count", n)
	}

	for range ticker.C {
		if n, err := orderService.AutoCloseShippedOrders(); err != nil {
			logger.Error("order auto-close job failed", "err", err)
		} else if n > 0 {
			logger.Info("order auto-close job completed", "closed_count", n)
		}
	}
}

// expireRFQs auto-cancels OP RFQs whose expired_date has passed.
// expired_date is set on creation as created_at + tconfig('rfq_expired') days.
// This represents system-initiated cancellation (status CC).
// Any pending quotations on those RFQs are also expired (PD → EX).
func expireRFQs(db *sqlx.DB) {
	// Step 1: collect RFQ IDs that are about to be cancelled so we can expire their quotations.
	type rfqRow struct {
		RFQID int64 `db:"rfq_id"`
	}
	var staleRFQs []rfqRow
	err := db.Select(&staleRFQs, `
		SELECT rfq_id
		FROM rfqs
		WHERE status = 'OP'
		  AND expired_date IS NOT NULL
		  AND expired_date < NOW()
	`)
	if err != nil {
		logger.Error("rfq auto-cancel: stale rfq query failed", "err", err)
		return
	}
	if len(staleRFQs) == 0 {
		return
	}

	ids := make([]int64, 0, len(staleRFQs))
	for _, r := range staleRFQs {
		ids = append(ids, r.RFQID)
	}

	// Step 2: expire pending quotations for those RFQs.
	query, args, err := sqlx.In(`
		UPDATE quotations
		SET status = 'EX', log_timestamp = NOW()
		WHERE rfq_id IN (?)
		  AND status = 'PD'
	`, ids)
	if err == nil {
		query = db.Rebind(query)
		if _, err2 := db.Exec(query, args...); err2 != nil {
			logger.Error("rfq auto-cancel: quotation expiry failed", "err", err2)
		}
	}

	// Step 3: cancel the RFQs.
	cancelQuery, cancelArgs, err := sqlx.In(`
		UPDATE rfqs
		SET status = 'CC', updated_at = NOW()
		WHERE rfq_id IN (?)
		  AND status = 'OP'
	`, ids)
	if err != nil {
		logger.Error("rfq auto-cancel: cancel query build failed", "err", err)
		return
	}
	cancelQuery = db.Rebind(cancelQuery)
	res, err := db.Exec(cancelQuery, cancelArgs...)
	if err != nil {
		logger.Error("rfq auto-cancel job failed", "err", err)
		return
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		logger.Info("rfq auto-cancel job completed", "cancelled_count", n, "from_status", "OP", "to_status", "CC", "reason", "expired_date_passed")
	}
}

// expireQuotations sets status = 'EX' for pending (PD) quotations where the validity window has passed.
// Expiry is determined by:
//   - valid_until column (set when factory creates/edits the quotation)
//   - fallback: create_time + COALESCE(validity_days, 7) days
//
// After expiring quotations, it also auto-closes (OP → CL) any RFQ whose
// remaining quotations are all EX/RJ (no active PD or AC left).
func expireQuotations(db *sqlx.DB) {
	// ขั้น 1: PD → EX เมื่อ validity window หมดแล้ว
	res, err := db.Exec(`
		UPDATE quotations
		SET status = 'EX', log_timestamp = NOW()
		WHERE status = 'PD'
		  AND COALESCE(is_locked, false) = false
		  AND COALESCE(
		        valid_until,
		        (create_time + COALESCE(validity_days, 7) * INTERVAL '1 day')::date
		      ) < NOW()::date
	`)
	if err != nil {
		logger.Error("quotation expiration job failed", "err", err)
		return
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		logger.Info("quotation expiration job completed", "expired_count", n, "from_status", "PD", "to_status", "EX")
	}

	// RFQ stays OP until the customer explicitly closes it (→ CL) or the system
	// auto-cancels it 30 days after creation (→ CC via expireRFQs).
	// Do NOT auto-close based on quotation state — customers may order from
	// multiple quotations and should remain in control of when the RFQ is closed.
}

func expirePendingDeposits(db *sqlx.DB) {
	type orderRow struct {
		OrderID int64 `db:"order_id"`
	}

	// ขั้น 1: PP → PE เมื่อครบกำหนดชำระ
	var peRows []orderRow
	err := db.Select(&peRows, `
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
		logger.Error("pending deposit expiration job failed", "err", err, "from_status", "PP", "to_status", "PE")
		return
	}
	for _, row := range peRows {
		payload, _ := json.Marshal(map[string]interface{}{"order_id": row.OrderID})
		if _, err := db.Exec(`INSERT INTO domain_events (event_type, payload) VALUES ($1, $2)`, "order.deposit_expired", payload); err != nil {
			logger.Error("deposit expired domain event insert failed", "order_id", row.OrderID, "err", err)
		}
	}
	if len(peRows) > 0 {
		logger.Info("pending deposit expiration job completed", "expired_count", len(peRows), "from_status", "PP", "to_status", "PE")
	}

	// ขั้น 2: PE → CL เมื่อผ่าน grace period (due_date + 3 วัน) แล้วยังไม่ได้ชำระ
	// พร้อมกันนั้น unlock quotation กลับเป็น PD เพื่อให้ลูกค้า re-order ได้จาก quotation ใบเดิม
	type clOrderRow struct {
		OrderID int64 `db:"order_id"`
		QuoteID int64 `db:"quote_id"`
	}
	var clRows []clOrderRow
	err = db.Select(&clRows, `
		WITH overdue_orders AS (
			SELECT o.order_id, o.quote_id
			FROM orders o
			WHERE o.status = 'PE'
			  AND COALESCE(
				(
					SELECT ps.due_date::timestamp + TIME '23:59:59'
					FROM payment_schedules ps
					WHERE ps.order_id = o.order_id
					ORDER BY ps.installment_no ASC, ps.schedule_id ASC
					LIMIT 1
				),
				o.created_at + INTERVAL '3 days'
			  ) + INTERVAL '3 days' < NOW()
		)
		UPDATE orders o
		SET status = 'CL',
		    updated_at = NOW()
		FROM overdue_orders e
		WHERE o.order_id = e.order_id
		RETURNING o.order_id, o.quote_id
	`)
	if err != nil {
		logger.Error("expired deposit cancellation job failed", "err", err, "from_status", "PE", "to_status", "CL")
		return
	}
	for _, row := range clRows {
		// Unlock quotation กลับเป็น PD ให้ลูกค้า re-order ได้จาก quotation ใบเดิม
		// (quotation จะ expire เองตาม logic expireQuotations ถ้าไม่มีการสั่งซื้อใหม่ภายใน 7 วัน)
		if _, err := db.Exec(`
			UPDATE quotations
			SET status = 'PD', is_locked = FALSE, log_timestamp = NOW()
			WHERE quote_id = $1 AND status = 'AC'
		`, row.QuoteID); err != nil {
			logger.Error("quotation unlock failed during deposit cancellation", "order_id", row.OrderID, "quote_id", row.QuoteID, "err", err)
		}
		payload, _ := json.Marshal(map[string]interface{}{"order_id": row.OrderID, "quote_id": row.QuoteID})
		if _, err := db.Exec(`INSERT INTO domain_events (event_type, payload) VALUES ($1, $2)`, "order.auto_cancelled", payload); err != nil {
			logger.Error("auto cancelled domain event insert failed", "order_id", row.OrderID, "quote_id", row.QuoteID, "err", err)
		}
	}
	if len(clRows) > 0 {
		logger.Info("expired deposit cancellation job completed", "cancelled_count", len(clRows), "from_status", "PE", "to_status", "CL")
	}

	// ขั้น 3: WS → CL — ลูกค้าไม่แนบสลีปภายในกำหนดชำระ (due date ของงวดแรก
	// ตั้งไว้ตอนสร้าง order = created_at + 3 วัน) ให้ยกเลิกคำสั่งซื้ออัตโนมัติ
	// พร้อม unlock quotation กลับเป็น PD เช่นเดียวกับ flow PE → CL
	var wsRows []clOrderRow
	err = db.Select(&wsRows, `
		WITH overdue_orders AS (
			SELECT o.order_id, o.quote_id
			FROM orders o
			WHERE o.status = 'WS'
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
		SET status = 'CL',
		    updated_at = NOW()
		FROM overdue_orders e
		WHERE o.order_id = e.order_id
		RETURNING o.order_id, o.quote_id
	`)
	if err != nil {
		logger.Error("wait-slip expiration job failed", "err", err, "from_status", "WS", "to_status", "CL")
		return
	}
	for _, row := range wsRows {
		if _, err := db.Exec(`
			UPDATE quotations
			SET status = 'PD', is_locked = FALSE, log_timestamp = NOW()
			WHERE quote_id = $1 AND status = 'AC'
		`, row.QuoteID); err != nil {
			logger.Error("quotation unlock failed during wait-slip cancellation", "order_id", row.OrderID, "quote_id", row.QuoteID, "err", err)
		}
		payload, _ := json.Marshal(map[string]interface{}{"order_id": row.OrderID, "quote_id": row.QuoteID})
		if _, err := db.Exec(`INSERT INTO domain_events (event_type, payload) VALUES ($1, $2)`, "order.auto_cancelled", payload); err != nil {
			logger.Error("auto cancelled domain event insert failed", "order_id", row.OrderID, "quote_id", row.QuoteID, "err", err)
		}
	}
	if len(wsRows) > 0 {
		logger.Info("wait-slip expiration job completed", "cancelled_count", len(wsRows), "from_status", "WS", "to_status", "CL")
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
		logger.Error("matching notification rfq query failed", "err", err)
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
			  AND n.type = 'RFQ'
			  AND n.link_to = '/factory/rfqs/' || $3::text
		  )
	`, categoryID, subCatArg, rfqID)
	if err != nil {
		logger.Error("matching notification factory query failed", "rfq_id", rfqID, "category_id", categoryID, "err", err)
		return
	}
	if len(factories) == 0 {
		return
	}

	title := "มี RFQ ใหม่ตรงกับหมวดของคุณ"
	body := "มีคำขอ RFQ ใหม่: " + rfqTitle + " กดเพื่อดูรายละเอียด"

	for _, f := range factories {
		_, err := db.Exec(`
			INSERT INTO notifications (user_id, type, title, message, link_to, is_read)
			VALUES ($1, 'RFQ', $2, $3, '/factory/rfqs/' || $4::text, FALSE)
		`, f.UserID, title, body, rfqID)
		if err != nil {
			logger.Error("matching notification insert failed", "factory_id", f.UserID, "rfq_id", rfqID, "err", err)
		}
	}
}

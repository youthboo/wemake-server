package cron

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/yourusername/wemake/internal/mailer"
	adminrepo "github.com/yourusername/wemake/internal/repository/admin"
)

// CommissionCron generates monthly commission invoices and emails factories.
type CommissionCron struct {
	db       *sqlx.DB
	invoices *adminrepo.CommissionInvoiceRepository
	mail     *mailer.Mailer
	stopCh   chan struct{}
}

func NewCommissionCron(db *sqlx.DB, invoices *adminrepo.CommissionInvoiceRepository, mail *mailer.Mailer) *CommissionCron {
	return &CommissionCron{db: db, invoices: invoices, mail: mail, stopCh: make(chan struct{})}
}

// Start runs the cron loop. Call in a goroutine.
func (c *CommissionCron) Start() {
	log.Println("[CRON] commission cron started")
	c.tick() // run once on startup
	for {
		next := c.nextRun()
		log.Printf("[CRON] next commission run at %s", next.Format("2006-01-02 15:04"))
		select {
		case <-time.After(time.Until(next)):
			c.tick()
		case <-c.stopCh:
			log.Println("[CRON] commission cron stopped")
			return
		}
	}
}

func (c *CommissionCron) Stop() {
	close(c.stopCh)
}

// nextRun calculates the next run time: 1st of next month at the configured hour.
func (c *CommissionCron) nextRun() time.Time {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)

	hour := 9 // default: 9 AM
	if raw := c.configVal("cron_commission_hour"); raw != "" {
		if h, err := strconv.Atoi(raw); err == nil && h >= 0 && h <= 23 {
			hour = h
		}
	}

	// 1st of next month at configured hour
	next := time.Date(now.Year(), now.Month()+1, 1, hour, 0, 0, 0, loc)
	// If we're already past the 1st of current month at that hour and haven't run yet,
	// run today (catches startup on the 1st)
	firstThisMonth := time.Date(now.Year(), now.Month(), 1, hour, 0, 0, 0, loc)
	if now.After(firstThisMonth) && !c.alreadyRanThisMonth(now) {
		return now.Add(10 * time.Second) // run almost immediately
	}
	return next
}

// alreadyRanThisMonth checks if invoices for the previous month already exist.
func (c *CommissionCron) alreadyRanThisMonth(now time.Time) bool {
	prev := now.AddDate(0, -1, 0)
	var count int
	_ = c.db.Get(&count, `SELECT COUNT(*) FROM commission_invoices WHERE period_month = $1 AND period_year = $2`,
		int(prev.Month()), prev.Year())
	return count > 0
}

func (c *CommissionCron) configVal(key string) string {
	var val string
	_ = c.db.Get(&val, `SELECT value FROM tconfig WHERE key = $1`, key)
	return strings.TrimSpace(val)
}

// tick generates invoices for the previous month and emails all factories.
func (c *CommissionCron) tick() {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	prev := now.AddDate(0, -1, 0)
	month := int(prev.Month())
	year := prev.Year()

	log.Printf("[CRON] generating commission invoices for %d/%d", month, year)

	created, err := c.invoices.GenerateForMonth(month, year)
	if err != nil {
		log.Printf("[CRON] generate error: %v", err)
		return
	}
	log.Printf("[CRON] invoices created: %d (period %d/%d)", created, month, year)

	// Fetch all draft invoices for this period and send email to each factory
	invoices, err := c.invoices.ListAll(month, year)
	if err != nil {
		log.Printf("[CRON] list error: %v", err)
		return
	}

	webURL := c.mail.WebURL()
	sent := 0
	for _, inv := range invoices {
		status := strings.TrimSpace(inv.Status)
		if status != "DR" {
			continue // already sent or processed
		}

		// Mark as sent
		if err := c.invoices.MarkSent(inv.InvoiceID); err != nil {
			log.Printf("[CRON] mark sent error invoice=%d: %v", inv.InvoiceID, err)
			continue
		}

		// Send email
		factoryEmail := c.mail.UserEmail(inv.FactoryID)
		if factoryEmail == "" {
			log.Printf("[CRON] no email for factory_id=%d, skipping", inv.FactoryID)
			continue
		}
		factoryName := c.mail.FactoryName(inv.FactoryID)

		c.mail.SendAsync("COMMISSION_INVOICE", factoryEmail, map[string]string{
			"FactoryName":      factoryName,
			"PeriodMonth":      fmt.Sprintf("%d", inv.PeriodMonth),
			"PeriodYear":       fmt.Sprintf("%d", inv.PeriodYear),
			"TotalOrders":      fmt.Sprintf("%d", inv.TotalOrders),
			"CommissionAmount": fmt.Sprintf("%.2f", inv.CommissionAmount),
			"VatAmount":        fmt.Sprintf("%.2f", inv.VatAmount),
			"GrandTotal":       fmt.Sprintf("%.2f", inv.GrandTotal),
			"Link":             webURL + fmt.Sprintf("/factory/invoices/%d", inv.InvoiceID),
		}, "invoice", inv.InvoiceID)
		sent++
	}

	log.Printf("[CRON] commission emails sent: %d (period %d/%d)", sent, month, year)
}

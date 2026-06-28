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

// CronJobConfig holds schedule config from tcronjob table.
type CronJobConfig struct {
	ScheduleType  string // "day_of_month" | "day_of_week"
	ScheduleValue string // "1"-"28" or "MON","TUE","WED","THU","FRI","SAT","SUN"
	Hour          int
	Enabled       bool
}

var weekdays = map[string]time.Weekday{
	"SUN": time.Sunday, "MON": time.Monday, "TUE": time.Tuesday,
	"WED": time.Wednesday, "THU": time.Thursday, "FRI": time.Friday, "SAT": time.Saturday,
}

// CommissionCron generates commission invoices and emails factories.
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
	cfg := c.loadJobConfig("commission_invoice")
	if !cfg.Enabled {
		log.Println("[CRON] commission_invoice job disabled, skipping startup tick")
	} else {
		c.tick(cfg)
	}
	for {
		cfg = c.loadJobConfig("commission_invoice")
		if !cfg.Enabled {
			log.Println("[CRON] commission_invoice disabled — sleeping 1h before recheck")
			select {
			case <-time.After(time.Hour):
				continue
			case <-c.stopCh:
				return
			}
		}
		next := c.nextRun(cfg)
		log.Printf("[CRON] next commission run at %s (type=%s value=%s hour=%d)",
			next.Format("2006-01-02 15:04"), cfg.ScheduleType, cfg.ScheduleValue, cfg.Hour)
		select {
		case <-time.After(time.Until(next)):
			cfg = c.loadJobConfig("commission_invoice")
			if cfg.Enabled {
				c.tick(cfg)
			}
		case <-c.stopCh:
			log.Println("[CRON] commission cron stopped")
			return
		}
	}
}

func (c *CommissionCron) Stop() { close(c.stopCh) }

// loadJobConfig reads schedule config from tcronjob table.
func (c *CommissionCron) loadJobConfig(jobKey string) CronJobConfig {
	cfg := CronJobConfig{ScheduleType: "day_of_month", ScheduleValue: "1", Hour: 9, Enabled: true}
	var row struct {
		ScheduleType  string `db:"schedule_type"`
		ScheduleValue string `db:"schedule_value"`
		Hour          int    `db:"hour"`
		Enabled       bool   `db:"enabled"`
	}
	if err := c.db.Get(&row, `SELECT schedule_type, schedule_value, hour, enabled FROM tcronjob WHERE job_key = $1`, jobKey); err == nil {
		cfg.ScheduleType = strings.TrimSpace(row.ScheduleType)
		cfg.ScheduleValue = strings.ToUpper(strings.TrimSpace(row.ScheduleValue))
		cfg.Hour = row.Hour
		cfg.Enabled = row.Enabled
	}
	return cfg
}

// nextRun calculates the next run time based on tcronjob config.
func (c *CommissionCron) nextRun(cfg CronJobConfig) time.Time {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	h := cfg.Hour

	switch cfg.ScheduleType {
	case "daily":
		// Every day at configured hour
		todayTarget := time.Date(now.Year(), now.Month(), now.Day(), h, 0, 0, 0, loc)
		if now.Before(todayTarget) && !c.alreadyRanToday(now) {
			return todayTarget
		}
		if now.After(todayTarget) && !c.alreadyRanToday(now) {
			return now.Add(10 * time.Second)
		}
		// Already ran today → schedule tomorrow
		return time.Date(now.Year(), now.Month(), now.Day()+1, h, 0, 0, 0, loc)

	case "day_of_week":
		wd, ok := weekdays[cfg.ScheduleValue]
		if !ok {
			wd = time.Friday
		}
		daysUntil := int(wd) - int(now.Weekday())
		if daysUntil < 0 {
			daysUntil += 7
		}
		if daysUntil == 0 && now.Hour() >= h {
			daysUntil = 7
		}
		target := time.Date(now.Year(), now.Month(), now.Day()+daysUntil, h, 0, 0, 0, loc)
		todayTarget := time.Date(now.Year(), now.Month(), now.Day(), h, 0, 0, 0, loc)
		if now.Weekday() == wd && now.Before(todayTarget) && !c.alreadyRanToday(now) {
			return todayTarget
		}
		if now.Weekday() == wd && now.After(todayTarget) && !c.alreadyRanToday(now) {
			return now.Add(10 * time.Second)
		}
		return target

	default: // day_of_month
		day := 1
		if d, err := strconv.Atoi(cfg.ScheduleValue); err == nil && d >= 1 && d <= 28 {
			day = d
		}
		thisMonth := time.Date(now.Year(), now.Month(), day, h, 0, 0, 0, loc)
		nextMonth := time.Date(now.Year(), now.Month()+1, day, h, 0, 0, 0, loc)

		if now.Before(thisMonth) {
			return thisMonth
		}
		// Startup: ran today or this period already?
		if now.Day() == day && now.Hour() < h && !c.alreadyRanThisMonth(now) {
			return thisMonth
		}
		if now.Day() == day && !c.alreadyRanThisMonth(now) {
			return now.Add(10 * time.Second)
		}
		return nextMonth
	}
}

func (c *CommissionCron) alreadyRanThisMonth(now time.Time) bool {
	prev := now.AddDate(0, -1, 0)
	var count int
	_ = c.db.Get(&count, `SELECT COUNT(*) FROM commission_invoices WHERE period_month=$1 AND period_year=$2`,
		int(prev.Month()), prev.Year())
	return count > 0
}

func (c *CommissionCron) alreadyRanToday(now time.Time) bool {
	var lastRun *time.Time
	_ = c.db.Get(&lastRun, `SELECT last_run_at FROM tcronjob WHERE job_key='commission_invoice'`)
	if lastRun == nil {
		return false
	}
	return lastRun.Year() == now.Year() && lastRun.YearDay() == now.YearDay()
}

func (c *CommissionCron) markLastRun() {
	_, _ = c.db.Exec(`UPDATE tcronjob SET last_run_at=NOW(), updated_at=NOW() WHERE job_key='commission_invoice'`)
}

func (c *CommissionCron) configVal(key string) string {
	var val string
	_ = c.db.Get(&val, `SELECT value FROM tconfig WHERE key=$1`, key)
	return strings.TrimSpace(val)
}

// tick generates invoices for the previous month and emails all factories.
func (c *CommissionCron) tick(cfg CronJobConfig) {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	if loc == nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)
	prev := now.AddDate(0, -1, 0)
	month := int(prev.Month())
	year := prev.Year()

	log.Printf("[CRON] generating commission invoices for %d/%d", month, year)
	c.markLastRun()

	created, err := c.invoices.GenerateForMonth(month, year)
	if err != nil {
		log.Printf("[CRON] generate error: %v", err)
		return
	}
	log.Printf("[CRON] invoices created: %d (period %d/%d)", created, month, year)

	invoices, err := c.invoices.ListAll(month, year)
	if err != nil {
		log.Printf("[CRON] list error: %v", err)
		return
	}

	webURL := c.mail.WebURL()
	bankName := c.configVal("tryly_bank_name")
	bankAccountNo := c.configVal("tryly_bank_account_no")
	accountHolder := c.configVal("tryly_account_holder")
	promptPay := c.configVal("tryly_promptpay")

	sent := 0
	for _, inv := range invoices {
		status := strings.TrimSpace(inv.Status)
		if status != "DR" {
			continue
		}
		if err := c.invoices.MarkSent(inv.InvoiceID); err != nil {
			log.Printf("[CRON] mark sent error invoice=%d: %v", inv.InvoiceID, err)
			continue
		}
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
			"BankName":         bankName,
			"BankAccountNo":    bankAccountNo,
			"AccountHolder":    accountHolder,
			"PromptPay":        promptPay,
		}, "invoice", inv.InvoiceID)
		sent++
	}
	log.Printf("[CRON] commission emails sent: %d (period %d/%d)", sent, month, year)
}

package mailer

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Mailer sends transactional emails via SMTP and logs every attempt.
type Mailer struct {
	db *sqlx.DB
}

// New creates a Mailer that reads SMTP config from tconfig on every send.
func New(db *sqlx.DB) *Mailer {
	return &Mailer{db: db}
}

// smtpConfig holds SMTP settings read from tconfig.
type smtpConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func (m *Mailer) loadConfig() (*smtpConfig, error) {
	rows, err := m.db.Queryx(`SELECT key, value FROM tconfig WHERE key IN ('smtp_host','smtp_port','smtp_user','smtp_password','smtp_from')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cfg := &smtpConfig{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			continue
		}
		switch k {
		case "smtp_host":
			cfg.Host = v
		case "smtp_port":
			cfg.Port = v
		case "smtp_user":
			cfg.User = v
		case "smtp_password":
			cfg.Password = v
		case "smtp_from":
			cfg.From = v
		}
	}
	if cfg.Host == "" || cfg.User == "" || cfg.Password == "" {
		return nil, fmt.Errorf("SMTP config incomplete: host=%q user=%q", cfg.Host, cfg.User)
	}
	if cfg.From == "" {
		cfg.From = cfg.User
	}
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	return cfg, nil
}

// Template holds a template loaded from tmail_templates.
type Template struct {
	Code    string
	Subject string
	Body    string
}

func (m *Mailer) loadTemplate(code string) (*Template, error) {
	t := &Template{Code: code}
	err := m.db.QueryRow(`SELECT subject, body FROM tmail_templates WHERE template_code = $1`, code).Scan(&t.Subject, &t.Body)
	if err != nil {
		return nil, fmt.Errorf("template %q not found: %w", code, err)
	}
	return t, nil
}

// render replaces {{.Key}} placeholders in text with values from data map.
func render(text string, data map[string]string) string {
	result := text
	for k, v := range data {
		result = strings.ReplaceAll(result, "{{."+k+"}}", v)
	}
	return result
}

// sendLog records a mail send attempt in mail_logs.
func (m *Mailer) sendLog(templateCode, recipient, subject, body, status, errMsg, refType string, refID int64) {
	_, _ = m.db.Exec(`
		INSERT INTO mail_logs (template_code, recipient, subject, body, status, error_message, ref_type, ref_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, templateCode, recipient, subject, body, status, errMsg, refType, refID)
}

// Send sends an email using template_code with data placeholders, and logs the result.
// refType/refID are for traceability (e.g. "order", orderID).
func (m *Mailer) Send(templateCode string, to string, data map[string]string, refType string, refID int64) error {
	tmpl, err := m.loadTemplate(templateCode)
	if err != nil {
		m.sendLog(templateCode, to, "", "", "FAIL", err.Error(), refType, refID)
		return err
	}

	subject := render(tmpl.Subject, data)
	body := render(tmpl.Body, data)

	cfg, err := m.loadConfig()
	if err != nil {
		log.Printf("[MAILER] SMTP config error: %v (template=%s to=%s)", err, templateCode, to)
		m.sendLog(templateCode, to, subject, body, "FAIL", err.Error(), refType, refID)
		return err
	}

	// Build HTML email with UTF-8 + base64-encoded subject and body for Thai chars
	encodedBody := base64.StdEncoding.EncodeToString([]byte(body))
	msg := "From: " + cfg.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: =?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?=\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		encodedBody

	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
	addr := cfg.Host + ":" + cfg.Port

	err = smtp.SendMail(addr, auth, cfg.User, []string{to}, []byte(msg))
	if err != nil {
		log.Printf("[MAILER] send failed: %v (to=%s template=%s)", err, to, templateCode)
		m.sendLog(templateCode, to, subject, body, "FAIL", err.Error(), refType, refID)
		return err
	}

	log.Printf("[MAILER] sent OK (to=%s template=%s ref=%s/%d)", to, templateCode, refType, refID)
	m.sendLog(templateCode, to, subject, body, "OK", "", refType, refID)
	return nil
}

// SendAsync sends email in a goroutine — fire and forget, errors logged.
func (m *Mailer) SendAsync(templateCode string, to string, data map[string]string, refType string, refID int64) {
	go func() {
		if err := m.Send(templateCode, to, data, refType, refID); err != nil {
			log.Printf("[MAILER] async send error: %v", err)
		}
	}()
}

// WebURL returns public_web_url from tconfig.
func (m *Mailer) WebURL() string {
	var url string
	_ = m.db.Get(&url, `SELECT value FROM tconfig WHERE key = 'public_web_url'`)
	if url == "" {
		return "https://tryly-web.vercel.app"
	}
	return strings.TrimRight(url, "/")
}

// UserEmail returns a user's email by user_id.
func (m *Mailer) UserEmail(userID int64) string {
	var email string
	_ = m.db.Get(&email, `SELECT email FROM users WHERE user_id = $1`, userID)
	return email
}

// FactoryName returns factory_name for a given factory user_id.
func (m *Mailer) FactoryName(factoryUserID int64) string {
	var name string
	_ = m.db.Get(&name, `SELECT COALESCE(factory_name, '') FROM factories WHERE user_id = $1`, factoryUserID)
	return name
}

// SAEmails returns all super-admin emails for admin notifications.
func (m *Mailer) SAEmails() []string {
	var emails []string
	_ = m.db.Select(&emails, `SELECT email FROM users WHERE TRIM(role) = 'SA'`)
	return emails
}

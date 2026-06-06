package mailer

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	htmltemplate "html/template"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
	texttemplate "text/template"
	"time"

	"github.com/jmoiron/sqlx"
)

const smtpTimeout = 30 * time.Second

// Mailer sends transactional emails via SMTP and logs every attempt.
type Mailer struct {
	db *sqlx.DB
}

// New creates a Mailer for transactional email.
func New(db *sqlx.DB) *Mailer {
	return &Mailer{db: db}
}

// smtpConfig holds SMTP settings read from environment variables.
type smtpConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func (m *Mailer) loadConfig() (*smtpConfig, error) {
	cfg := &smtpConfig{
		Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		Port:     getEnv("SMTP_PORT", "587"),
		User:     strings.TrimSpace(os.Getenv("SMTP_USER")),
		Password: strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		From:     strings.TrimSpace(os.Getenv("SMTP_FROM")),
	}
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	if cfg.From == "" {
		cfg.From = cfg.User
	}
	if cfg.Host == "" || cfg.User == "" || cfg.Password == "" {
		return nil, fmt.Errorf("SMTP config incomplete: host=%q user=%q", cfg.Host, cfg.User)
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// Template holds an in-code mail template.
type Template struct {
	Code    string
	Subject string
	Body    string
}

func (m *Mailer) loadTemplate(code string) (*Template, error) {
	t, ok := mailTemplates[code]
	if !ok {
		return nil, fmt.Errorf("template %q not found", code)
	}
	t.Code = code
	return &t, nil
}

func renderSubject(code string, text string, data map[string]string) (string, error) {
	tmpl, err := texttemplate.New(code + "_subject").Option("missingkey=error").Parse(text)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func renderBody(code string, text string, data map[string]string) (string, error) {
	tmpl, err := htmltemplate.New(code + "_body").Option("missingkey=error").Parse(text)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
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

	subject, err := renderSubject(templateCode, tmpl.Subject, data)
	if err != nil {
		m.sendLog(templateCode, to, "", "", "FAIL", err.Error(), refType, refID)
		return err
	}
	body, err := renderBody(templateCode, tmpl.Body, data)
	if err != nil {
		m.sendLog(templateCode, to, subject, "", "FAIL", err.Error(), refType, refID)
		return err
	}

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

	err = sendSMTP(cfg, to, []byte(msg))
	if err != nil {
		log.Printf("[MAILER] send failed: %v (to=%s template=%s)", err, to, templateCode)
		m.sendLog(templateCode, to, subject, body, "FAIL", err.Error(), refType, refID)
		return err
	}

	log.Printf("[MAILER] sent OK (to=%s template=%s ref=%s/%d)", to, templateCode, refType, refID)
	m.sendLog(templateCode, to, subject, body, "OK", "", refType, refID)
	return nil
}

func sendSMTP(cfg *smtpConfig, to string, msg []byte) error {
	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
	dialer := &net.Dialer{Timeout: smtpTimeout}

	var conn net.Conn
	var err error
	if cfg.Port == "465" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			ServerName: cfg.Host,
			MinVersion: tls.VersionTLS12,
		})
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(smtpTimeout))

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	if cfg.Port != "465" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{
				ServerName: cfg.Host,
				MinVersion: tls.VersionTLS12,
			}); err != nil {
				return err
			}
		}
	}

	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(cfg.User); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(msg); err != nil {
		_ = writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// SendAsync sends email in a goroutine — fire and forget, errors logged.
func (m *Mailer) SendAsync(templateCode string, to string, data map[string]string, refType string, refID int64) {
	go func() {
		if err := m.Send(templateCode, to, data, refType, refID); err != nil {
			log.Printf("[MAILER] async send error: %v", err)
		}
	}()
}

// WebURL returns the public web URL from environment variables.
func (m *Mailer) WebURL() string {
	url := getEnv("PUBLIC_WEB_URL", getEnv("PUBLIC_BASE_URL", "https://tryly-web.vercel.app"))
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

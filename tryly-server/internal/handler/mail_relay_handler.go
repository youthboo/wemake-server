package handler

import (
	"crypto/subtle"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/mailer"
)

type MailRelayHandler struct {
	mail *mailer.Mailer
}

func NewMailRelayHandler(mail *mailer.Mailer) *MailRelayHandler {
	return &MailRelayHandler{mail: mail}
}

type mailRelayRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	HTML    string `json:"html"`
}

// Send accepts trusted relay requests and sends them through the local SMTP config.
func (h *MailRelayHandler) Send(c *fiber.Ctx) error {
	expectedToken := strings.TrimSpace(os.Getenv("MAIL_RELAY_TOKEN"))
	if expectedToken == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "mail relay disabled"})
	}

	authHeader := strings.TrimSpace(c.Get("Authorization"))
	token := strings.TrimSpace(c.Get("X-Mail-Relay-Token"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req mailRelayRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	req.To = strings.TrimSpace(req.To)
	req.Subject = strings.TrimSpace(req.Subject)
	if req.To == "" || req.Subject == "" || strings.TrimSpace(req.HTML) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "to, subject, and html are required"})
	}

	if err := h.mail.SendRawSMTP(req.To, req.Subject, req.HTML); err != nil {
		log.Printf("[MAIL_RELAY] SMTP send failed: %v (to=%s)", err, req.To)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": err.Error()})
	}

	log.Printf("[MAIL_RELAY] sent OK (to=%s)", req.To)
	return c.JSON(fiber.Map{"status": "ok"})
}

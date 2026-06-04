package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/wemake/internal/helper"
)

// RequireJWT rejects requests that did not supply a valid Bearer JWT.
// X-User-ID header bypass is explicitly blocked here, so admin routes
// cannot be reached by anyone who cannot sign a real token.
func RequireJWT(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := strings.TrimSpace(c.Get("Authorization"))
		if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			return helper.WriteAPIError(c, helper.UnauthorizedAPIError("MISSING_TOKEN", "bearer token required"))
		}
		rawToken := strings.TrimSpace(authHeader[7:])
		userID, role, ok := parseUserAndRoleFromToken(rawToken, jwtSecret)
		if !ok || userID <= 0 {
			return helper.WriteAPIError(c, helper.UnauthorizedAPIError("INVALID_TOKEN", "invalid or expired token"))
		}
		// Set verified locals so downstream RequireRole reads from JWT, not X-User-ID
		c.Locals("user_id", userID)
		if role != "" {
			c.Locals("role", role)
		}
		return c.Next()
	}
}

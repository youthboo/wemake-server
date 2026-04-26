package middleware

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func AuthContext(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if userID, ok := parseUserIDFromHeader(c.Get("X-User-ID")); ok {
			c.Locals("user_id", userID)
			return c.Next()
		}

		authHeader := strings.TrimSpace(c.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			rawToken := strings.TrimSpace(authHeader[7:])
			if userID, role, ok := parseUserAndRoleFromToken(rawToken, jwtSecret); ok {
				c.Locals("user_id", userID)
				if role != "" {
					c.Locals("role", role)
				}
			}
		}

		return c.Next()
	}
}

func parseUserIDFromHeader(raw string) (int64, bool) {
	if raw == "" {
		return 0, false
	}
	userID, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, false
	}
	return userID, true
}

func parseUserAndRoleFromToken(rawToken string, jwtSecret string) (int64, string, bool) {
	token, err := jwt.Parse(rawToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return 0, "", false
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", false
	}
	role, _ := claims["role"].(string)

	switch value := claims["user_id"].(type) {
	case float64:
		return int64(value), role, true
	case string:
		userID, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, "", false
		}
		return userID, role, true
	default:
		return 0, "", false
	}
}

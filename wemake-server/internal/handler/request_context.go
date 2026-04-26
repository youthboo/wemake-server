package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func getUserIDFromHeader(c *fiber.Ctx) (int64, error) {
	if localValue := c.Locals("user_id"); localValue != nil {
		switch value := localValue.(type) {
		case int64:
			return value, nil
		case int:
			return int64(value), nil
		case string:
			return strconv.ParseInt(value, 10, 64)
		}
	}

	userIDStr := c.Get("X-User-ID")
	return strconv.ParseInt(userIDStr, 10, 64)
}

func getOptionalUserIDFromHeader(c *fiber.Ctx) int64 {
	userID, err := getUserIDFromHeader(c)
	if err != nil {
		return 0
	}
	return userID
}

func getOptionalRoleFromContext(c *fiber.Ctx) string {
	if localValue := c.Locals("role"); localValue != nil {
		if value, ok := localValue.(string); ok {
			return strings.TrimSpace(strings.ToUpper(value))
		}
	}
	return ""
}

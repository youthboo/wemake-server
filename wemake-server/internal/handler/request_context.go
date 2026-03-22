package handler

import (
	"strconv"

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

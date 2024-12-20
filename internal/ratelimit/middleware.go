package ratelimit

import (
	"github.com/gofiber/fiber/v2"
)

// Middleware creates a new rate limit middleware
func Middleware(limiter Limiter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		result, err := limiter.Allow(c)
		if err != nil {
			return err
		}

		if result.Limited {
			// Add rate limit headers if configured
			for header, value := range result.LimitHeaders {
				c.Set(header, value)
			}
			return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded")
		}

		// Add rate limit headers
		for header, value := range result.LimitHeaders {
			c.Set(header, value)
		}

		return c.Next()
	}
}

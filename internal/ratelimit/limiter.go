package ratelimit

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tuncerburak97/muhtar/internal/config"
)

// Result represents the result of a rate limit check
type Result struct {
	Limited      bool              // Whether the request is rate limited
	Remaining    int               // Remaining requests in the current window
	ResetTime    time.Time         // When the current window resets
	RetryAfter   time.Duration     // How long to wait before retrying
	LimitHeaders map[string]string // Rate limit headers to include in response
}

// Key represents a rate limit key
type Key struct {
	IP       string
	Path     string
	Method   string
	Group    string
	ClientID string // For API key based limiting
	UserID   string // For user based limiting
}

// Store defines the interface for rate limit storage
type Store interface {
	// Get retrieves the current count and window for a key
	Get(ctx context.Context, key string) (int, time.Time, error)

	// Increment increments the counter for a key and returns the new count
	Increment(ctx context.Context, key string, window time.Time) (int, error)

	// Reset resets the counter for a key
	Reset(ctx context.Context, key string) error

	// Close closes the store connection
	Close() error
}

// Limiter defines the interface for rate limiting
type Limiter interface {
	// Allow checks if a request should be allowed
	Allow(c *fiber.Ctx) (*Result, error)

	// Reset resets the rate limit for a specific key
	Reset(key *Key) error

	// Close closes the rate limiter and its resources
	Close() error
}

// Config represents the configuration for a rate limiter
type Config struct {
	RateLimit *config.RateLimitConfig
	Store     Store
}

// Headers for rate limiting
const (
	HeaderRateLimit     = "X-RateLimit-Limit"
	HeaderRateRemaining = "X-RateLimit-Remaining"
	HeaderRateReset     = "X-RateLimit-Reset"
	HeaderRetryAfter    = "Retry-After"
)

// Error types
var (
	ErrRateLimitExceeded  = fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded")
	ErrStorageUnavailable = fiber.NewError(fiber.StatusServiceUnavailable, "rate limit storage unavailable")
)

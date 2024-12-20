package ratelimit

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tuncerburak97/muhtar/internal/config"
)

// Service implements the Limiter interface
type Service struct {
	config *config.RateLimitConfig
	store  Store
}

// NewService creates a new rate limiter service
func NewService(cfg *config.RateLimitConfig, store Store) *Service {
	return &Service{
		config: cfg,
		store:  store,
	}
}

// Allow implements the Limiter interface
func (s *Service) Allow(c *fiber.Ctx) (*Result, error) {
	if !s.config.Enabled {
		return &Result{Limited: false}, nil
	}

	// Build rate limit key
	key := s.buildKey(c)

	// Check IP whitelist
	if s.config.PerIP.Enabled {
		ip := c.IP()
		if s.isWhitelisted(ip) {
			return &Result{Limited: false}, nil
		}
	}

	// Find matching route limit
	routeLimit := s.findRouteLimit(c.Method(), c.Path())

	// Apply rate limits in order: Route -> IP -> Global
	var result *Result
	var err error

	if routeLimit != nil {
		result, err = s.checkLimit(c.Context(), key.withSuffix("route"), routeLimit.Requests, routeLimit.Window, routeLimit.Burst)
		if err != nil || result.Limited {
			return result, err
		}
	}

	if s.config.PerIP.Enabled {
		result, err = s.checkLimit(c.Context(), key.withSuffix("ip"), s.config.PerIP.Requests, s.config.PerIP.Window, s.config.PerIP.Burst)
		if err != nil || result.Limited {
			return result, err
		}
	}

	result, err = s.checkLimit(c.Context(), key.withSuffix("global"), s.config.Global.Requests, s.config.Global.Window, s.config.Global.Burst)
	if err != nil || result.Limited {
		return result, err
	}

	return result, nil
}

// Reset implements the Limiter interface
func (s *Service) Reset(key *Key) error {
	return s.store.Reset(context.Background(), key.String())
}

// Close implements the Limiter interface
func (s *Service) Close() error {
	return s.store.Close()
}

// Helper methods

func (s *Service) buildKey(c *fiber.Ctx) *Key {
	return &Key{
		IP:     c.IP(),
		Path:   c.Path(),
		Method: c.Method(),
	}
}

func (s *Service) isWhitelisted(ip string) bool {
	for _, whitelistedIP := range s.config.PerIP.WhiteList {
		if strings.Contains(whitelistedIP, "/") {
			// CIDR notation
			_, ipNet, err := net.ParseCIDR(whitelistedIP)
			if err != nil {
				continue
			}
			if ipNet.Contains(net.ParseIP(ip)) {
				return true
			}
		} else {
			// Single IP
			if ip == whitelistedIP {
				return true
			}
		}
	}
	return false
}

func (s *Service) findRouteLimit(method, path string) *config.RouteLimit {
	var bestMatch *config.RouteLimit
	var bestPriority int
	var bestPattern string

	for _, route := range s.config.Routes {
		if route.Method != "*" && route.Method != method {
			continue
		}

		if ok := s.pathMatch(route.Path, path); !ok {
			continue
		}

		// If this is our first match or has higher priority
		if bestMatch == nil || route.Priority > bestPriority {
			bestMatch = &route
			bestPriority = route.Priority
			bestPattern = route.Path
			continue
		}

		// If same priority, more specific path wins
		if route.Priority == bestPriority && len(route.Path) > len(bestPattern) {
			bestMatch = &route
			bestPattern = route.Path
		}
	}

	return bestMatch
}

func (s *Service) pathMatch(pattern, path string) bool {
	if pattern == path {
		return true
	}

	if !strings.Contains(pattern, "*") {
		return false
	}

	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i := range patternParts {
		if patternParts[i] == "*" {
			continue
		}
		if patternParts[i] != pathParts[i] {
			return false
		}
	}

	return true
}

func (s *Service) checkLimit(ctx context.Context, key string, limit int, window time.Duration, burst int) (*Result, error) {
	count, resetTime, err := s.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	// If this is a new window
	if time.Now().After(resetTime) {
		resetTime = time.Now().Add(window)
		count = 0
	}

	// Check if we're within limits
	if count >= limit+burst {
		retryAfter := resetTime.Sub(time.Now())
		return &Result{
			Limited:    true,
			Remaining:  0,
			ResetTime:  resetTime,
			RetryAfter: retryAfter,
			LimitHeaders: map[string]string{
				HeaderRateLimit:     strconv.Itoa(limit),
				HeaderRateRemaining: "0",
				HeaderRateReset:     strconv.FormatInt(resetTime.Unix(), 10),
				HeaderRetryAfter:    strconv.FormatInt(int64(retryAfter.Seconds()), 10),
			},
		}, nil
	}

	// Increment counter
	newCount, err := s.store.Increment(ctx, key, resetTime)
	if err != nil {
		return nil, err
	}

	remaining := limit + burst - newCount
	if remaining < 0 {
		remaining = 0
	}

	return &Result{
		Limited:    false,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAfter: 0,
		LimitHeaders: map[string]string{
			HeaderRateLimit:     strconv.Itoa(limit),
			HeaderRateRemaining: strconv.Itoa(remaining),
			HeaderRateReset:     strconv.FormatInt(resetTime.Unix(), 10),
		},
	}, nil
}

// Key helper methods

func (k *Key) String() string {
	parts := []string{k.Method, k.Path}
	if k.IP != "" {
		parts = append(parts, k.IP)
	}
	if k.Group != "" {
		parts = append(parts, k.Group)
	}
	if k.ClientID != "" {
		parts = append(parts, k.ClientID)
	}
	if k.UserID != "" {
		parts = append(parts, k.UserID)
	}
	return strings.Join(parts, ":")
}

func (k *Key) withSuffix(suffix string) string {
	return fmt.Sprintf("%s:%s", k.String(), suffix)
}

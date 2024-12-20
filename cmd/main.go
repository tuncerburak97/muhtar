package main

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tuncerburak97/muhtar/internal/config"
	"github.com/tuncerburak97/muhtar/internal/logger"
	"github.com/tuncerburak97/muhtar/internal/metrics"
	"github.com/tuncerburak97/muhtar/internal/proxy"
	"github.com/tuncerburak97/muhtar/internal/ratelimit"
	"github.com/tuncerburak97/muhtar/internal/repository"
)

func main() {
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Initialize logger
	logger.Init(cfg.Log.Level)
	log := logger.GetLogger()

	// Initialize metrics collector
	metricsCollector := metrics.GetMetricsCollector("muhtar", "muhtar_proxy")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	})

	// Create repository
	repo, err := repository.NewRepository(&cfg.DB)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create repository")
	}

	// Run migrations
	if err := repo.Migrate(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Database migrations failed")
	}
	log.Info().Msg("Database migrations completed successfully")

	// Initialize rate limiter
	var rateLimiter ratelimit.Limiter
	if cfg.RateLimit.Enabled {
		var store ratelimit.Store
		if cfg.RateLimit.Storage.Type == "redis" {
			store, err = ratelimit.NewRedisStore(
				cfg.RateLimit.Storage.Redis.Host,
				cfg.RateLimit.Storage.Redis.Port,
				cfg.RateLimit.Storage.Redis.Password,
				cfg.RateLimit.Storage.Redis.DB,
				cfg.RateLimit.Storage.Redis.Timeout,
			)
		} else {
			store = ratelimit.NewMemoryStore(5 * time.Minute)
		}
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create rate limit store")
		}
		rateLimiter = ratelimit.NewService(&cfg.RateLimit, store)
	}

	// Create proxy handler
	proxyHandler, err := proxy.NewProxyHandler(&cfg.Proxy, log, repo, metricsCollector)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create proxy handler")
	}

	// Setup routes
	app.Get("/metrics", func(c *fiber.Ctx) error {
		jsonData, err := metricsCollector.GetMetricsJSON()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to get metrics: %v", err),
			})
		}

		c.Set("Content-Type", "application/json")
		return c.Send(jsonData)
	})

	// Apply rate limit middleware if enabled
	if cfg.RateLimit.Enabled {
		app.Use(ratelimit.Middleware(rateLimiter))
	}

	app.Use("/", proxyHandler.Handle)

	// Start server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Info().Msgf("Starting server at: %s", serverAddr)
	if err := app.Listen(serverAddr); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}

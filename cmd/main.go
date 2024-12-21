package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/tuncerburak97/muhtar/internal/config"
	"github.com/tuncerburak97/muhtar/internal/metrics"
	"github.com/tuncerburak97/muhtar/internal/proxy"
	"github.com/tuncerburak97/muhtar/internal/ratelimit"
	"github.com/tuncerburak97/muhtar/internal/repository"
	"github.com/tuncerburak97/muhtar/internal/transform"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config/config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Configure logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if cfg.Log.Format == "json" {
		log.Logger = log.Output(os.Stdout)
	}
	level, err := zerolog.ParseLevel(cfg.Log.Level)
	if err != nil {
		log.Warn().Err(err).Msg("Invalid log level, defaulting to info")
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Initialize metrics collector
	metricsCollector := metrics.GetMetricsCollector("muhtar", "muhtar_proxy")

	// Initialize repository
	repo, err := repository.NewRepository(cfg.DB)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize repository")
	}

	// Initialize rate limiter if enabled
	var rateLimiter *ratelimit.Service
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
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize rate limiter")
		}
	}

	// Initialize transform engine
	transformEngine, err := transform.NewEngine(cfg.Proxy.Transform)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize transform engine")
	}

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	})

	// Add rate limiting middleware if enabled
	if rateLimiter != nil {
		app.Use(ratelimit.Middleware(rateLimiter))
	}

	// Initialize and set up proxy handler
	proxyHandler, err := proxy.NewProxyHandler(&cfg.Proxy, &log.Logger, repo, metricsCollector, transformEngine)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize proxy handler")
	}

	// Set up routes
	app.All("/*", proxyHandler.Handle)

	// Start server
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := app.Listen(addr); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatal().Err(err).Msg("Failed to shutdown server")
	}

	// Close resources
	if err := repo.Close(); err != nil {
		log.Error().Err(err).Msg("Failed to close repository")
	}

	if rateLimiter != nil {
		if err := rateLimiter.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close rate limiter")
		}
	}
}

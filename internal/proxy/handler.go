package proxy

import (
	"bytes"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/tuncerburak97/muhtar/internal/config"
	"github.com/tuncerburak97/muhtar/internal/metrics"
	"github.com/tuncerburak97/muhtar/internal/model"
	"github.com/tuncerburak97/muhtar/internal/repository"
	"github.com/tuncerburak97/muhtar/internal/service"
)

type ProxyHandler struct {
	proxy       *httputil.ReverseProxy
	logger      *zerolog.Logger
	metrics     *metrics.MetricsCollector
	target      string
	config      *config.ProxyConfig
	logSvc      *service.LoggerService
	transformer *Transformer
}

func NewProxyHandler(cfg *config.ProxyConfig, logger *zerolog.Logger, repo repository.LogRepository, metrics *metrics.MetricsCollector) (*ProxyHandler, error) {
	target, err := url.Parse(cfg.Target)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Configure transport
	proxy.Transport = &http.Transport{
		MaxIdleConns:          cfg.MaxIdleConns,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		TLSHandshakeTimeout:   cfg.TLSTimeout,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		ExpectContinueTimeout: cfg.ExpectContinueTimeout,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
	}

	// Configure proxy timeouts
	proxy.ModifyResponse = func(r *http.Response) error {
		r.Header.Set("X-Proxy-Timeout", cfg.Timeout.String())
		return nil
	}

	logSvc := service.NewLoggerService(repo, 5, 1000)
	transformer := NewTransformer(cfg)

	return &ProxyHandler{
		proxy:       proxy,
		logger:      logger,
		metrics:     metrics,
		target:      cfg.Target,
		config:      cfg,
		logSvc:      logSvc,
		transformer: transformer,
	}, nil
}

func (h *ProxyHandler) Handle(c *fiber.Ctx) error {
	// Skip logging and proxying for metrics endpoint
	if c.Path() == "/metrics" {
		return c.Next()
	}

	// Increment active requests counter
	h.metrics.IncActiveRequests()
	defer h.metrics.DecActiveRequests()

	startTime := time.Now()
	traceID := uuid.New().String()
	var reqLog *model.RequestLog

	// Log initial request metrics
	method := string(c.Method())
	path := c.Path()
	h.logger.Info().
		Str("method", method).
		Str("path", path).
		Str("trace_id", traceID).
		Str("target_url", h.target).
		Msg("Proxying request")

	// Concurrent log ve metrik işlemleri
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Request logging
	wg.Add(1)
	go func() {
		defer wg.Done()
		reqLog = h.buildRequestLog(c, traceID)
		h.logSvc.LogRequest(reqLog)
	}()

	// Proxy işlemi
	responseWriter := newFiberResponseWriter(c.Response())
	proxyErr := h.proxyRequest(c, responseWriter)

	// Response logging
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := h.handleResponse(c, responseWriter, traceID, startTime, reqLog); err != nil {
			errChan <- err
		}
	}()

	// Wait for goroutines
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Error handling
	for err := range errChan {
		if err != nil {
			statusCode := fiber.StatusInternalServerError
			h.metrics.ObserveRequestDuration(method, path, strconv.Itoa(statusCode), time.Since(startTime))
			h.metrics.IncRequestCounter(method, path, strconv.Itoa(statusCode))
			return c.Status(statusCode).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	if proxyErr != nil {
		statusCode := fiber.StatusInternalServerError
		h.metrics.ObserveRequestDuration(method, path, strconv.Itoa(statusCode), time.Since(startTime))
		h.metrics.IncRequestCounter(method, path, strconv.Itoa(statusCode))
		return c.Status(statusCode).JSON(fiber.Map{
			"error": proxyErr.Error(),
		})
	}

	// Set response headers
	c.Response().Header.SetContentType("application/json")
	return nil
}

func (h *ProxyHandler) proxyRequest(c *fiber.Ctx, responseWriter *fiberResponseWriter) error {
	proxyReq, err := h.buildProxyRequest(c)
	if err != nil {
		return err
	}

	h.proxy.ServeHTTP(responseWriter, proxyReq)
	return nil
}

func (h *ProxyHandler) buildRequestLog(c *fiber.Ctx, traceID string) *model.RequestLog {
	reqLog := &model.RequestLog{
		TraceID:   traceID,
		Timestamp: time.Now(),
		Method:    string(c.Method()),
		URL:       c.OriginalURL(),
		Path:      c.Path(),
		Headers:   make(map[string]string),
		ClientIP:  c.IP(),
		UserAgent: string(c.Request().Header.UserAgent()),
	}

	c.Request().Header.VisitAll(func(key, value []byte) {
		reqLog.Headers[string(key)] = string(value)
	})

	if c.Body() != nil {
		reqLog.RequestBody = c.Body()
	}

	return reqLog
}

func (h *ProxyHandler) buildProxyRequest(c *fiber.Ctx) (*http.Request, error) {
	proxyReq, err := http.NewRequest(
		string(c.Method()),
		c.OriginalURL(),
		bytes.NewReader(c.Body()),
	)
	if err != nil {
		return nil, err
	}

	c.Request().Header.VisitAll(func(key, value []byte) {
		proxyReq.Header.Add(string(key), string(value))
	})

	return proxyReq, nil
}

func (h *ProxyHandler) handleResponse(c *fiber.Ctx, responseWriter *fiberResponseWriter, traceID string, startTime time.Time, reqLog *model.RequestLog) error {
	duration := time.Since(startTime)
	statusCode := strconv.Itoa(responseWriter.StatusCode())
	path := c.Path()
	method := string(c.Method())
	responseSize := int64(len(responseWriter.Body()))

	// Observe request metrics
	h.metrics.ObserveRequestDuration(method, path, statusCode, duration)
	h.metrics.IncRequestCounter(method, path, statusCode)
	h.metrics.ResponseSize.With(prometheus.Labels{
		"app":    h.metrics.AppName,
		"method": method,
		"path":   path,
		"status": statusCode,
	}).Observe(float64(responseSize))

	// Log detailed metrics
	h.logger.Info().
		Str("method", method).
		Str("path", path).
		Str("status", statusCode).
		Str("trace_id", traceID).
		Dur("duration", duration).
		Int64("response_size", responseSize).
		Msg("Request completed")

	// Log response
	respLog := &model.ResponseLog{
		TraceID:      traceID,
		RequestID:    reqLog.ID,
		Timestamp:    time.Now(),
		StatusCode:   responseWriter.StatusCode(),
		Headers:      make(map[string]string),
		ResponseBody: responseWriter.Body(),
		ResponseTime: duration,
	}

	for k, v := range responseWriter.Header() {
		if len(v) > 0 {
			respLog.Headers[k] = v[0]
		}
	}

	h.logSvc.LogResponse(respLog)

	return nil
}

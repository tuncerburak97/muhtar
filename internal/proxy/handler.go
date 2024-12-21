package proxy

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/tuncerburak97/muhtar/internal/config"
	"github.com/tuncerburak97/muhtar/internal/metrics"
	"github.com/tuncerburak97/muhtar/internal/model"
	"github.com/tuncerburak97/muhtar/internal/repository"
	"github.com/tuncerburak97/muhtar/internal/service"
	"github.com/tuncerburak97/muhtar/internal/transform"
)

type ProxyHandler struct {
	proxy                          *httputil.ReverseProxy
	logger                         *zerolog.Logger
	metrics                        *metrics.MetricsCollector
	target                         string
	config                         *config.ProxyConfig
	logSvc                         *service.LoggerService
	transformer                    *transform.Engine
	httpRequestResponseTransformer *HttpRequestResponseTransformer
}

func NewProxyHandler(cfg *config.ProxyConfig, logger *zerolog.Logger, repo repository.LogRepository, metrics *metrics.MetricsCollector, transformer *transform.Engine) (*ProxyHandler, error) {
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

	logSvc := service.NewLoggerService(repo, metrics, 5, 1000)
	httpRequestResponseTransformer := NewTransformer(cfg)
	return &ProxyHandler{
		proxy:                          proxy,
		logger:                         logger,
		metrics:                        metrics,
		target:                         cfg.Target,
		config:                         cfg,
		logSvc:                         logSvc,
		transformer:                    transformer,
		httpRequestResponseTransformer: httpRequestResponseTransformer,
	}, nil
}

// convertHeaders converts map[string][]string to map[string]string
func convertHeaders(headers map[string][]string) map[string]string {
	result := make(map[string]string)
	for k, v := range headers {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
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

	// Log initial request metrics
	method := string(c.Method())
	path := c.Path()
	h.logger.Info().
		Str("method", method).
		Str("path", path).
		Str("trace_id", traceID).
		Str("target_url", h.target).
		Msg("Proxying request")

	// Create target request
	targetURL := h.config.Target + c.OriginalURL()
	req, err := http.NewRequest(c.Method(), targetURL, bytes.NewReader(c.Body()))
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to create target request")
		return err
	}

	// Copy headers
	for k, v := range c.GetReqHeaders() {
		req.Header.Set(k, v[0])
	}

	// Transform request
	if err := h.transformer.TransformRequest(req); err != nil {
		h.logger.Error().Err(err).Msg("Failed to transform request")
		return err
	}

	h.httpRequestResponseTransformer.TransformRequest(req)

	// log request asynchronously
	reqLog := &model.Log{
		ID:          uuid.New().String(),
		TraceID:     traceID,
		ProcessType: model.ProcessTypeRequest,
		Timestamp:   startTime,
		Method:      c.Method(),
		Path:        c.Path(),
		Headers:     convertHeaders(c.GetReqHeaders()),
		ClientIP:    c.IP(),
		URL:         targetURL,
		UserAgent:   c.Get("User-Agent"),
		Body:        c.Body(),
	}
	go func(log *model.Log) {
		if err := h.logSvc.LogRequest(log); err != nil {
			h.logger.Error().Err(err).Msg("Failed to log request")
		}
	}(reqLog)

	// Send request
	resp, err := h.proxy.Transport.RoundTrip(req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to send request to target")
		return err
	}
	defer resp.Body.Close()

	// Transform response
	if err := h.transformer.TransformResponse(resp); err != nil {
		h.logger.Error().Err(err).Msg("Failed to transform response")
		return err
	}

	// Copy response headers
	for k, v := range resp.Header {
		c.Set(k, v[0])
	}

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to read response body")
		return err
	}
	duration := time.Since(startTime)

	h.logger.Info().
		Str("trace_id", traceID).
		Str("method", method).
		Str("path", path).
		Int("status_code", resp.StatusCode).
		Dur("duration", duration).
		Int("response_size", len(body)).
		Str("content_type", resp.Header.Get("Content-Type")).
		Str("cache_control", resp.Header.Get("Cache-Control")).
		Msg("Response completed")

	h.httpRequestResponseTransformer.TransformResponse(resp)

	respLog := &model.Log{
		ID:           uuid.New().String(),
		ProcessType:  model.ProcessTypeResponse,
		Method:       c.Method(),
		Path:         c.Path(),
		StatusCode:   resp.StatusCode,
		ClientIP:     c.IP(),
		Timestamp:    startTime,
		Headers:      convertHeaders(resp.Header),
		TraceID:      traceID,
		URL:          targetURL,
		UserAgent:    c.Get("User-Agent"),
		Body:         body,
		ResponseTime: duration,
	}
	go func(log *model.Log) {
		if err := h.logSvc.LogRequest(log); err != nil {
			h.logger.Error().Err(err).Msg("Failed to log response")
		}
	}(respLog)

	// Update metrics
	h.metrics.ObserveRequestDuration(method, path, strconv.Itoa(resp.StatusCode), duration)
	h.metrics.IncRequestCounter(method, path, strconv.Itoa(resp.StatusCode))

	// Send response
	c.Status(resp.StatusCode)
	// copy all response headers
	for k, v := range resp.Header {
		c.Set(k, v[0])
	}

	return c.Send(body)
}

package proxy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/tuncerburak97/muhtar/internal/config"
)

// HttpRequestResponseTransformer handles HTTP request/response transformations
type HttpRequestResponseTransformer struct {
	config *config.ProxyConfig
}

// NewTransformer creates a new transformer instance
func NewTransformer(cfg *config.ProxyConfig) *HttpRequestResponseTransformer {
	return &HttpRequestResponseTransformer{
		config: cfg,
	}
}

// TransformRequest modifies the outgoing request based on configuration
func (t *HttpRequestResponseTransformer) TransformRequest(req *http.Request) error {
	return t.transformRequestHeaders(req)
}

// TransformResponse modifies the incoming response based on configuration
func (t *HttpRequestResponseTransformer) TransformResponse(res *http.Response) error {
	return t.transformResponseHeaders(res)
}

// Header transformation functions
func (t *HttpRequestResponseTransformer) transformRequestHeaders(req *http.Request) error {
	// Remove unwanted headers
	headersToRemove := []string{
		"X-Powered-By",
		"Server",
		"X-AspNet-Version",
		"X-Internal-Token",
	}
	for _, header := range headersToRemove {
		req.Header.Del(header)
	}

	// Add standard headers
	requestID := uuid.New().String()
	standardHeaders := map[string]string{
		"X-Request-ID":     requestID,
		"X-Proxy-Version":  "1.0",
		"X-Correlation-ID": requestID,
		"Accept":           "application/json",
		"Content-Type":     "application/json",
	}
	for name, value := range standardHeaders {
		req.Header.Set(name, value)
	}

	// Rename B3 headers for distributed tracing
	b3Headers := map[string]string{
		"x-b3-traceid":      "X-B3-TraceId",
		"x-b3-spanid":       "X-B3-SpanId",
		"x-b3-parentspanid": "X-B3-ParentSpanId",
		"x-b3-sampled":      "X-B3-Sampled",
		"x-b3-flags":        "X-B3-Flags",
	}
	for oldName, newName := range b3Headers {
		if value := req.Header.Get(oldName); value != "" {
			req.Header.Del(oldName)
			req.Header.Set(newName, value)
		}
	}

	return nil
}

func (t *HttpRequestResponseTransformer) transformResponseHeaders(res *http.Response) error {
	// Remove internal headers
	headersToRemove := []string{
		"X-Internal-Server",
		"X-Debug-Info",
		"X-AspNet-Version",
		"X-Powered-By",
		"Server",
	}
	for _, header := range headersToRemove {
		res.Header.Del(header)
	}

	// Add standard response headers
	standardHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Cache-Control":          "no-store, no-cache, must-revalidate",
		"Pragma":                 "no-cache",
	}
	for name, value := range standardHeaders {
		res.Header.Set(name, value)
	}

	// Add security headers
	securityHeaders := map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Content-Security-Policy":   "default-src 'self'",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
	}
	for name, value := range securityHeaders {
		res.Header.Set(name, value)
	}

	// Add timing and tracing headers
	res.Header.Set("X-Response-Time", fmt.Sprintf("%d", time.Now().UnixNano()))
	if traceID := res.Request.Header.Get("X-B3-TraceId"); traceID != "" {
		res.Header.Set("X-Trace-ID", traceID)
	}

	return nil
}

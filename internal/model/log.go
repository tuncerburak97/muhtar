package model

import "time"

type ProcessType string

const (
	ProcessTypeRequest  ProcessType = "request"
	ProcessTypeResponse ProcessType = "response"
)

type Log struct {
	ID            string                 `json:"id"`
	TraceID       string                 `json:"trace_id"`
	ProcessType   ProcessType            `json:"process_type"`
	Timestamp     time.Time              `json:"timestamp"`
	Method        string                 `json:"method"`
	URL           string                 `json:"url"`
	Path          string                 `json:"path"`
	PathParams    map[string]string      `json:"path_params,omitempty"`
	QueryParams   map[string]string      `json:"query_params,omitempty"`
	Headers       map[string]string      `json:"headers"`
	Body          []byte                 `json:"body,omitempty"`
	ClientIP      string                 `json:"client_ip"`
	UserAgent     string                 `json:"user_agent"`
	StatusCode    int                    `json:"status_code,omitempty"`
	ResponseTime  time.Duration          `json:"response_time,omitempty"`
	ContentLength int64                  `json:"content_length,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

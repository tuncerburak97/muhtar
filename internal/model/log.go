package model

import (
	"encoding/json"
	"time"
)

const (
	ProcessTypeRequest  = "request"
	ProcessTypeResponse = "response"
)

type Log struct {
	ID            string            `json:"id" bson:"_id" db:"id"`
	TraceID       string            `json:"trace_id" bson:"trace_id" db:"trace_id"`
	ProcessType   string            `json:"process_type" bson:"process_type" db:"process_type"`
	Timestamp     time.Time         `json:"timestamp" bson:"timestamp" db:"timestamp"`
	Method        string            `json:"method,omitempty" bson:"method,omitempty" db:"method"`
	URL           string            `json:"url,omitempty" bson:"url,omitempty" db:"url"`
	Path          string            `json:"path,omitempty" bson:"path,omitempty" db:"path"`
	PathParams    map[string]string `json:"path_params,omitempty" bson:"path_params,omitempty" db:"path_params"`
	QueryParams   map[string]string `json:"query_params,omitempty" bson:"query_params,omitempty" db:"query_params"`
	Headers       map[string]string `json:"headers" bson:"headers" db:"headers"`
	Body          json.RawMessage   `json:"body,omitempty" bson:"body,omitempty" db:"body"`
	ClientIP      string            `json:"client_ip,omitempty" bson:"client_ip,omitempty" db:"client_ip"`
	UserAgent     string            `json:"user_agent,omitempty" bson:"user_agent,omitempty" db:"user_agent"`
	StatusCode    int               `json:"status_code,omitempty" bson:"status_code,omitempty" db:"status_code"`
	ResponseTime  time.Duration     `json:"response_time,omitempty" bson:"response_time,omitempty" db:"response_time"`
	ContentLength int64             `json:"content_length,omitempty" bson:"content_length,omitempty" db:"content_length"`
	Error         string            `json:"error,omitempty" bson:"error,omitempty" db:"error"`
	Metadata      json.RawMessage   `json:"metadata,omitempty" bson:"metadata,omitempty" db:"metadata"`
}

package model

import (
	"encoding/json"
	"time"
)

type RequestLog struct {
	ID             string            `json:"id"`
	Method         string            `json:"method"`
	Path           string            `json:"path"`
	StatusCode     int               `json:"status_code"`
	Duration       time.Duration     `json:"duration"`
	RequestSize    int               `json:"request_size"`
	ResponseSize   int               `json:"response_size"`
	ClientIP       string            `json:"client_ip"`
	Timestamp      time.Time         `json:"timestamp"`
	RequestHeaders map[string]string `json:"request_headers"`
	TraceID        string            `json:"trace_id"`
	URL            string            `json:"url"`
	PathParams     map[string]string `json:"path_params"`
	QueryParams    map[string]string `json:"query_params"`
	RequestBody    json.RawMessage   `json:"request_body"`
	ResponseBody   json.RawMessage   `json:"response_body"`
	Error          string            `json:"error,omitempty"`
	Headers        map[string]string `json:"headers"`
	UserAgent      string            `json:"user_agent"`
}

type ResponseLog struct {
	ID            string            `json:"id" bson:"_id" db:"id"`
	TraceID       string            `json:"trace_id" bson:"trace_id" db:"trace_id"`
	RequestID     string            `json:"request_id" bson:"request_id" db:"request_id"`
	Timestamp     time.Time         `json:"timestamp" bson:"timestamp" db:"timestamp"`
	StatusCode    int               `json:"status_code" bson:"status_code" db:"status_code"`
	Headers       map[string]string `json:"headers" bson:"headers" db:"headers"`
	ResponseBody  json.RawMessage   `json:"response_body" bson:"response_body" db:"response_body"`
	ResponseTime  time.Duration     `json:"response_time" bson:"response_time" db:"response_time"`
	ContentLength int64             `json:"content_length" bson:"content_length" db:"content_length"`
	Error         string            `json:"error,omitempty" bson:"error,omitempty" db:"error"`
}

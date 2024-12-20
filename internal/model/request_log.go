package model

import (
	"encoding/json"
	"time"
)

type RequestLog struct {
	ID          string            `json:"id" bson:"_id" db:"id"`
	TraceID     string            `json:"trace_id" bson:"trace_id" db:"trace_id"`
	Timestamp   time.Time         `json:"timestamp" bson:"timestamp" db:"timestamp"`
	Method      string            `json:"method" bson:"method" db:"method"`
	URL         string            `json:"url" bson:"url" db:"url"`
	Path        string            `json:"path" bson:"path" db:"path"`
	PathParams  map[string]string `json:"path_params" bson:"path_params" db:"path_params"`
	QueryParams map[string]string `json:"query_params" bson:"query_params" db:"query_params"`
	Headers     map[string]string `json:"headers" bson:"headers" db:"headers"`
	RequestBody json.RawMessage   `json:"request_body" bson:"request_body" db:"request_body"`
	ClientIP    string            `json:"client_ip" bson:"client_ip" db:"client_ip"`
	UserAgent   string            `json:"user_agent" bson:"user_agent" db:"user_agent"`
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

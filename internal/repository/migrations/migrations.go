package migrations

import (
	"context"
	"fmt"
)

type Migrator interface {
	Migrate(ctx context.Context) error
}

// PostgreSQL migrations
var PostgresSchema = `
CREATE TABLE IF NOT EXISTS http_log (
    id UUID PRIMARY KEY,
    trace_id UUID NOT NULL,
    process_type VARCHAR(10) NOT NULL, -- 'request' or 'response'
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    method VARCHAR(10),
    url TEXT,
    path TEXT,
    path_params JSONB,
    query_params JSONB,
    headers JSONB,
    body JSONB,
    client_ip VARCHAR(45),
    user_agent TEXT,
    status_code INTEGER,
    response_time INTERVAL,
    content_length BIGINT,
    error TEXT,
    metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_http_log_trace_id ON http_log(trace_id);
CREATE INDEX IF NOT EXISTS idx_http_log_process_type ON http_log(process_type);
CREATE INDEX IF NOT EXISTS idx_http_log_trace_process ON http_log(trace_id, process_type);
CREATE INDEX IF NOT EXISTS idx_http_log_timestamp ON http_log(timestamp);
`

// Oracle migrations
var OracleSchema = `
BEGIN
    EXECUTE IMMEDIATE 'CREATE TABLE logs (
        id RAW(16) PRIMARY KEY,
        trace_id RAW(16) NOT NULL,
        process_type VARCHAR2(10) NOT NULL,
        timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
        method VARCHAR2(10),
        url CLOB,
        path CLOB,
        path_params CLOB,
        query_params CLOB,
        headers CLOB,
        body CLOB,
        client_ip VARCHAR2(45),
        user_agent CLOB,
        status_code NUMBER,
        response_time INTERVAL DAY TO SECOND,
        content_length NUMBER,
        error CLOB,
        metadata CLOB
    )';
EXCEPTION
    WHEN OTHERS THEN
        IF SQLCODE != -955 THEN
            RAISE;
        END IF;
END;
/

CREATE INDEX idx_logs_trace_id ON logs(trace_id);
CREATE INDEX idx_logs_process_type ON logs(process_type);
CREATE INDEX idx_logs_trace_process ON logs(trace_id, process_type);
CREATE INDEX idx_logs_timestamp ON logs(timestamp);
`

// Couchbase indexes
func GetCouchbaseIndexes(bucketName string) []string {
	return []string{
		fmt.Sprintf("CREATE PRIMARY INDEX ON `%s`", bucketName),
		fmt.Sprintf("CREATE INDEX idx_logs_trace_id ON `%s`(trace_id)", bucketName),
		fmt.Sprintf("CREATE INDEX idx_logs_process_type ON `%s`(process_type)", bucketName),
		fmt.Sprintf("CREATE INDEX idx_logs_trace_process ON `%s`(trace_id, process_type)", bucketName),
		fmt.Sprintf("CREATE INDEX idx_logs_timestamp ON `%s`(timestamp)", bucketName),
	}
}

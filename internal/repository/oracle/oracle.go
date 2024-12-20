package oracle

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/tuncerburak97/muhtar/internal/model"
	"github.com/tuncerburak97/muhtar/internal/repository/migrations"
)

type OracleRepository struct {
	DB *sql.DB
}

func NewOracleRepository(connStr string) (*OracleRepository, error) {
	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to Oracle: %v", err)
	}

	return &OracleRepository{DB: db}, nil
}

func (r *OracleRepository) SaveRequestLog(ctx context.Context, log *model.RequestLog) error {
	headers, err := json.Marshal(log.Headers)
	if err != nil {
		return err
	}

	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO request_logs (
			id, trace_id, timestamp, method, url, path, 
			path_params, query_params, headers, request_body, 
			client_ip, user_agent
		) VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12)`,
		log.ID, log.TraceID, log.Timestamp, log.Method,
		log.URL, log.Path, log.PathParams, log.QueryParams,
		headers, log.RequestBody, log.ClientIP, log.UserAgent,
	)
	return err
}

func (r *OracleRepository) SaveRequestLogs(ctx context.Context, logs []*model.RequestLog) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO request_logs (
			id, trace_id, timestamp, method, url, path, 
			path_params, query_params, headers, request_body, 
			client_ip, user_agent
		) VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, log := range logs {
		headers, err := json.Marshal(log.Headers)
		if err != nil {
			return err
		}

		_, err = stmt.ExecContext(ctx,
			log.ID, log.TraceID, log.Timestamp, log.Method,
			log.URL, log.Path, log.PathParams, log.QueryParams,
			headers, log.RequestBody, log.ClientIP, log.UserAgent,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *OracleRepository) SaveResponseLog(ctx context.Context, log *model.ResponseLog) error {
	headers, err := json.Marshal(log.Headers)
	if err != nil {
		return err
	}

	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO response_logs (
			id, trace_id, request_id, timestamp, 
			status_code, headers, response_body, 
			response_time, content_length, error
		) VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10)`,
		log.ID, log.TraceID, log.RequestID, log.Timestamp,
		log.StatusCode, headers, log.ResponseBody,
		log.ResponseTime, log.ContentLength, log.Error,
	)
	return err
}

func (r *OracleRepository) SaveResponseLogs(ctx context.Context, logs []*model.ResponseLog) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO response_logs (
			id, trace_id, request_id, timestamp, 
			status_code, headers, response_body, 
			response_time, content_length, error
		) VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, log := range logs {
		headers, err := json.Marshal(log.Headers)
		if err != nil {
			return err
		}

		_, err = stmt.ExecContext(ctx,
			log.ID, log.TraceID, log.RequestID, log.Timestamp,
			log.StatusCode, headers, log.ResponseBody,
			log.ResponseTime, log.ContentLength, log.Error,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *OracleRepository) Close() error {
	return r.DB.Close()
}

func (r *OracleRepository) Migrate(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("Starting Oracle migrations")

	_, err := r.DB.ExecContext(ctx, migrations.OracleSchema)
	if err != nil {
		log.Error().Err(err).Msg("Oracle migrations failed")
		return fmt.Errorf("migration error: %v", err)
	}

	log.Info().Msg("Oracle migrations completed successfully")
	return nil
}

func (r *OracleRepository) SaveLog(ctx context.Context, log *model.Log) error {
	headers, err := json.Marshal(log.Headers)
	if err != nil {
		return err
	}

	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO logs (
			id, trace_id, process_type, timestamp, method, url, path,
			path_params, query_params, headers, body, client_ip,
			user_agent, status_code, response_time, content_length,
			error, metadata
		) VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14, :15, :16, :17, :18)`,
		log.ID, log.TraceID, log.ProcessType, log.Timestamp, log.Method,
		log.URL, log.Path, log.PathParams, log.QueryParams, headers,
		log.Body, log.ClientIP, log.UserAgent, log.StatusCode,
		log.ResponseTime, log.ContentLength, log.Error, log.Metadata,
	)
	return err
}

func (r *OracleRepository) SaveLogs(ctx context.Context, logs []*model.Log) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO logs (
			id, trace_id, process_type, timestamp, method, url, path,
			path_params, query_params, headers, body, client_ip,
			user_agent, status_code, response_time, content_length,
			error, metadata
		) VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14, :15, :16, :17, :18)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, log := range logs {
		headers, err := json.Marshal(log.Headers)
		if err != nil {
			return err
		}

		_, err = stmt.ExecContext(ctx,
			log.ID, log.TraceID, log.ProcessType, log.Timestamp, log.Method,
			log.URL, log.Path, log.PathParams, log.QueryParams, headers,
			log.Body, log.ClientIP, log.UserAgent, log.StatusCode,
			log.ResponseTime, log.ContentLength, log.Error, log.Metadata,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

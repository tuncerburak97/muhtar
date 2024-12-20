package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
	"github.com/tuncerburak97/muhtar/internal/model"
	"github.com/tuncerburak97/muhtar/internal/repository/migrations"
)

type PostgresRepository struct {
	Pool      *pgxpool.Pool
	BatchSize int
}

func NewPostgresRepository(connStr string) (*PostgresRepository, error) {
	pool, err := pgxpool.Connect(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	return &PostgresRepository{Pool: pool}, nil
}

func (r *PostgresRepository) SaveLog(ctx context.Context, log *model.Log) error {
	headers, err := json.Marshal(log.Headers)
	if err != nil {
		return err
	}

	_, err = r.Pool.Exec(ctx,
		`INSERT INTO http_log (
			id, trace_id, process_type, timestamp, method, url, path,
			path_params, query_params, headers, body, client_ip,
			user_agent, status_code, response_time, content_length,
			error, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		log.ID, log.TraceID, log.ProcessType, log.Timestamp, log.Method,
		log.URL, log.Path, log.PathParams, log.QueryParams, headers,
		log.Body, log.ClientIP, log.UserAgent, log.StatusCode,
		log.ResponseTime, log.ContentLength, log.Error, log.Metadata,
	)
	return err
}

func (r *PostgresRepository) SaveLogs(ctx context.Context, logs []*model.Log) error {
	batch := &pgx.Batch{}

	logger := zerolog.Ctx(ctx)

	logger.Debug().
		Int("count", len(logs)).
		Msg("Saving logs to database")

	for _, logEntry := range logs {
		headers, err := json.Marshal(logEntry.Headers)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to marshal headers")
			return err
		}
		batch.Queue(
			`INSERT INTO http_log (
				id, trace_id, process_type, timestamp, method, url, path,
				path_params, query_params, headers, body, client_ip,
					user_agent, status_code, response_time, content_length,
					error, metadata
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
			logEntry.ID, logEntry.TraceID, logEntry.ProcessType, logEntry.Timestamp, logEntry.Method,
			logEntry.URL, logEntry.Path, logEntry.PathParams, logEntry.QueryParams, headers,
			logEntry.Body, logEntry.ClientIP, logEntry.UserAgent, logEntry.StatusCode,
			logEntry.ResponseTime, logEntry.ContentLength, logEntry.Error, logEntry.Metadata,
		)
	}

	br := r.Pool.SendBatch(ctx, batch)
	defer br.Close()

	result := br.Close()
	if result != nil {
		logger.Error().Err(result).Msg("Failed to save logs")
		return result
	}

	logger.Debug().Msg("Successfully saved logs")
	return nil
}

func (r *PostgresRepository) Close() error {
	r.Pool.Close()
	return nil
}

func (r *PostgresRepository) Migrate(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("Starting PostgreSQL migrations")

	_, err := r.Pool.Exec(ctx, migrations.PostgresSchema)
	if err != nil {
		log.Error().Err(err).Msg("PostgreSQL migrations failed")
		return fmt.Errorf("migration error: %v", err)
	}

	log.Info().Msg("PostgreSQL migrations completed successfully")
	return nil
}

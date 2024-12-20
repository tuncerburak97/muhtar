package repository

import (
	"context"
	"fmt"
	"sync"
	"time"

	"database/sql"

	"github.com/couchbase/gocb/v2"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
	ora "github.com/sijms/go-ora/v2"
	"github.com/tuncerburak97/muhtar/internal/config"
	"github.com/tuncerburak97/muhtar/internal/repository/couchbase"
	"github.com/tuncerburak97/muhtar/internal/repository/oracle"
	"github.com/tuncerburak97/muhtar/internal/repository/postgres"
)

type RepositoryFactory struct {
	mu      sync.RWMutex
	pools   map[string]interface{}
	configs map[string]*config.DBConfig
}

func (f *RepositoryFactory) GetRepository(dbType string) (LogRepository, error) {
	f.mu.RLock()
	pool, exists := f.pools[dbType]
	f.mu.RUnlock()

	if exists {
		repo, err := f.createRepository(dbType, pool)
		if err != nil {
			return nil, err
		}

		// Run migrations
		if err := repo.Migrate(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to run migrations: %v", err)
		}

		return repo, nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if pool, exists = f.pools[dbType]; exists {
		repo, err := f.createRepository(dbType, pool)
		if err != nil {
			return nil, err
		}

		// Run migrations
		if err := repo.Migrate(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to run migrations: %v", err)
		}

		return repo, nil
	}

	// Create new pool
	pool, err := f.createPool(dbType)
	if err != nil {
		return nil, err
	}

	f.pools[dbType] = pool
	repo, err := f.createRepository(dbType, pool)
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := repo.Migrate(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %v", err)
	}

	return repo, nil
}

func (f *RepositoryFactory) createRepository(dbType string, pool interface{}) (LogRepository, error) {
	cfg := f.configs[dbType]
	switch dbType {
	case "postgres":
		if pgPool, ok := pool.(*pgxpool.Pool); ok {
			return &postgres.PostgresRepository{
				Pool:      pgPool,
				BatchSize: cfg.Pool.BatchSize,
			}, nil
		}
		return nil, fmt.Errorf("invalid pool type for postgres")

	case "oracle":
		if sqlDB, ok := pool.(*sql.DB); ok {
			return &oracle.OracleRepository{
				DB: sqlDB,
			}, nil
		}
		return nil, fmt.Errorf("invalid pool type for oracle")

	case "couchbase":
		if cbCluster, ok := pool.(*gocb.Cluster); ok {
			bucket := cbCluster.Bucket(cfg.Database)
			if err := bucket.WaitUntilReady(5*time.Second, nil); err != nil {
				return nil, err
			}
			return &couchbase.CouchbaseRepository{
				Cluster: cbCluster,
				Bucket:  bucket,
			}, nil
		}
		return nil, fmt.Errorf("invalid pool type for couchbase")

	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

func (f *RepositoryFactory) createPool(dbType string) (interface{}, error) {
	cfg := f.configs[dbType]
	switch dbType {
	case "postgres":
		connStr := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?pool_max_conns=%d&pool_min_conns=%d",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
			cfg.Pool.MaxConns, cfg.Pool.MinConns,
		)
		return pgxpool.Connect(context.Background(), connStr)

	case "oracle":
		connStr := ora.BuildUrl(cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password, nil)
		return sql.Open("oracle", connStr)

	case "couchbase":
		connStr := fmt.Sprintf(
			"couchbase://%s:%d",
			cfg.Host, cfg.Port,
		)
		return gocb.Connect(
			connStr,
			gocb.ClusterOptions{
				Username: cfg.User,
				Password: cfg.Password,
			},
		)

	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

func NewRepository(cfg *config.DBConfig) (LogRepository, error) {
	switch cfg.Type {
	case "postgres":
		log.Info().
			Str("type", "postgres").
			Str("host", cfg.Host).
			Int("port", cfg.Port).
			Str("database", cfg.Database).
			Msg("Connecting to database")

		connStr := fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?pool_max_conns=%d&pool_min_conns=%d",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
			cfg.Pool.MaxConns, cfg.Pool.MinConns,
		)
		return postgres.NewPostgresRepository(connStr)

	case "oracle":
		connStr := ora.BuildUrl(cfg.Host, cfg.Port, cfg.Database, cfg.User, cfg.Password, nil)
		return oracle.NewOracleRepository(connStr)

	case "couchbase":
		connStr := fmt.Sprintf(
			"couchbase://%s:%d",
			cfg.Host, cfg.Port,
		)
		return couchbase.NewCouchbaseRepository(connStr, cfg.Database, cfg.User, cfg.Password)

	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}

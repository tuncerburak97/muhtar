package couchbase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/rs/zerolog"
	"github.com/tuncerburak97/muhtar/internal/model"
	"github.com/tuncerburak97/muhtar/internal/repository/migrations"
)

type CouchbaseRepository struct {
	Cluster *gocb.Cluster
	Bucket  *gocb.Bucket
}

func NewCouchbaseRepository(connStr, bucketName, username, password string) (*CouchbaseRepository, error) {
	cluster, err := gocb.Connect(
		connStr,
		gocb.ClusterOptions{
			Username: username,
			Password: password,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to Couchbase: %v", err)
	}

	bucket := cluster.Bucket(bucketName)
	err = bucket.WaitUntilReady(5*time.Second, nil)
	if err != nil {
		return nil, fmt.Errorf("bucket not ready: %v", err)
	}

	return &CouchbaseRepository{
		Cluster: cluster,
		Bucket:  bucket,
	}, nil
}

func (r *CouchbaseRepository) SaveLog(ctx context.Context, log *model.Log) error {
	collection := r.Bucket.DefaultCollection()
	_, err := collection.Upsert(
		fmt.Sprintf("log_%s", log.ID),
		log,
		&gocb.UpsertOptions{},
	)
	return err
}

func (r *CouchbaseRepository) SaveLogs(ctx context.Context, logs []*model.Log) error {
	collection := r.Bucket.DefaultCollection()
	for _, log := range logs {
		_, err := collection.Upsert(
			fmt.Sprintf("log_%s", log.ID),
			log,
			&gocb.UpsertOptions{},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *CouchbaseRepository) Close() error {
	return r.Cluster.Close(nil)
}

func (r *CouchbaseRepository) Migrate(ctx context.Context) error {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("Starting Couchbase migrations")

	// Create required indexes for Couchbase
	indexes := migrations.GetCouchbaseIndexes(r.Bucket.Name())
	for _, indexQuery := range indexes {
		_, err := r.Cluster.Query(indexQuery, nil)
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			log.Error().Err(err).Str("query", indexQuery).Msg("Failed to create Couchbase index")
			return fmt.Errorf("index creation error: %v", err)
		}
	}

	log.Info().Msg("Couchbase migrations completed successfully")
	return nil
}

package mongo

import (
	"context"

	"github.com/tuncerburak97/muhtar/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoRepository struct {
	client *mongo.Client
	db     *mongo.Database
}

func NewMongoRepository(uri, dbName string) (*MongoRepository, error) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	return &MongoRepository{
		client: client,
		db:     client.Database(dbName),
	}, nil
}

func (r *MongoRepository) SaveRequestLog(ctx context.Context, log *model.RequestLog) error {
	_, err := r.db.Collection("request_logs").InsertOne(ctx, log)
	return err
}

func (r *MongoRepository) SaveResponseLog(ctx context.Context, log *model.ResponseLog) error {
	_, err := r.db.Collection("response_logs").InsertOne(ctx, log)
	return err
}

func (r *MongoRepository) Close() error {
	return r.client.Disconnect(context.Background())
}

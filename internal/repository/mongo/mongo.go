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

func (r *MongoRepository) Close() error {
	return r.client.Disconnect(context.Background())
}

func (r *MongoRepository) SaveLog(ctx context.Context, log *model.Log) error {
	_, err := r.db.Collection("logs").InsertOne(ctx, log)
	return err
}

func (r *MongoRepository) Migrate(ctx context.Context) error {
	return nil
}

func (r *MongoRepository) SaveLogs(ctx context.Context, logs []*model.Log) error {
	return nil
}

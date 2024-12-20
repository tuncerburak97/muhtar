package repository

import (
	"context"

	"github.com/tuncerburak97/muhtar/internal/model"
)

type LogRepository interface {
	SaveLog(ctx context.Context, log *model.Log) error
	SaveLogs(ctx context.Context, logs []*model.Log) error
	Migrate(ctx context.Context) error
	Close() error
}

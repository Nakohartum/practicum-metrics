package repository

import (
	"context"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

// DatabaseAdapter defines database operations required by metric repositories.
type DatabaseAdapter interface {
	Open(ctx context.Context) error
	Close(ctx context.Context) error
	CheckConnection(ctx context.Context) error
	SetData(ctx context.Context, metric models.Metrics) error
	GetAll(ctx context.Context) []models.Metrics
	GetData(ctx context.Context, metricType string, metricKey string) (models.Metrics, error)
	SetMultipleDataViaTransaction(context.Context, []models.Metrics) error
}

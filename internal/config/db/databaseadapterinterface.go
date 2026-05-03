package config

import (
	"context"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

type DatabaseAdapter interface {
	Open(ctx context.Context) error
	Close(ctx context.Context) error
	CheckConnection(ctx context.Context) error
	SetData(metric models.Metrics) error
	GetAll() []models.Metrics
	GetData(metricType string, metricKey string) (models.Metrics, error)
	SetMultipleDataViaTransaction(context.Context, []models.Metrics) error
}

package service

import (
	"context"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

// Service defines metric operations used by HTTP handlers.
type Service interface {
	GetData(context.Context, string, string) (models.Metrics, error)
	SetData(context.Context, string, string, string) error
	GetAll(context.Context) []models.Metrics
	SaveAllData(context.Context) error
	RunSaving(ctx context.Context)
	Ping(context.Context) error
	SaveDataAfterExit(context.Context) error
	SetDataUsingMetrics(context.Context, []models.Metrics) error
}

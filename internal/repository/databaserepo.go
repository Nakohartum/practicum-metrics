package repository

import (
	"context"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

// DatabaseRepository adapts a database adapter to the repository layer.
type DatabaseRepository struct {
	dbAdapter DatabaseAdapter
}

// SetAllData stores all provided metrics.
func (dr *DatabaseRepository) SetAllData(ctx context.Context, data []models.Metrics) error {
	return retryRetriablePostgresError(func() error {
		return dr.dbAdapter.SetMultipleDataViaTransaction(ctx, data)
	})
}

// GetData returns one metric by type and name.
func (dr *DatabaseRepository) GetData(ctx context.Context, metricType string, metricKey string) (models.Metrics, error) {
	return dr.dbAdapter.GetData(ctx, metricType, metricKey)
}

// GetAll returns all stored metrics.
func (dr *DatabaseRepository) GetAll(ctx context.Context) []models.Metrics {
	return dr.dbAdapter.GetAll(ctx)
}

// SetData stores one metric.
func (dr *DatabaseRepository) SetData(ctx context.Context, metric models.Metrics) error {
	return retryRetriablePostgresError(func() error {
		return dr.dbAdapter.SetData(ctx, metric)
	})
}

// NewDatabaseRepository creates a DatabaseRepository for the provided adapter.
func NewDatabaseRepository(dbAdapter DatabaseAdapter) *DatabaseRepository {
	return &DatabaseRepository{
		dbAdapter: dbAdapter,
	}
}

// Ping checks that database storage is available.
func (dr *DatabaseRepository) Ping(ctx context.Context) error {
	return dr.dbAdapter.CheckConnection(ctx)
}

// Open opens the underlying database connection.
func (dr *DatabaseRepository) Open(ctx context.Context) error {
	return dr.dbAdapter.Open(ctx)
}

// Close closes the underlying database connection.
func (dr *DatabaseRepository) Close(ctx context.Context) error {
	return dr.dbAdapter.Close(ctx)
}

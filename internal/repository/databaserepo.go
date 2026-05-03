package repository

import (
	"context"

	config "github.com/Nakohartum/practicum-metrics/internal/config/db"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

// DatabaseRepository adapts a database adapter to the repository layer.
type DatabaseRepository struct {
	dbAdapter config.DatabaseAdapter
}

// SetAllData stores all provided metrics.
func (dr *DatabaseRepository) SetAllData(data []models.Metrics) error {
	for _, v := range data {
		err := dr.SetData(v)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetData returns one metric by type and name.
func (dr *DatabaseRepository) GetData(metricType string, metricKey string) (models.Metrics, error) {
	return dr.dbAdapter.GetData(metricType, metricKey)
}

// GetAll returns all stored metrics.
func (dr *DatabaseRepository) GetAll() []models.Metrics {
	return dr.dbAdapter.GetAll()
}

// SetData stores one metric.
func (dr *DatabaseRepository) SetData(metric models.Metrics) error {
	return dr.dbAdapter.SetData(metric)
}

// NewDatabaseRepository creates a DatabaseRepository for the provided adapter.
func NewDatabaseRepository(dbAdapter config.DatabaseAdapter) *DatabaseRepository {
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

package repository

import (
	"context"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

// MemRepo adapts Storage to the repository layer.
type MemRepo struct {
	storage Storage
}

// NewMemRepo creates a MemRepo for the provided storage.
func NewMemRepo(config Storage) *MemRepo {
	return &MemRepo{
		storage: config,
	}
}

// GetData returns one metric by type and name.
func (mr *MemRepo) GetData(metricType, key string) (models.Metrics, error) {
	return mr.storage.GetData(metricType, key)
}

// SetData stores a metric value by type and name.
func (mr *MemRepo) SetData(metricType, key, value string) error {
	return mr.storage.SetData(metricType, key, value)
}

// GetAll returns all stored metrics.
func (mr *MemRepo) GetAll() []models.Metrics {
	return mr.storage.GetAll()
}

// Ping checks that the underlying storage is available.
func (mr *MemRepo) Ping(ctx context.Context) error {
	return mr.storage.Ping(ctx)
}

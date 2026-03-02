package repository

import (
	"context"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

type MemRepo struct {
	storage Storage
	
}

func NewMemRepo(config Storage) *MemRepo {
	return &MemRepo{
		storage: config,
	}
}

func (mr *MemRepo) GetData(metricType, key string) (models.Metrics, error) {
	return mr.storage.GetData(metricType, key)
}

func (mr *MemRepo) SetData(metricType, key, value string) error {
	return mr.storage.SetData(metricType, key, value)
}

func (mr *MemRepo) GetAll() []models.Metrics{
	return mr.storage.GetAll()
}

func (mr *MemRepo) Ping(ctx context.Context) error {
	return mr.storage.Ping(ctx)
}
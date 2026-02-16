package repository

import models "github.com/Nakohartum/practicum-metrics/internal/model"

type Storage interface {
	GetData(string, string) (models.Metrics, error)
	SetData(string, string, string) error
	GetAll() []models.Metrics
}

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

func (mr *MemRepo) GetAll() []models.Metrics {
	return mr.storage.GetAll()
}
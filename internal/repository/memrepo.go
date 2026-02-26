package repository

import (
	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

type MemRepo struct {
	storage Storage
	fileWorker FileWorker
}

func NewMemRepo(config Storage, fileWorker FileWorker) *MemRepo {
	return &MemRepo{
		storage: config,
		fileWorker: fileWorker,
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

func (mr *MemRepo) WriteData(data []models.Metrics) error {
	return mr.fileWorker.WriteData(data)
}

func (mr *MemRepo) ReadData() ([]models.Metrics, error) {
	return mr.fileWorker.ReadData()
}
package repository

import (
	"github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
)


type MemRepo struct {
	storage *config.MemStorage
}

func NewMemRepo(config config.MemStorage) *MemRepo {
	return &MemRepo{
		storage: &config,
	}
}

func (mr *MemRepo) GetData(metricType, key string) (string, bool){
	return mr.storage.GetData(metricType, key)
}

func (mr *MemRepo) SetData(metricType, key, value string) error{
	return mr.storage.SetData(metricType, key, value)
}
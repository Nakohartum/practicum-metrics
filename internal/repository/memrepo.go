package repository

import config "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"

type Storage interface {
	GetData(string, string) (config.StringAnswer, error)
	SetData(string, string, string) error
	GetAll() []config.StringAnswer
}

type MemRepo struct {
	storage Storage
}

func NewMemRepo(config Storage) *MemRepo {
	return &MemRepo{
		storage: config,
	}
}

func (mr *MemRepo) GetData(metricType, key string) (config.StringAnswer, error) {
	return mr.storage.GetData(metricType, key)
}

func (mr *MemRepo) SetData(metricType, key, value string) error {
	return mr.storage.SetData(metricType, key, value)
}

func (mr *MemRepo) GetAll() []config.StringAnswer{
	return mr.storage.GetAll()
}
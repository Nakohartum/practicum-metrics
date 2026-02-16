package service

import (
	"errors"

	config "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
)

type MetricsService struct {
	repo *repository.MemRepo
}

func NewMetricsService(r *repository.MemRepo) *MetricsService {
	return &MetricsService{
		repo: r,
	}
}

func (s *MetricsService) GetData(metricType, metricKey string) (config.StringAnswer, error) {
	if metricKey == "" {
		return config.StringAnswer{}, errors.New("no metric's name")
	}
	return s.repo.GetData(metricType, metricKey)
	
}

func (s *MetricsService) SetData(metricType, metricKey, metricValue string) error {
	return s.repo.SetData(metricType, metricKey, metricValue)
}

func (s *MetricsService) GetAll() []config.StringAnswer{
	return s.repo.GetAll()
}

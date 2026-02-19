package service

import (
	"errors"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
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

func (s *MetricsService) GetData(metricType, metricKey string) (models.Metrics, error) {
	if metricKey == "" {
		return models.Metrics{}, errors.New("no metric's name")
	}
	return s.repo.GetData(metricType, metricKey)
	
}

func (s *MetricsService) SetData(metricType, metricKey, metricValue string) error {
	return s.repo.SetData(metricType, metricKey, metricValue)
}

func (s *MetricsService) GetAll() []models.Metrics{
	return s.repo.GetAll()
}

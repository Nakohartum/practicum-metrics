package service

import (
	"context"
	"errors"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
)

type MetricsService struct {
	repo *repository.MemRepo
	databaseRepo *repository.DatabaseRepository
}

func NewMetricsService(r *repository.MemRepo, d *repository.DatabaseRepository) *MetricsService {
	return &MetricsService{
		repo: r,
		databaseRepo: d,
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

func (s *MetricsService) Ping() error {
	return s.databaseRepo.Ping(context.Background())
}

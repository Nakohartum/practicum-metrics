package service

import (
	"errors"
	"strings"

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

func (s *MetricsService) GetData(path string) (string, error) {
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] == "" {
		return "", errors.New("no metric's name")
	}

	if res, err := s.repo.GetData(parts[0], parts[1]); err == nil {
		return res, nil
	} else{
		return "", err
	}
	
}

func (s *MetricsService) SetData(metricType, metricKey, metricValue string) error {
	return s.repo.SetData(metricType, metricKey, metricValue)
}

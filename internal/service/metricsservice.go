package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
)

type MetricsService struct {
	repo *repository.MemRepo
	storeInterval time.Duration
}

func NewMetricsService(r *repository.MemRepo, storeInterval int) *MetricsService {
	return &MetricsService{
		repo: r,
		storeInterval: time.Duration(storeInterval) * time.Second,
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

func (s *MetricsService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *MetricsService) RunSaving(ctx context.Context){
	
	for {
		select{
		case <-ctx.Done():
			return
		default:
		}
		time.Sleep(s.storeInterval)
		data := s.repo.GetAll()
		for _, v := range data {
			switch v.MType{
			case models.Counter:
				s.repo.SetData(v.MType, v.ID, strconv.FormatInt(*v.Delta, 10))
			case models.Gauge:
				s.repo.SetData(v.MType, v.ID, strconv.FormatFloat(*v.Value, 'f', -1, 64))
			}
		}
	}
}

func (s* MetricsService) SaveAllData() error {
	return nil
}

func (s *MetricsService) SaveDataAfterExit(ctx context.Context) error {
	return nil
}
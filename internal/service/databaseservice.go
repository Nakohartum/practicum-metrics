package service

import (
	"context"
	"log"
	"strconv"
	"time"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
)

type DatabaseService struct {
	repo          *repository.DatabaseRepository
	memRepo       *repository.MemRepo
	storeInterval time.Duration
}

func NewDatabaseService(repo *repository.DatabaseRepository, mr *repository.MemRepo, storeInterval int) *DatabaseService {
	return &DatabaseService{
		repo:          repo,
		memRepo:       mr,
		storeInterval: time.Duration(storeInterval) * time.Second,
	}
}

func (s *DatabaseService) GetData(metricType, metricKey string) (models.Metrics, error) {
	if metricKey == "" {
		return models.Metrics{}, ErrMetricNameRequired
	}
	return s.repo.GetData(metricType, metricKey)

}

func (s *DatabaseService) SetData(metricType, metricKey, metricValue string) error {
	var metric models.Metrics
	metric.MType = metricType
	metric.ID = metricKey
	switch metricType {
	case models.Counter:
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return err
		}
		metric.Delta = &val
	case models.Gauge:
		val, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return err
		}
		metric.Value = &val
	default:
		return ErrMetricTypeNotSupported
	}
	return s.repo.SetData(metric)
}

func (s *DatabaseService) GetAll() []models.Metrics {
	return s.repo.GetAll()
}

func (s *DatabaseService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *DatabaseService) RunSaving(ctx context.Context) {
	ticker := time.NewTicker(s.storeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data := s.memRepo.GetAll()

			if len(data) == 0 {
				continue
			}
			err := s.repo.SetAllData(data)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func (s *DatabaseService) SaveDataAfterExit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.SaveAllData()
}

func (s *DatabaseService) SaveAllData() error {
	data := s.memRepo.GetAll()
	return s.repo.SetAllData(data)
}

func (s *DatabaseService) SetDataUsingMetrics(metrics []models.Metrics) error {
	var firstErr error
	for _, metric := range metrics {
		switch metric.MType {
		case models.Counter:
			if metric.Delta == nil {
				if firstErr == nil {
					firstErr = ErrCounterDeltaRequired
				}
				continue
			}
			if err := s.SetData(metric.MType, metric.ID, strconv.FormatInt(*metric.Delta, 10)); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			}
		case models.Gauge:
			if metric.Value == nil {
				if firstErr == nil {
					firstErr = ErrGaugeValueRequired
				}
				continue
			}
			if err := s.SetData(metric.MType, metric.ID, strconv.FormatFloat(*metric.Value, 'f', -1, 64)); err != nil {
				if firstErr == nil {
					firstErr = err
				}
			}
		default:
			if firstErr == nil {
				firstErr = ErrMetricTypeNotSupported
			}
			continue
		}
	}
	return firstErr
}

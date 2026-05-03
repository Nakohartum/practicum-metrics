package service

import (
	"context"
	"log"
	"strconv"
	"time"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
)

// DatabaseService provides metric operations backed by a database repository.
type DatabaseService struct {
	repo          *repository.DatabaseRepository
	memRepo       *repository.MemRepo
	storeInterval time.Duration
}

// NewDatabaseService creates a DatabaseService with a save interval in seconds.
func NewDatabaseService(repo *repository.DatabaseRepository, mr *repository.MemRepo, storeInterval int) *DatabaseService {
	return &DatabaseService{
		repo:          repo,
		memRepo:       mr,
		storeInterval: time.Duration(storeInterval) * time.Second,
	}
}

// GetData returns one metric from database storage by type and name.
func (s *DatabaseService) GetData(metricType, metricKey string) (models.Metrics, error) {
	if metricKey == "" {
		return models.Metrics{}, ErrMetricNameRequired
	}
	return s.repo.GetData(metricType, metricKey)

}

// SetData stores a metric value in database storage.
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

// GetAll returns all metrics from database storage.
func (s *DatabaseService) GetAll() []models.Metrics {
	return s.repo.GetAll()
}

// Ping checks that database storage is available.
func (s *DatabaseService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

// RunSaving periodically writes in-memory metrics to database storage.
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

// SaveDataAfterExit writes data during graceful shutdown.
func (s *DatabaseService) SaveDataAfterExit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.SaveAllData()
}

// SaveAllData writes all in-memory metrics to database storage.
func (s *DatabaseService) SaveAllData() error {
	data := s.memRepo.GetAll()
	return s.repo.SetAllData(data)
}

// SetDataUsingMetrics stores a batch of metric models.
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

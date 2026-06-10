package service

import (
	"context"
	"strconv"
	"time"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
)

// MetricsService provides metric operations backed by in-memory storage.
type MetricsService struct {
	repo          *repository.MemRepo
	storeInterval time.Duration
}

// NewMetricsService creates a MetricsService with a save interval in seconds.
func NewMetricsService(r *repository.MemRepo, storeInterval time.Duration) *MetricsService {
	return &MetricsService{
		repo:          r,
		storeInterval: storeInterval,
	}
}

// GetData returns one metric by type and name.
func (s *MetricsService) GetData(ctx context.Context, metricType, metricKey string) (models.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return models.Metrics{}, err
	}
	if metricKey == "" {
		return models.Metrics{}, ErrMetricNameRequired
	}
	return s.repo.GetData(metricType, metricKey)

}

// SetData stores a metric value by type and name.
func (s *MetricsService) SetData(ctx context.Context, metricType, metricKey, metricValue string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.repo.SetData(metricType, metricKey, metricValue)
}

// GetAll returns all stored metrics.
func (s *MetricsService) GetAll(ctx context.Context) []models.Metrics {
	if err := ctx.Err(); err != nil {
		return nil
	}
	return s.repo.GetAll()
}

// Ping checks that the underlying storage is available.
func (s *MetricsService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

// RunSaving periodically persists data until the context is canceled.
func (s *MetricsService) RunSaving(ctx context.Context) {
	ticker := time.NewTicker(s.storeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data := s.repo.GetAll()
			for _, v := range data {
				switch v.MType {
				case models.Counter:
					s.repo.SetData(v.MType, v.ID, strconv.FormatInt(*v.Delta, 10))
				case models.Gauge:
					s.repo.SetData(v.MType, v.ID, strconv.FormatFloat(*v.Value, 'f', -1, 64))
				}
			}
		}
	}
}

// SaveAllData persists all current metrics.
func (s *MetricsService) SaveAllData(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

// SaveDataAfterExit persists data during graceful shutdown.
func (s *MetricsService) SaveDataAfterExit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

// SetDataUsingMetrics stores a batch of metric models.
func (s *MetricsService) SetDataUsingMetrics(ctx context.Context, metrics []models.Metrics) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(metrics) == 0 {
		return nil
	}
	var firstErr error
	for _, v := range metrics {
		var err error
		switch v.MType {
		case models.Counter:
			if v.Delta == nil {
				if firstErr == nil {
					firstErr = ErrCounterDeltaRequired
				}
				continue
			}
			err = s.repo.SetData(v.MType, v.ID, strconv.FormatInt(*v.Delta, 10))
		case models.Gauge:
			if v.Value == nil {
				if firstErr == nil {
					firstErr = ErrGaugeValueRequired
				}
				continue
			}
			err = s.repo.SetData(v.MType, v.ID, strconv.FormatFloat(*v.Value, 'f', -1, 64))
		default:
			if firstErr == nil {
				firstErr = ErrMetricTypeNotSupported
			}
			continue
		}
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

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
func NewDatabaseService(repo *repository.DatabaseRepository, mr *repository.MemRepo, storeInterval time.Duration) *DatabaseService {
	return &DatabaseService{
		repo:          repo,
		memRepo:       mr,
		storeInterval: storeInterval,
	}
}

// GetData returns one metric from database storage by type and name.
func (s *DatabaseService) GetData(ctx context.Context, metricType, metricKey string) (models.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return models.Metrics{}, err
	}
	if metricKey == "" {
		return models.Metrics{}, ErrMetricNameRequired
	}
	return s.repo.GetData(ctx, metricType, metricKey)

}

// SetData stores a metric value in database storage.
func (s *DatabaseService) SetData(ctx context.Context, metricType, metricKey, metricValue string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
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
	return s.repo.SetData(ctx, metric)
}

// GetAll returns all metrics from database storage.
func (s *DatabaseService) GetAll(ctx context.Context) []models.Metrics {
	if err := ctx.Err(); err != nil {
		return nil
	}
	return s.repo.GetAll(ctx)
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
			err := s.repo.SetAllData(ctx, data)
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
	return s.SaveAllData(ctx)
}

// SaveAllData writes all in-memory metrics to database storage.
func (s *DatabaseService) SaveAllData(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data := s.memRepo.GetAll()
	if len(data) == 0 {
		return nil
	}
	return s.repo.SetAllData(ctx, data)
}

// SetDataUsingMetrics stores a batch of metric models.
func (s *DatabaseService) SetDataUsingMetrics(ctx context.Context, metrics []models.Metrics) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	normalized, err := normalizeMetricBatch(metrics)
	if err != nil {
		return err
	}
	if len(normalized) == 0 {
		return nil
	}

	return s.repo.SetAllData(ctx, normalized)
}

func normalizeMetricBatch(metrics []models.Metrics) ([]models.Metrics, error) {
	result := make([]models.Metrics, 0, len(metrics))
	counterIndex := make(map[string]int)
	gaugeIndex := make(map[string]int)

	for _, metric := range metrics {
		if metric.ID == "" {
			return nil, ErrMetricNameRequired
		}
		switch metric.MType {
		case models.Counter:
			if metric.Delta == nil {
				return nil, ErrCounterDeltaRequired
			}
			if idx, ok := counterIndex[metric.ID]; ok {
				delta := *result[idx].Delta + *metric.Delta
				result[idx].Delta = &delta
				continue
			}
			delta := *metric.Delta
			counterIndex[metric.ID] = len(result)
			result = append(result, models.Metrics{
				ID:    metric.ID,
				MType: metric.MType,
				Delta: &delta,
			})
		case models.Gauge:
			if metric.Value == nil {
				return nil, ErrGaugeValueRequired
			}
			value := *metric.Value
			if idx, ok := gaugeIndex[metric.ID]; ok {
				result[idx].Value = &value
				continue
			}
			gaugeIndex[metric.ID] = len(result)
			result = append(result, models.Metrics{
				ID:    metric.ID,
				MType: metric.MType,
				Value: &value,
			})
		default:
			return nil, ErrMetricTypeNotSupported
		}
	}

	return result, nil
}

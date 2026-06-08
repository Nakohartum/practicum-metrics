package service

import (
	"context"
	"log"
	"strconv"
	"time"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
)

// FileService provides metric operations backed by a file repository.
type FileService struct {
	repo          *repository.FileRepo
	memRepo       *repository.MemRepo
	storeInterval time.Duration
}

// NewFileService creates a FileService with a save interval in seconds.
func NewFileService(r *repository.FileRepo, mr *repository.MemRepo, storeInterval int) *FileService {
	return &FileService{
		repo:          r,
		memRepo:       mr,
		storeInterval: time.Duration(storeInterval) * time.Second,
	}
}

// WriteData writes all metrics to the file repository.
func (fs *FileService) WriteData(data []models.Metrics) error {
	return fs.repo.WriteData(data)
}

// GetData returns one metric from file storage by type and name.
func (fs *FileService) GetData(ctx context.Context, metricType, metricKey string) (models.Metrics, error) {
	if err := ctx.Err(); err != nil {
		return models.Metrics{}, err
	}
	values, err := fs.repo.ReadData()

	if err != nil {
		return models.Metrics{}, err
	}

	for _, v := range values {
		if v.MType == metricType && v.ID == metricKey {
			return v, nil
		}
	}
	return models.Metrics{}, ErrMetricNotFound
}

// GetAll returns all metrics from file storage.
func (fs *FileService) GetAll(ctx context.Context) []models.Metrics {
	if err := ctx.Err(); err != nil {
		return nil
	}
	values, err := fs.repo.ReadData()

	if err != nil {
		return make([]models.Metrics, 0)
	}
	return values
}

// SetData stores a metric in memory and persists it to file storage.
func (fs *FileService) SetData(ctx context.Context, metricType, metricKey, metricValue string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var model models.Metrics
	fs.memRepo.SetData(metricType, metricKey, metricValue)
	model, err := fs.memRepo.GetData(metricType, metricKey)
	if err != nil {
		return err
	}

	return fs.repo.WriteOneData(model)
}

// RunSaving periodically writes in-memory metrics to file storage.
func (fs *FileService) RunSaving(ctx context.Context) {
	ticker := time.NewTicker(fs.storeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data := fs.memRepo.GetAll()
			if len(data) == 0 {
				continue
			}
			err := fs.WriteData(data)
			if err != nil {
				log.Fatalf("Error writing data: %v", err)
			}
		}
	}
}

// Ping checks that file storage can be read.
func (fs *FileService) Ping(ctx context.Context) error {
	_, err := fs.repo.ReadData()
	return err
}

// SaveDataAfterExit writes data during graceful shutdown.
func (fs *FileService) SaveDataAfterExit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data := fs.memRepo.GetAll()
	err := fs.WriteData(data)
	if err != nil {
		return err
	}
	return nil
}

// SaveAllData writes all in-memory metrics to file storage.
func (fs *FileService) SaveAllData(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	values := fs.memRepo.GetAll()
	return fs.repo.WriteData(values)
}

// SetDataUsingMetrics stores a batch of metric models.
func (fs *FileService) SetDataUsingMetrics(ctx context.Context, metrics []models.Metrics) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(metrics) == 0 {
		return nil
	}
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
			if err := fs.SetData(ctx, metric.MType, metric.ID, strconv.FormatInt(*metric.Delta, 10)); err != nil {
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
			if err := fs.SetData(ctx, metric.MType, metric.ID, strconv.FormatFloat(*metric.Value, 'f', -1, 64)); err != nil {
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

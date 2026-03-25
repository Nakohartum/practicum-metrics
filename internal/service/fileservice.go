package service

import (
	"context"
	"log"
	"strconv"
	"time"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
)

type FileService struct {
	repo          *repository.FileRepo
	memRepo       *repository.MemRepo
	storeInterval time.Duration
}

func NewFileService(r *repository.FileRepo, mr *repository.MemRepo, storeInterval int) *FileService {
	return &FileService{
		repo:          r,
		memRepo:       mr,
		storeInterval: time.Duration(storeInterval) * time.Second,
	}
}

func (fs *FileService) WriteData(data []models.Metrics) error {
	return fs.repo.WriteData(data)
}

func (fs *FileService) GetData(metricType, metricKey string) (models.Metrics, error) {
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

func (fs *FileService) GetAll() []models.Metrics {
	values, err := fs.repo.ReadData()

	if err != nil {
		return make([]models.Metrics, 0)
	}
	return values
}

func (fs *FileService) SetData(metricType, metricKey, metricValue string) error {
	var model models.Metrics
	fs.memRepo.SetData(metricType, metricKey, metricValue)
	model, err := fs.memRepo.GetData(metricType, metricKey)
	if err != nil {
		return err
	}

	return fs.repo.WriteOneData(model)
}

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

func (fs *FileService) Ping(ctx context.Context) error {
	_, err := fs.repo.ReadData()
	return err
}

func (fs *FileService) SaveDataAfterExit(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	data, err := fs.repo.ReadData()
	if err != nil {
		return err
	}
	err = fs.WriteData(data)
	if err != nil {
		log.Fatalf("Error writing data: %v", err)
		return err
	}
	return nil
}

func (fs *FileService) SaveAllData() error {
	values := fs.memRepo.GetAll()
	return fs.repo.WriteData(values)
}

func (fs *FileService) SetDataUsingMetrics(metrics []models.Metrics) error {
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
			if err := fs.SetData(metric.MType, metric.ID, strconv.FormatInt(*metric.Delta, 10)); err != nil {
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
			if err := fs.SetData(metric.MType, metric.ID, strconv.FormatFloat(*metric.Value, 'f', -1, 64)); err != nil {
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

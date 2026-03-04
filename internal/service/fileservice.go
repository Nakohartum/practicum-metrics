package service

import (
	"context"
	"errors"
	"log"
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
	return models.Metrics{}, errors.New("metric not found error")
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

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		time.Sleep(fs.storeInterval)
		data := fs.memRepo.GetAll()
		if len(data) == 0 {
			return
		}
		err := fs.WriteData(data)
		if err != nil {
			log.Fatalf("Error writing data: %v", err)
		}
	}
}

func (fs *FileService) Ping(ctx context.Context) error {
	_, err := fs.repo.ReadData()
	return err
}

func (fs *FileService) SaveDataAfterExit(ctx context.Context) error {
	
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

func (fs* FileService) SetDataUsingMetrics(metrics []models.Metrics) error {
	return fs.repo.WriteData(metrics)
}
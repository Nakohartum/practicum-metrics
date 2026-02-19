package service

import (
	"log"
	"time"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
)

type FileService struct {
	repo *repository.MemRepo
	storeInterval time.Duration
}

func NewFileService(r *repository.MemRepo, storeInterval int) *FileService {
	return &FileService{
		repo: r,
		storeInterval: time.Duration(storeInterval) * time.Second,
	}
}

func (fs *FileService) WriteData(data []models.Metrics) error {
	return fs.repo.WriteData(data)
}

func (fs *FileService) ReadData() ([]models.Metrics, error) {
	return fs.repo.ReadData()
}

func (fs *FileService) RunSaving(){
	
	for {
		time.Sleep(fs.storeInterval)
		data := fs.repo.GetAll()
		err := fs.WriteData(data)
		if err != nil {
			log.Fatalf("Error writing data: %v", err)
		}
	}
}
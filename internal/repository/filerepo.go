package repository

import models "github.com/Nakohartum/practicum-metrics/internal/model"

type FileRepo struct {
	fileWorker FileWorker
}

func NewFileRepo(fileWorker FileWorker) *FileRepo {
	return &FileRepo{
		fileWorker: fileWorker,
	}
}

func (fr *FileRepo) Ping() error {
	return fr.fileWorker.FileExists()
}

func (fr *FileRepo) WriteData(data []models.Metrics) error {
	return fr.fileWorker.WriteData(data)
}

func (fr *FileRepo) ReadData() ([]models.Metrics, error) {
	return fr.fileWorker.ReadData()
}

func (fr *FileRepo) WriteOneData(data models.Metrics) error {
	return fr.fileWorker.WriteOneData(data)
}
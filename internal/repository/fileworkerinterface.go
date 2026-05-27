package repository

import models "github.com/Nakohartum/practicum-metrics/internal/model"

// FileWorker defines persistence operations for file-backed metric storage.
type FileWorker interface {
	WriteData(data []models.Metrics) error
	ReadData() ([]models.Metrics, error)
	WriteOneData(data models.Metrics) error
	FileExists() error
}

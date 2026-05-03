package repository

import models "github.com/Nakohartum/practicum-metrics/internal/model"

// FileRepo adapts FileWorker to the repository layer.
type FileRepo struct {
	fileWorker FileWorker
}

// NewFileRepo creates a FileRepo for the provided file worker.
func NewFileRepo(fileWorker FileWorker) *FileRepo {
	return &FileRepo{
		fileWorker: fileWorker,
	}
}

// Ping checks that file storage exists.
func (fr *FileRepo) Ping() error {
	return fr.fileWorker.FileExists()
}

// WriteData writes all metrics to file storage.
func (fr *FileRepo) WriteData(data []models.Metrics) error {
	return fr.fileWorker.WriteData(data)
}

// ReadData reads all metrics from file storage.
func (fr *FileRepo) ReadData() ([]models.Metrics, error) {
	return fr.fileWorker.ReadData()
}

// WriteOneData writes or replaces a single metric in file storage.
func (fr *FileRepo) WriteOneData(data models.Metrics) error {
	return fr.fileWorker.WriteOneData(data)
}

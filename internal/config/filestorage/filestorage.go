package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

// FileWriter writes metric data to a JSON file.
type FileWriter struct {
	file    *os.File
	encoder *json.Encoder
}

// NewFileWriter opens a metric file writer for the provided filename.
func NewFileWriter(filename string) (*FileWriter, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		return nil, err
	}

	return &FileWriter{
		file:    file,
		encoder: json.NewEncoder(file),
	}, nil
}

// WriteData replaces file contents with the provided metrics.
func (fw *FileWriter) WriteData(data []models.Metrics) error {
	err := fw.file.Truncate(0)
	if err != nil {
		return err
	}
	fw.file.Seek(0, 0)
	return fw.encoder.Encode(data)
}

// WriteOneData writes or replaces one metric in the provided metric slice.
func (fw *FileWriter) WriteOneData(models []models.Metrics, data models.Metrics) error {
	models = append(models, data)
	for i := range models {
		if models[i].MType == data.MType && models[i].ID == data.ID {
			models[i] = data
		}
	}
	return fw.WriteData(models)
}

// FileReader reads metric data from a JSON file.
type FileReader struct {
	file *os.File
}

// NewFileReader opens a metric file reader for the provided filename.
func NewFileReader(filename string) (*FileReader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)

	if err != nil {
		return nil, err
	}
	return &FileReader{
		file: file,
	}, nil
}

// ReadData reads all metrics from the file.
func (fr *FileReader) ReadData() ([]models.Metrics, error) {
	if _, err := fr.file.Seek(0, 0); err != nil {
		return nil, err
	}

	var data []models.Metrics
	dec := json.NewDecoder(fr.file)
	if err := dec.Decode(&data); err != nil {
		if errors.Is(err, io.EOF) {
			return []models.Metrics{}, nil
		}
		return nil, err
	}
	return data, nil
}

// FileManager combines file reading and writing operations.
type FileManager struct {
	fileReader FileReader
	fileWriter FileWriter
}

// NewFileManager creates a FileManager from reader and writer components.
func NewFileManager(reader *FileReader, writer *FileWriter) *FileManager {
	return &FileManager{
		fileReader: *reader,
		fileWriter: *writer,
	}
}

// WriteData writes all metrics to file storage.
func (fm *FileManager) WriteData(data []models.Metrics) error {
	return fm.fileWriter.WriteData(data)
}

// ReadData reads all metrics from file storage.
func (fm *FileManager) ReadData() ([]models.Metrics, error) {
	return fm.fileReader.ReadData()
}

// WriteOneData writes or replaces a single metric in file storage.
func (fm *FileManager) WriteOneData(data models.Metrics) error {
	metrics, err := fm.fileReader.ReadData()
	if err != nil {
		return err
	}
	return fm.fileWriter.WriteOneData(metrics, data)
}

// FileExists checks that both reader and writer files are available.
func (fm *FileManager) FileExists() error {
	if fm.fileReader.file == nil || fm.fileWriter.file == nil {
		return ErrFileDoesNotExist
	}
	return nil
}

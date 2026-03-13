package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

type FileWriter struct {
	file    *os.File
	encoder *json.Encoder
}

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

func (fw *FileWriter) WriteData(data []models.Metrics) error {
	err := fw.file.Truncate(0)
	if err != nil {
		return err
	}
	fw.file.Seek(0, 0)
	return fw.encoder.Encode(data)
}

func (fw *FileWriter) WriteOneData(models []models.Metrics, data models.Metrics) error {
	models = append(models, data)
	for i := range models {
		if models[i].MType == data.MType && models[i].ID == data.ID {
			models[i] = data
		}
	}
	return fw.WriteData(models)
}

type FileReader struct {
	file *os.File
}

func NewFileReader(filename string) (*FileReader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)

	if err != nil {
		return nil, err
	}
	return &FileReader{
		file: file,
	}, nil
}

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

type FileManager struct {
	fileReader FileReader
	fileWriter FileWriter
}

func NewFileManager(reader *FileReader, writer *FileWriter) *FileManager {
	return &FileManager{
		fileReader: *reader,
		fileWriter: *writer,
	}
}

func (fm *FileManager) WriteData(data []models.Metrics) error {
	return fm.fileWriter.WriteData(data)
}

func (fm *FileManager) ReadData() ([]models.Metrics, error) {
	return fm.fileReader.ReadData()
}

func (fm *FileManager) WriteOneData(data models.Metrics) error {
	metrics, err := fm.fileReader.ReadData()
	if err != nil {
		return err
	}
	return fm.fileWriter.WriteOneData(metrics, data)
}

func (fm *FileManager) FileExists() error {
	if fm.fileReader.file == nil || fm.fileWriter.file == nil {
		return ErrFileDoesNotExist
	}
	return nil
}

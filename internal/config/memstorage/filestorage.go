package config

import (
	"encoding/json"
	"os"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

type FileWriter struct {
	file *os.File
	encoder *json.Encoder
}

func NewFileWriter(filename string) (*FileWriter, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY, 0666)

	if err != nil {
		return nil, err
	}

	return &FileWriter{
		file: file,
		encoder: json.NewEncoder(file),
	}, nil
}

func (fw *FileWriter) WriteData (data []models.Metrics) error {
	return fw.encoder.Encode(data)
}

type FileReader struct {
	file *os.File
	decoder *json.Decoder
}

func NewFileReader(filename string) (*FileReader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY | os.O_CREATE, 0666)
	
	if err != nil {
		return nil, err
	}
	return &FileReader{
		file: file,
		decoder: json.NewDecoder(file),
	}, nil
}

func (fr *FileReader) ReadData() ([]models.Metrics, error) {
	var data []models.Metrics
	err := fr.decoder.Decode(&data)
	return data, err
}

type FileManager struct{
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
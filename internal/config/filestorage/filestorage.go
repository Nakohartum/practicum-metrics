package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

type FileWriter struct {
	file *os.File
	encoder *json.Encoder
}

func NewFileWriter(filename string) (*FileWriter, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)

	if err != nil {
		return nil, err
	}

	return &FileWriter{
		file: file,
		encoder: json.NewEncoder(file),
	}, nil
}

func (fw *FileWriter) WriteData (data []models.Metrics) error {
	err := fw.file.Truncate(0)
	if err != nil {
		return err
	}
	fw.file.Seek(0,0)
	return fw.encoder.Encode(data)
}

func (fw *FileWriter) WriteOneData (models []models.Metrics, data models.Metrics) error {
	models = append(models, data)
	return fw.WriteData(models)
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
	fr.file.Seek(0, 0)
	var data []models.Metrics
	err := fr.decoder.Decode(&data)
	if err != nil{
		if errors.Is(err, io.EOF){
			return make([]models.Metrics, 0), nil
		}
		return nil, err
	}
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

func (fm *FileManager) WriteOneData(data models.Metrics) error {
	metrics, err := fm.fileReader.ReadData()
	if err != nil {
		return err
	}
	return fm.fileWriter.WriteOneData(metrics, data)
}

func (fm *FileManager) FileExists() error {
	if fm.fileReader.file == nil || fm.fileWriter.file == nil {
		return errors.New("file does not exist")
	}
	return nil
}
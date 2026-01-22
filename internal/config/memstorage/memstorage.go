package config

import (
	"errors"
	"strconv"

	"github.com/Nakohartum/practicum-metrics/internal/model"
)

type MemStorage struct {
	data *models.StorageModel
}

func NewMemStorage(model models.StorageModel) *MemStorage {
	return &MemStorage{
		data: &model,
	}
}


func (ms *MemStorage) GetData(metricType, key string) (string, bool){
	switch metricType{
	case models.Counter:
		if val, exists := ms.data.Counters[key]; exists{
			return strconv.FormatInt(val, 10), true
		}
	case models.Gauge:
		if val, exists := ms.data.Gauges[key]; exists{
			return strconv.FormatFloat(val, 'f', 5, 64), true
		}
	}
	return "", false
}


func (ms *MemStorage) SetData(metricType, key, value string) error{
	switch metricType{
	case models.Counter:
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			
			return err
		}
		ms.data.Counters[key] += val
		return nil
	case models.Gauge:
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		ms.data.Gauges[key] = val
		return nil
	}

	return errors.New("no metric found")
}

package config

import (
	"context"
	"errors"
	"strconv"

	"github.com/Nakohartum/practicum-metrics/internal/model"
)

var (
	ErrNotExists = errors.New("item does not exist")
)

type MemStorage struct {
	data *models.StorageModel
}

func NewMemStorage(model *models.StorageModel) *MemStorage {
	return &MemStorage{
		data: model,
	}
}

func NewMemStubStorage() *MemStorage {
	return &MemStorage{
		data: models.NewStorageModel(),
	}
}

func (ms *MemStorage) GetData(metricType, key string) (models.Metrics, error){
	switch metricType{
	case models.Counter:
		if val, exists := ms.data.Counters[key]; exists{
			return models.Metrics{
				MType: models.Counter,
				ID: key,
				Delta: &val,
			}, nil
		}
	case models.Gauge:
		if val, exists := ms.data.Gauges[key]; exists{
			return models.Metrics{
				MType: models.Gauge,
				ID: key,
				Value: &val,
			}, nil 
		}
	}
	return models.Metrics{}, ErrNotExists
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

func (ms *MemStorage) GetAll() []models.Metrics{
	var res []models.Metrics;

	for k, v := range ms.data.Counters{
		res = append(res, models.Metrics{
			MType: models.Counter,
			ID: k,
			Delta: &v,
		})
	}

	for k, v := range ms.data.Gauges {
		res = append(res, models.Metrics{
			MType: models.Gauge,
			ID: k,
			Value: &v,
		})
	}

	return res
}

func (ms *MemStorage) Ping(ctx context.Context) error {
	return nil
}

package config

import (
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

type StringAnswer struct {
	Type  string
	Name  string
	Value string
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

func (ms *MemStorage) GetData(metricType, key string) (StringAnswer, error){
	switch metricType{
	case models.Counter:
		if val, exists := ms.data.Counters[key]; exists{
			return StringAnswer{
				Type: models.Counter,
				Name: key,
				Value: strconv.FormatInt(val, 10),
			}, nil
		}
	case models.Gauge:
		if val, exists := ms.data.Gauges[key]; exists{
			return StringAnswer{
				Type: models.Gauge,
				Name: key,
				Value: strconv.FormatFloat(val, 'f', -1, 64),
			}, nil 
		}
	}
	return StringAnswer{}, ErrNotExists
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

func (ms *MemStorage) GetAll() []StringAnswer{
	var res []StringAnswer;

	for k, v := range ms.data.Counters{
		res = append(res, StringAnswer{
			Type: models.Counter,
			Name: k,
			Value: strconv.FormatInt(v, 10),
		})
	}

	for k, v := range ms.data.Gauges {
		res = append(res, StringAnswer{
			Type: models.Gauge,
			Name: k,
			Value: strconv.FormatFloat(v, 'f', -1, 64),
		})
	}

	return res
}

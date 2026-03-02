package repository

import (
	"context"

	config "github.com/Nakohartum/practicum-metrics/internal/config/db"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

type DatabaseRepository struct {
	dbAdapter config.DatabaseAdapter
}

func (dr *DatabaseRepository) SetAllData(data []models.Metrics) error {
	for _, v := range data {
		err := dr.SetData(v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dr *DatabaseRepository) GetData(metricType string, metricKey string) (models.Metrics, error) {
	return dr.dbAdapter.GetData(metricType, metricKey)
}

func (dr *DatabaseRepository) GetAll() []models.Metrics {
	return dr.dbAdapter.GetAll()
}

func (dr *DatabaseRepository) SetData(metric models.Metrics) error {
	return dr.dbAdapter.SetData(metric)
}

func NewDatabaseRepository(dbAdapter config.DatabaseAdapter) *DatabaseRepository {
	return &DatabaseRepository{
		dbAdapter: dbAdapter,
	}
}

func (dr *DatabaseRepository) Ping(ctx context.Context) error {
	return dr.dbAdapter.CheckConnection(ctx)
}

func (dr *DatabaseRepository) Open(ctx context.Context) error {
	return dr.dbAdapter.Open(ctx)
}

func (dr *DatabaseRepository) Close(ctx context.Context) error {
	return dr.dbAdapter.Close(ctx)
}

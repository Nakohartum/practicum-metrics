package repository

import (
	"context"

	config "github.com/Nakohartum/practicum-metrics/internal/config/db"
)

type DatabaseRepository struct {
	dbAdapter config.DatabaseAdapter
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
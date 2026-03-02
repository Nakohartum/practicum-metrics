package service

import (
	"context"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

type Service interface {
	GetData(string, string) (models.Metrics, error)
	SetData(string, string, string) error
	GetAll() []models.Metrics
	SaveAllData() error
	RunSaving(ctx context.Context)
	Ping(context.Context) error
	SaveDataAfterExit(context.Context) error
}
package repository

import models "github.com/Nakohartum/practicum-metrics/internal/model"

type Storage interface {
	GetData(string, string) (models.Metrics, error)
	SetData(string, string, string) error
	GetAll() []models.Metrics
}
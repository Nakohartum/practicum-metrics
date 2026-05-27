package repository

import (
	"context"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

// Storage defines the low-level metric storage operations used by repositories.
type Storage interface {
	GetData(string, string) (models.Metrics, error)
	SetData(string, string, string) error
	GetAll() []models.Metrics
	Ping(ctx context.Context) error
}

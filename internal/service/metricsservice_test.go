package service

import (
	"testing"

	"github.com/Nakohartum/practicum-metrics/internal/mocks"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}

func TestMetricsServiceGetData(t *testing.T) {
	tests := []struct {
		name       string
		metricType string
		key        string
		mock       func(*mocks.MockStorage)
		wantMetric models.Metrics
		wantErr    error
	}{
		{
			name:       "returns error for empty key",
			metricType: models.Counter,
			key:        "",
			wantErr:    ErrMetricNameRequired,
		},
		{
			name:       "returns metric from repository",
			metricType: models.Counter,
			key:        "hits",
			mock: func(storage *mocks.MockStorage) {
				storage.EXPECT().GetData(models.Counter, "hits").Return(models.Metrics{
					ID:    "hits",
					MType: models.Counter,
					Delta: int64Ptr(3),
				}, nil)
			},
			wantMetric: models.Metrics{
				ID:    "hits",
				MType: models.Counter,
				Delta: int64Ptr(3),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			storage := mocks.NewMockStorage(ctrl)
			if tt.mock != nil {
				tt.mock(storage)
			}

			svc := NewMetricsService(repository.NewMemRepo(storage), 1)

			got, err := svc.GetData(tt.metricType, tt.key)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantMetric, got)
		})
	}
}

func TestMetricsServiceSetDataUsingMetrics(t *testing.T) {
	tests := []struct {
		name    string
		metrics []models.Metrics
		mock    func(*mocks.MockStorage)
		wantErr error
	}{
		{
			name: "sets counter and gauge",
			metrics: []models.Metrics{
				{ID: "c1", MType: models.Counter, Delta: int64Ptr(5)},
				{ID: "g1", MType: models.Gauge, Value: float64Ptr(1.5)},
			},
			mock: func(storage *mocks.MockStorage) {
				storage.EXPECT().SetData(models.Counter, "c1", "5").Return(nil)
				storage.EXPECT().SetData(models.Gauge, "g1", "1.5").Return(nil)
			},
		},
		{
			name: "skips nil counter delta",
			metrics: []models.Metrics{
				{ID: "c2", MType: models.Counter},
			},
			wantErr: ErrCounterDeltaRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			storage := mocks.NewMockStorage(ctrl)
			if tt.mock != nil {
				tt.mock(storage)
			}

			svc := NewMetricsService(repository.NewMemRepo(storage), 1)
			err := svc.SetDataUsingMetrics(tt.metrics)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

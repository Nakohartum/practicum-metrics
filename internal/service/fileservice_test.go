package service

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Nakohartum/practicum-metrics/internal/mocks"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
)

func TestFileServiceGetData(t *testing.T) {
	tests := []struct {
		name       string
		metricType string
		key        string
		mock       func(*mocks.MockFileWorker)
		wantMetric models.Metrics
		wantErr    error
	}{
		{
			name:       "returns metric from file worker",
			metricType: models.Gauge,
			key:        "cpu",
			mock: func(fileWorker *mocks.MockFileWorker) {
				fileWorker.EXPECT().ReadData().Return([]models.Metrics{{
					ID:    "cpu",
					MType: models.Gauge,
					Value: float64Ptr(0.7),
				}}, nil)
			},
			wantMetric: models.Metrics{
				ID:    "cpu",
				MType: models.Gauge,
				Value: float64Ptr(0.7),
			},
		},
		{
			name:       "returns error when metric missing",
			metricType: models.Gauge,
			key:        "missing",
			mock: func(fileWorker *mocks.MockFileWorker) {
				fileWorker.EXPECT().ReadData().Return([]models.Metrics{}, nil)
			},
			wantErr: ErrMetricNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			fileWorker := mocks.NewMockFileWorker(ctrl)
			memStorage := mocks.NewMockStorage(ctrl)
			tt.mock(fileWorker)

			svc := NewFileService(
				repository.NewFileRepo(fileWorker),
				repository.NewMemRepo(memStorage),
				1,
			)

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

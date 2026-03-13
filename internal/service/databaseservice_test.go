package service

import (
	"context"
	"testing"

	"github.com/Nakohartum/practicum-metrics/internal/mocks"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseServiceSetData(t *testing.T) {
	tests := []struct {
		name       string
		metricType string
		value      string
		mock       func(*mocks.MockDatabaseAdapter)
		wantErr    error
		assertErr  func(*testing.T, error)
	}{
		{
			name:       "counter is parsed and delegated",
			metricType: models.Counter,
			value:      "7",
			mock: func(adapter *mocks.MockDatabaseAdapter) {
				adapter.EXPECT().SetData(models.Metrics{
					ID:    "metric",
					MType: models.Counter,
					Delta: int64Ptr(7),
				}).Return(nil)
			},
		},
		{
			name:       "gauge is parsed and delegated",
			metricType: models.Gauge,
			value:      "7.5",
			mock: func(adapter *mocks.MockDatabaseAdapter) {
				adapter.EXPECT().SetData(models.Metrics{
					ID:    "metric",
					MType: models.Gauge,
					Value: float64Ptr(7.5),
				}).Return(nil)
			},
		},
		{
			name:       "invalid type returns error",
			metricType: "other",
			value:      "1",
			wantErr:    ErrMetricTypeNotSupported,
		},
		{
			name:       "invalid counter value returns error",
			metricType: models.Counter,
			value:      "bad",
			assertErr: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			adapter := mocks.NewMockDatabaseAdapter(ctrl)
			memStorage := mocks.NewMockStorage(ctrl)
			if tt.mock != nil {
				tt.mock(adapter)
			}

			svc := NewDatabaseService(
				repository.NewDatabaseRepository(adapter),
				repository.NewMemRepo(memStorage),
				1,
			)

			err := svc.SetData(tt.metricType, "metric", tt.value)
			if tt.assertErr != nil {
				tt.assertErr(t, err)
				return
			}
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestDatabaseServicePing(t *testing.T) {
	tests := []struct {
		name    string
		pingErr error
	}{
		{name: "returns nil when repo ping is ok"},
		{name: "returns ping error", pingErr: assert.AnError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			adapter := mocks.NewMockDatabaseAdapter(ctrl)
			memStorage := mocks.NewMockStorage(ctrl)
			adapter.EXPECT().CheckConnection(gomock.Any()).Return(tt.pingErr)

			svc := NewDatabaseService(
				repository.NewDatabaseRepository(adapter),
				repository.NewMemRepo(memStorage),
				1,
			)

			err := svc.Ping(context.Background())
			if tt.pingErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.pingErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

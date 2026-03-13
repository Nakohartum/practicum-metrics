package repository

import (
	"testing"

	"github.com/Nakohartum/practicum-metrics/internal/mocks"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func TestMemRepoSetData(t *testing.T) {
	tests := []struct {
		name    string
		mock    func(*mocks.MockStorage)
		wantErr error
	}{
		{
			name: "delegates without error",
			mock: func(storage *mocks.MockStorage) {
				storage.EXPECT().SetData("counter", "hits", "1").Return(nil)
			},
		},
		{
			name: "delegates with error",
			mock: func(storage *mocks.MockStorage) {
				storage.EXPECT().SetData("counter", "hits", "1").Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			storage := mocks.NewMockStorage(ctrl)
			tt.mock(storage)
			repo := NewMemRepo(storage)

			err := repo.SetData("counter", "hits", "1")
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestMemRepoGetData(t *testing.T) {
	tests := []struct {
		name       string
		mock       func(*mocks.MockStorage)
		wantMetric models.Metrics
		wantErr    error
	}{
		{
			name: "delegates success",
			mock: func(storage *mocks.MockStorage) {
				storage.EXPECT().GetData("counter", "hits").Return(models.Metrics{
					ID:    "hits",
					MType: "counter",
					Delta: int64Ptr(3),
				}, nil)
			},
			wantMetric: models.Metrics{
				ID:    "hits",
				MType: "counter",
				Delta: int64Ptr(3),
			},
		},
		{
			name: "delegates error",
			mock: func(storage *mocks.MockStorage) {
				storage.EXPECT().GetData("counter", "hits").Return(models.Metrics{}, assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			storage := mocks.NewMockStorage(ctrl)
			tt.mock(storage)
			repo := NewMemRepo(storage)

			got, err := repo.GetData("counter", "hits")
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

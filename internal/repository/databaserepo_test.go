package repository

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Nakohartum/practicum-metrics/internal/mocks"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

func TestDatabaseRepositorySetAllData(t *testing.T) {
	tests := []struct {
		name    string
		metrics []models.Metrics
		mock    func(*mocks.MockDatabaseAdapter)
		wantErr error
	}{
		{
			name:    "sets all metrics",
			metrics: []models.Metrics{{ID: "a"}, {ID: "b"}},
			mock: func(adapter *mocks.MockDatabaseAdapter) {
				adapter.EXPECT().SetData(models.Metrics{ID: "a"}).Return(nil)
				adapter.EXPECT().SetData(models.Metrics{ID: "b"}).Return(nil)
			},
		},
		{
			name:    "returns on first error",
			metrics: []models.Metrics{{ID: "bad"}, {ID: "ok"}},
			mock: func(adapter *mocks.MockDatabaseAdapter) {
				adapter.EXPECT().SetData(models.Metrics{ID: "bad"}).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			adapter := mocks.NewMockDatabaseAdapter(ctrl)
			tt.mock(adapter)
			repo := NewDatabaseRepository(adapter)

			err := repo.SetAllData(tt.metrics)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

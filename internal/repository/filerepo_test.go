package repository

import (
	"testing"

	"github.com/Nakohartum/practicum-metrics/internal/mocks"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileRepoWriteData(t *testing.T) {
	tests := []struct {
		name    string
		mock    func(*mocks.MockFileWorker)
		wantErr error
	}{
		{
			name: "delegates success",
			mock: func(fileWorker *mocks.MockFileWorker) {
				fileWorker.EXPECT().WriteData([]models.Metrics{{ID: "x"}}).Return(nil)
			},
		},
		{
			name: "delegates error",
			mock: func(fileWorker *mocks.MockFileWorker) {
				fileWorker.EXPECT().WriteData([]models.Metrics{{ID: "x"}}).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			fileWorker := mocks.NewMockFileWorker(ctrl)
			tt.mock(fileWorker)
			repo := NewFileRepo(fileWorker)

			err := repo.WriteData([]models.Metrics{{ID: "x"}})
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

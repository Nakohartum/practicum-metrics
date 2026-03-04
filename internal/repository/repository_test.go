package repository

import (
	"context"
	"errors"
	"testing"

	config "github.com/Nakohartum/practicum-metrics/internal/config/db"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStorage struct {
	getDataFn func(metricType, key string) (models.Metrics, error)
	setDataFn func(metricType, key, value string) error
	getAllFn  func() []models.Metrics
	pingFn    func(ctx context.Context) error
}

func (f *fakeStorage) GetData(metricType, key string) (models.Metrics, error) {
	return f.getDataFn(metricType, key)
}
func (f *fakeStorage) SetData(metricType, key, value string) error {
	return f.setDataFn(metricType, key, value)
}
func (f *fakeStorage) GetAll() []models.Metrics { return f.getAllFn() }
func (f *fakeStorage) Ping(ctx context.Context) error {
	return f.pingFn(ctx)
}

type fakeFileWorker struct {
	fileExistsFn func() error
	writeDataFn  func([]models.Metrics) error
	readDataFn   func() ([]models.Metrics, error)
	writeOneFn   func(models.Metrics) error
}

func (f *fakeFileWorker) FileExists() error { return f.fileExistsFn() }
func (f *fakeFileWorker) WriteData(data []models.Metrics) error { return f.writeDataFn(data) }
func (f *fakeFileWorker) ReadData() ([]models.Metrics, error) { return f.readDataFn() }
func (f *fakeFileWorker) WriteOneData(data models.Metrics) error { return f.writeOneFn(data) }

type fakeDBAdapter struct {
	checkConnectionFn func(context.Context) error
	openFn            func(context.Context) error
	closeFn           func(context.Context) error
	setDataFn         func(models.Metrics) error
	getAllFn          func() []models.Metrics
	getDataFn         func(string, string) (models.Metrics, error)
}

var _ config.DatabaseAdapter = (*fakeDBAdapter)(nil)

func (f *fakeDBAdapter) CheckConnection(ctx context.Context) error { return f.checkConnectionFn(ctx) }
func (f *fakeDBAdapter) Open(ctx context.Context) error            { return f.openFn(ctx) }
func (f *fakeDBAdapter) Close(ctx context.Context) error           { return f.closeFn(ctx) }
func (f *fakeDBAdapter) SetData(m models.Metrics) error            { return f.setDataFn(m) }
func (f *fakeDBAdapter) GetAll() []models.Metrics                  { return f.getAllFn() }
func (f *fakeDBAdapter) GetData(t, k string) (models.Metrics, error) {
	return f.getDataFn(t, k)
}

func TestSetDataMemRepo(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "delegates without error"},
		{name: "delegates with error", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			repo := NewMemRepo(&fakeStorage{
				setDataFn: func(metricType, key, value string) error {
					called = true
					if tt.wantErr {
						return errors.New("set error")
					}
					assert.Equal(t, "counter", metricType)
					assert.Equal(t, "hits", key)
					assert.Equal(t, "1", value)
					return nil
				},
				getDataFn: func(metricType, key string) (models.Metrics, error) { return models.Metrics{}, nil },
				getAllFn:  func() []models.Metrics { return nil },
				pingFn:    func(ctx context.Context) error { return nil },
			})

			err := repo.SetData("counter", "hits", "1")
			require.True(t, called)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetDataMemRepo(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "delegates success"},
		{name: "delegates error", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMemRepo(&fakeStorage{
				getDataFn: func(metricType, key string) (models.Metrics, error) {
					if tt.wantErr {
						return models.Metrics{}, errors.New("get error")
					}
					v := int64(3)
					return models.Metrics{ID: key, MType: metricType, Delta: &v}, nil
				},
				setDataFn: func(metricType, key, value string) error { return nil },
				getAllFn:  func() []models.Metrics { return nil },
				pingFn:    func(ctx context.Context) error { return nil },
			})

			got, err := repo.GetData("counter", "hits")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "hits", got.ID)
			assert.Equal(t, "counter", got.MType)
		})
	}
}

func TestWriteDataFileRepo(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{name: "delegates success"},
		{name: "delegates error", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			repo := NewFileRepo(&fakeFileWorker{
				fileExistsFn: func() error { return nil },
				writeDataFn: func(data []models.Metrics) error {
					called = true
					if tt.wantErr {
						return errors.New("write error")
					}
					return nil
				},
				readDataFn: func() ([]models.Metrics, error) { return nil, nil },
				writeOneFn: func(models.Metrics) error { return nil },
			})

			err := repo.WriteData([]models.Metrics{{ID: "x"}})
			require.True(t, called)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSetAllData(t *testing.T) {
	tests := []struct {
		name    string
		metrics []models.Metrics
		wantErr bool
	}{
		{
			name:    "sets all metrics",
			metrics: []models.Metrics{{ID: "a"}, {ID: "b"}},
		},
		{
			name:    "returns on first error",
			metrics: []models.Metrics{{ID: "bad"}, {ID: "ok"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			repo := NewDatabaseRepository(&fakeDBAdapter{
				checkConnectionFn: func(context.Context) error { return nil },
				openFn:            func(context.Context) error { return nil },
				closeFn:           func(context.Context) error { return nil },
				setDataFn: func(m models.Metrics) error {
					calls++
					if tt.wantErr && m.ID == "bad" {
						return errors.New("db error")
					}
					return nil
				},
				getAllFn:  func() []models.Metrics { return nil },
				getDataFn: func(string, string) (models.Metrics, error) { return models.Metrics{}, nil },
			})

			err := repo.SetAllData(tt.metrics)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, 1, calls)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, len(tt.metrics), calls)
		})
	}
}

func (f *fakeDBAdapter) SetMultipleDataViaTransaction(context.Context, []models.Metrics) error {
	return nil
}


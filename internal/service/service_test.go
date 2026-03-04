package service

import (
	"context"
	"errors"
	"os"
	"testing"

	dbconfig "github.com/Nakohartum/practicum-metrics/internal/config/db"
	fsconfig "github.com/Nakohartum/practicum-metrics/internal/config/filestorage"
	memconfig "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDBAdapter struct {
	setDataFn         func(models.Metrics) error
	checkConnectionFn func(context.Context) error
}

var _ dbconfig.DatabaseAdapter = (*fakeDBAdapter)(nil)

func (f *fakeDBAdapter) CheckConnection(ctx context.Context) error {
	if f.checkConnectionFn != nil {
		return f.checkConnectionFn(ctx)
	}
	return nil
}
func (f *fakeDBAdapter) Open(context.Context) error  { return nil }
func (f *fakeDBAdapter) Close(context.Context) error { return nil }
func (f *fakeDBAdapter) SetData(m models.Metrics) error {
	if f.setDataFn != nil {
		return f.setDataFn(m)
	}
	return nil
}
func (f *fakeDBAdapter) GetAll() []models.Metrics { return nil }
func (f *fakeDBAdapter) GetData(string, string) (models.Metrics, error) {
	return models.Metrics{}, nil
}
func (f *fakeDBAdapter) SetMultipleDataViaTransaction(context.Context, []models.Metrics) error {
	return nil
}

func TestGetData(t *testing.T) {
	tests := []struct {
		name       string
		metricType string
		key        string
		seed       func(*testing.T, *memconfig.MemStorage)
		wantErr    bool
	}{
		{
			name:       "returns error for empty key",
			metricType: models.Counter,
			key:        "",
			seed:       func(_ *testing.T, _ *memconfig.MemStorage) {},
			wantErr:    true,
		},
		{
			name:       "returns metric from repo",
			metricType: models.Counter,
			key:        "hits",
			seed: func(t *testing.T, ms *memconfig.MemStorage) {
				require.NoError(t, ms.SetData(models.Counter, "hits", "3"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memconfig.NewMemStubStorage()
			tt.seed(t, store)
			repo := repository.NewMemRepo(store)
			svc := NewMetricsService(repo, 1)

			metric, err := svc.GetData(tt.metricType, tt.key)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.key, metric.ID)
			assert.Equal(t, tt.metricType, metric.MType)
		})
	}
}

func TestSetDataUsingMetrics(t *testing.T) {
	tests := []struct {
		name      string
		input     []models.Metrics
		wantErr   bool
		wantPanic bool
	}{
		{
			name: "sets counter and gauge",
			input: []models.Metrics{
				{ID: "c1", MType: models.Counter, Delta: func() *int64 { v := int64(5); return &v }()},
				{ID: "g1", MType: models.Gauge, Value: func() *float64 { v := 1.5; return &v }()},
			},
		},
		{
			name:      "nil counter delta panics in current implementation",
			input:     []models.Metrics{{ID: "c2", MType: models.Counter, Delta: nil}},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memconfig.NewMemStubStorage()
			repo := repository.NewMemRepo(store)
			svc := NewMetricsService(repo, 1)

			if tt.wantPanic {
				assert.Panics(t, func() {
					_ = svc.SetDataUsingMetrics(tt.input)
				})
				return
			}

			err := svc.SetDataUsingMetrics(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSetData(t *testing.T) {
	tests := []struct {
		name       string
		metricType string
		value      string
		wantErr    bool
	}{
		{name: "counter is parsed and delegated", metricType: models.Counter, value: "7"},
		{name: "gauge is parsed and delegated", metricType: models.Gauge, value: "7.5"},
		{name: "invalid type returns error", metricType: "other", value: "1", wantErr: true},
		{name: "invalid counter value returns error", metricType: models.Counter, value: "bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			adapter := &fakeDBAdapter{
				setDataFn: func(metric models.Metrics) error {
					called = true
					return nil
				},
			}
			dbRepo := repository.NewDatabaseRepository(adapter)
			memRepo := repository.NewMemRepo(memconfig.NewMemStubStorage())
			svc := NewDatabaseService(dbRepo, memRepo, 1)

			err := svc.SetData(tt.metricType, "metric", tt.value)
			if tt.wantErr {
				require.Error(t, err)
				assert.False(t, called)
				return
			}
			require.NoError(t, err)
			assert.True(t, called)
		})
	}
}

func TestGetDataFileService(t *testing.T) {
	tests := []struct {
		name       string
		metricType string
		key        string
		seed       []models.Metrics
		wantErr    bool
	}{
		{
			name:       "returns metric from file",
			metricType: models.Gauge,
			key:        "cpu",
			seed: []models.Metrics{{
				ID:    "cpu",
				MType: models.Gauge,
				Value: func() *float64 { v := 0.7; return &v }(),
			}},
		},
		{
			name:       "returns error when metric missing",
			metricType: models.Gauge,
			key:        "missing",
			seed:       []models.Metrics{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "metrics-*.json")
			require.NoError(t, err)
			path := tmpFile.Name()
			require.NoError(t, tmpFile.Close())

			writer, err := fsconfig.NewFileWriter(path)
			require.NoError(t, err)
			reader, err := fsconfig.NewFileReader(path)
			require.NoError(t, err)
			manager := fsconfig.NewFileManager(reader, writer)

			fileRepo := repository.NewFileRepo(manager)
			memRepo := repository.NewMemRepo(memconfig.NewMemStubStorage())
			svc := NewFileService(fileRepo, memRepo, 1)
			require.NoError(t, svc.WriteData(tt.seed))

			metric, err := svc.GetData(tt.metricType, tt.key)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.key, metric.ID)
			assert.Equal(t, tt.metricType, metric.MType)
		})
	}
}

func TestPing(t *testing.T) {
	tests := []struct {
		name    string
		pingErr error
	}{
		{name: "returns nil when repo ping is ok"},
		{name: "returns ping error", pingErr: errors.New("ping fail")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &fakeDBAdapter{
				checkConnectionFn: func(context.Context) error {
					return tt.pingErr
				},
			}
			dbRepo := repository.NewDatabaseRepository(adapter)
			memRepo := repository.NewMemRepo(memconfig.NewMemStubStorage())
			svc := NewDatabaseService(dbRepo, memRepo, 1)

			err := svc.Ping(context.Background())
			if tt.pingErr != nil {
				require.Error(t, err)
				assert.EqualError(t, err, tt.pingErr.Error())
				return
			}
			require.NoError(t, err)
		})
	}
}

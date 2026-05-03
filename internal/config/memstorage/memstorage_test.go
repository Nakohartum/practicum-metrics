package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

func TestSetData(t *testing.T) {
	tests := []struct {
		name         string
		metricType   string
		key          string
		value        string
		startCounter int64
		wantCounter  int64
		wantGauge    float64
		checkCounter bool
		checkGauge   bool
		wantErr      bool
	}{
		{
			name:         "counter increments existing value",
			metricType:   models.Counter,
			key:          "requests_total",
			value:        "5",
			startCounter: 10,
			wantCounter:  15,
			checkCounter: true,
		},
		{
			name:       "gauge sets value",
			metricType: models.Gauge,
			key:        "cpu_usage",
			value:      "42.5",
			wantGauge:  42.5,
			checkGauge: true,
		},
		{
			name:       "counter invalid value returns error",
			metricType: models.Counter,
			key:        "requests_total",
			value:      "not-int",
			wantErr:    true,
		},
		{
			name:       "unknown metric type returns error",
			metricType: "unknown",
			key:        "x",
			value:      "1",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMemStubStorage()
			if tt.startCounter != 0 {
				ms.data.Counters[tt.key] = tt.startCounter
			}

			err := ms.SetData(tt.metricType, tt.key, tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.checkCounter {
				assert.Equal(t, tt.wantCounter, ms.data.Counters[tt.key])
			}
			if tt.checkGauge {
				assert.Equal(t, tt.wantGauge, ms.data.Gauges[tt.key])
			}
		})
	}
}

func TestGetData(t *testing.T) {
	tests := []struct {
		name       string
		metricType string
		key        string
		seed       func(*MemStorage)
		wantErr    error
		wantType   string
		wantID     string
		wantDelta  *int64
		wantValue  *float64
	}{
		{
			name:       "returns counter metric",
			metricType: models.Counter,
			key:        "hits",
			seed: func(ms *MemStorage) {
				ms.data.Counters["hits"] = 7
			},
			wantType:  models.Counter,
			wantID:    "hits",
			wantDelta: func() *int64 { v := int64(7); return &v }(),
		},
		{
			name:       "returns gauge metric",
			metricType: models.Gauge,
			key:        "load",
			seed: func(ms *MemStorage) {
				ms.data.Gauges["load"] = 1.25
			},
			wantType:  models.Gauge,
			wantID:    "load",
			wantValue: func() *float64 { v := 1.25; return &v }(),
		},
		{
			name:       "returns not exists error",
			metricType: models.Counter,
			key:        "missing",
			seed:       func(ms *MemStorage) {},
			wantErr:    ErrNotExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMemStubStorage()
			tt.seed(ms)

			got, err := ms.GetData(tt.metricType, tt.key)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantType, got.MType)
			assert.Equal(t, tt.wantID, got.ID)
			if tt.wantDelta != nil {
				require.NotNil(t, got.Delta)
				assert.Equal(t, *tt.wantDelta, *got.Delta)
			}
			if tt.wantValue != nil {
				require.NotNil(t, got.Value)
				assert.Equal(t, *tt.wantValue, *got.Value)
			}
		})
	}
}

func TestGetAll(t *testing.T) {
	tests := []struct {
		name          string
		seedCounters  map[string]int64
		seedGauges    map[string]float64
		wantLen       int
		wantByKeyType map[string]bool
	}{
		{
			name:         "returns all metrics from both maps",
			seedCounters: map[string]int64{"c1": 1, "c2": 2},
			seedGauges:   map[string]float64{"g1": 3.5},
			wantLen:      3,
			wantByKeyType: map[string]bool{
				"counter:c1": true,
				"counter:c2": true,
				"gauge:g1":   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMemStubStorage()
			for k, v := range tt.seedCounters {
				ms.data.Counters[k] = v
			}
			for k, v := range tt.seedGauges {
				ms.data.Gauges[k] = v
			}

			all := ms.GetAll()
			require.Len(t, all, tt.wantLen)

			actual := make(map[string]bool, len(all))
			for _, m := range all {
				actual[m.MType+":"+m.ID] = true
			}
			assert.Equal(t, tt.wantByKeyType, actual)
		})
	}
}

func TestPing(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "always returns nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := NewMemStubStorage()
			err := ms.Ping(context.Background())
			require.NoError(t, err)
		})
	}
}

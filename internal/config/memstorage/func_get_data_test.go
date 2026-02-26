package config

import (
	"testing"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)



func ptrInt64(v int64) *int64 { return &v }
func ptrFloat64(v float64) *float64 {return &v}



func TestGetData(t *testing.T) {
	tests := []struct{
		name string
		metricToFind models.Metrics
		errorNeeded bool
	}{
		{
			name: "positive test #1",
			metricToFind: models.Metrics{
				ID: "counter",
				MType: models.Counter,
				Delta: ptrInt64(5),
			},
			errorNeeded: false,
		},
		{
			name: "positive test #2",
			metricToFind: models.Metrics{
				ID: "gauge",
				MType: models.Gauge,
				Value: ptrFloat64(3.5),
			},
			errorNeeded: false,
		},
		{
			name: "metric does not exist (counter)",
			metricToFind: models.Metrics{
				ID: "counter1",
				MType: models.Counter,
				Delta: ptrInt64(5),
			},
			errorNeeded: true,
		},
		{
			name: "metric does not exist (gauge)",
			metricToFind: models.Metrics{
				ID: "gauge1",
				MType: models.Gauge,
				Value: ptrFloat64(3.5),
			},
			errorNeeded: true,
		},
	}

	memStorage := NewMemStubStorage()
	memStorage.data.Counters["counter"] = 5
	memStorage.data.Gauges["gauge"] = 3.5

	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T) {
			val, err := memStorage.GetData(tt.metricToFind.MType, tt.metricToFind.ID)
			if tt.errorNeeded {
				require.Error(t, err)
			} else{
				assert.Equal(t, tt.metricToFind, val)
				require.NoError(t, err)
			}
		})
	}
}
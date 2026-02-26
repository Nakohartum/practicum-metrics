package config

import (
	"testing"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/stretchr/testify/require"
)

func TestSetData(t *testing.T) {
	tests := []struct{
		name string
		metricType string
		metricKey string
		metricValue string
		wantError bool
	} {
		{
			name: "positive test #1",
			metricType: models.Counter,
			metricKey: "counter",
			metricValue: "5",
			wantError: false,
		},
		{
			name: "positive test #2",
			metricType: models.Gauge,
			metricKey: "gauge",
			metricValue: "3.4",
			wantError: false,
		},
		{
			name: "negative test #1",
			metricType: models.Counter,
			metricKey: "counter",
			metricValue: "f",
			wantError: true,
		},
		{
			name: "negative test #2",
			metricType: models.Gauge,
			metricKey: "gauge",
			metricValue: "f",
			wantError: true,
		},
		{
			name: "negative test #3",
			metricType: "ff",
			metricKey: "ff",
			metricValue: "f",
			wantError: true,
		},
	}

	memStorage := NewMemStubStorage()

	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T) {
			err := memStorage.SetData(tt.metricType, tt.metricKey, tt.metricValue)

			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
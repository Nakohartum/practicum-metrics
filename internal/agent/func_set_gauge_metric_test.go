package agent

import "testing"

func TestSetGaugeMetric(t *testing.T) {

	tests := []struct {
		name                string
		initialGaugeMetrics map[string]float64
		metricName          string
		value               float64
		expected            float64
		times               int
	}{
		{
			name: "Positive case #1",
			initialGaugeMetrics: map[string]float64{
				"testMetric": 2.4,
			},
			metricName: "testMetric",
			value:      3.5,
			expected:   3.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ma := NewAgentMetrics(0, 0)
			ma.gaugeMetrics = tt.initialGaugeMetrics
			ma.setGaugeMetric(tt.metricName, tt.value)
			if ma.gaugeMetrics[tt.metricName] != tt.expected {
				t.Errorf("Not correct value for key: %s. Value: %f. Expected: %f", tt.metricName, ma.gaugeMetrics[tt.metricName], tt.expected)
			}
		})
	}
}
package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAgentMetrics(t *testing.T) {
	tests := []struct {
		name           string
		pollInterval   int
		reportInterval int
		key            string
	}{
		{name: "creates agent", pollInterval: 1, reportInterval: 2, key: "secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgentMetrics(tt.pollInterval, tt.reportInterval, tt.key)

			require.NotNil(t, agent)
			assert.Equal(t, time.Second, agent.PollInterval)
			assert.Equal(t, 2*time.Second, agent.ReportInterval)
			assert.Equal(t, tt.key, agent.key)
			assert.NotNil(t, agent.client)
		})
	}
}

func TestMetricsAgentSetGaugeMetric(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value float64
	}{
		{name: "stores gauge metric", key: "Alloc", value: 10.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgentMetrics(1, 1, "")

			agent.setGaugeMetric(tt.key, tt.value)

			assert.Equal(t, tt.value, agent.gaugeMetrics[tt.key])
		})
	}
}

func TestMetricsAgentSetCounterMetrics(t *testing.T) {
	tests := []struct {
		name  string
		start int64
		want  int64
	}{
		{name: "increments poll count", start: 0, want: 1},
		{name: "increments existing poll count", start: 2, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgentMetrics(1, 1, "")
			agent.counterMetrics["PollCount"] = tt.start

			agent.setCounterMetrics()

			assert.Equal(t, tt.want, agent.counterMetrics["PollCount"])
		})
	}
}

func TestMetricsAgentSendGaugeMetrics(t *testing.T) {
	tests := []struct {
		name string
		seed map[string]float64
	}{
		{name: "marshals all gauge metrics", seed: map[string]float64{"Alloc": 1.5, "Sys": 2.5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgentMetrics(1, 1, "")
			agent.gaugeMetrics = tt.seed

			got := agent.sendGaugeMetrics()

			require.Len(t, got, len(tt.seed))
			for _, raw := range got {
				var metric models.Metrics
				require.NoError(t, json.Unmarshal(raw, &metric))
				assert.Equal(t, models.Gauge, metric.MType)
			}
		})
	}
}

func TestMetricsAgentSendCounterMetrics(t *testing.T) {
	tests := []struct {
		name string
		seed map[string]int64
	}{
		{name: "marshals all counter metrics", seed: map[string]int64{"PollCount": 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgentMetrics(1, 1, "")
			agent.counterMetrics = tt.seed

			got := agent.sendCounterMetrics()

			require.Len(t, got, len(tt.seed))
			for _, raw := range got {
				var metric models.Metrics
				require.NoError(t, json.Unmarshal(raw, &metric))
				assert.Equal(t, models.Counter, metric.MType)
			}
		})
	}
}

func TestMetricsAgentSendDataWithDeadline(t *testing.T) {
	tests := []struct {
		name             string
		key              string
		firstStatusCode  int
		secondStatusCode int
		wantErr          bool
		wantCalls        int
	}{
		{name: "sends gzip payload successfully", key: "secret", firstStatusCode: http.StatusOK, wantCalls: 1},
		{name: "falls back to plain payload", key: "secret", firstStatusCode: http.StatusInternalServerError, secondStatusCode: http.StatusOK, wantCalls: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				callCount++
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				if callCount == 1 {
					require.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
					reader, err := gzip.NewReader(bytes.NewReader(body))
					require.NoError(t, err)
					decoded, err := io.ReadAll(reader)
					require.NoError(t, err)
					require.NoError(t, reader.Close())
					assert.Equal(t, `{"id":"hits"}`, string(decoded))
					assert.NotEmpty(t, r.Header.Get("HashSHA256"))
					w.WriteHeader(tt.firstStatusCode)
					return
				}

				assert.Empty(t, r.Header.Get("Content-Encoding"))
				assert.Equal(t, `{"id":"hits"}`, string(body))
				assert.NotEmpty(t, r.Header.Get("HashSHA256"))
				w.WriteHeader(tt.secondStatusCode)
			}))
			defer server.Close()

			agent := NewAgentMetrics(1, 1, tt.key)
			err := agent.sendDataWithDeadline([]byte(`{"id":"hits"}`), server.URL, time.Second)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCalls, callCount)
		})
	}
}

func TestCompressData(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "compresses payload", body: `{"id":"hits"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := compressData([]byte(tt.body))

			require.NoError(t, err)
			reader, err := gzip.NewReader(bytes.NewReader(compressed))
			require.NoError(t, err)
			decoded, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.NoError(t, reader.Close())
			assert.Equal(t, tt.body, string(decoded))
		})
	}
}

func TestMakeHash(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		key  string
		want string
	}{
		{name: "returns empty hash without key", body: []byte("abc"), key: "", want: ""},
		{name: "returns stable hash", body: []byte("abc"), key: "key", want: "afcb12512ff218c850c8672f2f984f34d86f7739c5f4bee205442eb5a6ad6fff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, makeHash(tt.body, tt.key))
		})
	}
}

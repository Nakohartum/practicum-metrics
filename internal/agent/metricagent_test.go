package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Nakohartum/practicum-metrics/internal/cryptoutil"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
)

func TestNewAgentMetrics(t *testing.T) {
	agent := NewAgentMetrics(1, 2, 3, "secret", "")

	require.NotNil(t, agent)
	assert.Equal(t, time.Second, agent.PollInterval)
	assert.Equal(t, 2*time.Second, agent.ReportInterval)
	assert.Equal(t, 3, agent.rateLimit)
	assert.Equal(t, "secret", agent.key)
	assert.NotNil(t, agent.client)
	assert.Nil(t, agent.publicKey)
}

func TestCollectSnapshot(t *testing.T) {
	agent := NewAgentMetrics(1, 1, 1, "", "")

	got := agent.collectSnapshot()

	require.NotEmpty(t, got.Metrics)
	assert.Equal(t, 1, agent.pollCount)

	metricsByID := make(map[string]models.Metrics, len(got.Metrics))
	for _, metric := range got.Metrics {
		metricsByID[metric.ID] = metric
	}

	randomMetric, ok := metricsByID["RandomValue"]
	require.True(t, ok)
	assert.Equal(t, models.Gauge, randomMetric.MType)
	require.NotNil(t, randomMetric.Value)

	pollCountMetric, ok := metricsByID["PollCount"]
	require.True(t, ok)
	assert.Equal(t, models.Counter, pollCountMetric.MType)
	require.NotNil(t, pollCountMetric.Delta)
	assert.EqualValues(t, 1, *pollCountMetric.Delta)
}

func TestSendSnapshot(t *testing.T) {
	var got []models.Metrics
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		assert.Equal(t, "/updates", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		if r.Header.Get("Content-Encoding") == "gzip" {
			reader, err := gzip.NewReader(bytes.NewReader(body))
			require.NoError(t, err)
			body, err = io.ReadAll(reader)
			require.NoError(t, err)
			require.NoError(t, reader.Close())
		}

		require.NoError(t, json.Unmarshal(body, &got))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewAgentMetrics(1, 1, 1, "", "")
	snap := snapshot{
		Metrics: []models.Metrics{
			newGaugeMetric("Alloc", 1.5),
			newCounterMetric("PollCount", 2),
		},
	}

	agent.sendSnapshot(snap, server.URL)

	assert.Equal(t, 1, callCount)
	require.Len(t, got, 2)
	assert.Equal(t, "Alloc", got[0].ID)
	assert.Equal(t, models.Gauge, got[0].MType)
	assert.Equal(t, "PollCount", got[1].ID)
	assert.Equal(t, models.Counter, got[1].MType)
}

func TestRunSendsLastSnapshotAfterContextCancel(t *testing.T) {
	got := make(chan []models.Metrics, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		if r.Header.Get("Content-Encoding") == "gzip" {
			reader, err := gzip.NewReader(bytes.NewReader(body))
			require.NoError(t, err)
			body, err = io.ReadAll(reader)
			require.NoError(t, err)
			require.NoError(t, reader.Close())
		}

		var metrics []models.Metrics
		require.NoError(t, json.Unmarshal(body, &metrics))
		got <- metrics
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewAgentMetrics(1, 1, 1, "", "")
	agent.lastSnapshot = snapshot{
		Metrics: []models.Metrics{
			newGaugeMetric("Alloc", 1.5),
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	agent.Run(ctx, server.URL)

	select {
	case metrics := <-got:
		require.Len(t, metrics, 1)
		assert.Equal(t, "Alloc", metrics[0].ID)
	case <-time.After(time.Second):
		t.Fatal("agent did not send last snapshot before stopping")
	}
}

func TestSendDataWithDeadline(t *testing.T) {
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

			agent := NewAgentMetrics(1, 1, 1, tt.key, "")
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

func TestSendDataWithDeadlineEncrypted(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		body, err = cryptoutil.Decrypt(body, privateKey)
		require.NoError(t, err)

		require.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
		reader, err := gzip.NewReader(bytes.NewReader(body))
		require.NoError(t, err)
		decoded, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		assert.Equal(t, `{"id":"hits"}`, string(decoded))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewAgentMetrics(1, 1, 1, "", "")
	agent.publicKey = &privateKey.PublicKey

	err = agent.sendDataWithDeadline([]byte(`{"id":"hits"}`), server.URL, time.Second)
	require.NoError(t, err)
}

func TestCompressData(t *testing.T) {
	compressed, err := compressData([]byte(`{"id":"hits"}`))

	require.NoError(t, err)
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	decoded, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	assert.Equal(t, `{"id":"hits"}`, string(decoded))
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

package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	config "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMetricDataHandle(t *testing.T) {
	repo := repository.NewMemRepo(config.NewMemStubStorage())
	repo.SetData("counter", "cpu", "1")
	repo.SetData("gauge", "cpu", "1.5")
	ms := service.NewMetricsService(repo, 1)
	mh := NewMetricsHandler(ms)

	r := chi.NewRouter()

	r.Get("/value/{metricType}/{metricName}", func(w http.ResponseWriter, r *http.Request) { mh.GetMetricDataHandle().ServeHTTP(w, r) })

	server := httptest.NewServer(r)

	tests := []struct {
		name       string
		path       string
		statusCode int
		body       string
		method     string
	}{
		{
			name:       "positive test #1",
			path:       "/value/counter/cpu",
			statusCode: http.StatusOK,
			body:       "1",
			method:     http.MethodGet,
		},
		{
			name:       "positive test #2",
			path:       "/value/gauge/cpu",
			statusCode: http.StatusOK,
			body:       "1.5",
			method:     http.MethodGet,
		},
		{
			name:       "method not allowed",
			path:       "/value/counter/cpu",
			statusCode: http.StatusMethodNotAllowed,
			body:       "",
			method:     http.MethodPost,
		},
		{
			name:       "no metric found #1",
			path:       "/value//",
			statusCode: http.StatusNotFound,
			body:       "",
			method:     http.MethodGet,
		},
		{
			name:       "no metric found #2",
			path:       "/value/gauge/video",
			statusCode: http.StatusNotFound,
			body:       "",
			method:     http.MethodGet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(tt.method, server.URL+tt.path, http.NoBody)

			require.NoError(t, err)
			defer request.Body.Close()

			resp, err := server.Client().Do(request)
			require.NoError(t, err)

			defer resp.Body.Close()

			res, err := io.ReadAll(resp.Body)

			require.NoError(t, err)

			if tt.body != "" {
				assert.Equal(t, tt.body, string(res))
			}

			require.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}
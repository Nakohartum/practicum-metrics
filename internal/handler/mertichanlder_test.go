package handler

import (
	"fmt"
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

func TestServeHTTP(t *testing.T) {

	repo := repository.NewMemRepo(config.NewMemStubStorage())
	ms := service.NewMetricsService(repo)
	mh := NewMetricsHandler(ms)

	r := chi.NewRouter()
	r.Post("/update/{metricType}/{metricName}/{metricValue}", mh.ServeHTTP)

	server := httptest.NewServer(r)

	tests := []struct {
		name string // description of this test case
		statusCode int
		path string
		method string
	}{
		{
			name: "positive test #1",
			statusCode: http.StatusOK,
			path: "/update/gauge/cpu/0.75",
			method: http.MethodPost,
		},
		{
			name: "positive test #2",
			statusCode: http.StatusOK,
			path: "/update/counter/cpu/1",
			method: http.MethodPost,
		},
		{
			name: "method not allowed test",
			statusCode: http.StatusMethodNotAllowed,
			path: "/update/gauge/cpu/0.75",
			method: http.MethodGet,
		},
		{
			name: "status not found test",
			statusCode: http.StatusNotFound,
			path: "/update/counter//1",
			method: http.MethodPost,
		},
		{
			name: "status not correct value test",
			statusCode: http.StatusBadRequest,
			path: "/update/counter/cpu/1.5",
			method: http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmt.Println("URL "+server.URL+tt.path)
			request, err := http.NewRequest(tt.method, server.URL+tt.path, http.NoBody)
			require.NoError(t, err)
			defer request.Body.Close()

			resp, err := server.Client().Do(request)
			require.NoError(t, err)
			defer resp.Body.Close()

			_, errFromResp := io.ReadAll(resp.Body)
			require.NoError(t, errFromResp)


			assert.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}

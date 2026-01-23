package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	config "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestServeHTTP(t *testing.T) {
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
			path: "/",
			method: http.MethodGet,
		},
		{
			name: "status not found test",
			statusCode: http.StatusNotFound,
			path: "/update/counter//1",
			method: http.MethodPost,
		},
		{
			name: "status not found test",
			statusCode: http.StatusBadRequest,
			path: "/update/counter/cpu/1.5",
			method: http.MethodPost,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMemRepo(config.NewMemStubStorage())
			ms := service.NewMetricsService(repo)
			mh := NewMetricsHandler(ms)
			
			request := httptest.NewRequest(tt.method, tt.path, nil)

			w := httptest.NewRecorder()

			mh.ServeHTTP(w, request)
			

			res := w.Result()

			assert.Equal(t, tt.statusCode, res.StatusCode)
		})
	}
}

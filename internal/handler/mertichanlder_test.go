package handler

import (
	"bytes"
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

func TestSetMetricDataHandle(t *testing.T) {

	repo := repository.NewMemRepo(config.NewMemStubStorage())
	ms := service.NewMetricsService(repo)
	mh := NewMetricsHandler(ms)

	r := chi.NewRouter()
	r.Post("/update/{metricType}/{metricName}/{metricValue}", func(w http.ResponseWriter, r *http.Request) {mh.SetMetricDataHandle().ServeHTTP(w, r)})

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

func TestGetMetricDataHandle(t *testing.T){
	repo := repository.NewMemRepo(config.NewMemStubStorage())
	repo.SetData("counter", "cpu", "1")
	repo.SetData("gauge", "cpu", "1.5")
	ms := service.NewMetricsService(repo)
	mh := NewMetricsHandler(ms)

	r := chi.NewRouter()

	r.Get("/value/{metricType}/{metricName}", func(w http.ResponseWriter, r *http.Request) {mh.GetMetricDataHandle().ServeHTTP(w,r)})

	server := httptest.NewServer(r)

	tests := []struct{
		name string
		path string
		statusCode int
		body string
		method string
	}{
		{
			name: "positive test #1",
			path: "/value/counter/cpu",
			statusCode: http.StatusOK,
			body: "1",
			method: http.MethodGet,
		},
		{
			name: "positive test #2",
			path: "/value/gauge/cpu",
			statusCode: http.StatusOK,
			body: "1.5",
			method: http.MethodGet,
		},
		{
			name: "method not allowed",
			path: "/value/counter/cpu",
			statusCode: http.StatusMethodNotAllowed,
			body: "",
			method: http.MethodPost,
		},
		{
			name: "no metric found #1",
			path: "/value//",
			statusCode: http.StatusNotFound,
			body: "",
			method: http.MethodGet,
		},
		{
			name: "no metric found #2",
			path: "/value/gauge/video",
			statusCode: http.StatusNotFound,
			body: "",
			method: http.MethodGet,
		},
	}

	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(tt.method, server.URL + tt.path, http.NoBody)

			require.NoError(t, err)
			defer request.Body.Close()

			resp, err := server.Client().Do(request)
			require.NoError(t, err)

			defer resp.Body.Close()

			res, err := io.ReadAll(resp.Body)

			require.NoError(t, err)

			if tt.body != ""{
				assert.Equal(t, tt.body, string(res))
			}

			require.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}


func TestServePage(t *testing.T){
	repo := repository.NewMemRepo(config.NewMemStubStorage())
	ms := service.NewMetricsService(repo)
	mh := NewMetricsHandler(ms)

	r := chi.NewRouter()

	r.Get("/", mh.ServePage)

	server := httptest.NewServer(r)

	tests := []struct{
		name string
		statusCode int
		contentType string
		method string
	}{
		{
			name: "positive test #1",
			statusCode: http.StatusOK,
			contentType: "text/html; charset=utf-8",
			method: http.MethodGet,
		},
		{
			name: "method not allowed",
			statusCode: http.StatusMethodNotAllowed,
			contentType: "",
			method: http.MethodPost,
		},
	}

	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(tt.method, server.URL, http.NoBody)

			require.NoError(t, err)
			defer request.Body.Close()

			resp, err := server.Client().Do(request)
			require.NoError(t, err)

			defer resp.Body.Close()

			if tt.contentType != ""{
				require.Equal(t, tt.contentType, resp.Header.Get("Content-Type"))
			}

			require.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}

func TestUpdateMetricsDataHandler(t *testing.T){
	repo := repository.NewMemRepo(config.NewMemStubStorage())
	ms := service.NewMetricsService(repo)
	mh := NewMetricsHandler(ms)

	r := chi.NewRouter()

	r.Post("/update", func(w http.ResponseWriter, r *http.Request) {mh.UpdateMetricsDataHandle().ServeHTTP(w, r)})

	server := httptest.NewServer(r)

	tests := []struct{
		name string
		statusCode int
		body string
	} {
		{
			name: "positive test #1",
			statusCode: http.StatusCreated,
			body: `{"id":"cpu","type":"gauge","value":0.75}`,
		},
		{
			name: "positive test #2",
			statusCode: http.StatusCreated,
			body: `{"id":"cpu","type":"counter","delta":1}`,
		},
		{
			name: "no metric's name",
			statusCode: http.StatusBadRequest,
			body: `{"type":"gauge","value":0.75}`,
		},
		{
			name: "no metric's type",
			statusCode: http.StatusBadRequest,
			body: `{"id":"cpu","value":0.75}`,
		},
	}

	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodPost, server.URL + "/update", bytes.NewBufferString(tt.body))
			require.NoError(t, err)
			defer request.Body.Close()

			resp, err := server.Client().Do(request)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}

func TestGetMetricsByName(t *testing.T){
	repo := repository.NewMemRepo(config.NewMemStubStorage())
	ms := service.NewMetricsService(repo)
	mh := NewMetricsHandler(ms)
	mh.service.SetData("gauge", "cpu", "1.1")
	mh.service.SetData("counter", "cpu", "1")
	r := chi.NewRouter()
	r.Post("/value", func(w http.ResponseWriter, r *http.Request) {mh.GetMetricsByNameHandle().ServeHTTP(w, r)})
	server := httptest.NewServer(r)

	tests := []struct{
		name string
		statusCode int
		body string
	}{
		{
			name: "positive test #1",
			statusCode: http.StatusOK,
			body: `{"id":"cpu","type":"gauge","value":1.1}`,
		},
	}

	for _, tt := range tests{
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodPost, server.URL + "/value", bytes.NewBufferString(tt.body))
			require.NoError(t, err)
			defer request.Body.Close()

			resp, err := server.Client().Do(request)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}
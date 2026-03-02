package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	config "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestUpdateMetricsDataHandler(t *testing.T){
	repo := repository.NewMemRepo(config.NewMemStubStorage())
	ms := service.NewMetricsService(repo, 1)
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
			statusCode: http.StatusOK,
			body: `{"id":"cpu","type":"gauge","value":0.75}`,
		},
		{
			name: "positive test #2",
			statusCode: http.StatusOK,
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
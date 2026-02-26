package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	config "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	"github.com/Nakohartum/practicum-metrics/internal/config/mocks"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestGetMetricsByName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileReader, err := config.NewFileReader("test.json")
	require.NoError(t, err)
	fileWriter, err := config.NewFileWriter("test.json")
	require.NoError(t, err)
	fileWorker := config.NewFileManager(fileReader, fileWriter)
	repo := repository.NewMemRepo(config.NewMemStubStorage(), fileWorker)
	dbAdapter := mocks.NewMockDatabaseAdapter(ctrl)
	databaseRepo := repository.NewDatabaseRepository(dbAdapter)
	ms := service.NewMetricsService(repo, databaseRepo)
	mh := NewMetricsHandler(ms)
	mh.service.SetData("gauge", "cpu", "1.1")
	mh.service.SetData("counter", "cpu", "1")
	r := chi.NewRouter()
	r.Post("/value", func(w http.ResponseWriter, r *http.Request) { mh.GetMetricsByNameHandle().ServeHTTP(w, r) })
	server := httptest.NewServer(r)

	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{
			name:       "positive test #1",
			statusCode: http.StatusOK,
			body:       `{"id":"cpu","type":"gauge","value":1.1}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodPost, server.URL+"/value", bytes.NewBufferString(tt.body))
			require.NoError(t, err)
			defer request.Body.Close()

			resp, err := server.Client().Do(request)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}
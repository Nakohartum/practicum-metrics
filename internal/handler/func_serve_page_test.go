package handler

import (
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

func TestServePage(t *testing.T) {
	ctrl := gomock.NewController(t)
	fileReader, err := config.NewFileReader("test.json")
	require.NoError(t, err)
	fileWriter, err := config.NewFileWriter("test.json")
	require.NoError(t, err)
	fileWorker := config.NewFileManager(fileReader, fileWriter)
	repo := repository.NewMemRepo(config.NewMemStubStorage(), fileWorker)
	dbAdapter := mocks.NewMockDatabaseAdapter(ctrl)
	dbRepo := repository.NewDatabaseRepository(dbAdapter)
	ms := service.NewMetricsService(repo, dbRepo)
	mh := NewMetricsHandler(ms)

	r := chi.NewRouter()

	r.Get("/", mh.ServePage)

	server := httptest.NewServer(r)

	tests := []struct {
		name        string
		statusCode  int
		contentType string
		method      string
	}{
		{
			name:        "positive test #1",
			statusCode:  http.StatusOK,
			contentType: "text/html; charset=utf-8",
			method:      http.MethodGet,
		},
		{
			name:        "method not allowed",
			statusCode:  http.StatusMethodNotAllowed,
			contentType: "",
			method:      http.MethodPost,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(tt.method, server.URL, http.NoBody)

			require.NoError(t, err)
			defer request.Body.Close()

			resp, err := server.Client().Do(request)
			require.NoError(t, err)

			defer resp.Body.Close()

			if tt.contentType != "" {
				require.Equal(t, tt.contentType, resp.Header.Get("Content-Type"))
			}

			require.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}
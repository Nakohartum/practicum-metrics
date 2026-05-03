package handler_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"

	memconfig "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	"github.com/Nakohartum/practicum-metrics/internal/handler"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
)

func exampleRouter() http.Handler {
	storage := memconfig.NewMemStorage(models.NewStorageModel())
	repo := repository.NewMemRepo(storage)
	svc := service.NewMetricsService(repo, 0)
	metricsHandler := handler.NewMetricsHandler(svc)

	router := chi.NewRouter()
	router.Get("/ping", metricsHandler.Ping().ServeHTTP)
	router.Post("/update", metricsHandler.UpdateMetricsDataHandle().ServeHTTP)
	router.Post("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.SetMetricDataHandle().ServeHTTP)
	router.Post("/updates", metricsHandler.SetMetricsDataHandle().ServeHTTP)
	router.Get("/value/{metricType}/{metricName}", metricsHandler.GetMetricDataHandle().ServeHTTP)
	router.Post("/value", metricsHandler.GetMetricsByNameHandle().ServeHTTP)

	return router
}

func performRequest(router http.Handler, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func ExampleMetricsHandler_SetMetricDataHandle() {
	router := exampleRouter()

	update := performRequest(router, http.MethodPost, "/update/counter/hits/3", "")
	value := performRequest(router, http.MethodGet, "/value/counter/hits", "")

	fmt.Println(update.Code)
	fmt.Println(value.Code, value.Body.String())

	// Output:
	// 200
	// 200 3
}

func ExampleMetricsHandler_UpdateMetricsDataHandle() {
	router := exampleRouter()

	updateBody := `{"id":"temperature","type":"gauge","value":23.5}`
	update := performRequest(router, http.MethodPost, "/update", updateBody)

	valueBody := `{"id":"temperature","type":"gauge"}`
	value := performRequest(router, http.MethodPost, "/value", valueBody)

	fmt.Println(update.Code)
	fmt.Println(value.Code, value.Body.String())

	// Output:
	// 200
	// 200 {"id":"temperature","type":"gauge","value":23.5}
}

func ExampleMetricsHandler_SetMetricsDataHandle() {
	router := exampleRouter()

	batch := `[
		{"id":"hits","type":"counter","delta":2},
		{"id":"hits","type":"counter","delta":3},
		{"id":"load","type":"gauge","value":1.25}
	]`
	update := performRequest(router, http.MethodPost, "/updates", batch)
	hits := performRequest(router, http.MethodGet, "/value/counter/hits", "")
	load := performRequest(router, http.MethodGet, "/value/gauge/load", "")

	fmt.Println(update.Code)
	fmt.Println(hits.Body.String())
	fmt.Println(load.Body.String())

	// Output:
	// 200
	// 5
	// 1.25
}

func ExampleMetricsHandler_Ping() {
	router := exampleRouter()

	ping := performRequest(router, http.MethodGet, "/ping", "")

	fmt.Println(ping.Code)

	// Output:
	// 200
}

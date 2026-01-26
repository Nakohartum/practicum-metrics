package main

import (
	"net/http"

	config "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	"github.com/Nakohartum/practicum-metrics/internal/handler"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

func main() {
	router := chi.NewRouter()

	model := models.NewStorageModel()
	conf := config.NewMemStorage(model)
	repo := repository.NewMemRepo(conf)
	metricsService := service.NewMetricsService(repo)
	metricsHandler := handler.NewMetricsHandler(metricsService)
	router.Post("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.ServeHTTP)

	http.ListenAndServe(":8080", router)
}

package main

import (
	"net/http"

	config "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	"github.com/Nakohartum/practicum-metrics/internal/handler"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
)

func main() {

	mux := http.NewServeMux()
	model := models.NewStorageModel()
	conf := config.NewMemStorage(*model)
	repo := repository.NewMemRepo(*conf)
	metricsService := service.NewMetricsService(*repo)
	metricsHandler := handler.NewMetricsHandler(metricsService)
	mux.Handle("/update/", metricsHandler)

	http.ListenAndServe(":8080", mux)
}

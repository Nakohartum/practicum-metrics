package main

import (
	"net/http"

	config "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	"github.com/Nakohartum/practicum-metrics/internal/handler"
	"github.com/Nakohartum/practicum-metrics/internal/logger"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

func main() {
	parseFlags()
	router := chi.NewRouter()

	model := models.NewStorageModel()
	conf := config.NewMemStorage(model)
	repo := repository.NewMemRepo(conf)
	metricsService := service.NewMetricsService(repo)
	metricsHandler := handler.NewMetricsHandler(metricsService)

	router.Post("/update/{metricType}/{metricName}/{metricValue}", logger.AttachLoggingToResponse(metricsHandler.SetMetricDataHandle()))
	router.Route("/", func(r chi.Router) {
		r.Get("/", metricsHandler.ServePage)
		r.Route("/value", func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", metricsHandler.GetMetricDataHandle)
		})
	})

	http.ListenAndServe(address.String(), router)
}

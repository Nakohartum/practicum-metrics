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

	r := chi.NewRouter()

	model := models.NewStorageModel()
	conf := config.NewMemStorage(model)
	repo := repository.NewMemRepo(conf)
	service := service.NewMetricsService(repo)
	h := handler.NewMetricsHandler(service)

	// главная страница
	r.Get("/", h.ServePage)

	// JSON update (iteration7)
	r.Post("/update",  logger.AttachLoggingToResponse(h.UpdateMetricsDataHandle()))
	r.Post("/update/", logger.AttachLoggingToResponse(h.UpdateMetricsDataHandle()))

	// старый формат: path-параметры
	r.Post("/update/{metricType}/{metricName}/{metricValue}",
		logger.AttachLoggingToResponse(h.SetMetricDataHandle()))
	r.Post("/update/{metricType}/{metricName}/{metricValue}/",
		logger.AttachLoggingToResponse(h.SetMetricDataHandle()))

	// JSON получение значения
	r.Post("/value",  logger.AttachLoggingToResponse(h.GetMetricsByNameHandle()))
	r.Post("/value/", logger.AttachLoggingToResponse(h.GetMetricsByNameHandle()))

	// старый формат получения
	r.Get("/value/{metricType}/{metricName}",
		logger.AttachLoggingToResponse(h.GetMetricDataHandle()))
	r.Get("/value/{metricType}/{metricName}/",
		logger.AttachLoggingToResponse(h.GetMetricDataHandle()))

	http.ListenAndServe(address.String(), r)
}


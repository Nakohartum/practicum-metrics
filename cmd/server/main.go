package main

import (
	"log"
	"net/http"
	"strconv"

	config "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	"github.com/Nakohartum/practicum-metrics/internal/handler"
	"github.com/Nakohartum/practicum-metrics/internal/logger"
	models "github.com/Nakohartum/practicum-metrics/internal/model"
	"github.com/Nakohartum/practicum-metrics/internal/repository"
	"github.com/Nakohartum/practicum-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	parseFlags()
	router := chi.NewRouter()

	model := models.NewStorageModel()
	conf := config.NewMemStorage(model)
	reader, err := config.NewFileReader(fileWork.fileStoragePath)	
	if err != nil {
		log.Printf("can't open file for restore: %v", err)
	}
	writer, err := config.NewFileWriter(fileWork.fileStoragePath)
	if err != nil {
		log.Printf("cont open file for writing: %v", err)
	}
	fileManager := config.NewFileManager(reader, writer)
	repo := repository.NewMemRepo(conf, fileManager)
	fileService := service.NewFileService(repo, int(fileWork.storeInterval))
	metricsService := service.NewMetricsService(repo)
	metricsHandler := handler.NewMetricsHandler(metricsService)

	if fileWork.restore {
		
		res, err := fileService.ReadData()
		if err != nil {
			log.Printf("can't read data from file: %v", err)
		}
		for _, metric := range res {
			switch metric.MType{
				case models.Gauge:
					metricsService.SetData(metric.MType, metric.ID, strconv.FormatFloat(*metric.Value, 'f', -1, 64))
				case models.Counter:
					metricsService.SetData(metric.MType, metric.ID, strconv.FormatInt(*metric.Delta, 10))
			}
		}
	}

	router.Use(middleware.StripSlashes)
	router.Use(handler.GetZippedDataMiddleware)
	router.Use(handler.GiveZippedDataMiddleware)
	
	router.Route("/", func(r chi.Router) {
		r.Get("/", metricsHandler.ServePage)
		r.Post("/update", logger.AttachLoggingToResponse(metricsHandler.UpdateMetricsDataHandle()))
		r.Post("/update/{metricType}/{metricName}/{metricValue}", logger.AttachLoggingToResponse(metricsHandler.SetMetricDataHandle()))
		r.Route("/value", func(r chi.Router) {
			r.Post("/", logger.AttachLoggingToResponse(metricsHandler.GetMetricsByNameHandle()))
			r.Get("/{metricType}/{metricName}", logger.AttachLoggingToResponse(metricsHandler.GetMetricDataHandle()))
		})
	})
	go fileService.RunSaving()
	http.ListenAndServe(address.String(), router)
}

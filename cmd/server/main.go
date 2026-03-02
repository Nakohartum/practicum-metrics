package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	mConfig "github.com/Nakohartum/practicum-metrics/internal/config/memstorage"
	dbConfig "github.com/Nakohartum/practicum-metrics/internal/config/db"
	fConfig "github.com/Nakohartum/practicum-metrics/internal/config/filestorage"
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
	memRepo := setupMemRepo()
	var service service.Service = setupMemService(memRepo)

	if configData.FileWork.fileStoragePath != ""{
		service = setupFileService(memRepo)
	}

	if configData.DatabaseAddress.connectionString != ""{
		service = setupDatabaseService(memRepo)
	}

	server := setupServer(service)
	go func() {
		log.Println("Server started")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", configData.Address.String(), err)
		}
	}()
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()
	
	err := service.SaveDataAfterExit(ctx)

	if err != nil {
		log.Println(err)
	}

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server shut down")
}

func setupMemRepo() *repository.MemRepo {
	model := models.NewStorageModel()
	config := mConfig.NewMemStorage(model)
	repo := repository.NewMemRepo(config)
	return repo
}

func setupMemService(repo *repository.MemRepo) *service.MetricsService {
	service := service.NewMetricsService(repo, int(configData.FileWork.storeInterval))
	return service
}

func setupDatabaseService(memRepo *repository.MemRepo) *service.DatabaseService {
	dbAdapter := dbConfig.NewPgDatabaseAdapter(configData.DatabaseAddress.connectionString)
	dbAdapter.Open(context.Background())
	repo := repository.NewDatabaseRepository(dbAdapter)
	dbService := service.NewDatabaseService(repo, memRepo, int(configData.FileWork.storeInterval))
	return dbService
}

func setupFileService(memRepo *repository.MemRepo) *service.FileService{
	fWriter, err := fConfig.NewFileWriter(configData.FileWork.fileStoragePath)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	fReader, err := fConfig.NewFileReader(configData.FileWork.fileStoragePath)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	fWorker := fConfig.NewFileManager(fReader, fWriter)
	fRepo := repository.NewFileRepo(fWorker)
	fService := service.NewFileService(fRepo, memRepo, int(configData.FileWork.storeInterval))
	return fService
}


func setupRouter(service service.Service) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.StripSlashes)
	router.Use(handler.GetZippedDataMiddleware)
	router.Use(handler.GiveZippedDataMiddleware)
	if configData.FileWork.storeInterval == 0 {
		router.Use(handler.SaveAfterPostMiddleware(service))
	}
	return router
}

func setupServer(service service.Service) http.Server {
	router := setupRouter(service)
	
	metricsHandler := handler.NewMetricsHandler(service)

	if configData.FileWork.restore {
		
		res := service.GetAll()
		if len(res) == 0 {
			log.Printf("no data")
		}
		for _, metric := range res {
			switch metric.MType{
				case models.Gauge:
					service.SetData(metric.MType, metric.ID, strconv.FormatFloat(*metric.Value, 'f', -1, 64))
				case models.Counter:
					service.SetData(metric.MType, metric.ID, strconv.FormatInt(*metric.Delta, 10))
			}
		}
	}

	router.Route("/", func(r chi.Router) {
		r.Get("/", metricsHandler.ServePage)
		r.Get("/ping", metricsHandler.Ping(context.Background()).ServeHTTP)
		r.Post("/update", logger.AttachLoggingToResponse(metricsHandler.UpdateMetricsDataHandle()))
		r.Post("/update/{metricType}/{metricName}/{metricValue}", logger.AttachLoggingToResponse(metricsHandler.SetMetricDataHandle()))
		r.Route("/value", func(r chi.Router) {
			r.Post("/", logger.AttachLoggingToResponse(metricsHandler.GetMetricsByNameHandle()))
			r.Get("/{metricType}/{metricName}", logger.AttachLoggingToResponse(metricsHandler.GetMetricDataHandle()))
		})
	})

	if configData.FileWork.storeInterval != 0 {
		go service.RunSaving(context.Background())
	}

	return http.Server{
		Addr:    configData.Address.String(),
		Handler: router,
	}
}